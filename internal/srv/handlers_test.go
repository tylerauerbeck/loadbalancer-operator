package srv

import (
	"context"
	"os"
	"time"

	"github.com/lestrrat-go/backoff/v2"
	"github.com/stretchr/testify/assert"

	lbapi "go.infratographer.com/load-balancer-api/pkg/client"

	"go.infratographer.com/loadbalanceroperator/internal/utils"
	"go.infratographer.com/loadbalanceroperator/internal/utils/mock"

	"go.infratographer.com/x/echox"
	"go.infratographer.com/x/events"
	"go.infratographer.com/x/gidx"

	"go.uber.org/zap"
	"helm.sh/helm/v3/pkg/chart/loader"
)

func (suite *srvTestSuite) TestLocationCheck() { //nolint:govet
	lb, _ := gidx.Parse("testloc-abcd1234")

	srv := Server{
		Locations: []string{"abcd1234", "defg5678"},
	}

	check := srv.locationCheck(lb)
	assert.Equal(suite.T(), true, check)

	lb, _ = gidx.Parse("testloc-efgh5678")
	check = srv.locationCheck(lb)
	assert.Equal(suite.T(), false, check)
}

func (suite *srvTestSuite) TestProcessChange() { //nolint:govet
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

	loc, _ := gidx.Parse("testloc-abcd1234")

	backoffPolicy := backoff.Exponential(
		backoff.WithMinInterval(time.Second),
		backoff.WithMaxInterval(2*time.Minute),
		backoff.WithJitterFactor(0.05),
		backoff.WithMaxRetries(5),
	)

	srv := Server{
		APIClient:     lbapi.NewClient(api.URL),
		BackoffConfig: backoffPolicy,
		Echo:          eSrv,
		Context:       context.TODO(),
		Logger:        zap.NewNop().Sugar(),
		KubeClient:    suite.Kubeconfig,
		// SubscriberConfig: suite.SubConfig,
		EventsConnection: suite.Connection,
		ChangeTopics:     []string{"*.load-balancer"},
		Chart:            ch,
		ValuesPath:       pwd + "/../../hack/ci/values.yaml",
		Locations:        []string{"abcd1234"},
	}

	// TODO: check that namespace does not exist
	// TODO: check that release does not exist

	// publish a message to the change channel
	_, err = srv.EventsConnection.PublishChange(context.TODO(), "load-balancer", events.ChangeMessage{
		EventType:            string(events.CreateChangeType),
		SubjectID:            id,
		AdditionalSubjectIDs: []gidx.PrefixedID{loc},
	})
	if err != nil {
		utils.ErrPanic("unable to publish message", err)
	}

	err = srv.configureSubscribers(context.TODO())
	if err != nil {
		utils.ErrPanic("unable to configure subscribers", err)
	}

	go srv.listenChange(srv.changeChannels[0])
	// TODO: check that namespace exists
	// TODO: check that release exists

	_, err = srv.EventsConnection.PublishChange(context.TODO(), "load-balancer", events.ChangeMessage{
		EventType:            string(events.UpdateChangeType),
		AdditionalSubjectIDs: []gidx.PrefixedID{loc},
		SubjectID:            id,
	})
	if err != nil {
		utils.ErrPanic("unable to publish message -2", err)
	}

	// TODO: check that namespace exists
	// TODO: check that release exists
	// TODO: verify some update, maybe with values file

	_, err = srv.EventsConnection.PublishChange(context.TODO(), "load-balancer", events.ChangeMessage{
		EventType:            string(events.UpdateChangeType),
		AdditionalSubjectIDs: []gidx.PrefixedID{id, loc},
		SubjectID:            gidx.MustNewID("loadprt"),
	})
	if err != nil {
		utils.ErrPanic("unable to publish message -3", err)
	}

	//TODO: verify some update exists

	_, err = srv.EventsConnection.PublishChange(context.TODO(), "load-balancer", events.ChangeMessage{
		EventType:            string(events.DeleteChangeType),
		AdditionalSubjectIDs: []gidx.PrefixedID{loc},
		SubjectID:            id,
	})
	if err != nil {
		utils.ErrPanic("unable to publish message -4", err)
	}
}

// TODO: add more extensive tests once we start processing event messages
func (suite *srvTestSuite) TestProcessEvent() { //nolint:govet
	id := gidx.MustNewID("loadbal")

	api := mock.DummyAPI(id.String())
	api.Start()

	eSrv, _ := echox.NewServer(zap.NewNop(), echox.Config{}, nil)

	testDir, err := os.MkdirTemp("", "test-process-event")
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

	loc, _ := gidx.Parse("testloc-abcd1234")

	srv := Server{
		APIClient:        lbapi.NewClient(api.URL),
		Echo:             eSrv,
		Context:          context.TODO(),
		Logger:           zap.NewNop().Sugar(),
		KubeClient:       suite.Kubeconfig,
		EventsConnection: suite.Connection,
		EventTopics:      []string{"*.load-balancer-event"},
		Chart:            ch,
		ValuesPath:       pwd + "/../../hack/ci/values.yaml",
		Locations:        []string{"abcd1234"},
	}

	_, err = srv.EventsConnection.PublishEvent(context.TODO(), "load-balancer-event", events.EventMessage{
		EventType:            "create",
		SubjectID:            id,
		AdditionalSubjectIDs: []gidx.PrefixedID{loc},
	})
	if err != nil {
		utils.ErrPanic("unable to publish message", err)
	}

	_ = srv.configureSubscribers(context.TODO())

	go srv.listenEvent(srv.eventChannels[0])

	_, err = srv.EventsConnection.PublishEvent(context.TODO(), "load-balancer-event", events.EventMessage{
		EventType:            "update",
		AdditionalSubjectIDs: []gidx.PrefixedID{loc},
		SubjectID:            id,
	})
	if err != nil {
		utils.ErrPanic("unable to publish message", err)
	}

	_, err = srv.EventsConnection.PublishEvent(context.TODO(), "load-balancer-event", events.EventMessage{
		EventType:            "delete",
		AdditionalSubjectIDs: []gidx.PrefixedID{loc},
		SubjectID:            id,
	})
	if err != nil {
		utils.ErrPanic("unable to publish message", err)
	}
}
