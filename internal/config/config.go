// Package config provides a struct to store the applications config
package config

import (
	"go.infratographer.com/x/echojwtx"
	"go.infratographer.com/x/echox"
	"go.infratographer.com/x/events"
	"go.infratographer.com/x/loggingx"
	"go.infratographer.com/x/otelx"
)

// AppConfig contains the application configuration structure.
var AppConfig struct {
	OIDC    echojwtx.AuthConfig
	Logging loggingx.Config
	Events  EventsConfig
	Server  echox.Config
	Tracing otelx.Config
}

// EventsConfig stores the configuration for a load-balancer-api events config
type EventsConfig struct {
	Subscriber events.SubscriberConfig
}
