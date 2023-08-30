package srv

import (
	"errors"
	"fmt"
	"math/rand"
	"time"

	lock "github.com/viney-shih/go-lock"
	"go.infratographer.com/loadbalancer-manager-haproxy/pkg/lbapi"
	"go.infratographer.com/x/gidx"
)

func (s *Server) newLoadBalancer(subj gidx.PrefixedID, adds []gidx.PrefixedID, timestamp time.Time) (*loadBalancer, error) {
	l := new(loadBalancer)
	l.isLoadBalancer(subj, adds)

	time.Sleep(time.Duration(rand.Int63n(int64(2*time.Second))) / 2)
	// check

	lock := s.checkLock(l.loadBalancerID.String())

	ok := lock.lock.TryLockWithTimeout(1 * time.Minute)

	if !ok || !timestamp.After(lock.Timestamp) {
		fmt.Println("something bad happened")
		// need to nak or error or whatevs
		// return specific error so we can say something different
	} else {
		lock.lock.Lock()
	}

	if l.lbType != typeNoLB {
		data, err := s.APIClient.GetLoadBalancer(s.Context, l.loadBalancerID.String())
		if err != nil {
			if errors.Is(err, lbapi.ErrLBNotfound) {
				// ack and drop msg
				return nil, nil
			}

			s.Logger.Errorw("unable to get loadbalancer from API", "error", err, "loadBalancer", l.loadBalancerID.String())

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

func (s *Server) checkLock(id string) *lbLock {
	lk, ok := s.loadbalancers[id]
	if !ok {
		lock := lock.NewCASMutex()
		s.loadbalancers[id] = lbLock{lock: lock}

		return &lk
	}

	return &lk
}
