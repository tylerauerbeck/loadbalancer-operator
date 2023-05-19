package srv

import (
	"context"
	"os"
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.infratographer.com/loadbalanceroperator/internal/utils"
	"go.infratographer.com/x/echox"
	"go.infratographer.com/x/pubsubx"
	"go.uber.org/zap"
)

func (suite srvTestSuite) TestProcessLoadBalancerPort() { //nolint:govet
	type testCase struct {
		name           string
		msg            interface{}
		expectedErrors []error
		eventType      string
	}

	js := utils.GetJetstreamConnection(suite.NATSServer)

	_, _ = js.AddStream(&nats.StreamConfig{
		Name:     "TestProcessLBPort",
		Subjects: []string{"plbp.foo", "plbp.bar"},
		MaxBytes: 1024,
	})

	dir, cp, ch, pwd := utils.CreateWorkspace("test-process-lb-port")
	defer os.RemoveAll(dir)

	eSrv, err := echox.NewServer(zap.NewNop(), echox.Config{}, nil)

	require.NoError(suite.T(), err, "unexpected error creating new server")

	srv := Server{
		Echo:            eSrv,
		Context:         context.TODO(),
		StreamName:      "TestProcessLBPort",
		Logger:          zap.NewNop().Sugar(),
		KubeClient:      suite.Kubeenv.Config,
		JetstreamClient: js,
		Debug:           false,
		Prefix:          "plbp",
		Subjects:        []string{"foo", "bar"},
		Subscriptions:   []*nats.Subscription{},
		Chart:           ch,
		ChartPath:       cp,
		ValuesPath:      pwd + "/../../hack/ci/values.yaml",
	}

	testCases := []testCase{
		{
			name:           "process change message",
			expectedErrors: nil,
			eventType:      "change",
			msg: pubsubx.ChangeMessage{
				EventType: create,
				SubjectID: "loadprt-mzxcvxcv",
			},
		},
		{
			name:           "process change failure",
			expectedErrors: []error{errUnknownEventType},
			eventType:      "change",
			msg: pubsubx.ChangeMessage{
				EventType: "unknown",
				SubjectID: "thisisnonsense",
			},
		},
		{
			name:           "process change mismatch",
			expectedErrors: []error{errMessageTypeMismatch},
			eventType:      "change",
			msg: pubsubx.EventMessage{
				EventType: "create",
				SubjectID: "loadprt-mzxcvxcv",
			},
		},
		{
			name:           "process event message",
			expectedErrors: nil,
			eventType:      "event",
			msg: pubsubx.EventMessage{
				EventType: "create",
				SubjectID: "loadprt-mzxcvxcv",
			},
		},
		{
			name:           "process event failure",
			expectedErrors: []error{errUnknownEventType},
			eventType:      "event",
			msg: pubsubx.EventMessage{
				EventType: "unknown",
				SubjectID: "thisisnonsense",
			},
		},
		{
			name:           "process event mismatch",
			expectedErrors: []error{errMessageTypeMismatch},
			eventType:      "event",
			msg: pubsubx.ChangeMessage{
				EventType: "create",
				SubjectID: "loadprt-mzxcvxcv",
			},
		},
		{
			name:           "process unknown message",
			expectedErrors: []error{errUnknownMessageEventType},
			eventType:      "unknown",
			msg: pubsubx.EventMessage{
				EventType: "unknown",
				SubjectID: "thisisnonsense",
			},
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			s := srv
			err := s.processLoadBalancerPort(tc.eventType, tc.msg)

			if len(tc.expectedErrors) > 0 {
				assert.Error(suite.T(), err)
				for _, e := range tc.expectedErrors {
					assert.ErrorIs(suite.T(), err, e)
				}
			} else {
				assert.Nil(suite.T(), err)
			}
		})
	}
}

func (suite srvTestSuite) TestProcessLoadBalancerPortChange() { //nolint:govet
	type testCase struct {
		name           string
		msg            pubsubx.ChangeMessage
		expectedErrors []error
		eventType      string
	}

	js := utils.GetJetstreamConnection(suite.NATSServer)

	_, _ = js.AddStream(&nats.StreamConfig{
		Name:     "TestProcessLBPortChange",
		Subjects: []string{"plbpc.foo", "plbpc.bar"},
		MaxBytes: 1024,
	})

	dir, cp, ch, pwd := utils.CreateWorkspace("test-process-lb-port-change")
	defer os.RemoveAll(dir)

	eSrv, err := echox.NewServer(zap.NewNop(), echox.Config{}, nil)

	require.NoError(suite.T(), err, "unexpected error creating new server")

	srv := Server{
		Echo:            eSrv,
		Context:         context.TODO(),
		StreamName:      "TestProcessLBPortChange",
		Logger:          zap.NewNop().Sugar(),
		KubeClient:      suite.Kubeenv.Config,
		JetstreamClient: js,
		Debug:           false,
		Prefix:          "plbpc",
		Subjects:        []string{"foo", "bar"},
		Subscriptions:   []*nats.Subscription{},
		Chart:           ch,
		ChartPath:       cp,
		ValuesPath:      pwd + "/../../hack/ci/values.yaml",
	}

	testCases := []testCase{
		{
			name:           "process change create",
			expectedErrors: nil,
			eventType:      "change",
			msg: pubsubx.ChangeMessage{
				EventType: create,
				SubjectID: "loadprt-asasasasasa",
			},
		},
		{
			name:           "process change update",
			expectedErrors: nil,
			eventType:      "change",
			msg: pubsubx.ChangeMessage{
				EventType: "update",
				SubjectID: "loadprt-lkjasdlfkj",
			},
		},
		{
			name:           "process change delete",
			expectedErrors: nil,
			eventType:      "change",
			msg: pubsubx.ChangeMessage{
				EventType: "delete",
				SubjectID: "loadprt-mzxcvxcv",
			},
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			s := srv
			err := s.processLoadBalancerChange(tc.msg)

			if len(tc.expectedErrors) > 0 {
				assert.Error(suite.T(), err)
				for _, e := range tc.expectedErrors {
					assert.ErrorIs(suite.T(), err, e)
				}
			} else {
				assert.Nil(suite.T(), err)
			}
		})
	}
}
