package srv

import (
	"context"
	"encoding/hex"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	lbapi "go.infratographer.com/load-balancer-api/pkg/client"

	"go.infratographer.com/x/gidx"
	"go.uber.org/zap"

	"helm.sh/helm/v3/pkg/cli/values"
	"k8s.io/client-go/rest"

	"go.infratographer.com/loadbalanceroperator/internal/utils/mock"
)

// var (
// 	dummyKey = "hello"
// 	dummyVal = "world"
// )

func (suite *srvTestSuite) TestGenerateLBHelmVals() {
	id := gidx.MustNewID("loadbal")
	api := mock.DummyAPI(id.String())
	api.Start()

	defer api.Close()

	s := &Server{
		Context:          context.TODO(),
		APIClient:        lbapi.NewClient(api.URL),
		ContainerPortKey: "containerPorts",
		ServicePortKey:   "service.ports",
		Logger:           zap.NewNop().Sugar(),
	}

	lb, _ := s.newLoadBalancer(context.TODO(), id, nil)

	hash := hex.EncodeToString([]byte(lb.loadBalancerID.String()))

	opts := helmvalues{&values.Options{}}

	assert.Nil(suite.T(), opts.StringValues)

	opts.generateLBHelmVals(lb, s)

	assert.NotNil(suite.T(), opts.StringValues)
	assert.Contains(suite.T(), opts.StringValues, managedHelmKeyPrefix+".lbID="+lb.loadBalancerID.String())
	assert.Contains(suite.T(), opts.StringValues, managedHelmKeyPrefix+".lbIDEnc="+hash)

	assert.NotNil(suite.T(), opts.JSONValues)
}

// func (suite *srvTestSuite) TestAddValue() { //nolint:stylecheck
// 	opts := helmvalues{&values.Options{}}

// 	assert.Nil(suite.T(), opts.StringValues)

// 	opts.addValue(dummyKey, dummyVal)

// 	assert.NotNil(suite.T(), opts.StringValues)
// 	assert.Len(suite.T(), opts.StringValues, 1)
// 	assert.Contains(suite.T(), opts.StringValues, managedHelmKeyPrefix+"."+dummyKey+"="+dummyVal)
// }

func (suite *srvTestSuite) TestNewHelmValues() {
	type testCase struct {
		name        string
		valuesPath  string
		expectError bool
		lb          *loadBalancer
	}

	pwd, err := os.Getwd()
	if err != nil {
		suite.T().Fatal(err)
	}

	id := gidx.MustNewID("loadbal")
	api := mock.DummyAPI(id.String())
	api.Start()

	defer api.Close()

	testCases := []testCase{
		{
			name:        "valid values path",
			expectError: false,
			valuesPath:  pwd + "/../../hack/ci/values.yaml",
			lb: &loadBalancer{
				loadBalancerID: gidx.MustNewID("loadbal"),
			},
		},
		{
			name:        "valid overrides",
			expectError: false,
			valuesPath:  pwd + "/../../hack/ci/values.yaml",
			lb: &loadBalancer{
				loadBalancerID: gidx.MustNewID("loadbal"),
			},
		},
		{
			name:        "missing values path",
			expectError: true,
			valuesPath:  "",
			lb: &loadBalancer{
				loadBalancerID: gidx.MustNewID("loadbal"),
			},
		},
	}

	for _, tcase := range testCases {
		suite.T().Run(tcase.name, func(t *testing.T) {
			srv := Server{
				Context:          context.TODO(),
				APIClient:        lbapi.NewClient(api.URL),
				Logger:           zap.NewNop().Sugar(),
				ValuesPath:       tcase.valuesPath,
				ContainerPortKey: "containerPorts",
				ServicePortKey:   "service.ports",
			}

			lb, _ := srv.newLoadBalancer(context.TODO(), tcase.lb.loadBalancerID, nil)

			values, err := srv.newHelmValues(lb)
			if tcase.expectError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, values)
			}
		})
	}
}

func (suite *srvTestSuite) TestNewHelmClient() {
	type testCase struct {
		name         string
		appNamespace string
		kubeClient   *rest.Config
		expectError  bool
	}

	testCases := []testCase{
		{
			name:         "valid client",
			appNamespace: "launchpad",
			kubeClient:   suite.Kubeconfig,
			expectError:  false,
		},
		{
			name:         "invalid client",
			appNamespace: "",
			kubeClient:   suite.Kubeconfig,
			expectError:  true,
		},
	}

	for _, tcase := range testCases {
		suite.T().Run(tcase.name, func(t *testing.T) {
			srv := Server{
				Context:    context.TODO(),
				Logger:     zap.NewNop().Sugar(),
				KubeClient: tcase.kubeClient,
			}

			_, err := srv.newHelmClient(tcase.appNamespace)

			if tcase.expectError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}
