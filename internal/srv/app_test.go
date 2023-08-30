package srv

import (
	"context"
	"os"
	"testing"

	"github.com/gobuffalo/packr/v2/file/resolver/encoding/hex"
	"github.com/stretchr/testify/assert"
	"go.infratographer.com/loadbalancer-manager-haproxy/pkg/lbapi"

	"go.infratographer.com/loadbalanceroperator/internal/utils"
	"go.infratographer.com/loadbalanceroperator/internal/utils/mock"

	"go.infratographer.com/x/gidx"
	"go.uber.org/zap"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd/api"
)

var (
	dummyLBID = "loadbal-lkjasdlfkjasdf"
)

func (suite *srvTestSuite) TestHashLBName() {
	hash := hashLBName(dummyLBID)

	assert.NotNil(suite.T(), hash)
	assert.IsType(suite.T(), "", hash)

	dec, _ := hex.DecodeString(hash)

	assert.Equal(suite.T(), dummyLBID, string(dec))
}

func (suite *srvTestSuite) TestCheckNameLength() {
	type testCase struct {
		name   string
		check  string
		length int
		expect bool
	}

	testCases := []testCase{
		{
			name:   "valid helm release",
			check:  gidx.MustNewID("loadbal").String(),
			length: helmReleaseLength,
			expect: true,
		},
		{
			name:   "invalid helm release",
			check:  hex.EncodeToString([]byte(gidx.MustNewID("loadbal"))),
			length: helmReleaseLength,
			expect: false,
		},
		{
			name:   "valid kube namespace length",
			check:  hex.EncodeToString([]byte(gidx.MustNewID("loadbal"))),
			length: 63,
			expect: true,
		},
		{
			name:   "invalid kube namespace length",
			check:  hex.EncodeToString([]byte(gidx.MustNewID("loadbal"))) + hex.EncodeToString([]byte(gidx.MustNewID("loadbal"))),
			length: 63,
			expect: false,
		},
	}

	for _, tcase := range testCases {
		suite.T().Run(tcase.name, func(t *testing.T) {
			c := checkNameLength(tcase.check, tcase.length)

			assert.Equal(t, tcase.expect, c)
		})
	}
}

func (suite *srvTestSuite) TestStrPt() {
	s := "string"
	p := strPt(s)

	assert.NotNil(suite.T(), p)
	assert.IsType(suite.T(), new(string), p)
}

func (suite *srvTestSuite) TestCreateNamespace() {
	type testCase struct {
		name        string
		id          gidx.PrefixedID
		expectError bool
		kubeclient  *rest.Config
	}

	testCases := []testCase{
		{
			name:        "valid yaml",
			expectError: false,
			id:          gidx.MustNewID("loadbal"),
			kubeclient:  suite.Kubeconfig,
		},
		{
			name:        "invalid kubeclient",
			expectError: true,
			id:          gidx.MustNewID("loadbal"),
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
			api := mock.DummyAPI(tcase.id.String())
			api.Start()

			defer api.Close()

			srv := Server{
				APIClient:  lbapi.NewClient(api.URL),
				Context:    context.TODO(),
				Logger:     zap.NewNop().Sugar(),
				KubeClient: tcase.kubeclient,
			}

			subj := tcase.id
			adds := []gidx.PrefixedID{}

			lb, err := srv.newLoadBalancer(context.TODO(), subj, adds)

			assert.Nil(t, err)

			hash := hashLBName(lb.loadBalancerID.String())
			ns, err := srv.CreateNamespace(context.TODO(), hash)

			if tcase.expectError {
				assert.NotNil(t, err)
				assert.Nil(t, ns)
			} else {
				assert.Nil(t, err)
				assert.Contains(t, ns.Labels, "com.infratographer.lb-operator/managed")
				assert.Contains(t, ns.Labels, "com.infratographer.lb-operator/lb-id")
			}
		})
	}
}

// TODO: add test for bad binding
// func (suite *srvTestSuite) TestCreateNamespace_BadBinding(){

// }

