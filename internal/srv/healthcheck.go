package srv

import (
	"context"

	"github.com/nats-io/nats.go"
)

func subscriptionReadiness(sub []*nats.Subscription) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		for _, s := range sub {
			if !s.IsValid() {
				return errInactiveSubscription
			}
		}

		return nil
	}
}

func (s *Server) configureHealthcheck() error {
	s.Gin.AddReadinessCheck("nats", subscriptionReadiness(s.Subscriptions))

	return nil
}
