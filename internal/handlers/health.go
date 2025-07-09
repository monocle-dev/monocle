package handlers

import (
	"time"

	"github.com/gin-gonic/gin"
)

func HealthCheck(c *gin.Context) {
	c.JSON(200, gin.H{
		"status":    "ok",
		"message":   "Monocle is running",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}
