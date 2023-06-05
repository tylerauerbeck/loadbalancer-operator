package srv

import (
	"go.infratographer.com/loadbalancer-manager-haproxy/pkg/lbapi"
	"go.infratographer.com/x/gidx"
)

const (
	LBPrefix = "loadbal"

	typeLB      = 1
	typeAssocLB = 2
	typeNoLB    = 0
)

type loadBalancer struct {
	loadBalancerID gidx.PrefixedID
	lbData         *lbapi.GetLoadBalancer
	lbType         int
}
