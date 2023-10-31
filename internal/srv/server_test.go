package srv

import (
	"context"
	"os"

	"github.com/stretchr/testify/assert"
	lbapi "go.infratographer.com/load-balancer-api/pkg/client"

	"go.infratographer.com/load-balancer-operator/internal/utils"
	"go.infratographer.com/load-balancer-operator/internal/utils/mock"

	"go.infratographer.com/x/echox"
	"go.infratographer.com/x/gidx"
	"go.uber.org/zap"
	"helm.sh/helm/v3/pkg/chart/loader"
)

func (suite *srvTestSuite) TestRun() { //nolint:govet
	id := gidx.MustNewID("loadbal")

	api := mock.DummyAPI(id.String())
	api.Start()

	eSrv, _ := echox.NewServer(zap.NewNop(), echox.Config{}, nil)

	testDir, err := os.MkdirTemp("", "test-process-change")
	if err != nil {
		suite.T().Fatal(err)
	}

	defer os.RemoveAll(testDir)

	chartPath, err := utils.CreateTestChart(testDir)
	if err != nil {
		suite.T().Fatal(err)
	}

	ch, err := loader.Load(chartPath)
	if err != nil {
		suite.T().Fatal(err)
	}

	pwd, err := os.Getwd()
	if err != nil {
		suite.T().Fatal(err)
	}

	srv := Server{
		APIClient:        lbapi.NewClient(api.URL),
		Echo:             eSrv,
		Context:          context.TODO(),
		Logger:           zap.NewNop().Sugar(),
		KubeClient:       suite.Kubeconfig,
		EventsConnection: suite.Connection,
		ChangeTopics:     []string{"*.load-balancer-run"},
		Chart:            ch,
		ValuesPath:       pwd + "/../../hack/ci/values.yaml",
		Locations:        []string{"abcd1234"},
	}

	err = srv.Run(srv.Context)

	assert.Nil(suite.T(), err)
	assert.Len(suite.T(), srv.changeChannels, len(srv.ChangeTopics))
}

// TODO: add test for consumer that is already bound
