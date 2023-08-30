package srv

import (
	"context"

	"github.com/lestrrat-go/backoff/v2"
	"go.infratographer.com/loadbalancer-manager-haproxy/pkg/lbapi"
	"go.infratographer.com/x/echox"
	"go.infratographer.com/x/events"
	"go.uber.org/zap"
	"helm.sh/helm/v3/pkg/chart"
	"k8s.io/client-go/rest"

	"go.infratographer.com/ipam-api/pkg/ipamclient"
)

// instrumentationName is a unique package name used for tracing
const instrumentationName = "go.infratographer.com/loadbalanceroperator/srv"

// Server holds options for server connectivity and settings
type Server struct {
	APIClient        *lbapi.Client
	BackoffConfig    backoff.Policy
	IPAMClient       *ipamclient.Client
	Echo             *echox.Server
	Context          context.Context
	EventsConnection events.Connection
	eventChannels    []<-chan events.Message[events.EventMessage]
	changeChannels   []<-chan events.Message[events.ChangeMessage]
	Logger           *zap.SugaredLogger
	KubeClient       *rest.Config
	Debug            bool
	EventTopics      []string
	ChangeTopics     []string
	Chart            *chart.Chart
	ChartPath        string
	ValuesPath       string
	Locations        []string
	ServicePortKey   string
	ContainerPortKey string
	MetricsPort      int
}

// Run will start the server queue connections and healthcheck endpoints
func (s *Server) Run(ctx context.Context) error {
	s.Echo.AddHandler(s)

	go func() {
		if err := s.Echo.Run(); err != nil {
			s.Logger.Error("unable to start healthcheck server", zap.Error(err))
		}
	}()

	s.Logger.Infow("starting subscribers")

	if err := s.configureSubscribers(ctx); err != nil {
		s.Logger.Errorw("unable to configure subscribers", "error", err)
		return err
	}

	for _, ch := range s.changeChannels {
		go s.listenChange(ch)
	}

	for _, ev := range s.eventChannels {
		go s.listenEvent(ev)
	}

	return nil
}

func (s *Server) Shutdown() error {
	if err := s.EventsConnection.Shutdown(s.Context); err != nil {
		s.Logger.Debugw("Unable to shutdown connection", "error", err)
		return err
	}

	return nil
}

func (s *Server) configureSubscribers(ctx context.Context) error {
	var ch []<-chan events.Message[events.ChangeMessage]

	var ev []<-chan events.Message[events.EventMessage]

	for _, topic := range s.ChangeTopics {
		s.Logger.Debugw("subscribing to change topic", "topic", topic)

		c, err := s.EventsConnection.SubscribeChanges(ctx, topic)
		if err != nil {
			s.Logger.Errorw("unable to subscribe to change topic", "error", err, "topic", topic, "type", "change")
			return errSubscriptionCreate
		}

		ch = append(ch, c)
	}

	for _, topic := range s.EventTopics {
		s.Logger.Debugw("subscribing to event topic", "topic", topic)

		e, err := s.EventsConnection.SubscribeEvents(ctx, topic)
		if err != nil {
			s.Logger.Errorw("unable to subscribe to event topic", "error", err, "topic", topic, "type", "event")
			return errSubscriptionCreate
		}

		ev = append(ev, e)
	}

	s.changeChannels = ch
	s.eventChannels = ev

	return nil
}
