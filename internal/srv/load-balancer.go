package srv

import (
	"go.infratographer.com/x/gidx"
)

func (s *Server) newLoadBalancer(subj gidx.PrefixedID, adds []gidx.PrefixedID) (*loadBalancer, error) {
	l := new(loadBalancer)
	l.isLoadBalancer(subj, adds)

	if l.lbType != typeNoLB {
		data, err := s.APIClient.GetLoadBalancer(s.Context, l.loadBalancerID.String())
		if err != nil {
			s.Logger.Errorw("unable to get loadbalancer from API", "error", err)
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
