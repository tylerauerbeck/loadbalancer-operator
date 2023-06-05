package srv

import (
	"context"
	"os"

	"github.com/stretchr/testify/assert"
	"go.infratographer.com/loadbalancer-manager-haproxy/pkg/lbapi"

	"go.infratographer.com/loadbalanceroperator/internal/utils"
	"go.infratographer.com/loadbalanceroperator/internal/utils/mock"

	"go.infratographer.com/x/echox"
	"go.infratographer.com/x/gidx"
	"go.uber.org/zap"
	"helm.sh/helm/v3/pkg/chart/loader"
)

func (s srvTestSuite) TestRun() { //nolint:govet
	id := gidx.MustNewID("loadbal")

	api := mock.DummyAPI(id.String())
	api.Start()

	eSrv, _ := echox.NewServer(zap.NewNop(), echox.Config{}, nil)

	testDir, err := os.MkdirTemp("", "test-process-change")
	if err != nil {
		s.T().Fatal(err)
	}

	defer os.RemoveAll(testDir)

	chartPath, err := utils.CreateTestChart(testDir)
	if err != nil {
		s.T().Fatal(err)
	}

	ch, err := loader.Load(chartPath)
	if err != nil {
		s.T().Fatal(err)
	}

	pwd, err := os.Getwd()
	if err != nil {
		s.T().Fatal(err)
	}

	srv := Server{
		APIClient:        lbapi.NewClient(api.URL),
		Echo:             eSrv,
		Context:          context.TODO(),
		Logger:           zap.NewNop().Sugar(),
		KubeClient:       s.Kubeconfig,
		SubscriberConfig: s.SubConfig,
		Topics:           []string{"*.load-balancer-run"},
		Chart:            ch,
		ValuesPath:       pwd + "/../../hack/ci/values.yaml",
		Locations:        []string{"abcd1234"},
	}

	err = srv.Run(srv.Context)

	assert.Nil(s.T(), err)
	assert.Len(s.T(), srv.eventChannels, len(srv.Topics))
	assert.Len(s.T(), srv.changeChannels, len(srv.Topics))
}

// TODO: add test for consumer that is already bound
