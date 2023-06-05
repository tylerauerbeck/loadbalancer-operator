package srv

import (
	"context"
	"os"

	"github.com/stretchr/testify/assert"

	"go.infratographer.com/loadbalancer-manager-haproxy/pkg/lbapi"

	"go.infratographer.com/loadbalanceroperator/internal/utils"
	"go.infratographer.com/loadbalanceroperator/internal/utils/mock"

	"go.infratographer.com/x/echox"
	"go.infratographer.com/x/events"
	"go.infratographer.com/x/gidx"

	"go.uber.org/zap"
	"helm.sh/helm/v3/pkg/chart/loader"
)

func (s *srvTestSuite) TestLocationCheck() { //nolint:govet
	lb, _ := gidx.Parse("testloc-abcd1234")

	srv := Server{
		Locations: []string{"abcd1234", "defg5678"},
	}

	check := srv.locationCheck(lb)
	assert.Equal(s.T(), true, check)

	lb, _ = gidx.Parse("testloc-efgh5678")
	check = srv.locationCheck(lb)
	assert.Equal(s.T(), false, check)
}

func (s srvTestSuite) TestProcessChange() { //nolint:govet
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

	loc, _ := gidx.Parse("testloc-abcd1234")

	srv := Server{
		APIClient:        lbapi.NewClient(api.URL),
		Echo:             eSrv,
		Context:          context.TODO(),
		Logger:           zap.NewNop().Sugar(),
		KubeClient:       s.Kubeconfig,
		SubscriberConfig: s.SubConfig,
		Topics:           []string{"*.load-balancer"},
		Chart:            ch,
		ValuesPath:       pwd + "/../../hack/ci/values.yaml",
		Locations:        []string{"abcd1234"},
	}

	// TODO: check that namespace does not exist
	// TODO: check that release does not exist

	// publish a message to the change channel
	pub := s.PubConfig
	p, _ := events.NewPublisher(pub)
	_ = p.PublishChange(context.TODO(), "load-balancer", events.ChangeMessage{
		EventType:            string(events.CreateChangeType),
		SubjectID:            id,
		AdditionalSubjectIDs: []gidx.PrefixedID{loc},
	})

	_ = srv.configureSubscribers()

	go srv.processChange(srv.changeChannels[0])
	// TODO: check that namespace exists
	// TODO: check that release exists

	_ = p.PublishChange(context.TODO(), "load-balancer", events.ChangeMessage{
		EventType:            string(events.UpdateChangeType),
		AdditionalSubjectIDs: []gidx.PrefixedID{loc},
		SubjectID:            id,
	})

	// TODO: check that namespace exists
	// TODO: check that release exists
	// TODO: verify some update, maybe with values file

	_ = p.PublishChange(context.TODO(), "load-balancer", events.ChangeMessage{
		EventType:            string(events.UpdateChangeType),
		AdditionalSubjectIDs: []gidx.PrefixedID{id, loc},
		SubjectID:            gidx.MustNewID("loadprt"),
	})

	//TODO: verify some update exists

	_ = p.PublishChange(context.TODO(), "load-balancer", events.ChangeMessage{
		EventType:            string(events.DeleteChangeType),
		AdditionalSubjectIDs: []gidx.PrefixedID{loc},
		SubjectID:            id,
	})
}

// TODO: add more extensive tests once we start processing event messages
func (s srvTestSuite) TestProcessEvent() { //nolint:govet
	id := gidx.MustNewID("loadbal")

	api := mock.DummyAPI(id.String())
	api.Start()

	eSrv, _ := echox.NewServer(zap.NewNop(), echox.Config{}, nil)

	testDir, err := os.MkdirTemp("", "test-process-event")
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

	loc, _ := gidx.Parse("testloc-abcd1234")

	srv := Server{
		APIClient:        lbapi.NewClient(api.URL),
		Echo:             eSrv,
		Context:          context.TODO(),
		Logger:           zap.NewNop().Sugar(),
		KubeClient:       s.Kubeconfig,
		SubscriberConfig: s.SubConfig,
		Topics:           []string{"*.load-balancer-event"},
		Chart:            ch,
		ValuesPath:       pwd + "/../../hack/ci/values.yaml",
		Locations:        []string{"abcd1234"},
	}

	pub := s.PubConfig
	p, _ := events.NewPublisher(pub)
	_ = p.PublishEvent(context.TODO(), "load-balancer-event", events.EventMessage{
		EventType:            "create",
		SubjectID:            id,
		AdditionalSubjectIDs: []gidx.PrefixedID{loc},
	})

	_ = srv.configureSubscribers()

	go srv.processEvent(srv.eventChannels[0])

	_ = p.PublishEvent(context.TODO(), "load-balancer-event", events.EventMessage{
		EventType:            "update",
		AdditionalSubjectIDs: []gidx.PrefixedID{loc},
		SubjectID:            id,
	})

	_ = p.PublishEvent(context.TODO(), "load-balancer-event", events.EventMessage{
		EventType:            "delete",
		AdditionalSubjectIDs: []gidx.PrefixedID{loc},
		SubjectID:            id,
	})
}
