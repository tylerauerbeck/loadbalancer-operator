package srv

import (
	"errors"

	"go.infratographer.com/x/gidx"
)

const (

	// Event types
	loadbalancer = "load-balancer"
	port         = "load-balancer-port"

	// Event actions
	create = "create"
	update = "update"
	delete = "delete"
)

var (
	errUnknownEventType = errors.New("unknown event type")
	// errUnableToProcess  = errors.New("unable to process message")
)

type loadBalancer struct {
	loadBalancerID         gidx.PrefixedID
	loadBalancerTenantID   gidx.PrefixedID
	loadBalancerLocationID gidx.PrefixedID
}
