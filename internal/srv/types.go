package srv

import (
	lbapi "go.infratographer.com/load-balancer-api/pkg/client"
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
	lbData         *lbapi.LoadBalancer
	lbType         int
}
