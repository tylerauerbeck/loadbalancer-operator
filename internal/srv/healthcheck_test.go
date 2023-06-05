package srv

import (
	"net/http"
	"net/http/httptest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.infratographer.com/x/echox"
)

func (suite srvTestSuite) TestVersionHandler() { // nolint:govet
	srv, err := echox.NewServer(zap.NewNop(), echox.Config{}, nil)

	require.NoError(suite.T(), err, "unexpected error creating new server")

	s := &Server{
		Echo: srv,
	}

	s.Echo.AddHandler(s)

	req := httptest.NewRequest("GET", "/version", nil)
	rec := httptest.NewRecorder()
	s.Echo.Handler().ServeHTTP(rec, req)

	assert.Equal(suite.T(), http.StatusOK, rec.Code)
}
