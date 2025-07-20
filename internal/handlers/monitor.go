package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/monocle-dev/monocle/db"
	"github.com/monocle-dev/monocle/internal/models"
	"github.com/monocle-dev/monocle/internal/scheduler"
	"github.com/monocle-dev/monocle/internal/utils"
	"gorm.io/gorm"
)

type CreateMonitorRequest struct {
	Name     string                 `json:"name" binding:"required"`
	Type     string                 `json:"type" binding:"required"`     // "http", "https", "ssl", "dns", "database"
	Interval int                    `json:"interval" binding:"required"` // Interval in seconds
	Config   map[string]interface{} `json:"config" binding:"required"`   // Configuration specific to the monitor type
}

type UpdateMonitorRequest struct {
	Name     string                 `json:"name" binding:"required"`
	Type     string                 `json:"type" binding:"required"`
	Interval int                    `json:"interval" binding:"required"`
	Config   map[string]interface{} `json:"config" binding:"required"`
}

func CreateMonitor(ctx *gin.Context) {
	var req CreateMonitorRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	projectID, err := utils.GetProjectID(ctx)

	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	userID, err := utils.GetCurrentUserID(ctx)

	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var project models.Project

	if err := db.DB.Where("id = ? AND owner_id = ?", projectID, userID).First(&project).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		} else {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve project"})
		}
		return
	}

	configJSON, err := json.Marshal(req.Config)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid config format"})
		return
	}

	monitor := models.Monitor{
		ProjectID: uint(projectID),
		Name:      req.Name,
		Type:      req.Type,
		Status:    "active",
		Interval:  req.Interval,
		Config:    configJSON,
	}

	if err := db.DB.Create(&monitor).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create monitor"})
		return
	}

	// Add monitor to scheduler
	scheduler.AddMonitor(monitor)
	ctx.JSON(http.StatusCreated, gin.H{"message": "Monitor created successfully", "monitor_id": monitor.ID})
}

func DeleteMonitor(ctx *gin.Context) {
	userID, err := utils.GetCurrentUserID(ctx)

	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	projectID, monitorID, err := utils.GetProjectMonitorID(ctx)

	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	var monitor models.Monitor

	if err := db.DB.Where("id = ? AND project_id = ? AND project.owner_id = ?", monitorID, projectID, userID).
		Joins("JOIN projects ON projects.id = monitors.project_id").
		First(&monitor).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Monitor not found"})
		} else {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve monitor"})
		}
		return
	}

	if err := db.DB.Delete(&monitor).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete monitor"})
		return
	}

	// Remove monitor from scheduler
	scheduler.RemoveMonitor(monitor.ID)

	ctx.JSON(http.StatusNoContent, gin.H{"message": "Monitor deleted successfully"})
}

func GetMonitors(ctx *gin.Context) {
	userID, err := utils.GetCurrentUserID(ctx)

	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	projectID, monitorID, err := utils.GetProjectMonitorID(ctx)

	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err})
	}

	var project models.Project

	if err := db.DB.Where("id = ? AND owner_id = ?", projectID, userID).First(&project).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		} else {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve project"})
		}

		return
	}

	var monitors []models.Monitor

	if err := db.DB.Where("project_id = ? AND id = ?", projectID, monitorID).Find(&monitors).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Monitor not found"})
		} else {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve monitor"})
		}
		return
	}

	ctx.JSON(http.StatusOK, monitors)
}

func GetMonitorChecks(ctx *gin.Context) {
	projectIDStr := ctx.Param("project_id")
	monitorIDStr := ctx.Param("monitor_id")

	projectID, err := strconv.ParseUint(projectIDStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Project ID"})
		return
	}

	monitorID, err := strconv.ParseUint(monitorIDStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Monitor ID"})
		return
	}

	userID, err := utils.GetCurrentUserID(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Verify ownership through project
	var monitor models.Monitor
	if err := db.DB.Joins("JOIN projects ON projects.id = monitors.project_id").
		Where("monitors.id = ? AND monitors.project_id = ? AND projects.owner_id = ?", monitorID, projectID, userID).
		First(&monitor).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Monitor not found"})
		return
	}

	var checks []models.MonitorCheck
	if err := db.DB.Select("id, monitor_id, status, response_time, message, checked_at, created_at").
		Where("monitor_id = ?", monitorID).
		Order("checked_at DESC").
		Limit(50).
		Find(&checks).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get checks"})
		return
	}

	ctx.JSON(http.StatusOK, checks)
}

func UpdateMonitor(ctx *gin.Context) {
	userID, err := utils.GetCurrentUserID(ctx)

	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	projectID, monitorID, err := utils.GetProjectMonitorID(ctx)

	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	var req UpdateMonitorRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var monitor models.Monitor

	if err := db.DB.Joins("JOIN projects ON projects.id = monitors.project_id").
		Where("monitors.id = ? AND monitors.project_id = ? AND projects.owner_id = ?", monitorID, projectID, userID).
		First(&monitor).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Monitor not found"})
		return
	}

	monitor.Name = req.Name
	monitor.Type = req.Type
	monitor.Interval = req.Interval
	configJSON, err := json.Marshal(req.Config)

	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid config format"})
		return
	}

	monitor.Config = configJSON

	if err := db.DB.Save(&monitor).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update monitor"})
		return
	}

	scheduler.UpdateMonitor(monitor)

	ctx.JSON(http.StatusOK, gin.H{"message": "Monitor updated successfully", "monitor_id": monitor.ID})
}
