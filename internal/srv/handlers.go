package srv

import (
	"context"
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
	m := msg.Message()

	ctx, span := otel.Tracer(instrumentationName).Start(m.GetTraceContext(s.Context), "processEvent")
	defer span.End()

	lb, err := prepareLoadBalancer[events.EventMessage](ctx, m, s)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	if err == nil && lb != nil && lb.lbType != typeNoLB {
		span.SetAttributes(
			attribute.String("loadbalancer.id", lb.loadBalancerID.String()),
			attribute.String("message.event", m.EventType),
			attribute.String("message.id", msg.ID()),
			attribute.String("message.subject", m.SubjectID.String()),
		)

		ch := s.checkChannel(ctx, lb)
		ch.writer <- &lbTask{lb: lb, ctx: ctx, evt: m.EventType, srv: s}
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
	m := msg.Message()

	ctx, span := otel.Tracer(instrumentationName).Start(m.GetTraceContext(s.Context), "processChange")
	defer span.End()

	lb, err := prepareLoadBalancer[events.ChangeMessage](ctx, m, s)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	if err == nil && lb != nil && lb.lbType != typeNoLB {
		span.SetAttributes(
			attribute.String("loadbalancer.id", lb.loadBalancerID.String()),
			attribute.String("message.event", m.EventType),
			attribute.String("message.id", msg.ID()),
			attribute.String("message.subject", m.SubjectID.String()),
		)

		ch := s.checkChannel(ctx, lb)
		ch.writer <- &lbTask{lb: lb, ctx: ctx, evt: m.EventType, srv: s}
	}

	// we need to Acknowledge that we received and processed the message,
	// otherwise, it will be resent over and over again.
	if err := msg.Ack(); err != nil {
		s.Logger.Errorw("unable to acknowledge message", "error", err, "messageID", msg.ID())
	}
}

func (s *Server) checkChannel(ctx context.Context, lb *loadBalancer) *runner {
	ctx, span := otel.Tracer(instrumentationName).Start(ctx, "checkChannel")
	defer span.End()

	ch, ok := s.LoadBalancers[lb.loadBalancerID.String()]
	if !ok {
		span.SetAttributes(attribute.Bool("channel-exists", false))

		r := NewRunner(ctx, process)

		s.LoadBalancers[lb.lbData.ID] = r
		ch = r
	} else {
		span.SetAttributes(attribute.Bool("channel-exists", true))
	}

	return ch
}

func prepareLoadBalancer[M Message](ctx context.Context, msg M, s *Server) (*loadBalancer, error) {
	ctx, span := otel.Tracer(instrumentationName).Start(ctx, "prepareLoadBalancer")
	defer span.End()

	if slices.ContainsFunc(M.GetAddSubjects(msg), s.locationCheck) || len(s.Locations) == 0 {
		var (
			lb  = new(loadBalancer)
			err error
		)

		// TODO: this is a hack to get around the fact that we can't lookup a loadbalancer
		// that has already been deleted. So if we have a delete event, we just grab the LB ID
		// from the message and don't attempt to look it up as we don't need the actual data.
		if msg.GetEventType() == string(events.DeleteChangeType) {
			lb.isLoadBalancer(msg.GetSubject(), msg.GetAddSubjects())

			span.SetAttributes(attribute.Bool("lbdata-lookup", false))
		} else {
			lb, err = s.newLoadBalancer(ctx, msg.GetSubject(), msg.GetAddSubjects())
			if err != nil {
				s.Logger.Errorw("unable to initialize loadbalancer", "error", err, "subjectID", msg.GetSubject().String())
				err = errLoadBalancerInit
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				return nil, err
			}
			span.SetAttributes(attribute.Bool("lbdata-lookup", true))
		}

		span.SetAttributes(
			attribute.String("loadbalancer.id", lb.loadBalancerID.String()),
			attribute.String("message.event", msg.GetEventType()),
			attribute.String("message.subject", msg.GetSubject().String()),
			attribute.Bool("trackedLocation", true),
		)

		return lb, nil
	}

	span.SetAttributes(attribute.Bool("trackedLocation", false))

	return nil, errNotMyMessage
}

func process(t *lbTask) {
	switch {
	case t.evt == "ip-address.assigned":
		t.srv.Logger.Debugw("ip address processed. updating loadbalancer", "loadbalancer", t.lb.loadBalancerID.String())

		if err := t.srv.createDeployment(t.ctx, t.lb); err != nil {
			t.srv.Logger.Errorw("unable to update loadbalancer", "error", err, "loadbalancer", t.lb.loadBalancerID.String())
		}

		return
	case t.evt == string(events.CreateChangeType) && t.lb.lbType == typeLB:
		t.srv.Logger.Debugw("creating loadbalancer", "loadbalancer", t.lb.loadBalancerID.String())

		if err := t.srv.processLoadBalancerChangeCreate(t.ctx, t.lb); err != nil {
			t.srv.Logger.Errorw("handler unable to create/update loadbalancer", "error", err, "loadbalancerID", t.lb.loadBalancerID.String())
		}

		return
	case t.evt == string(events.DeleteChangeType) && t.lb.lbType == typeLB:
		t.srv.Logger.Debugw("deleting loadbalancer", "loadbalancer", t.lb.loadBalancerID.String())

		if err := t.srv.processLoadBalancerChangeDelete(t.ctx, t.lb); err != nil {
			t.srv.Logger.Errorw("handler unable to delete loadbalancer", "error", err, "loadbalancerID", t.lb.loadBalancerID.String())
		}

		ch, ok := t.srv.LoadBalancers[t.lb.loadBalancerID.String()]
		if ok {
			ch.stop()
		}

		delete(t.srv.LoadBalancers, t.lb.loadBalancerID.String())

		return
	case t.evt == "ip-address.unassigned":
		t.srv.Logger.Debugw("ip address unassigned. updating loadbalancer", "loadbalancer", t.lb.loadBalancerID.String())
	default:
		t.srv.Logger.Debugw("updating loadbalancer", "loadbalancer", t.lb.loadBalancerID.String())

		if err := t.srv.processLoadBalancerChangeUpdate(t.ctx, t.lb); err != nil {
			t.srv.Logger.Errorw("handler unable to update loadbalancer", "error", err, "loadbalancerID", t.lb.loadBalancerID.String())
		}
	}
}
