package srv

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/lestrrat-go/backoff/v2"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/storage/driver"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	applyv1 "k8s.io/client-go/applyconfigurations/core/v1"
	applymetav1 "k8s.io/client-go/applyconfigurations/meta/v1"
	rbacapplyv1 "k8s.io/client-go/applyconfigurations/rbac/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	helmReleaseLength = 53
	kubeNSLength      = 63
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
func (s *Server) CreateNamespace(hash string) (*v1.Namespace, error) {
	s.Logger.Debugw("ensuring namespace exists", "namespace", hash)

	if !checkNameLength(hash, kubeNSLength) {
		s.Logger.Debugw("namespace name is empty or too long", "namespace", hash, "limit", kubeNSLength)
		return nil, errInvalidObjectNameLength
	}

	kc, err := kubernetes.NewForConfig(s.KubeClient)
	if err != nil {
		s.Logger.Debugw("unable to authenticate against kubernetes cluster", "error", err)
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
				"com.infratographer.lb-operator/lb-id": hash},
		},
		Spec:   &applyv1.NamespaceSpecApplyConfiguration{},
		Status: &applyv1.NamespaceStatusApplyConfiguration{},
	}

	ns, err := kc.CoreV1().Namespaces().Apply(s.Context, &apSpec, metav1.ApplyOptions{FieldManager: "loadbalanceroperator"})
	if err != nil {
		s.Logger.Debugw("unable to create namespace", "error", err, "namespace", hash)
		return nil, errors.Join(err, errInvalidNamespace)
	}

	if err := attachRoleBinding(s.Context, kc, hash); err != nil {
		s.Logger.Debugw("unable to attach namespace manager rolebinding to namespace", "error", err)
		return nil, errors.Join(err, errInvalidRoleBinding)
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

// newDeployment deploys a loadBalancer based upon the configuration provided
// from the event that is processed.
func (s *Server) newDeployment(lb *loadBalancer) error {
	hash := hashLBName(lb.loadBalancerID.String())

	if _, err := s.CreateNamespace(hash); err != nil {
		s.Logger.Errorw("unable to create namespace", "error", err, "namespace", hash, "loadBalancer", lb.loadBalancerID.String())
		return err
	}

	releaseName := fmt.Sprintf("lb-%s", hash)
	if !checkNameLength(releaseName, helmReleaseLength) {
		releaseName = releaseName[0:helmReleaseLength]
	}

	values, err := s.newHelmValues(lb)
	if err != nil {
		s.Logger.Errorw("unable to prepare chart values", "error", err, "loadBalancer", lb.loadBalancerID.String(), "namespace", hash)
		return err
	}

	client, err := s.newHelmClient(hash)
	if err != nil {
		s.Logger.Errorw("unable to initialize helm client", "err", err, "loadBalancer", lb.loadBalancerID.String(), "namespace", hash)
		return err
	}

	hc := action.NewInstall(client)
	hc.ReleaseName = releaseName
	hc.Namespace = hash
	_, err = hc.Run(s.Chart, values)

	switch err {
	case nil:
		s.Logger.Infow("loadbalancer deployed successfully", "namespace", hash, "releaseName", releaseName, "loadBalancer", lb.loadBalancerID.String())
	case driver.ErrReleaseExists:
		s.Logger.Debugw("loadbalancer already exists, proceeding to upgrade...", "namespace", hash, "releaseName", releaseName, "loadBalancer", lb.loadBalancerID.String())
	default:
		s.Logger.Errorw("unable to deploy loadbalancer", "error", err, "namespace", hash, "releaseName", releaseName, "loadBalancer", lb.loadBalancerID.String())
		return err
	}

	return nil
}

func (s *Server) updateDeployment(lb *loadBalancer) error {
	hash := hashLBName(lb.loadBalancerID.String())

	releaseName := fmt.Sprintf("lb-%s", hash)
	if !checkNameLength(releaseName, helmReleaseLength) {
		releaseName = releaseName[0:helmReleaseLength]
	}

	values, err := s.newHelmValues(lb)
	if err != nil {
		s.Logger.Errorw("unable to prepare chart values", "error", err, "loadBalancer", lb.loadBalancerID.String())
		return err
	}

	client, err := s.newHelmClient(hash)
	if err != nil {
		s.Logger.Errorw("unable to initialize helm client", "error", err, "loadBalancer", lb.loadBalancerID.String())
		return err
	}

	hc := action.NewUpgrade(client)
	hc.Namespace = hash
	_, err = hc.Run(releaseName, s.Chart, values)

	if err != nil {
		s.Logger.Debugw("unable to upgrade loadbalancer", "error", err, "namespace", hash, "releaseName", releaseName, "loadBalancer", lb.loadBalancerID.String())
		return err
	}

	s.Logger.Infow("loadbalancer upgraded successfully", "namespace", hash, "releaseName", releaseName, "loadBalancer", lb.loadBalancerID.String())

	return nil
}

func (s *Server) removeDeployment(lb *loadBalancer) error {
	hash := hashLBName(lb.loadBalancerID.String())

	releaseName := fmt.Sprintf("lb-%s", hash)
	if !checkNameLength(releaseName, helmReleaseLength) {
		releaseName = releaseName[0:helmReleaseLength]
	}

	client, err := s.newHelmClient(hash)
	if err != nil {
		s.Logger.Errorw("unable to initialize helm client", "error", err, "loadBalancer", lb.loadBalancerID.String(), "namespace", hash, "releaseName", releaseName)
		return err
	}

	hc := action.NewUninstall(client)
	_, err = hc.Run(releaseName)

	if err != nil {
		s.Logger.Errorw("unable to remove loadBalancer", "error", err, "releaseName", releaseName, "loadBalancer", lb.loadBalancerID.String(), "namespace", hash)
		return err
	}

	s.Logger.Infow("loadbalancer removed successfully", "releaseName", releaseName, "loadBalancer", lb.loadBalancerID.String())

	err = s.removeNamespace(hash)
	if err != nil {
		s.Logger.Errorw("unable to remove namespace", "error", err, "namespace", hash, "loadBalancer", lb.loadBalancerID.String())
		return err
	}

	s.Logger.Infow("namespace removed successfully", "namespace", hash, "loadBalancer", lb.loadBalancerID.String())

	return nil
}

func strPt(s string) *string {
	return &s
}

func checkNameLength(name string, limit int) bool {
	return len(name) <= limit && len(name) > 0
}

func hashLBName(name string) string {
	return hex.EncodeToString([]byte(name))
}

func (s *Server) createDeployment(_ context.Context, lb *loadBalancer) error {
	hash := hashLBName(lb.loadBalancerID.String())

	releaseName := fmt.Sprintf("lb-%s", hash)
	if !checkNameLength(releaseName, helmReleaseLength) {
		releaseName = releaseName[0:helmReleaseLength]
	}

	client, err := s.newHelmClient(hash)
	if err != nil {
		s.Logger.Debugw("unable to initialize helm client", "error", err, "loadBalancer", lb.loadBalancerID.String())
		return err
	}

	histClient := action.NewHistory(client)
	histClient.Max = 1

	if _, err := histClient.Run(releaseName); errors.Is(err, driver.ErrReleaseNotFound) {
		err = s.newDeployment(lb)
		if err != nil && !errors.Is(err, driver.ErrReleaseExists) {
			return err
		}
	}

	b := s.BackoffConfig.Start(s.Context)
	for backoff.Continue(b) {
		err = s.updateDeployment(lb)
		if err == nil {
			return nil
		} else {
			s.Logger.Debugw("unable to update loadbalancer, retrying...", "error", err, "loadBalancer", lb.loadBalancerID.String())
		}
	}

	s.Logger.Errorw("failed to update loadbalancer", "error", err, "loadBalancer", lb.loadBalancerID.String())

	return err
}
