package handlers

import (
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

type CreateProjectRequest struct {
	Name           string `json:"name" binding:"required"`
	Description    string `json:"description"`
	DiscordWebhook string `json:"discord_webhook"`
	SlackWebhook   string `json:"slack_webhook"`
}

type UpdateProjectRequest struct {
	Name           string `json:"name" binding:"required"`
	Description    string `json:"description"`
	DiscordWebhook string `json:"discord_webhook"`
	SlackWebhook   string `json:"slack_webhook"`
}

type GetProjectResponse struct {
	ID             uint   `json:"id"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	OwnerID        uint   `json:"owner_id"`
	DiscordWebhook string `json:"discord_webhook"`
	SlackWebhook   string `json:"slack_webhook"`
}

func CreateProject(ctx *gin.Context) {
	var body CreateProjectRequest

	if err := ctx.BindJSON(&body); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	userID, err := utils.GetCurrentUserID(ctx)

	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	project := models.Project{
		Name:           body.Name,
		Description:    body.Description,
		OwnerID:        userID,
		DiscordWebhook: body.DiscordWebhook,
		SlackWebhook:   body.SlackWebhook,
	}

	if err := db.DB.Create(&project).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create project"})
		return
	}

	ctx.JSON(http.StatusCreated, GetProjectResponse{
		ID:             project.ID,
		Name:           project.Name,
		Description:    project.Description,
		OwnerID:        project.OwnerID,
		DiscordWebhook: project.DiscordWebhook,
		SlackWebhook:   project.SlackWebhook,
	})
}

func ListProjects(ctx *gin.Context) {
	userID, err := utils.GetCurrentUserID(ctx)

	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var projects []models.Project

	if err := db.DB.Select("id, name, description, owner_id, discord_webhook, slack_webhook").Where("owner_id = ?", userID).Find(&projects).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve projects"})
		return
	}

	response := make([]GetProjectResponse, 0, len(projects))

	for _, project := range projects {
		response = append(response, GetProjectResponse{
			ID:             project.ID,
			Name:           project.Name,
			Description:    project.Description,
			OwnerID:        project.OwnerID,
			DiscordWebhook: project.DiscordWebhook,
			SlackWebhook:   project.SlackWebhook,
		})
	}

	ctx.JSON(http.StatusOK, response)
}

func UpdateProject(ctx *gin.Context) {
	userID, err := utils.GetCurrentUserID(ctx)

	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var body UpdateProjectRequest

	if err := ctx.BindJSON(&body); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	var project models.Project

	projectIDStr := ctx.Param("project_id")
	projectID, err := strconv.ParseUint(projectIDStr, 10, 64)

	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	if err := db.DB.Where("id = ? AND owner_id = ?", projectID, userID).First(&project).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		} else {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve project"})
		}
		return
	}

	project.Name = body.Name
	project.Description = body.Description
	project.DiscordWebhook = body.DiscordWebhook
	project.SlackWebhook = body.SlackWebhook

	if err := db.DB.Save(&project).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update project"})
		return
	}

	ctx.JSON(http.StatusOK, GetProjectResponse{
		ID:             project.ID,
		Name:           project.Name,
		Description:    project.Description,
		OwnerID:        project.OwnerID,
		DiscordWebhook: project.DiscordWebhook,
		SlackWebhook:   project.SlackWebhook,
	})
}

func DeleteProject(ctx *gin.Context) {
	var project models.Project
	projectIDStr := ctx.Param("project_id")

	projectID, err := strconv.ParseUint(projectIDStr, 10, 64)

	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	userID, err := utils.GetCurrentUserID(ctx)

	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	if err := db.DB.Where("id = ? AND owner_id = ?", projectID, userID).First(&project).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		} else {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve project"})
		}
		return
	}

	tx := db.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Get all monitors for this project to stop them from scheduler
	var monitors []models.Monitor
	if err := tx.Where("project_id = ?", projectID).Find(&monitors).Error; err != nil {
		tx.Rollback()
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve monitors"})
		return
	}

	// Stop all monitors from scheduler
	for _, monitor := range monitors {
		scheduler.RemoveMonitor(monitor.ID)
	}

	// Delete all monitor checks for monitors in this project
	if err := tx.Where("monitor_id IN (SELECT id FROM monitors WHERE project_id = ?)", projectID).Delete(&models.MonitorCheck{}).Error; err != nil {
		tx.Rollback()
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete monitor checks"})
		return
	}

	// Delete all monitors for this project
	if err := tx.Where("project_id = ?", projectID).Delete(&models.Monitor{}).Error; err != nil {
		tx.Rollback()
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete monitors"})
		return
	}

	// Delete the project itself
	if err := tx.Delete(&project).Error; err != nil {
		tx.Rollback()
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete project"})
		return
	}

	if err := tx.Commit().Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit deletion"})
		return
	}

	ctx.Status(http.StatusNoContent)
}
