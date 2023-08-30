package srv

import (
	"context"
	"errors"

	"go.infratographer.com/loadbalancer-manager-haproxy/pkg/lbapi"
	"go.infratographer.com/x/gidx"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

func (s *Server) newLoadBalancer(ctx context.Context, subj gidx.PrefixedID, adds []gidx.PrefixedID) (*loadBalancer, error) {
	l := new(loadBalancer)
	l.isLoadBalancer(subj, adds)

	ctx, span := otel.Tracer(instrumentationName).Start(ctx, "newLoadBalancer")
	defer span.End()

	if l.lbType != typeNoLB {
		data, err := s.APIClient.GetLoadBalancer(ctx, l.loadBalancerID.String())
		if err != nil {
			if errors.Is(err, lbapi.ErrLBNotfound) {
				// ack and drop msg
				return nil, nil
			}

			s.Logger.Errorw("unable to get loadbalancer from API", "error", err, "loadBalancer", l.loadBalancerID.String())
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())

			return nil, err
		}

		l.lbData = data
	}

	return l, nil
}

func (l *loadBalancer) isLoadBalancer(subj gidx.PrefixedID, adds []gidx.PrefixedID) {
	check, subs := getLBFromAddSubjs(adds)

	switch {
	case subj.Prefix() == LBPrefix:
		l.loadBalancerID = subj
		l.lbType = typeLB

		return
	case check:
		l.loadBalancerID = subs
		l.lbType = typeAssocLB

		return
	default:
		l.lbType = typeNoLB
		return
	}
}

func getLBFromAddSubjs(adds []gidx.PrefixedID) (bool, gidx.PrefixedID) {
	for _, i := range adds {
		if i.Prefix() == LBPrefix {
			return true, i
		}
	}

	id := new(gidx.PrefixedID)

	return false, *id
}
