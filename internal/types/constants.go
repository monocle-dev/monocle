package types

import (
	"os"
	"strings"
)

const ContextUserKey = "user"

var (
	// Default allowed origins for development
	defaultOrigins = []string{
		"http://localhost:3000",
		"http://localhost:5173",
	}

	AllowedOrigins = initAllowedOrigins()
)

func initAllowedOrigins() []string {
	origins := make([]string, len(defaultOrigins))
	copy(origins, defaultOrigins)

	if clientURL := os.Getenv("CLIENT_URL"); clientURL != "" {
		origins = append(origins, clientURL)
	}

	if allowedOrigins := os.Getenv("ALLOWED_ORIGINS"); allowedOrigins != "" {
		envOrigins := strings.Split(allowedOrigins, ",")
		for _, origin := range envOrigins {
			trimmed := strings.TrimSpace(origin)
			if trimmed != "" {
				origins = append(origins, trimmed)
			}
		}
	}

	return origins
}
