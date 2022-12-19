package srv

import (
	"context"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func TestGetHelmValues(t *testing.T) {
	type testCase struct {
		name        string
		content     *fstest.MapFS
		file        string
		expectError bool
	}

	testCases := []testCase{
		{
			name:        "valid yaml",
			expectError: false,
			file:        "values.yaml",
			content: &fstest.MapFS{
				"values.yaml": {
					Data: []byte("hello: world"),
				},
			},
		},
		{
			name:        "invalid yaml",
			expectError: true,
			file:        "values.yaml",
			content: &fstest.MapFS{
				"values.yaml": {
					Data: []byte("hello there"),
				},
			},
		},
	}

	for _, tcase := range testCases {
		t.Run(tcase.name, func(t *testing.T) {
			content, _ := tcase.content.ReadFile(tcase.file)
			values, err := getHelmValues(content)

			if tcase.expectError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, values)
			}
		})
	}
}

func TestCreateNamespace(t *testing.T) {
	type testCase struct {
		name         string
		appNamespace string
		expectError  bool
		kubeclient   *rest.Config
	}

	env := envtest.Environment{}

	cfg, err := env.Start()
	if err != nil {
		panic(err)
	}

	testCases := []testCase{
		{
			name:         "valid yaml",
			expectError:  false,
			appNamespace: "flintlock",
			kubeclient:   cfg,
		},
		{
			name:         "invalid namespace",
			expectError:  true,
			appNamespace: "DarkwingDuck",
			kubeclient:   cfg,
		},
	}

	for _, tcase := range testCases {
		t.Run(tcase.name, func(t *testing.T) {
			srv := Server{
				Context:    context.TODO(),
				Logger:     setupTestLogger(t, tcase.name),
				KubeClient: tcase.kubeclient,
			}

			err := srv.CreateNamespace(tcase.appNamespace)

			if tcase.expectError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}

	err = env.Stop()
	if err != nil {
		panic(err)
	}
}

func TestCreateApp(t *testing.T) {
	type testCase struct {
		name         string
		appNamespace string
		appName      string
		expectError  bool
	}

	env := envtest.Environment{}

	cfg, err := env.Start()
	if err != nil {
		panic(err)
	}

	testCases := []testCase{
		{
			name:         "valid yaml",
			expectError:  false,
			appNamespace: uuid.New().String(),
			appName:      uuid.New().String(),
		},
		{
			name:         "invalid namespace",
			expectError:  true,
			appNamespace: "DarkwingDuck",
			appName:      uuid.New().String(),
		},
	}

	for _, tcase := range testCases {
		t.Run(tcase.name, func(t *testing.T) {
			srv := Server{
				Context:    context.TODO(),
				Logger:     setupTestLogger(t, tcase.name),
				KubeClient: cfg,
				ChartPath:  "/tmp/chart.tgz",
				ValuesPath: "/tmp/values.yaml",
			}

			_ = srv.CreateNamespace(tcase.appNamespace)
			err = srv.CreateApp(tcase.appName, srv.ChartPath, tcase.appNamespace)

			if tcase.expectError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}

	err = env.Stop()
	if err != nil {
		panic(err)
	}
}
