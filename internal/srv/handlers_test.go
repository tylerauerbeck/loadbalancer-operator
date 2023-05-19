package srv

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"

	"go.infratographer.com/x/pubsubx"

	"go.infratographer.com/loadbalanceroperator/internal/utils"
)

func (suite srvTestSuite) TestMessageRouter() { //nolint:govet
	type testCase struct {
		name        string
		msg         interface{}
		eventType   string
		expectError bool
	}

	js := utils.GetJetstreamConnection(suite.NATSServer)

	_, _ = js.AddStream(&nats.StreamConfig{
		Name:     "TestMessageRouter",
		Subjects: []string{"mr.foo", "mr.bar"},
		MaxBytes: 1024,
	})

	dir, cp, ch, pwd := utils.CreateWorkspace("test-message-router")
	defer os.RemoveAll(dir)

	srv := Server{
		Context:         context.TODO(),
		StreamName:      "TestMessageRouter",
		Logger:          zap.NewNop().Sugar(),
		KubeClient:      suite.Kubeenv.Config,
		JetstreamClient: js,
		Debug:           false,
		Prefix:          "mr",
		Subjects:        []string{"foo", "bar"},
		Subscriptions:   []*nats.Subscription{},
		Chart:           ch,
		ChartPath:       cp,
		ValuesPath:      pwd + "/../../hack/ci/values.yaml",
	}

	testCases := []testCase{
		{
			name:        "test change",
			expectError: false,
			eventType:   "change",
			msg: pubsubx.ChangeMessage{
				SubjectID: "loadbal-lkjasdlfkjad",
				EventType: "create",
			},
		},
		{
			name:        "test event",
			expectError: false,
			eventType:   "event",
			msg: pubsubx.EventMessage{
				SubjectID: "loadbal-kjasdlkjf",
				EventType: "create",
			},
		},
		{
			name:        "test unknown subject",
			expectError: false,
			eventType:   "change",
			msg: pubsubx.ChangeMessage{
				SubjectID: "unknown-kljsdlfkj",
				EventType: "unknown",
			},
		},
		{
			name:      "unknown event type",
			eventType: "somethingunknown",
			msg: pubsubx.ChangeMessage{
				SubjectID: "loadbal-lkjadlfjk",
				EventType: "create",
			},
		},
		{
			name:      "test port",
			eventType: "event",
			msg: pubsubx.EventMessage{
				SubjectID: "loadprt-kjasdlkjf",
				EventType: "create",
			},
		},
		{
			name:      "bad port",
			eventType: "change",
			msg: pubsubx.ChangeMessage{
				SubjectID: "loadprt-lkjasdlfkj",
				EventType: "unknown",
			},
		},
		{
			name:      "bad lb",
			eventType: "event",
			msg: pubsubx.EventMessage{
				SubjectID: "loadbal-lkjasdflkj",
				EventType: "unknown",
			},
		},
	}

	for _, tcase := range testCases {
		suite.T().Run(tcase.name, func(t *testing.T) {
			msgstr, _ := json.Marshal(tcase.msg)
			nmsg := nats.Msg{
				Data:   []byte(string(msgstr)),
				Header: make(nats.Header),
			}

			nmsg.Header.Set("X-INFRA9-MSG-TYPE", tcase.eventType)
			srv.messageRouter(&nmsg)
		})
	}
}

func (suite srvTestSuite) TestPrefixLookup() { //nolint:govet
	type testCase struct {
		name   string
		prefix string
		value  string
	}

	testCases := []testCase{
		{
			name:   "load-balancer lookup",
			prefix: "loadbal",
			value:  loadbalancer,
		},
		{
			name:   "load-balancer-port lookup",
			prefix: "loadprt",
			value:  port,
		},
		{
			name:   "unknown prefix",
			prefix: "unknown",
			value:  "",
		},
	}

	for _, tcase := range testCases {
		suite.T().Run(tcase.name, func(t *testing.T) {

			subject := PrefixLookup(tcase.prefix)
			suite.Equal(tcase.value, subject)
		})
	}

}
