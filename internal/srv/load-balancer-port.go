package srv

import (
	"context"
	"fmt"

	"go.infratographer.com/x/pubsubx"
	"go.infratographer.com/x/urnx"
)

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
	// lbdata := events.LoadBalancerData{}
	urn, err := urnx.Parse(msg.SubjectURN)
	if err != nil {
		s.Logger.Errorw("unable to parse load-balancer URN", "error", err)
		return err
	}

	lb, err := s.APIClient.GetLoadBalancer(context.TODO(), urn.ResourceID.String())

	if err != nil {
		s.Logger.Errorw("unable to get load balancer", "error", err)
		return err
	}

	fmt.Println(lb)

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
