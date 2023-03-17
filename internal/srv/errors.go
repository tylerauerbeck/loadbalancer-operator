package srv

import "errors"

var (
	// ErrPortsRequired is returned when a healthcheck port has not been provided
	ErrPortsRequired = errors.New("no ports provided")
)
