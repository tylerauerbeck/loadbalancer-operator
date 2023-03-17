package srv

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"go.infratographer.com/loadbalanceroperator/internal/utils"
)

func (suite srvTestSuite) TestLivezHandler() { //nolint:govet
	s := &Server{
		Gin: gin.Default(),
	}

	s.configureHealthcheck()

	req := httptest.NewRequest("GET", "/livez", nil)
	rec := httptest.NewRecorder()
	s.Gin.ServeHTTP(rec, req)

	assert.Equal(suite.T(), 200, rec.Code)
}

func (suite srvTestSuite) TestHealthzHandler() { //nolint:govet
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
				Gin:             gin.Default(),
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

			req := httptest.NewRequest("GET", "/healthz", nil)
			rec := httptest.NewRecorder()
			s.Gin.ServeHTTP(rec, req)

			if tc.expectError {
				assert.Equal(suite.T(), http.StatusInternalServerError, rec.Code)
			} else {
				assert.Equal(suite.T(), http.StatusOK, rec.Code)
			}
		})
	}
}

func (suite srvTestSuite) TestVersionHandler() { // nolint:govet
	s := &Server{
		Gin: gin.Default(),
	}

	s.configureHealthcheck()

	req := httptest.NewRequest("GET", "/version", nil)
	rec := httptest.NewRecorder()
	s.Gin.ServeHTTP(rec, req)

	assert.Equal(suite.T(), 200, rec.Code)
}
