package srv

import "errors"

var (
	// ErrPortsRequired is returned when a healthcheck port has not been provided
	ErrPortsRequired = errors.New("no ports provided")
	// errInactiveSubscription is returned when a NATS subscription is not valid
	errInactiveSubscription = errors.New("inactive subscription")
)
