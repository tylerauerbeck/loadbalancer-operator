package srv

import (
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"

	"go.infratographer.com/x/gidx"
	"go.infratographer.com/x/pubsubx"
)

func (s *Server) messageRouter(m *nats.Msg) {
	subjString, data := getSubject(m)

	subj, err := gidx.Parse(subjString)
	if err != nil {
		// TODO: handle error and requeue or send to dead letter queue
		s.Logger.Errorw("Unable to parse subject ID: %s", "error", err)
		return
	}

	eventType := m.Header.Get("X-INFRA9-MSG-TYPE")

	switch prefixLookup(subj.Prefix()) {
	case loadbalancer:
		if err := s.processLoadBalancer(eventType, data); err != nil {
			s.Logger.Errorw("Unable to process load balancer", "error", err)
		}
	default:
		s.Logger.Errorw("Unknown resource type: %s", "resource_type", subj.Prefix())
	}
}

func getSubject(m *nats.Msg) (string, interface{}) {
	t := m.Header.Get("X-INFRA9-MSG-TYPE")
	switch t {
	case "change":
		msg := pubsubx.ChangeMessage{}
		if err := json.Unmarshal(m.Data, &msg); err != nil {
			fmt.Println("Error unmarshalling change message")
		}

		return msg.SubjectID.String(), msg
	case "event":
		msg := pubsubx.EventMessage{}
		if err := json.Unmarshal(m.Data, &msg); err != nil {
			fmt.Println("Error unmarshalling event message")
		}

		return msg.SubjectID.String(), msg
	default:
		fmt.Println("Unknown")
		return "", nil
	}
}

func prefixLookup(s string) string {
	switch s {
	case "loadbal":
		return loadbalancer
	case "loadprt":
		return port
	default:
		return ""
	}
}
