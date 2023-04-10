package srv

import "errors"

const (

	// Event types
	loadbalancer = "load-balancer"
	// pools        = "pools"
	// ports        = "ports"

	// Event actions
	create = "create"
	update = "update"
	delete = "delete"
)

var (
	errUnknownEventType = errors.New("unknown event type")
	// errUnableToProcess  = errors.New("unable to process message")
)
