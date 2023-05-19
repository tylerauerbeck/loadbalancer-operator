package srv

import (
	"reflect"

	"go.infratographer.com/x/pubsubx"
)

func (s *Server) processLoadBalancerPort(msgType string, data interface{}) error {
	switch msgType {
	case "change":
		if reflect.TypeOf(data).String() != "pubsubx.ChangeMessage" {
			return errMessageTypeMismatch
		}

		msg := data.(pubsubx.ChangeMessage)
		if err := s.processLoadBalancerPortChange(msg); err != nil {
			return err
		}
	case "event":
		if reflect.TypeOf(data).String() != "pubsubx.EventMessage" {
			return errMessageTypeMismatch
		}

		msg := data.(pubsubx.EventMessage)
		if err := s.processLoadBalancerPortEvent(msg); err != nil {
			return err
		}
	default:
		return errUnknownMessageEventType
	}

	return nil
}

func (s *Server) processLoadBalancerPortChange(msg pubsubx.ChangeMessage) error {
	switch msg.EventType {
	case create:
		if err := s.processLoadBalancerPortChangeCreate(msg); err != nil {
			return err
		}
	case update:
		if err := s.processLoadBalancerPortChangeUpdate(msg); err != nil {
			return err
		}
	case delete:
		if err := s.processLoadBalancerPortChangeDelete(msg); err != nil {
			return err
		}
	default:
		return errUnknownEventType
	}

	return nil
}

func (s *Server) processLoadBalancerPortChangeCreate(msg pubsubx.ChangeMessage) error {
	lbID := msg.SubjectID.String()

	if err := s.newDeployment(lbID, nil); err != nil {
		s.Logger.Errorw("handler unable to create loadbalancer", "error", err)
		return err
	}

	return nil
}

func (s *Server) processLoadBalancerPortChangeDelete(msg pubsubx.ChangeMessage) error {
	lbID := msg.SubjectID.String()

	if err := s.removeDeployment(lbID); err != nil {
		s.Logger.Errorw("handler unable to delete loadbalancer", "error", err)
		return err
	}

	return nil
}

func (s *Server) processLoadBalancerPortChangeUpdate(msg pubsubx.ChangeMessage) error {
	_ = msg.SubjectID.String()
	return nil
}

func (s *Server) processLoadBalancerPortEvent(msg pubsubx.EventMessage) error {
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
