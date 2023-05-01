package srv

import "errors"

var (
	// ErrPortsRequired is returned when a healthcheck port has not been provided
	ErrPortsRequired = errors.New("no ports provided")

	// errNoAssocLB is returned when an associated load balancer cannot be found in the message
	errNoAssocLB = errors.New("no associated load balancer found in message")
)
