package srv

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"k8s.io/client-go/rest"
)

// Server holds options for server connectivity and settings
type Server struct {
	Context         context.Context
	StreamName      string
	Logger          *zap.SugaredLogger
	KubeClient      *rest.Config
	JetstreamClient nats.JetStreamContext
	Debug           bool
	Metro           string
	Prefix          string
	ChartPath       string
}

// Run will start the server queue connections and healthcheck endpoints
func (s *Server) Run(ctx context.Context) error {
	subscription, err := s.JetstreamClient.QueueSubscribe(fmt.Sprintf("%s.>", s.Prefix), "loadbalanceroperator-workers", s.MessageHandler, nats.BindStream(s.StreamName))
	if err != nil {
		s.Logger.Errorf("unable to subscribe to queue: %s", err)
		return err
	}

	if err := s.ExposeEndpoint(subscription, viper.GetString("healthcheck-port")); err != nil {
		return err
	}

	return nil
}
