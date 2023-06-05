package srv

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"go.infratographer.com/x/versionx"
)

func (s *Server) versionHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, echo.Map{
		"version": versionx.BuildDetails(),
	})
}

func (s *Server) Routes(g *echo.Group) {
	g.GET("/version", s.versionHandler)
}
