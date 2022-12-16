package srv

import (
	"encoding/json"
	"net/http"

	"github.com/nats-io/nats.go"

	"go.infratographer.com/x/pubsubx"

	events "go.infratographer.com/loadbalanceroperator/pkg/events/v1alpha1"
)

// MessageHandler handles the routing of events from specified queues
func (s *Server) MessageHandler(m *nats.Msg) {
	msg := pubsubx.Message{}
	if err := json.Unmarshal(m.Data, &msg); err != nil {
		s.Logger.Errorln("Unable to process data in message: %s", err)
	}

	switch msg.EventType {
	case events.EVENTCREATE:
		if err := s.createMessageHandler(&msg); err != nil {
			s.Logger.Errorf("unable to process create: %s", err)
		}
	case events.EVENTUPDATE:
		err := s.updateMessageHandler(&msg)
		if err != nil {
			s.Logger.Errorf("unable to process update: %s", err)
		}
	default:
		s.Logger.Debug("This is some other set of queues that we don't know about.")
	}
}

func (s *Server) createMessageHandler(m *pubsubx.Message) error {
	lbdata := events.LoadBalancerData{}

	if err := parseLBData(&m.AdditionalData, &lbdata); err != nil {
		return err
	}

	if err := s.CreateNamespace(m.SubjectURN); err != nil {
		return err
	}

	if err := s.CreateApp(lbdata.LoadBalancerID.String(), s.ChartPath, m.SubjectURN); err != nil {
		return err
	}

	return nil
}

func (s *Server) updateMessageHandler(m *pubsubx.Message) error {
	return nil
}

// ExposeEndpoint exposes a specified port for various checks
func (s *Server) ExposeEndpoint(subscription *nats.Subscription, port string) error {
	if port == "" {
		return ErrPortsRequired
	}

	go func() {
		s.Logger.Infof("Starting endpoints on %s", port)

		checkConfig := http.NewServeMux()
		checkConfig.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("ok"))
		})
		checkConfig.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
			if !subscription.IsValid() {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte("500 - Queue subscription is inactive"))
			} else {
				_, _ = w.Write([]byte("ok"))
			}
		})

		checks := http.Server{
			Handler: checkConfig,
			Addr:    port,
		}

		_ = checks.ListenAndServe()
	}()

	return nil
}

func parseLBData(data *map[string]interface{}, lbdata *events.LoadBalancerData) error {
	d, err := json.Marshal(data)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(d, &lbdata); err != nil {
		return err
	}

	return nil
}
