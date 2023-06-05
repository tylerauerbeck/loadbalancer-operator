package srv

import (
	"encoding/hex"
	"errors"
	"fmt"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/getter"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
)

type helmvalues struct {
	*values.Options
}

const (
	managedHelmKeyPrefix = "operator.managed"
)

func (v helmvalues) generateLBHelmVals(lb *loadBalancer) {
	// add loadbalancer id values
	v.addValue("loadBalancerID", lb.loadBalancerID.String())
	v.addValue("loadBalancerIDEnc", hex.EncodeToString([]byte(lb.loadBalancerID.String())))
}

func (v helmvalues) addValue(key string, value interface{}) {
	val := fmt.Sprintf("%s.%s=%s", managedHelmKeyPrefix, key, value)
	v.StringValues = append(v.StringValues, val)
}

func (s *Server) newHelmClient(namespace string) (*action.Configuration, error) {
	config := &action.Configuration{}
	cliopt := genericclioptions.NewConfigFlags(false)
	wrapper := func(*rest.Config) *rest.Config { return s.KubeClient }
	cliopt.WithWrapConfigFn(wrapper)

	err := config.Init(cliopt, namespace, "secret", s.Logger.Debugf)
	if err != nil || namespace == "" {
		s.Logger.Debugw("unable to initialize helm client", "error", err, "namespace", namespace)
		err = errors.Join(err, errInvalidHelmClient)

		return nil, err
	}

	return config, nil
}

func (s *Server) newHelmValues(lb *loadBalancer) (map[string]interface{}, error) {
	provider := getter.All(&cli.EnvSettings{})

	opts := helmvalues{&values.Options{
		ValueFiles: []string{s.ValuesPath},
	}}

	opts.generateLBHelmVals(lb)

	values, err := opts.MergeValues(provider)
	if err != nil {
		s.Logger.Debugw("unable to load values data", "error", err, "loadBalancer", lb.loadBalancerID.String())
		return nil, errors.Join(err, errInvalidHelmValues)
	}

	return values, nil
}
