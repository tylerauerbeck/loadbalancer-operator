package srv

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"go.uber.org/zap"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd/api"

	"go.infratographer.com/loadbalanceroperator/internal/utils"

	"go.infratographer.com/x/gidx"
)

func (suite *srvTestSuite) TestNewHelmValues() {
	type testCase struct {
		name        string
		valuesPath  string
		overrides   []valueSet
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
			overrides:   nil,
			lb: &loadBalancer{
				loadBalancerID:         gidx.MustNewID("loadbal"),
				loadBalancerTenantID:   "tnnttnt-lkjasdflkj",
				loadBalancerLocationID: "lctnloc-lkjasdflkj",
			},
		},
		{
			name:        "valid overrides",
			expectError: false,
			valuesPath:  pwd + "/../../hack/ci/values.yaml",
			overrides: []valueSet{
				{
					helmKey: "hello",
					value:   "world",
				},
			},
			lb: &loadBalancer{
				loadBalancerID:         gidx.MustNewID("loadbal"),
				loadBalancerTenantID:   "tnnttnt-lkjasdflkj",
				loadBalancerLocationID: "lctnloc-lkjasdflkj",
			},
		},
		{
			name:        "missing values path",
			expectError: true,
			valuesPath:  "",
			overrides:   nil,
			lb:          nil,
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

func (suite *srvTestSuite) TestCreateNamespace() {
	type testCase struct {
		name         string
		appNamespace string
		expectError  bool
		kubeclient   *rest.Config
	}

	testCases := []testCase{
		{
			name:         "valid yaml",
			expectError:  false,
			appNamespace: "flintlock",
			kubeclient:   suite.Kubeconfig,
		},
		{
			name:         "invalid namespace",
			expectError:  true,
			appNamespace: "",
			kubeclient:   suite.Kubeconfig,
		},
		{
			name:         "invalid kubeclient",
			expectError:  true,
			appNamespace: "flintlock",
			kubeclient: &rest.Config{
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
		},
	}

	for _, tcase := range testCases {
		suite.T().Run(tcase.name, func(t *testing.T) {
			srv := Server{
				Context:    context.TODO(),
				Logger:     zap.NewNop().Sugar(),
				KubeClient: tcase.kubeclient,
			}

			hash := hashName(tcase.appNamespace)
			ns, err := srv.CreateNamespace(tcase.appNamespace, hash)

			if tcase.expectError {
				assert.NotNil(t, err)
				assert.Nil(t, ns)
			} else {
				assert.Nil(t, err)
				assert.Contains(t, ns.Annotations, "com.infratographer.lb-operator/managed")
				assert.Contains(t, ns.Annotations, "com.infratographer.lb-operator/lb-id")
			}
		})
	}
}

func (suite *srvTestSuite) TestNewDeployment() {
	type testCase struct {
		name         string
		appNamespace string
		appName      string
		expectError  bool
		chart        *chart.Chart
		kubeClient   *rest.Config
		valPath      string
		lb           *loadBalancer
	}

	testDir, err := os.MkdirTemp("", "test-new-deployment")
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

	testCases := []testCase{
		{
			name:         "valid yaml",
			expectError:  false,
			appNamespace: uuid.New().String(),
			appName:      uuid.New().String(),
			chart:        ch,
			valPath:      pwd + "/../../hack/ci/values.yaml",
			kubeClient:   suite.Kubeconfig,
			lb: &loadBalancer{
				loadBalancerID:         gidx.MustNewID("loadbal"),
				loadBalancerTenantID:   "tnnttnt-lkjasdflkj",
				loadBalancerLocationID: "lctnloc-lkjasdflkj",
			},
		},
		{
			name:         "missing values path",
			expectError:  true,
			appNamespace: uuid.New().String(),
			appName:      uuid.New().String(),
			chart:        ch,
			valPath:      "",
			kubeClient:   suite.Kubeconfig,
			lb: &loadBalancer{
				loadBalancerID:         gidx.MustNewID("loadbal"),
				loadBalancerTenantID:   "tnnttnt-lkjasdflkj",
				loadBalancerLocationID: "lctnloc-lkjasdflkj",
			},
		},
		{
			name:         "invalid chart",
			expectError:  true,
			appNamespace: uuid.New().String(),
			appName:      uuid.New().String(),
			chart: &chart.Chart{
				Raw:       []*chart.File{},
				Metadata:  &chart.Metadata{},
				Lock:      &chart.Lock{},
				Templates: []*chart.File{},
				Values:    map[string]interface{}{},
				Schema:    []byte{},
				Files:     []*chart.File{},
			},
			valPath:    pwd + "/../../hack/ci/values.yaml",
			kubeClient: suite.Kubeconfig,
			lb: &loadBalancer{
				loadBalancerID:         gidx.MustNewID("loadbal"),
				loadBalancerTenantID:   "tnnttnt-lkjasdflkj",
				loadBalancerLocationID: "lctnloc-lkjasdflkj",
			},
		},
	}

	for _, tcase := range testCases {
		suite.T().Run(tcase.name, func(t *testing.T) {
			srv := Server{
				Context:    context.TODO(),
				Logger:     zap.NewNop().Sugar(),
				KubeClient: tcase.kubeClient,
				ValuesPath: tcase.valPath,
				Chart:      tcase.chart,
			}

			hash := hashName(tcase.appNamespace)
			_, _ = srv.CreateNamespace(tcase.appName, hash)
			err = srv.newDeployment(tcase.lb)

			if tcase.expectError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
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

func (suite *srvTestSuite) TestAttachRoleBinding() {
	type testCase struct {
		name       string
		namespace  string
		kubeClient *rest.Config
		expectErr  bool
	}

	testCases := []testCase{
		{
			name:       "valid rolebinding",
			namespace:  "default",
			kubeClient: suite.Kubeconfig,
			expectErr:  false,
		},
		{
			name:       "invalid rolebinding",
			namespace:  "thisThingDoesNotExist",
			kubeClient: suite.Kubeconfig,
			expectErr:  true,
		},
	}

	for _, tcase := range testCases {
		suite.T().Run(tcase.name, func(t *testing.T) {
			srv := Server{
				Context:    context.TODO(),
				Logger:     zap.NewNop().Sugar(),
				KubeClient: tcase.kubeClient,
			}

			cli, err := kubernetes.NewForConfig(tcase.kubeClient)
			if err != nil {
				t.Fatal(err)
			}

			err = attachRoleBinding(srv.Context, cli, tcase.namespace)

			if tcase.expectErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func (suite *srvTestSuite) TestRemoveNamespace() {
	type testCase struct {
		name         string
		appNamespace string
		expectError  bool
		kubeclient   *rest.Config
	}

	testCases := []testCase{
		{
			name:         "valid yaml",
			expectError:  false,
			appNamespace: "flintlock",
			kubeclient:   suite.Kubeconfig,
		},
		{
			name:         "invalid namespace",
			expectError:  true,
			appNamespace: "this-does-not-exist",
			kubeclient:   suite.Kubeconfig,
		},
		{
			name:         "bad kubeclient",
			expectError:  true,
			appNamespace: "this-should-fail",
			kubeclient: &rest.Config{
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
		},
	}

	for _, tcase := range testCases {
		suite.T().Run(tcase.name, func(t *testing.T) {
			srv := Server{
				Context:    context.TODO(),
				Logger:     zap.NewNop().Sugar(),
				KubeClient: tcase.kubeclient,
			}

			hash := hashName(tcase.appNamespace)

			if !tcase.expectError {
				_, err := srv.CreateNamespace(tcase.appNamespace, hash)
				if err != nil {
					t.Fatal(err)
				}
			}

			err := srv.removeNamespace(hash)

			if tcase.expectError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func (suite *srvTestSuite) TestRemoveDeployment() {
	type testCase struct {
		name         string
		appNamespace string
		appName      string
		expectError  bool
		chart        *chart.Chart
		kubeClient   *rest.Config
		valPath      string
		lb           *loadBalancer
	}

	testDir, err := os.MkdirTemp("", "test-remove-deployment")
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

	testCases := []testCase{
		{
			name:         "valid deployment",
			expectError:  false,
			appNamespace: uuid.New().String(),
			appName:      uuid.New().String(),
			chart:        ch,
			valPath:      pwd + "/../../hack/ci/values.yaml",
			kubeClient:   suite.Kubeconfig,
			lb: &loadBalancer{
				loadBalancerID:         gidx.MustNewID("loadbal"),
				loadBalancerTenantID:   "tnnttnt-lkjasdflkj",
				loadBalancerLocationID: "lctnloc-lkjasdflkj",
			},
		},
		{
			name:         "invalid deployment",
			expectError:  true,
			appNamespace: uuid.New().String(),
			appName:      uuid.New().String(),
			chart:        ch,
			valPath:      pwd + "/../../hack/ci/values.yaml",
			kubeClient:   suite.Kubeconfig,
			lb: &loadBalancer{
				loadBalancerID:         gidx.MustNewID("loadbal"),
				loadBalancerTenantID:   "tnnttnt-lkjasdflkj",
				loadBalancerLocationID: "lctnloc-lkjasdflkj",
			},
		},
	}

	for _, tcase := range testCases {
		suite.T().Run(tcase.name, func(t *testing.T) {
			srv := Server{
				Context:    context.TODO(),
				Logger:     zap.NewNop().Sugar(),
				KubeClient: suite.Kubeconfig,
				ValuesPath: tcase.valPath,
				Chart:      tcase.chart,
			}

			if !tcase.expectError {
				hash := hashName(tcase.lb.loadBalancerID.String())
				_, _ = srv.CreateNamespace(tcase.lb.loadBalancerID.String(), hash)
				err = srv.newDeployment(tcase.lb)
				if err != nil {
					t.Fatal(err)
				}
			}
			err = srv.removeDeployment(tcase.lb.loadBalancerID.String())

			if tcase.expectError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}
