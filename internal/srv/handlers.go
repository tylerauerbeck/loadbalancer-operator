package srv

import (
	"strings"

	"go.infratographer.com/x/events"
	"go.infratographer.com/x/gidx"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
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

func (s *Server) listenEvent(messages <-chan events.Message[events.EventMessage]) {
	for msg := range messages {
		s.processEvent(msg)
	}
}

func (s *Server) processEvent(msg events.Message[events.EventMessage]) {
	var lb *loadBalancer

	var err error

	m := msg.Message()
	s.Logger.Infow("messageType", "event", "messageID", msg.ID())
	s.Logger.Debugw("messageType", "event", "messageID", msg.ID(), "data", m)

	ctx, span := otel.Tracer(instrumentationName).Start(m.GetTraceContext(s.Context), "processEvent")
	defer span.End()

	span.SetAttributes(
		attribute.String("loadbalancer.id", m.SubjectID.String()),
	)

	if slices.ContainsFunc(m.AdditionalSubjectIDs, s.locationCheck) || len(s.Locations) == 0 {
		if m.EventType == string("ip-address.unassigned") {
			lb = &loadBalancer{loadBalancerID: m.SubjectID, lbData: nil, lbType: typeLB}
		} else {
			lb, err = s.newLoadBalancer(ctx, m.SubjectID, m.AdditionalSubjectIDs)
			if err != nil {
				s.Logger.Errorw("unable to initialize loadbalancer", "error", err, "messageID", msg.ID(), "loadbalancerID", m.SubjectID.String())
			}
		}

		if lb != nil && lb.lbType != typeNoLB {
			switch {
			case m.EventType == "ip-address.assigned":
				s.Logger.Debugw("ip address processed. updating loadbalancer", "loadbalancer", lb.loadBalancerID.String())

				if err := s.createDeployment(ctx, lb); err != nil {
					s.Logger.Errorw("unable to update loadbalancer", "error", err, "messageID", msg.ID(), "loadbalancer", lb.loadBalancerID.String())
				}
			case m.EventType == "ip-address.unassigned":
				s.Logger.Debugw("ip address unassigned. updating loadbalancer", "loadbalancer", lb.loadBalancerID.String())
			default:
				s.Logger.Debugw("unknown event", "loadbalancer", lb.loadBalancerID.String(), "event", m.EventType)
			}
		}
	}

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	// we need to Acknowledge that we received and processed the message,
	// otherwise, it will be resent over and over again.
	if err := msg.Ack(); err != nil {
		s.Logger.Errorw("unable to acknowledge message", "error", err, "messageID", msg.ID())
	}
}

func (s *Server) listenChange(messages <-chan events.Message[events.ChangeMessage]) {
	for msg := range messages {
		s.processChange(msg)
	}
}

func (s *Server) processChange(msg events.Message[events.ChangeMessage]) {
	var lb *loadBalancer

	var err error

	m := msg.Message()

	ctx, span := otel.Tracer(instrumentationName).Start(m.GetTraceContext(s.Context), "processChange")
	defer span.End()

	span.SetAttributes(
		attribute.String("loadbalancer.id", m.SubjectID.String()),
	)

	if slices.ContainsFunc(m.AdditionalSubjectIDs, s.locationCheck) || len(s.Locations) == 0 {
		if m.EventType == string(events.DeleteChangeType) && m.SubjectID.Prefix() == LBPrefix {
			lb = &loadBalancer{loadBalancerID: m.SubjectID, lbData: nil, lbType: typeLB}
		} else {
			lb, err = s.newLoadBalancer(ctx, m.SubjectID, m.AdditionalSubjectIDs)
			if err != nil {
				s.Logger.Errorw("unable to initialize loadbalancer", "error", err, "messageID", msg.ID(), "subjectID", m.SubjectID.String())
			}
		}

		if lb != nil && lb.lbType != typeNoLB {
			switch {
			case m.EventType == string(events.CreateChangeType) && lb.lbType == typeLB:
				s.Logger.Debugw("creating loadbalancer", "loadbalancer", lb.loadBalancerID.String())

				if err := s.processLoadBalancerChangeCreate(ctx, lb); err != nil {
					s.Logger.Errorw("handler unable to create/update loadbalancer", "error", err, "loadbalancerID", lb.loadBalancerID.String())
				}
			case m.EventType == string(events.DeleteChangeType) && lb.lbType == typeLB:
				s.Logger.Debugw("deleting loadbalancer", "loadbalancer", lb.loadBalancerID.String())

				if err := s.processLoadBalancerChangeDelete(ctx, lb); err != nil {
					s.Logger.Errorw("handler unable to delete loadbalancer", "error", err, "loadbalancerID", lb.loadBalancerID.String())
				}
			default:
				s.Logger.Debugw("updating loadbalancer", "loadbalancer", lb.loadBalancerID.String())

				if err := s.processLoadBalancerChangeUpdate(ctx, lb); err != nil {
					s.Logger.Errorw("handler unable to update loadbalancer", "error", err, "loadbalancerID", lb.loadBalancerID.String())
				}
			}
		}
	}

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	// we need to Acknowledge that we received and processed the message,
	// otherwise, it will be resent over and over again.
	if err := msg.Ack(); err != nil {
		s.Logger.Errorw("unable to acknowledge message", "error", err, "messageID", msg.ID())
	}
}
