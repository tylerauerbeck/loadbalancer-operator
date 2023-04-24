package srv

import (
	"context"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
)

func (s *Server) natsHealthcheck(ctx context.Context) error {
	for _, s := range s.Subscriptions {
		if !s.IsValid() {
			return fmt.Errorf("subscription %s is unavailable", s.Subject) //nolint:goerr113 // allowing for healthcheck
		}
	}

	return nil
}

func (s *Server) versionHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, echo.Map{
		"version": "0.0.1",
	})
}

func (s *Server) configureHealthcheck() {
	s.Echo.AddReadinessCheck("nats", s.natsHealthcheck)
}

func (s *Server) Routes(g *echo.Group) {
	g.GET("/version", s.versionHandler)
}
