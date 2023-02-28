package srv

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd/api"

	"github.com/stretchr/testify/assert"

	"go.infratographer.com/x/pubsubx"

	"go.infratographer.com/loadbalanceroperator/internal/utils"
	events "go.infratographer.com/loadbalanceroperator/pkg/events/v1alpha1"
)

var (
	msg = pubsubx.Message{
		SubjectURN: uuid.NewString(),
		EventType:  "create",
		Source:     "lbapi",
		Timestamp:  time.Now(),
		ActorURN:   uuid.NewString(),
	}
)

func (suite *srvTestSuite) TestParseLBData() {
	suite.T().Parallel()

	type testCase struct {
		name        string
		data        map[string]interface{}
		expectError bool
	}

	testCases := []testCase{
		{
			name:        "valid data",
			expectError: false,
			data: map[string]interface{}{
				"load_balancer_id": uuid.New(),
				"location_id":      uuid.New(),
			},
		},
		{
			name:        "unable to parse event data",
			expectError: true,
			data: map[string]interface{}{
				"load_balancer_id": 1,
				"location_id":      2,
			},
		},
		{
			name:        "unable to load event data",
			expectError: true,
			data: map[string]interface{}{
				"other field": make(chan int),
			},
		},
	}

	for _, tcase := range testCases {
		suite.T().Run(tcase.name, func(t *testing.T) {
			lbData := events.LoadBalancerData{}
			srv := &Server{
				Logger: zap.NewNop().Sugar(),
			}
			msg.AdditionalData = tcase.data
			err := srv.parseLBData(&tcase.data, &lbData)

			if tcase.expectError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, lbData)
			}
		})
	}
}

