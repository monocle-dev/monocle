package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/monocle-dev/monocle/db"
	"github.com/monocle-dev/monocle/internal/auth"
	"github.com/monocle-dev/monocle/internal/models"
)

type AuthenticatedUser struct {
	ID    uint   `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func AuthMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authHeader := ctx.GetHeader("Authorization")

		if authHeader == "" {
			ctx.JSON(401, gin.H{"error": "Authorization token is required"})
			ctx.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)

		if len(parts) != 2 || parts[0] != "Bearer" {
			ctx.JSON(401, gin.H{"error": "Invalid authorization header"})
			ctx.Abort()
			return
		}

		tokenString := parts[1]

		token, err := auth.VerifyJWT(tokenString)

		if err != nil || !token.Valid {
			ctx.JSON(401, gin.H{"error": "Invalid or expired token"})
			ctx.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)

		if !ok {
			ctx.JSON(401, gin.H{"error": "Invalid token claims"})
			ctx.Abort()
			return
		}

		userIDFloat, ok := claims["user_id"].(float64)

		if !ok {
			ctx.JSON(401, gin.H{"error": "Invalid user ID in token claims"})
			ctx.Abort()
			return
		}

		userID := uint(userIDFloat)

		var user models.User

		if err := db.DB.Where("id = ?", userID).First(&user).Error; err != nil {
			ctx.JSON(401, gin.H{"error": "User not found"})
			ctx.Abort()
			return
		}

		ctx.Set("user", AuthenticatedUser{
			ID:    user.ID,
			Name:  user.Name,
			Email: user.Email,
		})
		ctx.Next()
	}
}
