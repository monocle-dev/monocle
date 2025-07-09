package router

import (
	"github.com/gin-gonic/gin"
	"github.com/monocle-dev/monocle/internal/handlers"
)

func NewRouter() *gin.Engine {
	r := gin.Default()

	r.GET("/health", handlers.HealthCheck)

	return r
}