func (suite *srvTestSuite) TestDeleteMessageHandler() {
	type testCase struct {
		name           string
		msg            pubsubx.Message
		chart          *chart.Chart
		valPath        string
		expectError    bool
		kubeclient     *rest.Config
		missingRelease bool
	}

	testDir, err := os.MkdirTemp("", "test-delete-handler")
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
			name:        "valid data",
			expectError: false,
			chart:       ch,
			kubeclient:  suite.Kubeconfig,
			valPath:     pwd + "/../../hack/ci/values.yaml",
			msg: pubsubx.Message{
				SubjectURN:     "urn:infratographer:load-balancer:" + uuid.NewString(),
				EventType:      "delete",
				Source:         "lbapi",
				Timestamp:      time.Now(),
				ActorURN:       uuid.NewString(),
				AdditionalData: map[string]interface{}{},
			},
		},
		{
			name:           "unable to remove release (missing release)",
			expectError:    true,
			missingRelease: true,
			kubeclient:     suite.Kubeconfig,
			chart:          ch,
			valPath:        pwd + "/../../hack/ci/values.yaml",
			msg: pubsubx.Message{
				SubjectURN:     uuid.NewString(),
				EventType:      "delete",
				Source:         "lbapi",
				Timestamp:      time.Now(),
				ActorURN:       uuid.NewString(),
				AdditionalData: map[string]interface{}{},
			},
		},
		{
			name:        "unable to parse data",
			expectError: true,
			chart:       ch,
			kubeclient:  suite.Kubeconfig,
			valPath:     pwd + "/../../hack/ci/values.yaml",
			msg: pubsubx.Message{
				SubjectURN: uuid.NewString(),
				EventType:  "delete",
				Source:     "lbapi",
				Timestamp:  time.Now(),
				ActorURN:   uuid.NewString(),
				AdditionalData: map[string]interface{}{
					"load_balancer_id": 1,
					"location_id":      2,
				},
			},
		},
		{
			name:        "unable to remove deployment",
			msg:         msg,
			chart:       ch,
			valPath:     pwd + "/../../hack/ci/values.yaml",
			expectError: true,
			kubeclient: &rest.Config{
				Host:                "http://localhost:45678",
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
			missingRelease: false,
		},
	}

	for _, tcase := range testCases {
		suite.T().Run(tcase.name, func(t *testing.T) {
			srv := &Server{
				Context:    context.TODO(),
				Logger:     zap.NewNop().Sugar(),
				KubeClient: suite.Kubeconfig,
				ValuesPath: tcase.valPath,
				Chart:      tcase.chart,
			}

			if !tcase.missingRelease {
				_ = srv.createMessageHandler(&tcase.msg)
			}
			err := srv.deleteMessageHandler(&tcase.msg)

			if tcase.expectError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func (suite *srvTestSuite) TestUpdateMessageHandler() {
	msg := pubsubx.Message{}
	srv := &Server{}
	err := srv.updateMessageHandler(&msg)
	assert.Nil(suite.T(), err)
}

func (suite *srvTestSuite) TestCreateMessageHandler() {
	type testCase struct {
		name        string
		msg         pubsubx.Message
		chart       *chart.Chart
		valPath     string
		expectError bool
		kubeclient  *rest.Config
		viperFlags  map[string]string
	}

	testDir, err := os.MkdirTemp("", "test-create-handler")
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
			name:        "valid data",
			expectError: false,
			chart:       ch,
			valPath:     pwd + "/../../hack/ci/values.yaml",
			kubeclient:  suite.Kubeconfig,
			msg: pubsubx.Message{
				SubjectURN: "urn:infratographer:load-balancer:" + uuid.NewString(),
				EventType:  "create",
				Source:     "lbapi",
				Timestamp:  time.Now(),
				ActorURN:   uuid.NewString(),
				AdditionalData: map[string]interface{}{
					"load_balancer_id": uuid.New(),
					"location_id":      uuid.New(),
				},
			},
		},
		{
			name: "valid overrides",
			msg: pubsubx.Message{
				SubjectURN:     "urn:infratographer:load-balancer:" + uuid.NewString(),
				EventType:      "create",
				Source:         "lbapi",
				Timestamp:      time.Now(),
				ActorURN:       uuid.NewString(),
				AdditionalData: map[string]interface{}{},
			},
			expectError: false,
			chart:       ch,
			valPath:     pwd + "/../../hack/ci/values.yaml",
			kubeclient:  suite.Kubeconfig,
			viperFlags: map[string]string{
				"helm-cpu-flag":    "resources.limits.cpu",
				"helm-memory-flag": "resources.limits.memory",
			},
		},
		{
			name:        "unable to create namespace",
			expectError: true,
			chart:       ch,
			valPath:     pwd + "/../../hack/ci/values.yaml",
			msg: pubsubx.Message{
				SubjectURN: "urn:infratographer:load-balancer:" + uuid.NewString(),
				EventType:  "create",
				Source:     "lbapi",
				Timestamp:  time.Now(),
				ActorURN:   uuid.NewString(),
				AdditionalData: map[string]interface{}{
					"load_balancer_id": uuid.New(),
					"location_id":      uuid.New(),
				},
			},
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
		{
			name:        "unable to create deployment",
			expectError: true,
			chart:       ch,
			valPath:     "",
			kubeclient:  suite.Kubeconfig,
			msg: pubsubx.Message{
				SubjectURN:     "urn:infratographer:load-balancer" + uuid.NewString(),
				EventType:      "create",
				Source:         "lbapi",
				Timestamp:      time.Now(),
				ActorURN:       uuid.NewString(),
				AdditionalData: map[string]interface{}{},
			},
		},
	}

	for _, tcase := range testCases {
		suite.T().Run(tcase.name, func(t *testing.T) {
			srv := &Server{
				Context:    context.TODO(),
				Logger:     zap.NewNop().Sugar(),
				KubeClient: tcase.kubeclient,
				ValuesPath: tcase.valPath,
				Chart:      tcase.chart,
			}

			if tcase.viperFlags != nil {
				viper.Reset()
				for key, val := range tcase.viperFlags {
					viper.Set(key, val)
				}
			}

			err := srv.createMessageHandler(&tcase.msg)

			if tcase.expectError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}
