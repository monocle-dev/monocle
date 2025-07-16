package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/monocle-dev/monocle/db"
	"github.com/monocle-dev/monocle/internal/auth"
	"github.com/monocle-dev/monocle/internal/models"
	"github.com/monocle-dev/monocle/internal/types"
)

type AuthenticatedUser struct {
	ID    uint   `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func AuthMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		cookie, err := ctx.Cookie("token")

		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization token is required"})
			return
		}

		token, err := auth.VerifyJWT(cookie)

		if err != nil || !token.Valid {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)

		if !ok {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			return
		}

		userIDFloat, ok := claims["user_id"].(float64)

		if !ok {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID in token claims"})
			return
		}

		userID := uint(userIDFloat)

		var user models.User

		if err := db.DB.Where("id = ?", userID).First(&user).Error; err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
			return
		}

		ctx.Set(types.ContextUserKey, AuthenticatedUser{
			ID:    user.ID,
			Name:  user.Name,
			Email: user.Email,
		})
		ctx.Next()
	}
}
