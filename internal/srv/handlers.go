package srv

import (
	"encoding/json"

	"github.com/nats-io/nats.go"
	"github.com/spf13/viper"

	"go.infratographer.com/x/pubsubx"

	events "go.infratographer.com/loadbalanceroperator/pkg/events/v1alpha1"

	"go.infratographer.com/x/urnx"
)

type valueSet struct {
	helmKey string
	value   string
}

// MessageHandler handles the routing of events from specified queues
func (s *Server) MessageHandler(m *nats.Msg) {
	msg := pubsubx.Message{}
	if err := json.Unmarshal(m.Data, &msg); err != nil {
		s.Logger.Errorw("Unable to process data in message: %s", "error", err)
	}

	switch msg.EventType {
	case events.EVENTCREATE:
		if err := s.createMessageHandler(&msg); err != nil {
			s.Logger.Errorw("unable to process create: %s", "error", err)
		}
	case events.EVENTUPDATE:
		if err := s.updateMessageHandler(&msg); err != nil {
			s.Logger.Errorw("unable to process update", "error", err.Error())
		}
	case events.EVENTDELETE:
		if err := s.deleteMessageHandler(&msg); err != nil {
			s.Logger.Errorw("unable to process delete", "error", err.Error())
		}
	default:
		s.Logger.Debug("This is some other set of queues that we don't know about.")
	}
}

func (s *Server) createMessageHandler(m *pubsubx.Message) error {
	lbdata := events.LoadBalancerData{}

	lbURN, err := urnx.Parse(m.SubjectURN)
	if err != nil {
		s.Logger.Errorw("unable to parse load-balancer URN", "error", err)
		return err
	}

	if err := s.parseLBData(&m.AdditionalData, &lbdata); err != nil {
		s.Logger.Errorw("handler unable to parse loadbalancer data", "error", err)
		return err
	}

	if _, err := s.CreateNamespace(lbURN.ResourceID.String()); err != nil {
		s.Logger.Errorw("handler unable to create required namespace", "error", err)
		return err
	}

	overrides := []valueSet{}
	for _, cpuFlag := range viper.GetStringSlice("helm-cpu-flag") {
		overrides = append(overrides, valueSet{
			helmKey: cpuFlag,
			value:   lbdata.Resources.CPU,
		})
	}

	for _, memFlag := range viper.GetStringSlice("helm-memory-flag") {
		overrides = append(overrides, valueSet{
			helmKey: memFlag,
			value:   lbdata.Resources.Memory,
		})
	}

	if err := s.newDeployment(lbURN.ResourceID.String(), overrides); err != nil {
		s.Logger.Errorw("handler unable to create loadbalancer", "error", err)
		return err
	}

	return nil
}

func (s *Server) deleteMessageHandler(m *pubsubx.Message) error {
	lbURN, err := urnx.Parse(m.SubjectURN)
	if err != nil {
		s.Logger.Errorw("unable to parse load-balancer URN", "error", err)
		return err
	}

	if err := s.removeDeployment(lbURN.ResourceID.String()); err != nil {
		s.Logger.Errorw("handler unable to delete loadbalancer", "error", err)
		return err
	}

	return nil
}

func (s *Server) updateMessageHandler(m *pubsubx.Message) error {
	return nil
}

func (s *Server) parseLBData(data *map[string]interface{}, lbdata *events.LoadBalancerData) error {
	d, err := json.Marshal(data)
	if err != nil {
		s.Logger.Errorw("unable to load data from event", "error", err.Error())
		return err
	}

	if err := json.Unmarshal(d, &lbdata); err != nil {
		s.Logger.Errorw("unable to parse event data", "error", err.Error())
		return err
	}

	return nil
}
