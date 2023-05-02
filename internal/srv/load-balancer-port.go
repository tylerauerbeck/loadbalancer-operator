package srv

import (
	"context"
	"fmt"

	"go.infratographer.com/x/pubsubx"
	"go.infratographer.com/x/urnx"
	"helm.sh/helm/v3/pkg/strvals"
)

type ports struct {
	Name       string
	Protocol   string
	Port       int64
	TargetPort int64
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

	svcports := []ports{}
	for _, port := range lb.LoadBalancer.Ports {
		svcports = append(svcports, ports{
			Name:       port.Name,
			Protocol:   "TCP",
			Port:       port.Port,
			TargetPort: port.Port,
		})
	}

	fmt.Printf("%+v\n", svcports)

	values, err := s.newHelmValues(nil)
	if err != nil {
		fmt.Println(err)
	}

	err = newHelmOverrides(&values, "service.ports", svcports)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(values)

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
	if err := strvals.ParseInto(fmt.Sprintf("%s=%s", key, value), *values); err != nil {
		return err
	}

	return nil
}
