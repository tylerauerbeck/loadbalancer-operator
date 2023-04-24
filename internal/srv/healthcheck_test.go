package srv

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"go.infratographer.com/x/echox"

	"go.infratographer.com/loadbalanceroperator/internal/utils"
)

func (suite srvTestSuite) TestNatsHealthcheck() { //nolint:govet
	type testCase struct {
		name        string
		subjects    []string
		expectError bool
	}

	js := utils.GetJetstreamConnection(suite.NATSServer)

	_, _ = js.AddStream(&nats.StreamConfig{
		Name:     "TestHealthcheck",
		Subjects: []string{"test.foo", "test.bar"},
		MaxBytes: 1024,
	})

	testCases := []testCase{
		{
			name: "active subscriptions",
			subjects: []string{
				"foo",
				"bar",
			},
			expectError: false,
		},
		{
			name: "no active subscriptions",
			subjects: []string{
				"foo",
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			s := &Server{
				JetstreamClient: js,
				Echo:            echox.NewServer(zap.NewNop(), echox.Config{}, nil),
				Subjects:        tc.subjects,
				Prefix:          "test",
				StreamName:      "TestHealthcheck",
				Logger:          zap.NewNop().Sugar(),
			}

			_ = s.configureSubscribers()
			s.configureHealthcheck()

			if tc.expectError {
				_ = s.Subscriptions[0].Unsubscribe()
			}

			req := httptest.NewRequest("GET", "/readyz", nil)
			rec := httptest.NewRecorder()

			s.Echo.Handler().ServeHTTP(rec, req)

			if tc.expectError {
				assert.Equal(suite.T(), http.StatusServiceUnavailable, rec.Code)
			} else {
				assert.Equal(suite.T(), http.StatusOK, rec.Code)
			}
		})
	}
}

func (suite srvTestSuite) TestVersionHandler() { // nolint:govet
	s := &Server{
		Echo: echox.NewServer(zap.NewNop(), echox.Config{}, nil),
	}

	s.Echo.AddHandler(s)

	req := httptest.NewRequest("GET", "/version", nil)
	rec := httptest.NewRecorder()
	s.Echo.Handler().ServeHTTP(rec, req)

	assert.Equal(suite.T(), http.StatusOK, rec.Code)
}
