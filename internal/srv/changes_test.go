package srv

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.infratographer.com/loadbalancer-manager-haproxy/pkg/lbapi"
	"go.infratographer.com/x/echox"
	"go.infratographer.com/x/events"
	"go.infratographer.com/x/gidx"
	"go.infratographer.com/x/pubsubx"
	"go.uber.org/zap"
	"helm.sh/helm/v3/pkg/chart"
	"k8s.io/client-go/rest"

	"go.infratographer.com/loadbalanceroperator/internal/utils"
	"go.infratographer.com/loadbalanceroperator/internal/utils/mock"
)

func (suite *srvTestSuite) TestProcessLoadBalancerChangeCreate() { //nolint:govet
	type testCase struct {
		name           string
		msg            pubsubx.ChangeMessage
		cfg            *rest.Config
		chart          *chart.Chart
		expectedErrors []error
	}

	dir, cp, ch, pwd := utils.CreateWorkspace("test-create-lb")
	defer os.RemoveAll(dir)

	eSrv, err := echox.NewServer(zap.NewNop(), echox.Config{}, nil)

	api := mock.DummyAPI(dummyLB.String())
	api.Start()

	defer api.Close()

	require.NoError(suite.T(), err, "unexpected error creating new server")

	srv := Server{
		APIClient:    lbapi.NewClient(api.URL),
		Echo:         eSrv,
		Context:      context.TODO(),
		Logger:       zap.NewNop().Sugar(),
		Debug:        false,
		ChangeTopics: []string{"foo", "bar"},
		ChartPath:    cp,
		ValuesPath:   pwd + "/../../hack/ci/values.yaml",
	}

	testCases := []testCase{
		{
			name:           "loadbalancer create",
			expectedErrors: nil,
			cfg:            suite.Kubeenv.Config,
			chart:          ch,
			msg: pubsubx.ChangeMessage{
				EventType: "create",
				SubjectID: "loadbal-lkjasdlfkjasdf",
			},
		},
		{
			name:           "invalid loadbalancer - long name",
			expectedErrors: []error{errInvalidObjectNameLength},
			msg: pubsubx.ChangeMessage{
				EventType: "create",
				SubjectID: "loadbal-reallyreallyreallyreallyreallyreallylongreallylong",
			},
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			s := srv
			s.KubeClient = tc.cfg
			s.Chart = tc.chart

			lb, err := s.newLoadBalancer(tc.msg.SubjectID, tc.msg.AdditionalSubjectIDs)

			assert.Nil(suite.T(), err)

			err = s.processLoadBalancerChangeCreate(lb)

			if len(tc.expectedErrors) > 0 {
				assert.Error(suite.T(), err)
			} else {
				assert.Nil(suite.T(), err)
			}

			//TODO: check if the namespace was created
			//TODO: check if the helm release exists
		})
	}
}

func (suite *srvTestSuite) TestProcessLoadBalancerDelete() { //nolint:govet
	type testCase struct {
		name        string
		msg         pubsubx.ChangeMessage
		cfg         *rest.Config
		chart       *chart.Chart
		expectError bool
	}

	dir, cp, ch, pwd := utils.CreateWorkspace("test-delete-lb")
	defer os.RemoveAll(dir)

	eSrv, err := echox.NewServer(zap.NewNop(), echox.Config{}, nil)

	require.NoError(suite.T(), err, "unexpected error creating new server")

	api := mock.DummyAPI(dummyLB.String())
	api.Start()

	defer api.Close()

	srv := Server{
		APIClient:    lbapi.NewClient(api.URL),
		Echo:         eSrv,
		Context:      context.TODO(),
		Logger:       zap.NewNop().Sugar(),
		Debug:        false,
		ChangeTopics: []string{"foo", "bar"},
		ChartPath:    cp,
		ValuesPath:   pwd + "/../../hack/ci/values.yaml",
	}

	testCases := []testCase{
		{
			name:        "delete lb",
			expectError: false,
			cfg:         suite.Kubeenv.Config,
			chart:       ch,
			msg: pubsubx.ChangeMessage{
				EventType: string(events.DeleteChangeType),
				SubjectID: dummyLB,
			},
		},
		{
			name:        "unable to remove deployment",
			expectError: true,
			cfg:         &rest.Config{},
			chart:       ch,
			msg: pubsubx.ChangeMessage{
				EventType: string(events.DeleteChangeType),
				SubjectID: "loadbal-kljasdlkfjasdf",
			},
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			s := srv
			s.KubeClient = tc.cfg
			s.Chart = tc.chart

			lb, err := s.newLoadBalancer(tc.msg.SubjectID, tc.msg.AdditionalSubjectIDs)

			assert.Nil(suite.T(), err)

			_ = s.processLoadBalancerChangeCreate(lb)

			// TODO: check if the namespace was created
			// TODO: check if helm release exists

			err = s.processLoadBalancerChangeDelete(lb)

			if tc.expectError {
				assert.Error(suite.T(), err)
			} else {
				assert.Nil(suite.T(), err)

				// TODO: check if the release is missing
				// TODO: check if the namespace is missing
			}
		})
	}
}

func (suite *srvTestSuite) TestProcessLoadBalancerUpdate() { //nolint:govet
	dir, cp, ch, pwd := utils.CreateWorkspace("test-delete-lb")
	defer os.RemoveAll(dir)

	eSrv, err := echox.NewServer(zap.NewNop(), echox.Config{}, nil)

	require.NoError(suite.T(), err, "unexpected error creating new server")

	api := mock.DummyAPI(dummyLB.String())
	api.Start()

	defer api.Close()

	srv := Server{
		APIClient:    lbapi.NewClient(api.URL),
		KubeClient:   suite.Kubeenv.Config,
		Echo:         eSrv,
		Context:      context.TODO(),
		Logger:       zap.NewNop().Sugar(),
		Chart:        ch,
		Debug:        false,
		ChangeTopics: []string{"foo", "bar"},
		ChartPath:    cp,
		ValuesPath:   pwd + "/../../hack/ci/values.yaml",
	}

	id := gidx.MustNewID("loadbal")
	lb, _ := srv.newLoadBalancer(id, nil)

	err = srv.processLoadBalancerChangeCreate(lb)

	assert.NoError(suite.T(), err)

	u := srv.processLoadBalancerChangeUpdate(lb)

	assert.NoError(suite.T(), u)
}
