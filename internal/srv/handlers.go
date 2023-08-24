package srv

import (
	"context"
	"strings"

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

func (s *Server) processEvent(messages <-chan events.Message[events.EventMessage]) {
	var lb *loadBalancer

	var err error

	for msg := range messages {
		m := msg.Message()
		s.Logger.Infow("messageType", "event", "messageID", msg.ID())
		s.Logger.Debugw("messageType", "event", "messageID", msg.ID(), "data", m)

		if slices.ContainsFunc(m.AdditionalSubjectIDs, s.locationCheck) || len(s.Locations) == 0 {
			if m.EventType == string("ip-address.unassigned") {
				lb = &loadBalancer{loadBalancerID: m.SubjectID, lbData: nil, lbType: typeLB}
			} else {
				lb, err = s.newLoadBalancer(m.SubjectID, m.AdditionalSubjectIDs)
				if err != nil {
					s.Logger.Errorw("unable to initialize loadbalancer", "error", err, "messageID", msg.ID(), "loadbalancerID", m.SubjectID.String())
				}
			}

			if lb != nil && lb.lbType != typeNoLB {
				switch {
				case m.EventType == "ip-address.assigned":
					s.Logger.Debugw("ip address processed. updating loadbalancer", "loadbalancer", lb.loadBalancerID.String())
					// TODO: plumb context through appropriately when we add tracing
					if err := s.createDeployment(context.TODO(), lb); err != nil {
						s.Logger.Errorw("unable to update loadbalancer", "error", err, "messageID", msg.ID(), "loadbalancer", lb.loadBalancerID.String())
					}
				case m.EventType == "ip-address.unassigned":
					s.Logger.Debugw("ip address unassigned. updating loadbalancer", "loadbalancer", lb.loadBalancerID.String())
				default:
					s.Logger.Debugw("unknown event", "loadbalancer", lb.loadBalancerID.String(), "event", m.EventType)
				}
			}
		}
		// we need to Acknowledge that we received and processed the message,
		// otherwise, it will be resent over and over again.
		if err := msg.Ack(); err != nil {
			s.Logger.Errorw("unable to acknowledge message", "error", err, "messageID", msg.ID())
		}
	}
}

func (s *Server) processChange(messages <-chan events.Message[events.ChangeMessage]) {
	var lb *loadBalancer

	var err error

	for msg := range messages {
		m := msg.Message()

		if slices.ContainsFunc(m.AdditionalSubjectIDs, s.locationCheck) || len(s.Locations) == 0 {
			if m.EventType == string(events.DeleteChangeType) && m.SubjectID.Prefix() == LBPrefix {
				lb = &loadBalancer{loadBalancerID: m.SubjectID, lbData: nil, lbType: typeLB}
			} else {
				lb, err = s.newLoadBalancer(m.SubjectID, m.AdditionalSubjectIDs)
				if err != nil {
					s.Logger.Errorw("unable to initialize loadbalancer", "error", err, "messageID", msg.ID(), "subjectID", m.SubjectID.String())
				}
			}

			if lb != nil && lb.lbType != typeNoLB {
				switch {
				case m.EventType == string(events.CreateChangeType) && lb.lbType == typeLB:
					s.Logger.Debugw("creating loadbalancer", "loadbalancer", lb.loadBalancerID.String())

					if err := s.processLoadBalancerChangeCreate(lb); err != nil {
						s.Logger.Errorw("handler unable to create/update loadbalancer", "error", err, "loadbalancerID", lb.loadBalancerID.String())
					}
				case m.EventType == string(events.DeleteChangeType) && lb.lbType == typeLB:
					s.Logger.Debugw("deleting loadbalancer", "loadbalancer", lb.loadBalancerID.String())

					if err := s.processLoadBalancerChangeDelete(lb); err != nil {
						s.Logger.Errorw("handler unable to delete loadbalancer", "error", err, "loadbalancerID", lb.loadBalancerID.String())
					}
				default:
					s.Logger.Debugw("updating loadbalancer", "loadbalancer", lb.loadBalancerID.String())

					if err := s.processLoadBalancerChangeUpdate(lb); err != nil {
						s.Logger.Errorw("handler unable to update loadbalancer", "error", err, "loadbalancerID", lb.loadBalancerID.String())
					}
				}
			}
		}
		// we need to Acknowledge that we received and processed the message,
		// otherwise, it will be resent over and over again.
		if err := msg.Ack(); err != nil {
			s.Logger.Errorw("unable to acknowledge message", "error", err, "messageID", msg.ID())
		}
	}
}
