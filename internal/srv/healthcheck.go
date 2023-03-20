package srv

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) healthzHandler(c *gin.Context) {
	for _, s := range s.Subscriptions {
		if !s.IsValid() {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status": fmt.Sprintf("Subscription %s is unavailable", s.Subject),
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "all subscriptions available",
	})
}

func (s *Server) livezHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

func (s *Server) versionHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"version": "0.0.1",
	})
}

func (s *Server) configureHealthcheck() {
	s.Gin.GET("/healthz", s.healthzHandler)
	s.Gin.GET("/livez", s.livezHandler)
	s.Gin.GET("/version", s.versionHandler)
}
