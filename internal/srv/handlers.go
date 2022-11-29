package srv

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/Moby/Moby/pkg/namesgenerator"
	"github.com/nats-io/nats.go"
)

// MessageHandler handles the routing of events from specified queues
func (s *Server) MessageHandler(m *nats.Msg) {
	switch m.Subject {
	case fmt.Sprintf("%s.create", s.Prefix):
		err := s.createMessageHandler(m)
		if err != nil {
			s.Logger.Errorln("unable to process create")
			// Redeliver message
			if err := m.Nak(); err != nil {
				s.Logger.Errorln("unable to process message", err)
			}
		}
	case fmt.Sprintf("%s.update", s.Prefix):
		err := s.updateMessageHandler(m)
		if err != nil {
			s.Logger.Errorln("unable to process update")
			// Redeliver message
			if err := m.Nak(); err != nil {
				s.Logger.Errorln("unable to process message", err)
			}
		}
	default:
		s.Logger.Debug("This is some other set of queues that we don't know about.")
	}
}

func (s *Server) createMessageHandler(m *nats.Msg) error {
	name := strings.ReplaceAll(namesgenerator.GetRandomName(0), "_", "-")

	if err := s.CreateNamespace(string(m.Data)); err != nil {
		return err
	}

	if err := s.CreateApp(name, s.ChartPath, string(m.Data)); err != nil {
		return err
	}

	return nil
}

func (s *Server) updateMessageHandler(m *nats.Msg) error {
	s.Logger.Infoln("updating")
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
