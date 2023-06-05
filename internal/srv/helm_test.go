package srv

import (
	"context"
	"encoding/hex"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.infratographer.com/x/gidx"
	"go.uber.org/zap"

	"helm.sh/helm/v3/pkg/cli/values"
	"k8s.io/client-go/rest"
)

var (
	dummyKey = "hello"
	dummyVal = "world"
)

func (suite *srvTestSuite) TestGenerateLBHelmVals() {
	lb := &loadBalancer{
		loadBalancerID: gidx.MustNewID("loadbal"),
	}

	hash := hex.EncodeToString([]byte(lb.loadBalancerID.String()))

	opts := helmvalues{&values.Options{}}

	assert.Nil(suite.T(), opts.StringValues)

	opts.generateLBHelmVals(lb)

	assert.NotNil(suite.T(), opts.StringValues)
	assert.Len(suite.T(), opts.StringValues, 2)
	assert.Contains(suite.T(), opts.StringValues, managedHelmKeyPrefix+".loadBalancerID="+lb.loadBalancerID.String())
	assert.Contains(suite.T(), opts.StringValues, managedHelmKeyPrefix+".loadBalancerIDEnc="+hash)
}

func (suite *srvTestSuite) TestAddValue() { //nolint:stylecheck
	opts := helmvalues{&values.Options{}}

	assert.Nil(suite.T(), opts.StringValues)

	opts.addValue(dummyKey, dummyVal)

	assert.NotNil(suite.T(), opts.StringValues)
	assert.Len(suite.T(), opts.StringValues, 1)
	assert.Contains(suite.T(), opts.StringValues, managedHelmKeyPrefix+"."+dummyKey+"="+dummyVal)
}

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
				Logger:     zap.NewNop().Sugar(),
				ValuesPath: tcase.valuesPath,
			}
			values, err := srv.newHelmValues(tcase.lb)
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
