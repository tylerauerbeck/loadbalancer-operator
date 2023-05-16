package srv

import "errors"

var (
	// ErrPortsRequired is returned when a healthcheck port has not been provided
	ErrPortsRequired           = errors.New("no ports provided")
	errUnknownMessageEventType = errors.New("unknown message event type")
	errMessageTypeMismatch     = errors.New("message type does not match event type")
	errInvalidObjectNameLength = errors.New("object name must be less than 53 characters")
)
