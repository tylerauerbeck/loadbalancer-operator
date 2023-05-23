// Package srv provides server connectivity for loadbalanceroperator
package srv

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"reflect"

	"go.infratographer.com/x/gidx"
	"golang.org/x/exp/slices"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/getter"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	applyv1 "k8s.io/client-go/applyconfigurations/core/v1"
	applymetav1 "k8s.io/client-go/applyconfigurations/meta/v1"
	rbacapplyv1 "k8s.io/client-go/applyconfigurations/rbac/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/helm/pkg/strvals"
)

const (
	nameLength           = 53
	managedHelmKeyPrefix = "operator.managed"
)

func (s *Server) removeNamespace(ns string) error {
	s.Logger.Debugw("removing namespace", "namespace", ns)
	kc, err := kubernetes.NewForConfig(s.KubeClient)

	if err != nil {
		s.Logger.Errorw("unable to authenticate against kubernetes cluster", "error", err)
		return err
	}

	err = kc.CoreV1().Namespaces().Delete(s.Context, ns, metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	return nil
}

// CreateNamespace creates namespaces for the specified group that is
// provided in the event received
func (s *Server) CreateNamespace(hash string, encoded string) (*v1.Namespace, error) {
	s.Logger.Debugf("ensuring namespace %s exists", hash)

	if !checkNameLength(hash) || !checkNameLength(encoded) {
		return nil, errInvalidObjectNameLength
	}

	kc, err := kubernetes.NewForConfig(s.KubeClient)

	if err != nil {
		s.Logger.Errorw("unable to authenticate against kubernetes cluster", "error", err)
		return nil, err
	}

	apSpec := applyv1.NamespaceApplyConfiguration{
		TypeMetaApplyConfiguration: applymetav1.TypeMetaApplyConfiguration{
			Kind:       strPt("Namespace"),
			APIVersion: strPt("v1"),
		},
		ObjectMetaApplyConfiguration: &applymetav1.ObjectMetaApplyConfiguration{
			Name: &hash,
			Labels: map[string]string{"com.infratographer.lb-operator/managed": "true",
				"com.infratographer.lb-operator/lb-id": encoded},
		},
		Spec:   &applyv1.NamespaceSpecApplyConfiguration{},
		Status: &applyv1.NamespaceStatusApplyConfiguration{},
	}
	ns, err := kc.CoreV1().Namespaces().Apply(s.Context, &apSpec, metav1.ApplyOptions{FieldManager: "loadbalanceroperator"})

	if err != nil {
		s.Logger.Errorw("unable to create namespace", "error", err)
		return nil, err
	}

	if err := attachRoleBinding(s.Context, kc, hash); err != nil {
		s.Logger.Errorw("unable to attach namespace manager rolebinding to namespace", "error", err)
		return nil, err
	}

	return ns, nil
}

func attachRoleBinding(ctx context.Context, client *kubernetes.Clientset, namespace string) error {
	apSpec := rbacapplyv1.RoleBindingApplyConfiguration{
		ObjectMetaApplyConfiguration: &applymetav1.ObjectMetaApplyConfiguration{
			Name: strPt("load-balancer-operator-admin"),
		},
		TypeMetaApplyConfiguration: applymetav1.TypeMetaApplyConfiguration{
			Kind:       strPt("RoleBinding"),
			APIVersion: strPt("rbac.authorization.k8s.io/v1"),
		},
		RoleRef: &rbacapplyv1.RoleRefApplyConfiguration{Kind: strPt("ClusterRole"), Name: strPt("cluster-admin")},
		Subjects: []rbacapplyv1.SubjectApplyConfiguration{
			{
				Kind:      strPt("ServiceAccount"),
				Name:      strPt("load-balancer-operator"),
				Namespace: &namespace,
			},
		},
	}

	_, err := client.RbacV1().RoleBindings(namespace).Apply(ctx, &apSpec, metav1.ApplyOptions{FieldManager: "loadbalanceroperator"})

	if err != nil {
		return err
	}

	return nil
}

func (s *Server) newHelmValues(lb *loadBalancer) (map[string]interface{}, error) {
	provider := getter.All(&cli.EnvSettings{})

	valOpts := &values.Options{
		ValueFiles: []string{s.ValuesPath},
	}

	values, err := valOpts.MergeValues(provider)
	if err != nil {
		s.Logger.Errorw("unable to load values data", "error", err)
		return nil, err
	}

	additionalValues := generateLBHelmValues(lb)

	for _, override := range additionalValues {
		if err := strvals.ParseInto(override, values); err != nil {
			s.Logger.Errorw("unable to parse values", "error", err)
			return nil, err
		}
	}

	return values, nil
}

func (s *Server) removeDeployment(name string) error {
	n := hashName(name)

	releaseName := fmt.Sprintf("lb-%s", n)
	if len(releaseName) > nameLength {
		releaseName = releaseName[0:nameLength]
	}

	client, err := s.newHelmClient(n)
	if err != nil {
		s.Logger.Errorln("unable to initialize helm client: %s", err)
		return err
	}

	hc := action.NewUninstall(client)
	_, err = hc.Run(releaseName)

	if err != nil {
		s.Logger.Errorw("unable to remove deployment", "error", err)
		return err
	}

	s.Logger.Infof("%s removed successfully", releaseName)

	err = s.removeNamespace(n)
	if err != nil {
		s.Logger.Errorw("unable to remove namespace", "error", err)
		return err
	}

	return nil
}

// newDeployment deploys a loadBalancer based upon the configuration provided
// from the event that is processed.
func (s *Server) newDeployment(lb *loadBalancer) error {
	n := hashName(lb.loadBalancerID.String())
	enc := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString([]byte(lb.loadBalancerID.String()))

	if _, err := s.CreateNamespace(n, enc); err != nil {
		s.Logger.Errorw("unable to create namespace", "error", err)
		return err
	}

	releaseName := fmt.Sprintf("lb-%s", n)
	if len(releaseName) > nameLength {
		releaseName = releaseName[0:nameLength]
	}

	values, err := s.newHelmValues(lb)
	if err != nil {
		s.Logger.Errorw("unable to prepare chart values", "error", err)
		return err
	}

	client, err := s.newHelmClient(n)
	if err != nil {
		s.Logger.Errorln("unable to initialize helm client: %s", err)
		return err
	}

	hc := action.NewInstall(client)
	hc.ReleaseName = releaseName
	hc.Namespace = n
	_, err = hc.Run(s.Chart, values)

	if err != nil {
		s.Logger.Errorf("unable to deploy %s to %s", releaseName, n)
		return err
	}

	s.Logger.Infof("%s deployed to %s successfully", releaseName, n)

	return nil
}

func (s *Server) newHelmClient(namespace string) (*action.Configuration, error) {
	config := &action.Configuration{}
	cliopt := genericclioptions.NewConfigFlags(false)
	wrapper := func(*rest.Config) *rest.Config {
		return s.KubeClient
	}
	cliopt.WithWrapConfigFn(wrapper)

	err := config.Init(cliopt, namespace, "secret", func(format string, v ...interface{}) {
		// fmt.Println(v)

	})
	if err != nil {
		s.Logger.Errorw("unable to initialize helm client", "error", err)
		return nil, err
	}

	return config, nil
}

func strPt(s string) *string {
	return &s
}

func checkNameLength(name string) bool {
	return len(name) <= nameLength && len(name) > 0
}

func hashName(name string) string {
	n := md5.Sum([]byte(name))
	return hex.EncodeToString(n[:])
}

func generateLBHelmValues(lb *loadBalancer) []string {
	var vals []string

	v := reflect.ValueOf(lb).Elem()
	for i := 0; i < v.NumField(); i++ {
		field := fmt.Sprintf("%s", v.Field(i))
		val := fmt.Sprintf("%s.%s=%s", managedHelmKeyPrefix, v.Type().Field(i).Name, base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString([]byte(field)))
		vals = append(vals, val)
	}

	return vals
}

func (s *Server) newLoadBalancer(lbID gidx.PrefixedID, additionalSubjs []gidx.PrefixedID) *loadBalancer {
	lb := &loadBalancer{
		loadBalancerID: lbID,
	}

	for _, sub := range additionalSubjs {
		switch {
		// TODO: clean this up once we have a better way to handle this
		case sub.Prefix() == "tnnttnt":
			lb.loadBalancerTenantID = sub
		case slices.Contains(s.Locations, sub.String()):
			lb.loadBalancerLocationID = sub
		}
	}

	return lb
}
