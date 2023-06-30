package srv

import (
	"encoding/hex"
	"encoding/json"
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

func (v helmvalues) generateLBHelmVals(lb *loadBalancer, s *Server) {
	// add loadbalancer id values
	v.StringValues = append(v.StringValues, fmt.Sprintf("%s=%s", managedHelmKeyPrefix+".loadBalancerID", lb.loadBalancerID.String()))
	v.StringValues = append(v.StringValues, fmt.Sprintf("%s=%s", managedHelmKeyPrefix+".loadBalancerIDEnc", hex.EncodeToString([]byte(lb.loadBalancerID.String()))))

	// add IP address if it is available; will be empty if an IP is not yet assigned
	// while multiple addresses are possible, we only support one for now
	if len(lb.lbData.LoadBalancer.IPAddresses) > 0 {
		v.StringValues = append(v.StringValues, fmt.Sprintf("%s=%s", managedHelmKeyPrefix+".loadBalancerIP", lb.lbData.LoadBalancer.IPAddresses[0].IP))
	}

	// add port values
	var cport, sport []interface{}

	for _, port := range lb.lbData.LoadBalancer.Ports.Edges {
		cport = append(cport, map[string]interface{}{"name": port.Node.Name, "containerPort": port.Node.Number})
		sport = append(sport, map[string]interface{}{"name": port.Node.Name, "port": port.Node.Number})
	}

	if cport != nil {
		if cports, err := json.Marshal(cport); err != nil {
			s.Logger.Debugw("unable to marshal container ports", "error", err, "loadbalancer", lb.loadBalancerID.String())
		} else {
			v.JSONValues = append(v.JSONValues, fmt.Sprintf("%s=%s", s.ContainerPortKey, string(cports)))
		}
	}

	if sport != nil {
		if sports, err := json.Marshal(sport); err != nil {
			s.Logger.Debugw("unable to marshal service ports", "error", err, "loadbalancer", lb.loadBalancerID.String())
		} else {
			v.JSONValues = append(v.JSONValues, fmt.Sprintf("%s=%s", s.ServicePortKey, string(sports)))
		}
	}
}

// func (v helmvalues) addValue(key string, value interface{}) {
// 	val := fmt.Sprintf("%s.%s=%s", managedHelmKeyPrefix, key, value)
// 	v.StringValues = append(v.StringValues, val)
// }

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

	opts.generateLBHelmVals(lb, s)

	values, err := opts.MergeValues(provider)
	if err != nil {
		s.Logger.Debugw("unable to load values data", "error", err, "loadBalancer", lb.loadBalancerID.String())
		return nil, errors.Join(err, errInvalidHelmValues)
	}

	return values, nil
}
