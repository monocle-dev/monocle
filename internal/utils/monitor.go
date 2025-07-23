package utils

import (
	"errors"
	"net/url"
	"strconv"
	"strings"

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

	return projectID, nil
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

	return monitorID, nil
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

	return projectID, monitorID, nil
}

func ExtractRawDomain(input string) (string, error) {
	if input == "" {
		return "", errors.New("input cannot be empty")
	}

	// Clean up the input
	domain := strings.TrimSpace(input)

	// If it looks like a URL, parse it
	if strings.Contains(domain, "://") {
		parsedURL, err := url.Parse(domain)
		if err != nil {
			return "", errors.New("invalid URL format")
		}

		if parsedURL.Hostname() == "" {
			return "", errors.New("no hostname found in URL")
		}

		domain = parsedURL.Hostname()
	}

	// Remove trailing slashes
	domain = strings.TrimSuffix(domain, "/")

	// Remove www. prefix if present
	if strings.HasPrefix(strings.ToLower(domain), "www.") {
		domain = domain[4:] // Remove "www."
	}

	// Final validation - ensure we have a valid domain
	if domain == "" {
		return "", errors.New("invalid domain after processing")
	}

	return domain, nil
}
