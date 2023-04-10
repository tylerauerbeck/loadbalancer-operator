package srv

import (
	"encoding/json"

	"github.com/nats-io/nats.go"

	"go.infratographer.com/x/pubsubx"

	events "go.infratographer.com/loadbalanceroperator/pkg/events/v1alpha1"

	"go.infratographer.com/x/urnx"
)

type valueSet struct {
	helmKey string
	value   string
}

func (s *Server) messageRouter(m *nats.Msg) {
	msg := pubsubx.Message{}
	if err := json.Unmarshal(m.Data, &msg); err != nil {
		s.Logger.Errorw("Unable to process data in message: %s", "error", err)
		return
	}

	actor, err := urnx.Parse(msg.SubjectURN)
	if err != nil {
		// TODO: handle error and requeue or send to dead letter queue
		s.Logger.Errorw("Unable to parse actor URN: %s", "error", err)
		return
	}

	switch actor.ResourceType {
	case loadbalancer:
		if err := s.processLoadBalancer(msg); err != nil {
			s.Logger.Errorw("Unable to process load balancer", "error", err)
		}
	default:
		s.Logger.Errorw("Unknown resource type: %s", "resource_type", actor.ResourceType)
	}
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
