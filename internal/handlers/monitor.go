package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

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

type MonitorSummary struct {
	ID           uint                   `json:"id"`
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`
	Status       string                 `json:"status"`
	Interval     int                    `json:"interval"`
	Config       map[string]interface{} `json:"config"`
	LastCheck    *MonitorCheckSummary   `json:"last_check"`
	Uptime       float64                `json:"uptime_percentage"`
	ResponseTime float64                `json:"avg_response_time"`
}

type MonitorCheckSummary struct {
	ID           uint      `json:"id"`
	Status       string    `json:"status"`
	ResponseTime int       `json:"response_time"`
	Message      string    `json:"message"`
	CheckedAt    time.Time `json:"checked_at"`
}

type DashboardResponse struct {
	Project         ProjectSummary    `json:"project"`
	MonitorsSummary MonitorsSummary   `json:"monitors_summary"`
	Monitors        []MonitorSummary  `json:"monitors"`
	RecentIncidents []IncidentSummary `json:"recent_incidents"`
}

type ProjectSummary struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type MonitorsSummary struct {
	Total   int `json:"total"`
	Active  int `json:"active"`
	Down    int `json:"down"`
	Warning int `json:"warning"`
}

type IncidentSummary struct {
	ID          uint      `json:"id"`
	MonitorName string    `json:"monitor_name"`
	Severity    string    `json:"severity"`
	Status      string    `json:"status"`
	StartedAt   time.Time `json:"started_at"`
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
	projectIDStr := ctx.Param("project_id")

	projectID, err := strconv.ParseUint(projectIDStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Project ID"})
		return
	}

	userID, err := utils.GetCurrentUserID(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Verify project ownership
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
	if err := db.DB.Where("project_id = ?", projectID).Find(&monitors).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve monitors"})
		return
	}

	var monitorSummaries []MonitorSummary
	for _, monitor := range monitors {
		summary, err := buildMonitorSummary(monitor)
		if err != nil {
			log.Printf("Failed to build summary for monitor %d: %v", monitor.ID, err)
			continue
		}
		monitorSummaries = append(monitorSummaries, summary)
	}

	ctx.JSON(http.StatusOK, monitorSummaries)
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

func buildMonitorSummary(monitor models.Monitor) (MonitorSummary, error) {
	// Get last check
	var lastCheck models.MonitorCheck
	lastCheckFound := true
	if err := db.DB.Where("monitor_id = ?", monitor.ID).
		Order("checked_at DESC").
		First(&lastCheck).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return MonitorSummary{}, err
		}
		lastCheckFound = false
	}

	// Calculate uptime (last 24 hours)
	uptime := calculateUptime(monitor.ID)

	// Calculate average response time (last 24 hours)
	avgResponseTime := calculateAverageResponseTime(monitor.ID)

	// Parse config
	var config map[string]interface{}
	if err := json.Unmarshal(monitor.Config, &config); err != nil {
		config = make(map[string]interface{})
	}

	summary := MonitorSummary{
		ID:           monitor.ID,
		Name:         monitor.Name,
		Type:         monitor.Type,
		Status:       monitor.Status,
		Interval:     monitor.Interval,
		Config:       config,
		Uptime:       uptime,
		ResponseTime: avgResponseTime,
	}

	if lastCheckFound {
		summary.LastCheck = &MonitorCheckSummary{
			ID:           lastCheck.ID,
			Status:       lastCheck.Status,
			ResponseTime: lastCheck.ResponseTime,
			Message:      lastCheck.Message,
			CheckedAt:    lastCheck.CheckedAt,
		}
	}

	return summary, nil
}

func calculateUptime(monitorID uint) float64 {
	var total, successful int64

	// Count total checks in last 24 hours
	db.DB.Model(&models.MonitorCheck{}).
		Where("monitor_id = ? AND checked_at > ?", monitorID, time.Now().Add(-24*time.Hour)).
		Count(&total)

	// Count successful checks
	db.DB.Model(&models.MonitorCheck{}).
		Where("monitor_id = ? AND status = 'success' AND checked_at > ?", monitorID, time.Now().Add(-24*time.Hour)).
		Count(&successful)

	if total == 0 {
		return 100.0
	}

	return float64(successful) / float64(total) * 100
}

func calculateAverageResponseTime(monitorID uint) float64 {
	var avg float64

	db.DB.Model(&models.MonitorCheck{}).
		Select("AVG(response_time)").
		Where("monitor_id = ? AND status = 'success' AND checked_at > ?", monitorID, time.Now().Add(-24*time.Hour)).
		Scan(&avg)

	return avg
}

func GetDashboard(ctx *gin.Context) {
	projectIDStr := ctx.Param("project_id")

	projectID, err := strconv.ParseUint(projectIDStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Project ID"})
		return
	}

	userID, err := utils.GetCurrentUserID(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Verify project ownership
	var project models.Project
	if err := db.DB.Where("id = ? AND owner_id = ?", projectID, userID).First(&project).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	// Get monitors with enhanced data
	var monitors []models.Monitor
	if err := db.DB.Where("project_id = ?", projectID).Find(&monitors).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve monitors"})
		return
	}

	var monitorSummaries []MonitorSummary
	var totalMonitors, activeMonitors, downMonitors, warningMonitors int

	for _, monitor := range monitors {
		summary, err := buildMonitorSummary(monitor)
		if err != nil {
			continue
		}

		monitorSummaries = append(monitorSummaries, summary)
		totalMonitors++

		if monitor.Status == "active" {
			if summary.LastCheck != nil {
				switch summary.LastCheck.Status {
				case "success":
					activeMonitors++
				case "failure":
					downMonitors++
				default:
					warningMonitors++
				}
			} else {
				warningMonitors++
			}
		}
	}

	// Get recent incidents
	var incidents []models.Incident
	db.DB.Joins("JOIN monitors ON monitors.id = incidents.monitor_id").
		Where("monitors.project_id = ? AND incidents.created_at > ?", projectID, time.Now().Add(-7*24*time.Hour)).
		Order("incidents.created_at DESC").
		Limit(10).
		Find(&incidents)

	var incidentSummaries []IncidentSummary
	for _, incident := range incidents {
		var monitor models.Monitor
		db.DB.First(&monitor, incident.MonitorID)

		startedAt := time.Time{}
		if incident.StartedAt != nil {
			startedAt = *incident.StartedAt
		}

		incidentSummaries = append(incidentSummaries, IncidentSummary{
			ID:          incident.ID,
			MonitorName: monitor.Name,
			Severity:    incident.Severity,
			Status:      incident.Status,
			StartedAt:   startedAt,
		})
	}

	response := DashboardResponse{
		Project: ProjectSummary{
			ID:          project.ID,
			Name:        project.Name,
			Description: project.Description,
		},
		MonitorsSummary: MonitorsSummary{
			Total:   totalMonitors,
			Active:  activeMonitors,
			Down:    downMonitors,
			Warning: warningMonitors,
		},
		Monitors:        monitorSummaries,
		RecentIncidents: incidentSummaries,
	}

	ctx.JSON(http.StatusOK, response)
}
