package srv

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"
	"go.infratographer.com/loadbalancer-manager-haproxy/pkg/lbapi"
	"go.infratographer.com/x/echox"
	"go.infratographer.com/x/events"
	"go.uber.org/zap"
	"helm.sh/helm/v3/pkg/chart"
	"k8s.io/client-go/rest"
)

// Server holds options for server connectivity and settings
type Server struct {
	APIClient        *lbapi.Client
	Echo             *echox.Server
	Context          context.Context
	eventChannels    []<-chan *message.Message
	changeChannels   []<-chan *message.Message
	Logger           *zap.SugaredLogger
	KubeClient       *rest.Config
	SubscriberConfig events.SubscriberConfig
	Debug            bool
	Topics           []string
	Chart            *chart.Chart
	ChartPath        string
	ValuesPath       string
	Locations        []string
	ServicePortKey   string
	ContainerPortKey string
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

	if err := s.configureSubscribers(); err != nil {
		s.Logger.Errorw("unable to configure subscribers", "error", err)
		return err
	}

	for _, ch := range s.changeChannels {
		go s.processChange(ch)
	}

	for _, ev := range s.eventChannels {
		go s.processEvent(ev)
	}

	return nil
}

func (s *Server) configureSubscribers() error {
	var ev, ch []<-chan *message.Message

	for _, topic := range s.Topics {
		s.Logger.Debugw("subscribing to topic", "topic", topic)

		csub, err := events.NewSubscriber(s.SubscriberConfig)
		if err != nil {
			s.Logger.Errorw("unable to create change subscriber", "error", err, "topic", topic)
			return errSubscriberCreate
		}

		c, err := csub.SubscribeChanges(s.Context, topic)
		if err != nil {
			s.Logger.Errorw("unable to subscribe to change topic", "error", err, "topic", topic, "type", "change")
			return errSubscriptionCreate
		}

		ch = append(ch, c)

		esub, err := events.NewSubscriber(s.SubscriberConfig)
		if err != nil {
			s.Logger.Errorw("unable to create event subscriber", "error", err, "topic", topic)
			return errSubscriberCreate
		}

		e, err := esub.SubscribeEvents(s.Context, topic)
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
