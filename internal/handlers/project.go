package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/monocle-dev/monocle/db"
	"github.com/monocle-dev/monocle/internal/models"
	"github.com/monocle-dev/monocle/internal/utils"
	"gorm.io/gorm"
)

type CreateProjectRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

type UpdateProjectRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

type GetProjectResponse struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	OwnerID     uint   `json:"owner_id"`
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
		Name:        body.Name,
		Description: body.Description,
		OwnerID:     userID,
	}

	if err := db.DB.Create(&project).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create project"})
		return
	}

	ctx.JSON(http.StatusCreated, GetProjectResponse{
		ID:          project.ID,
		Name:        project.Name,
		Description: project.Description,
		OwnerID:     project.OwnerID,
	})
}

func ListProjects(ctx *gin.Context) {
	userID, err := utils.GetCurrentUserID(ctx)

	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var projects []models.Project

	if err := db.DB.Where("owner_id = ?", userID).Find(&projects).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve projects"})
		return
	}

	var response []GetProjectResponse

	for _, project := range projects {
		response = append(response, GetProjectResponse{
			ID:          project.ID,
			Name:        project.Name,
			Description: project.Description,
			OwnerID:     project.OwnerID,
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
	projectID := ctx.Param("id")

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

	if err := db.DB.Save(&project).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update project"})
		return
	}

	ctx.JSON(http.StatusOK, GetProjectResponse{
		ID:          project.ID,
		Name:        project.Name,
		Description: project.Description,
		OwnerID:     project.OwnerID,
	})
}

func DeleteProject(ctx *gin.Context) {
	var project models.Project
	projectID := ctx.Param("id")

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

	if err := db.DB.Delete(&project).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete project"})
		return
	}

	ctx.Status(http.StatusNoContent)
}
