package srv

import (
	"context"
	"os"
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.infratographer.com/x/echox"
	"go.infratographer.com/x/pubsubx"
	"go.uber.org/zap"
	"helm.sh/helm/v3/pkg/chart"
	"k8s.io/client-go/rest"

	"go.infratographer.com/loadbalanceroperator/internal/utils"
)

func (suite srvTestSuite) TestProcessLoadBalancer() { //nolint:govet
	type testCase struct {
		name           string
		msg            interface{}
		expectedErrors []error
		eventType      string
	}

	js := utils.GetJetstreamConnection(suite.NATSServer)

	_, _ = js.AddStream(&nats.StreamConfig{
		Name:     "TestProcessLB",
		Subjects: []string{"plb.foo", "plb.bar"},
		MaxBytes: 1024,
	})

	dir, cp, ch, pwd := utils.CreateWorkspace("test-process-lb")
	defer os.RemoveAll(dir)

	eSrv, err := echox.NewServer(zap.NewNop(), echox.Config{}, nil)

	require.NoError(suite.T(), err, "unexpected error creating new server")

	srv := Server{
		Echo:            eSrv,
		Context:         context.TODO(),
		StreamName:      "TestProcessLB",
		Logger:          zap.NewNop().Sugar(),
		KubeClient:      suite.Kubeenv.Config,
		JetstreamClient: js,
		Debug:           false,
		Prefix:          "plb",
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
				SubjectID: "loadbal-mzxcvxcv",
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
				SubjectID: "loadbal-mzxcvxcv",
			},
		},
		{
			name:           "process event message",
			expectedErrors: nil,
			eventType:      "event",
			msg: pubsubx.EventMessage{
				EventType: "create",
				SubjectID: "loadbal-mzxcvxcv",
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
				SubjectID: "loadbal-mzxcvxcv",
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
			err := s.processLoadBalancer(tc.eventType, tc.msg)

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

func (suite srvTestSuite) TestProcessLoadBalancerChange() { //nolint:govet
	type testCase struct {
		name           string
		msg            pubsubx.ChangeMessage
		expectedErrors []error
		eventType      string
	}

	js := utils.GetJetstreamConnection(suite.NATSServer)

	_, _ = js.AddStream(&nats.StreamConfig{
		Name:     "TestProcessLBChange",
		Subjects: []string{"plbc.foo", "plbc.bar"},
		MaxBytes: 1024,
	})

	dir, cp, ch, pwd := utils.CreateWorkspace("test-process-lb")
	defer os.RemoveAll(dir)

	eSrv, err := echox.NewServer(zap.NewNop(), echox.Config{}, nil)

	require.NoError(suite.T(), err, "unexpected error creating new server")

	srv := Server{
		Echo:            eSrv,
		Context:         context.TODO(),
		StreamName:      "TestProcessLBChange",
		Logger:          zap.NewNop().Sugar(),
		KubeClient:      suite.Kubeenv.Config,
		JetstreamClient: js,
		Debug:           false,
		Prefix:          "plbc",
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
				SubjectID: "loadbal-asasasasasa",
			},
		},
		{
			name:           "process long namespace",
			expectedErrors: []error{errInvalidObjectNameLength},
			eventType:      "change",
			msg: pubsubx.ChangeMessage{
				EventType: create,
				SubjectID: "loadbal-reallyreallyreallyreallyreallyreallylongreallylong",
			},
		},
		{
			name:           "process change update",
			expectedErrors: nil,
			eventType:      "change",
			msg: pubsubx.ChangeMessage{
				EventType: "update",
				SubjectID: "loadbal-lkjasdlfkj",
			},
		},
		{
			name:           "process change delete",
			expectedErrors: nil,
			eventType:      "change",
			msg: pubsubx.ChangeMessage{
				EventType: "delete",
				SubjectID: "loadbal-mzxcvxcv",
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

func (suite srvTestSuite) TestProcessLoadBalancerChangeCreate() { //nolint:govet
	type testCase struct {
		name           string
		msg            pubsubx.ChangeMessage
		cfg            *rest.Config
		chart          *chart.Chart
		expectedErrors []error
	}

	js := utils.GetJetstreamConnection(suite.NATSServer)

	_, _ = js.AddStream(&nats.StreamConfig{
		Name:     "TestCreateLB",
		Subjects: []string{"clb.foo", "clb.bar"},
		MaxBytes: 1024,
	})

	dir, cp, ch, pwd := utils.CreateWorkspace("test-create-lb")
	defer os.RemoveAll(dir)

	eSrv, err := echox.NewServer(zap.NewNop(), echox.Config{}, nil)

	require.NoError(suite.T(), err, "unexpected error creating new server")

	srv := Server{
		Echo:            eSrv,
		Context:         context.TODO(),
		StreamName:      "TestCreateLB",
		Logger:          zap.NewNop().Sugar(),
		JetstreamClient: js,
		Debug:           false,
		Prefix:          "clb",
		Subjects:        []string{"foo", "bar"},
		Subscriptions:   []*nats.Subscription{},
		ChartPath:       cp,
		ValuesPath:      pwd + "/../../hack/ci/values.yaml",
	}

	testCases := []testCase{
		{
			name:           "create message",
			expectedErrors: nil,
			cfg:            suite.Kubeenv.Config,
			chart:          ch,
			msg: pubsubx.ChangeMessage{
				EventType: create,
				SubjectID: "loadbal-lkjasdlfkjasdf",
			},
		},
		{
			name:           "fail create long namespace",
			expectedErrors: []error{errInvalidObjectNameLength},
			msg: pubsubx.ChangeMessage{
				EventType: create,
				SubjectID: "loadbal-reallyreallyreallyreallyreallyreallylongreallylong",
			},
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			s := srv
			s.KubeClient = tc.cfg
			s.Chart = tc.chart
			err := s.processLoadBalancerChangeCreate(tc.msg)

			if len(tc.expectedErrors) > 0 {
				assert.Error(suite.T(), err)
			} else {
				assert.Nil(suite.T(), err)
			}
		})
	}
}

func (suite srvTestSuite) TestProcessLoadBalancerDelete() { //nolint:govet
	type testCase struct {
		name        string
		msg         pubsubx.ChangeMessage
		cfg         *rest.Config
		chart       *chart.Chart
		expectError bool
	}

	js := utils.GetJetstreamConnection(suite.NATSServer)

	_, _ = js.AddStream(&nats.StreamConfig{
		Name:     "TestDeleteLB",
		Subjects: []string{"dlb.foo", "dlb.bar"},
		MaxBytes: 1024,
	})

	dir, cp, ch, pwd := utils.CreateWorkspace("test-delete-lb")
	defer os.RemoveAll(dir)

	eSrv, err := echox.NewServer(zap.NewNop(), echox.Config{}, nil)

	require.NoError(suite.T(), err, "unexpected error creating new server")

	srv := Server{
		Echo:       eSrv,
		Context:    context.TODO(),
		StreamName: "TestDeleteLB",
		Logger:     zap.NewNop().Sugar(),
		// KubeClient:      suite.Kubeenv.Config,
		JetstreamClient: js,
		Debug:           false,
		Prefix:          "dlb",
		Subjects:        []string{"foo", "bar"},
		Subscriptions:   []*nats.Subscription{},
		// Chart:           ch,
		ChartPath:  cp,
		ValuesPath: pwd + "/../../hack/ci/values.yaml",
	}

	testCases := []testCase{
		{
			name:        "delete lb",
			expectError: false,
			cfg:         suite.Kubeenv.Config,
			chart:       ch,
			msg: pubsubx.ChangeMessage{
				EventType: delete,
				SubjectID: "loadbal-oiuqweroiu",
			},
		},
		{
			name:        "unable to remove deployment",
			expectError: true,
			cfg:         &rest.Config{},
			chart:       ch,
			msg: pubsubx.ChangeMessage{
				EventType: delete,
				SubjectID: "loadbal-kljasdlkfjasdf",
			},
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			s := srv
			s.KubeClient = tc.cfg
			s.Chart = tc.chart
			_ = s.processLoadBalancerChangeCreate(pubsubx.ChangeMessage{
				EventType: create,
				SubjectID: tc.msg.SubjectID,
			})
			err := s.processLoadBalancerChangeDelete(tc.msg)

			if tc.expectError {
				assert.Error(suite.T(), err)
			} else {
				assert.Nil(suite.T(), err)
			}
		})
	}
}

func (suite srvTestSuite) TestProcessLoadBalancerEvent() { //nolint:govet
	type testCase struct {
		name           string
		msg            pubsubx.EventMessage
		expectedErrors []error
		eventType      string
	}

	js := utils.GetJetstreamConnection(suite.NATSServer)

	_, _ = js.AddStream(&nats.StreamConfig{
		Name:     "TestProcessLBEvent",
		Subjects: []string{"plbe.foo", "plbe.bar"},
		MaxBytes: 1024,
	})

	dir, cp, ch, pwd := utils.CreateWorkspace("test-process-lb-event")
	defer os.RemoveAll(dir)

	eSrv, err := echox.NewServer(zap.NewNop(), echox.Config{}, nil)

	require.NoError(suite.T(), err, "unexpected error creating new server")

	srv := Server{
		Echo:            eSrv,
		Context:         context.TODO(),
		StreamName:      "TestProcessLBEvent",
		Logger:          zap.NewNop().Sugar(),
		KubeClient:      suite.Kubeenv.Config,
		JetstreamClient: js,
		Debug:           false,
		Prefix:          "plbe",
		Subjects:        []string{"foo", "bar"},
		Subscriptions:   []*nats.Subscription{},
		Chart:           ch,
		ChartPath:       cp,
		ValuesPath:      pwd + "/../../hack/ci/values.yaml",
	}

	testCases := []testCase{
		{
			name:           "process event create",
			expectedErrors: nil,
			eventType:      "event",
			msg: pubsubx.EventMessage{
				EventType: create,
				SubjectID: "loadbal-asasasasasa",
			},
		},
		{
			name:           "process event update",
			expectedErrors: nil,
			eventType:      "event",
			msg: pubsubx.EventMessage{
				EventType: "update",
				SubjectID: "loadbal-lkjasdlfkj",
			},
		},
		{
			name:           "process event delete",
			expectedErrors: nil,
			eventType:      "event",
			msg: pubsubx.EventMessage{
				EventType: "delete",
				SubjectID: "loadbal-mzxcvxcv",
			},
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			s := srv
			err := s.processLoadBalancerEvent(tc.msg)

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
