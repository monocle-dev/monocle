package router

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/monocle-dev/monocle/internal/handlers"
	"github.com/monocle-dev/monocle/internal/middleware"
)

func NewRouter() *gin.Engine {
	r := gin.Default()

	// Add CORS middleware
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:3001", "http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization", "Accept", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	api := r.Group("/api")
	{
		api.GET("/health", handlers.HealthCheck)
		api.GET("/ws/:project_id", middleware.AuthMiddleware(), handlers.WebSocket)
		auth := api.Group("/auth")
		{
			auth.POST("/register", handlers.CreateUser)
			auth.POST("/login", handlers.LoginUser)
			auth.GET("/me", middleware.AuthMiddleware(), handlers.Me)
		}

		projects := api.Group("/projects", middleware.AuthMiddleware())
		{
			projects.POST("", handlers.CreateProject)
			projects.GET("", handlers.ListProjects)
			projects.PATCH("/:project_id", handlers.UpdateProject)
			projects.DELETE("/:project_id", handlers.DeleteProject)

			// Dashboard endpoint
			projects.GET("/:project_id/dashboard", handlers.GetDashboard)

			// Monitor endpoints
			projects.POST("/:project_id/monitors", handlers.CreateMonitor)
			projects.GET("/:project_id/monitors", handlers.GetMonitors)
			projects.PUT("/:project_id/monitors/:monitor_id", handlers.UpdateMonitor)
			projects.GET("/:project_id/monitors/:monitor_id/checks", handlers.GetMonitorChecks)
			projects.DELETE("/:project_id/monitors/:monitor_id", handlers.DeleteMonitor)
		}
	}

	return r
}
