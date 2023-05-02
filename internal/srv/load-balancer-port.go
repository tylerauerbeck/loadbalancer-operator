package srv

import (
	"context"
	"encoding/json"
	"fmt"

	"go.infratographer.com/x/pubsubx"
	"go.infratographer.com/x/urnx"
	"helm.sh/helm/v3/pkg/strvals"
)

type svcport struct {
	Name       string
	Protocol   string
	Port       int64
	TargetPort string
}

type cport struct {
	Name          string
	ContainerPort string
}

func (s *Server) processLoadBalancerPort(msg pubsubx.Message) error {
	switch msg.EventType {
	case create:
		if err := s.processLoadBalancerPortCreate(msg); err != nil {
			return err
		}
	case update:
		if err := s.processLoadBalancerPortUpdate(msg); err != nil {
			return err
		}
	case delete:
		if err := s.processLoadBalancerPortDelete(msg); err != nil {
			return err
		}
	default:
		s.Logger.Errorw("Unknown action: %s", "action", msg.EventType)
		return errUnknownEventType
	}

	return nil
}

func (s *Server) processLoadBalancerPortCreate(msg pubsubx.Message) error {
	lbid, err := s.getAssocLBUUID(msg.AdditionalSubjectURNs)
	if err != nil {
		s.Logger.Errorw("unable to find associated load balancer in message", "error", err, "associated_urns", msg.AdditionalSubjectURNs)
		return err
	}

	lb, err := s.APIClient.GetLoadBalancer(context.TODO(), lbid.String())
	if err != nil {
		s.Logger.Errorw("unable to get load balancer", "error", err)
		return err
	}

	svcports := []map[string]interface{}{}
	cports := []map[string]interface{}{}

	for _, port := range lb.LoadBalancer.Ports {
		sp := map[string]interface{}{
			"name":       port.Name,
			"protocol":   "TCP",
			"port":       port.Port,
			"targetPort": port.Port,
		}
		svcports = append(svcports, sp)

		cp := map[string]interface{}{
			"name":          port.Name,
			"containerPort": port.Port,
		}
		cports = append(cports, cp)
	}

	values, err := s.newHelmValues(nil)
	if err != nil {
		fmt.Println(err)
	}

	jsondata, err := json.Marshal(&svcports)
	if err != nil {
		s.Logger.Errorw("unable to marshal service ports", "error", err)
		return err
	}

	// // TODO: just make this a generic helm val that any chart could combine in their chart for usage
	// err = newHelmOverrides(&values, "service.ports", fmt.Sprintf("%+v", svcports))
	err = newHelmOverrides(&values, "service.ports", jsondata)
	if err != nil {
		fmt.Println(err)
	}

	jsondata, err = json.Marshal(&cports)
	if err != nil {
		s.Logger.Errorw("unable to marshal container ports", "error", err)
		return err
	}

	// TODO: just make this a generic helm val that any chart could combine in their chart for usage
	err = newHelmOverrides(&values, "containerPorts", jsondata)
	if err != nil {
		fmt.Println(err)
	}

	err = s.updateDeployment(lbid.String(), values)
	if err != nil {
		fmt.Println(err)
	}

	return nil
}

func (s *Server) processLoadBalancerPortDelete(msg pubsubx.Message) error {
	_, err := urnx.Parse(msg.SubjectURN)
	if err != nil {
		s.Logger.Errorw("unable to parse load-balancer URN", "error", err)
		return err
	}

	return nil
}

func (s *Server) processLoadBalancerPortUpdate(msg pubsubx.Message) error {
	_, err := urnx.Parse(msg.SubjectURN)
	if err != nil {
		s.Logger.Errorw("unable to parse load-balancer URN", "error", err)
		return err
	}

	return nil
}

func newHelmOverrides(values *map[string]interface{}, key string, value interface{}) error {
	if err := strvals.ParseJSON(fmt.Sprintf("%s=%s", key, value), *values); err != nil {
		return err
	}

	return nil
}
