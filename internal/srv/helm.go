package srv

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

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
	v.StringValues = append(v.StringValues, fmt.Sprintf("%s=%s", managedHelmKeyPrefix+".lbID", lb.loadBalancerID.String()))
	v.StringValues = append(v.StringValues, fmt.Sprintf("%s=%s", managedHelmKeyPrefix+".lbIDEnc", hex.EncodeToString([]byte(lb.loadBalancerID.String()))))

	// add IP address if it is available; will be empty if an IP is not yet assigned
	// while multiple addresses are possible, we only support one for now
	if len(lb.lbData.LoadBalancer.IPAddresses) > 0 {
		v.StringValues = append(v.StringValues, fmt.Sprintf("%s=%s", managedHelmKeyPrefix+".lbIP", lb.lbData.LoadBalancer.IPAddresses[0].IP))
	}

	// add port values
	var cport, sport []interface{}

	for _, port := range lb.lbData.LoadBalancer.Ports.Edges {
		cport = append(cport, map[string]interface{}{"name": "p" + strconv.Itoa(int(port.Node.Number)), "containerPort": port.Node.Number})
		sport = append(sport, map[string]interface{}{"name": "p" + strconv.Itoa(int(port.Node.Number)), "port": port.Node.Number})
	}

	// add metrics port
	cport = append(cport, map[string]interface{}{"name": "infra9-metrics", "containerPort": s.MetricsPort})
	sport = append(sport, map[string]interface{}{"name": "infra9-metrics", "port": s.MetricsPort})

	if cport != nil {
		if cports, err := json.Marshal(cport); err != nil {
			s.Logger.Warnw("unable to marshal container ports", "error", err, "loadbalancer", lb.loadBalancerID.String())
		} else {
			v.JSONValues = append(v.JSONValues, fmt.Sprintf("%s=%s", s.ContainerPortKey, string(cports)))
		}

		s.Logger.Debugw("patching deployment containterPorts", "loadBalancer", lb.loadBalancerID.String(), "key", s.ContainerPortKey, "values", cport)
	}

	if sport != nil {
		if sports, err := json.Marshal(sport); err != nil {
			s.Logger.Warnw("unable to marshal service ports", "error", err, "loadbalancer", lb.loadBalancerID.String())
		} else {
			v.JSONValues = append(v.JSONValues, fmt.Sprintf("%s=%s", s.ServicePortKey, string(sports)))
		}

		s.Logger.Debugw("patching serivceports", "loadBalancer", lb.loadBalancerID.String(), "key", s.ServicePortKey, "values", sport)
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
