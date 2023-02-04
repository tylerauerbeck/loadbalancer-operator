// Package srv provides server connectivity for loadbalanceroperator
package srv

import (
	"context"
	"fmt"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/getter"
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
	nameLength = 53
)

// CreateNamespace creates namespaces for the specified group that is
// provided in the event received
func (s *Server) CreateNamespace(groupID string) error {
	s.Logger.Debugf("ensuring namespace %s exists", groupID)
	kc, err := kubernetes.NewForConfig(s.KubeClient)

	if err != nil {
		s.Logger.Errorw("unable to authenticate against kubernetes cluster", "error", err)
		return err
	}

	apSpec := applyv1.NamespaceApplyConfiguration{
		TypeMetaApplyConfiguration: applymetav1.TypeMetaApplyConfiguration{
			Kind:       strPt("Namespace"),
			APIVersion: strPt("v1"),
		},
		ObjectMetaApplyConfiguration: &applymetav1.ObjectMetaApplyConfiguration{
			Name: &groupID,
		},
		Spec:   &applyv1.NamespaceSpecApplyConfiguration{},
		Status: &applyv1.NamespaceStatusApplyConfiguration{},
	}
	_, err = kc.CoreV1().Namespaces().Apply(s.Context, &apSpec, metav1.ApplyOptions{FieldManager: "loadbalanceroperator"})

	if err != nil {
		s.Logger.Errorw("unable to create namespace", "error", err)
		return err
	}

	if err := attachRoleBinding(s.Context, kc, groupID); err != nil {
		s.Logger.Errorw("unable to attach namespace manager rolebinding to namespace", "error", err)
		return err
	}

	return nil
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

func (s *Server) newHelmValues(overrides []valueSet) (map[string]interface{}, error) {
	provider := getter.All(&cli.EnvSettings{})

	valOpts := &values.Options{
		ValueFiles: []string{s.ValuesPath},
	}

	values, err := valOpts.MergeValues(provider)
	if err != nil {
		s.Logger.Errorw("unable to load values data", "error", err)
		return nil, err
	}

	for _, override := range overrides {
		if err := strvals.ParseInto(override.helmKey+"="+override.value, values); err != nil {
			s.Logger.Errorw("unable to parse values", "error", err)
			return nil, err
		}
	}

	return values, nil
}

// newDeployment deploys a loadBalancer based upon the configuration provided
// from the event that is processed.
func (s *Server) newDeployment(name string, overrides []valueSet) error {
	releaseName := fmt.Sprintf("lb-%s", name)
	if len(releaseName) > nameLength {
		releaseName = releaseName[0:nameLength]
	}

	values, err := s.newHelmValues(overrides)
	if err != nil {
		s.Logger.Errorw("unable to prepare chart values", "error", err)
		return err
	}

	client, err := s.newHelmClient(name)
	if err != nil {
		s.Logger.Errorln("unable to initialize helm client: %s", err)
		return err
	}

	hc := action.NewInstall(client)
	hc.ReleaseName = releaseName
	hc.Namespace = name
	_, err = hc.Run(s.Chart, values)

	if err != nil {
		s.Logger.Errorf("unable to deploy %s to %s", releaseName, name)
		return err
	}

	s.Logger.Infof("%s deployed to %s successfully", releaseName, name)

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
