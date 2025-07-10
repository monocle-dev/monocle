package router

import (
	"github.com/gin-gonic/gin"
	"github.com/monocle-dev/monocle/internal/handlers"
	"github.com/monocle-dev/monocle/internal/middleware"
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
			auth.GET("/me", middleware.AuthMiddleware(), handlers.GetCurrentUser)
		}
	}

	return r
}
