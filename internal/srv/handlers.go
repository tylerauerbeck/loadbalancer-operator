package srv

import (
	"strings"

	"github.com/ThreeDotsLabs/watermill/message"
	"go.infratographer.com/x/events"
	"go.infratographer.com/x/gidx"
	"golang.org/x/exp/slices"
)

func (s *Server) locationCheck(i gidx.PrefixedID) bool {
	for _, s := range s.Locations {
		if strings.HasSuffix(i.String(), s) {
			return true
		}
	}

	return false
}

func (s *Server) processEvent(messages <-chan *message.Message) {
	for msg := range messages {
		s.Logger.Infof("received event message: %s, payload: %s\n", msg.UUID, string(msg.Payload))

		m, err := events.UnmarshalEventMessage(msg.Payload)
		if err != nil {
			s.Logger.Errorw("unable to unmarshal event message", "error", err)
			msg.Nack()
		}

		if slices.ContainsFunc(m.AdditionalSubjectIDs, s.locationCheck) || len(s.Locations) == 0 {
			lb, err := s.newLoadBalancer(m.SubjectID, m.AdditionalSubjectIDs)
			if err != nil {
				s.Logger.Errorw("unable to initialize loadbalancer", "error", err, "messageID", msg.UUID)
				msg.Nack()
			}

			if lb.lbType != typeNoLB {
				switch {
				case m.EventType == "create" && lb.lbType == typeLB:
					s.Logger.Debugw("stub for creating loadbalancer", "loadbalancer", lb.loadBalancerID.String())
				case m.EventType == "delete" && lb.lbType == typeLB:
					s.Logger.Debugw("stub for deleting loadbalancer", "loadbalancer", lb.loadBalancerID.String())
				default:
					s.Logger.Debugw("stub for updating loadbalancer", "loadbalancer", lb.loadBalancerID.String())
				}
			}
		}
		// we need to Acknowledge that we received and processed the message,
		// otherwise, it will be resent over and over again.
		msg.Ack()
	}
}

func (s *Server) processChange(messages <-chan *message.Message) {
	for msg := range messages {
		m, err := events.UnmarshalChangeMessage(msg.Payload)
		if err != nil {
			s.Logger.Errorw("unable to unmarshal change message", "error", err, "messageID", msg.UUID)
			msg.Nack()
		}

		if slices.ContainsFunc(m.AdditionalSubjectIDs, s.locationCheck) || len(s.Locations) == 0 {
			lb, err := s.newLoadBalancer(m.SubjectID, m.AdditionalSubjectIDs)
			if err != nil {
				s.Logger.Errorw("unable to initialize loadbalancer", "error", err, "messageID", msg.UUID)
				msg.Nack()
			}

			if lb.lbType != typeNoLB {
				switch {
				case m.EventType == string(events.CreateChangeType) && lb.lbType == typeLB:
					s.Logger.Debugw("creating loadbalancer", "loadbalancer", lb.loadBalancerID.String())

					if err := s.processLoadBalancerChangeCreate(lb); err != nil {
						s.Logger.Errorw("handler unable to create loadbalancer", "error", err)
						msg.Nack()
					}
				case m.EventType == string(events.DeleteChangeType) && lb.lbType == typeLB:
					s.Logger.Debugw("deleting loadbalancer", "loadbalancer", lb.loadBalancerID.String())

					if err := s.processLoadBalancerChangeDelete(lb); err != nil {
						s.Logger.Errorw("handler unable to delete loadbalancer", "error", err)
						msg.Nack()
					}
				default:
					s.Logger.Debugw("updating loadbalancer", "loadbalancer", lb.loadBalancerID.String())

					if err := s.processLoadBalancerChangeUpdate(lb); err != nil {
						s.Logger.Errorw("handler unable to update loadbalancer", "error", err)
						msg.Nack()
					}
				}
			}
		}
		// we need to Acknowledge that we received and processed the message,
		// otherwise, it will be resent over and over again.
		msg.Ack()
	}
}
