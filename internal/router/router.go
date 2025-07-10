package router

import (
	"github.com/gin-gonic/gin"
	"github.com/monocle-dev/monocle/internal/handlers"
)

func NewRouter() *gin.Engine {
	r := gin.Default()

	r.GET("/health", handlers.HealthCheck)

	api := r.Group("/api")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", handlers.CreateUser)
			auth.POST("/login", handlers.LoginUser)

		}
	}

	return r
}
