package utils

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
)

func GetProjectID(ctx *gin.Context) (uint64, error) {
	var err error

	projectIDStr := ctx.Param("project_id")

	if projectIDStr == "" {
		return 0, errors.New("Project ID not found")
	}

	projectID, err := strconv.ParseUint(projectIDStr, 10, 32)

	if err != nil {
		return 0, errors.New("Invalid Project ID")
	}

	return projectID, err
}

func GetMonitorID(ctx *gin.Context) (uint64, error) {
	var err error

	monitorIDStr := ctx.Param("monitor_id")

	if monitorIDStr == "" {
		return 0, errors.New("Monitor ID not found")
	}

	monitorID, err := strconv.ParseUint(monitorIDStr, 10, 32)

	if err != nil {
		return 0, errors.New("Invalid Monitor ID")
	}

	return monitorID, err
}

func GetProjectMonitorID(ctx *gin.Context) (uint64, uint64, error) {
	var err error

	projectID, err := GetProjectID(ctx)

	if err != nil {
		return 0, 0, err
	}

	monitorID, err := GetMonitorID(ctx)

	if err != nil {
		return 0, 0, err
	}

	return projectID, monitorID, err
}