func (suite *srvTestSuite) TestRemoveNamespace() {
	type testCase struct {
		name        string
		id          gidx.PrefixedID
		expectError bool
		kubeclient  *rest.Config
	}

	testCases := []testCase{
		{
			name:        "valid yaml",
			expectError: false,
			id:          gidx.MustNewID("loadbal"),
			kubeclient:  suite.Kubeconfig,
		},
		{
			name:        "bad kubeclient",
			expectError: true,
			id:          gidx.MustNewID("loadbal"),
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
			api := mock.DummyAPI(tcase.id.String())
			api.Start()

			defer api.Close()

			srv := Server{
				APIClient:  lbapi.NewClient(api.URL),
				Context:    context.TODO(),
				Logger:     zap.NewNop().Sugar(),
				KubeClient: tcase.kubeclient,
			}

			lb, err := srv.newLoadBalancer(context.TODO(), tcase.id, []gidx.PrefixedID{})

			assert.Nil(t, err)

			hash := hashLBName(lb.loadBalancerID.String())

			// TODO: check that namespace doesn't exist

			_, _ = srv.CreateNamespace(context.TODO(), hash)

			// TODO: check that namespace does exist

			err = srv.removeNamespace(context.TODO(), hash)

			// TODO: check that namespace does not exist

			if tcase.expectError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

// TODO: add test for bad namespace
// func (suite *srvTestSuite) TestRemoveNamespace_BadNamespace() {

// }

func (suite *srvTestSuite) TestNewDeployment() {
	type testCase struct {
		name        string
		id          gidx.PrefixedID
		expectError bool
		chart       *chart.Chart
		kubeClient  *rest.Config
		valPath     string
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
			name:        "valid yaml",
			expectError: false,
			id:          gidx.MustNewID("loadbal"),
			chart:       ch,
			valPath:     pwd + "/../../hack/ci/values.yaml",
			kubeClient:  suite.Kubeconfig,
		},
		{
			name:        "missing values path",
			expectError: true,
			id:          gidx.MustNewID("loadbal"),
			chart:       ch,
			valPath:     "",
			kubeClient:  suite.Kubeconfig,
		},
		{
			name:        "invalid chart",
			expectError: true,
			id:          gidx.MustNewID("loadbal"),
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
		},
	}

	for _, tcase := range testCases {
		suite.T().Run(tcase.name, func(t *testing.T) {
			api := mock.DummyAPI(tcase.id.String())
			api.Start()

			defer api.Close()

			srv := Server{
				APIClient:  lbapi.NewClient(api.URL),
				Context:    context.TODO(),
				Logger:     zap.NewNop().Sugar(),
				KubeClient: tcase.kubeClient,
				ValuesPath: tcase.valPath,
				Chart:      tcase.chart,
			}

			lb, err := srv.newLoadBalancer(context.TODO(), tcase.id, []gidx.PrefixedID{})

			assert.Nil(t, err)

			hash := hashLBName(lb.loadBalancerID.String())

			// TODO: check that namespace doesn't exist

			_, _ = srv.CreateNamespace(context.TODO(), hash)

			// TODO: check that namespace does exist

			// TODO: check that deployment doesn't exist
			err = srv.newDeployment(context.TODO(), lb)

			// TODO: check that deployment exists

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
		name        string
		id          gidx.PrefixedID
		expectError bool
		chart       *chart.Chart
		kubeClient  *rest.Config
		valPath     string
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
			name:        "valid deployment",
			expectError: false,
			id:          gidx.MustNewID("loadbal"),
			chart:       ch,
			valPath:     pwd + "/../../hack/ci/values.yaml",
			kubeClient:  suite.Kubeconfig,
		},
	}

	for _, tcase := range testCases {
		suite.T().Run(tcase.name, func(t *testing.T) {
			api := mock.DummyAPI(tcase.id.String())
			api.Start()

			defer api.Close()

			srv := Server{
				APIClient:  lbapi.NewClient(api.URL),
				Context:    context.TODO(),
				Logger:     zap.NewNop().Sugar(),
				KubeClient: tcase.kubeClient,
				ValuesPath: tcase.valPath,
				Chart:      tcase.chart,
			}

			lb, err := srv.newLoadBalancer(context.TODO(), tcase.id, []gidx.PrefixedID{})

			assert.Nil(t, err)

			hash := hashLBName(lb.loadBalancerID.String())

			// TODO: check that namespace does not exist

			_, _ = srv.CreateNamespace(context.TODO(), hash)

			// TODO: check that namespace does exist
			// TODO: check that release does not exist
			_ = srv.newDeployment(context.TODO(), lb)
			// TODO: check that release does exist

			err = srv.removeDeployment(context.TODO(), lb)

			// TODO: check that release does not exist
			// tODO: check that namespace does not exist

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
