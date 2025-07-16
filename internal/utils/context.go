package utils

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/monocle-dev/monocle/internal/middleware"
	"github.com/monocle-dev/monocle/internal/types"
)

func GetCurrentUser(ctx *gin.Context) (middleware.AuthenticatedUser, error) {
	user, exists := ctx.Get(types.ContextUserKey)

	if !exists {
		return middleware.AuthenticatedUser{}, fmt.Errorf("User not authenticated")
	}

	authenticatedUser, ok := user.(middleware.AuthenticatedUser)

	if !ok {
		return middleware.AuthenticatedUser{}, fmt.Errorf("Invalid user type in context")
	}

	return authenticatedUser, nil
}

func GetCurrentUserID(ctx *gin.Context) (uint, error) {
	user, err := GetCurrentUser(ctx)

	if err != nil {
		return 0, err
	}

	return user.ID, nil
}
