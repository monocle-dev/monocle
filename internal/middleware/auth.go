package middleware

import (
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/monocle-dev/monocle/db"
	"github.com/monocle-dev/monocle/internal/handlers"
	"github.com/monocle-dev/monocle/internal/models"
)

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
		jwtSecret := os.Getenv("JWT_SECRET")

		if jwtSecret == "" {
			ctx.JSON(500, gin.H{"error": "JWT_SECRET environment variable is not set"})
			ctx.Abort()
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}

			return []byte(jwtSecret), nil
		})

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

		ctx.Set("user", handlers.UserResponse{
			ID:    user.ID,
			Name:  user.Name,
			Email: user.Email,
		})
		ctx.Next()
	}
}
