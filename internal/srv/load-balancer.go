package srv

import (
	"reflect"

	"go.infratographer.com/x/pubsubx"
)

func (s *Server) processLoadBalancer(msgType string, data interface{}) error {
	switch msgType {
	case "change":
		if reflect.TypeOf(data).String() != "pubsubx.ChangeMessage" {
			return errMessageTypeMismatch
		}

		msg := data.(pubsubx.ChangeMessage)
		if err := s.processLoadBalancerChange(msg); err != nil {
			return err
		}
	case "event":
		if reflect.TypeOf(data).String() != "pubsubx.EventMessage" {
			return errMessageTypeMismatch
		}

		msg := data.(pubsubx.EventMessage)
		if err := s.processLoadBalancerEvent(msg); err != nil {
			return err
		}
	default:
		return errUnknownMessageEventType
	}

	return nil
}

func (s *Server) processLoadBalancerChange(msg pubsubx.ChangeMessage) error {
	switch msg.EventType {
	case create:
		if err := s.processLoadBalancerChangeCreate(msg); err != nil {
			return err
		}
	case update:
		if err := s.processLoadBalancerChangeUpdate(msg); err != nil {
			return err
		}
	case delete:
		if err := s.processLoadBalancerChangeDelete(msg); err != nil {
			return err
		}
	default:
		return errUnknownEventType
	}

	return nil
}

func (s *Server) processLoadBalancerChangeCreate(msg pubsubx.ChangeMessage) error {
	// lbID := msg.SubjectID.String()
	lb := s.newLoadBalancer(msg.SubjectID, msg.AdditionalSubjectIDs)

	if err := s.newDeployment(lb); err != nil {
		s.Logger.Errorw("handler unable to create loadbalancer", "error", err)
		return err
	}

	return nil
}

func (s *Server) processLoadBalancerChangeDelete(msg pubsubx.ChangeMessage) error {
	lbID := msg.SubjectID.String()

	if err := s.removeDeployment(lbID); err != nil {
		s.Logger.Errorw("handler unable to delete loadbalancer", "error", err)
		return err
	}

	return nil
}

func (s *Server) processLoadBalancerChangeUpdate(msg pubsubx.ChangeMessage) error {
	_ = msg.SubjectID.String()
	return nil
}

func (s *Server) processLoadBalancerEvent(msg pubsubx.EventMessage) error {
	switch msg.EventType {
	case create:
		return nil
	case update:
		return nil
	case delete:
		return nil
	default:
		return errUnknownEventType
	}
}
