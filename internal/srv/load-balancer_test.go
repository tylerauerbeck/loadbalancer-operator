package srv

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.infratographer.com/x/echox"
	"go.infratographer.com/x/pubsubx"
	"go.uber.org/zap"
	"helm.sh/helm/v3/pkg/chart"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd/api"

	"go.infratographer.com/loadbalanceroperator/internal/utils"
)

func (suite srvTestSuite) TestProcessLoadBalancer() { //nolint:govet
	type testCase struct {
		name        string
		msg         pubsubx.Message
		expectError bool
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
			name:        "create message",
			expectError: false,
			msg: pubsubx.Message{
				EventType:  create,
				SubjectURN: "urn:infratographer:load-balancer:07442309-182E-4498-BC41-EBD679A9A792",
			},
		},
		{
			name:        "create failure",
			expectError: true,
			msg: pubsubx.Message{
				EventType:  create,
				SubjectURN: "thisisnonsense",
			},
		},
		{
			name:        "update message",
			expectError: false,
			msg: pubsubx.Message{
				EventType:  update,
				SubjectURN: "urn:infratographer:load-balancer:07442309-182E-4498-BC41-EBD679A9A792",
			},
		},
		{
			name:        "update failure",
			expectError: true,
			msg: pubsubx.Message{
				EventType:  update,
				SubjectURN: "thisisnonsense",
			},
		},
		{
			name:        "delete message",
			expectError: false,
			msg: pubsubx.Message{
				EventType:  delete,
				SubjectURN: "urn:infratographer:load-balancer:07442309-182E-4498-BC41-EBD679A9A792",
			},
		},
		{
			name:        "delete failure",
			expectError: true,
			msg: pubsubx.Message{
				EventType:  delete,
				SubjectURN: "thisisnonsense",
			},
		},
		{
			name:        "unknown message",
			expectError: true,
			msg: pubsubx.Message{
				EventType:  "unknown",
				SubjectURN: "urn:infratographer:load-balancer:" + uuid.NewString(),
			},
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			s := srv
			err := s.processLoadBalancer(tc.msg)

			if tc.expectError {
				assert.Error(suite.T(), err)
			} else {
				assert.Nil(suite.T(), err)
			}
		})
	}
}

func (suite srvTestSuite) TestProcessLoadBalancerCreate() { //nolint:govet
	type testCase struct {
		name        string
		msg         pubsubx.Message
		cfg         *rest.Config
		chart       *chart.Chart
		expectError bool
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
			name:        "create message",
			expectError: false,
			cfg:         suite.Kubeenv.Config,
			chart:       ch,
			msg: pubsubx.Message{
				EventType:  create,
				SubjectURN: "urn:infratographer:load-balancer:07442309-182E-4498-BC41-EBD679A9A793",
			},
		},
		{
			name:        "failed namespace",
			expectError: true,
			cfg: &rest.Config{
				Host:                "localhost:45678",
				APIPath:             "",
				ContentConfig:       rest.ContentConfig{},
				Username:            "",
				Password:            "",
				BearerToken:         "",
				BearerTokenFile:     "",
				Impersonate:         rest.ImpersonationConfig{},
				AuthProvider:        &api.AuthProviderConfig{},
				AuthConfigPersister: nil,
				ExecProvider:        &api.ExecConfig{},
				TLSClientConfig:     rest.TLSClientConfig{},
				UserAgent:           "",
				DisableCompression:  false,
				Transport:           nil,
				QPS:                 0,
				Burst:               0,
				RateLimiter:         nil,
				WarningHandler:      nil,
				Timeout:             0,
			},
			msg: pubsubx.Message{
				EventType:  create,
				SubjectURN: "urn:infratographer:load-balancer:07442309-182E-4498-BC41-EBD679A9A793",
			},
		},
		{
			name:        "failed deployment",
			expectError: true,
			cfg:         suite.Kubeenv.Config,
			chart:       &chart.Chart{},
			msg: pubsubx.Message{
				EventType:  create,
				SubjectURN: "urn:infratographer:load-balancer:07442309-182E-4498-BC41-EBD679A9A793",
			},
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			s := srv
			s.KubeClient = tc.cfg
			s.Chart = tc.chart
			err := s.processLoadBalancerCreate(tc.msg)

			if tc.expectError {
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
		msg         pubsubx.Message
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
			msg: pubsubx.Message{
				EventType:  delete,
				SubjectURN: "urn:infratographer:load-balancer:07442309-182E-4498-BC41-EBD679A9A794",
			},
		},
		{
			name:        "unable to remove deployment",
			expectError: true,
			cfg:         &rest.Config{},
			chart:       ch,
			msg: pubsubx.Message{
				EventType:  delete,
				SubjectURN: "urn:infratographer:load-balancer:07442309-182E-4498-BC41-EBD679A9A794",
			},
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			s := srv
			s.KubeClient = tc.cfg
			s.Chart = tc.chart
			_ = s.processLoadBalancerCreate(pubsubx.Message{
				EventType:  create,
				SubjectURN: tc.msg.SubjectURN,
			})
			err := s.processLoadBalancerDelete(tc.msg)

			if tc.expectError {
				assert.Error(suite.T(), err)
			} else {
				assert.Nil(suite.T(), err)
			}
		})
	}
}
