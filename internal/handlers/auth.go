package handlers

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/monocle-dev/monocle/db"
	"github.com/monocle-dev/monocle/internal/models"
	"golang.org/x/crypto/bcrypt"
)

type CreateUserRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type LoginUserRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type UserResponse struct {
	ID    uint   `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func generateJWT(userID uint, email string) (string, error) {
	var jwtSecret = os.Getenv("JWT_SECRET")

	if jwtSecret == "" {
		return "", fmt.Errorf("JWT_SECRET environment variable is not set")
	}

	claims := jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"exp":     time.Now().Add(time.Hour * 168).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtSecret))
}

func CreateUser(ctx *gin.Context) {
	var user CreateUserRequest

	if err := ctx.BindJSON(&user); err != nil {
		log.Printf("Failed to bind JSON: %v", err)
		ctx.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	var existingUser models.User

	if err := db.DB.Where("email = ?", user.Email).First(&existingUser).Error; err == nil {
		ctx.JSON(400, gin.H{"error": "User with this email already exists"})
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)

	if err != nil {
		log.Printf("Failed to hash password: %v", err)
		ctx.JSON(500, gin.H{"error": "Internal server error"})
		return
	}

	newUser := models.User{
		Name:         user.Name,
		Email:        user.Email,
		PasswordHash: string(passwordHash),
	}

	if err := db.DB.Create(&newUser).Error; err != nil {
		log.Printf("Failed to create user: %v", err)
		ctx.JSON(500, gin.H{"error": "Internal server error"})
		return
	}

	token, err := generateJWT(newUser.ID, newUser.Email)

	if err != nil {
		log.Printf("Failed to generate JWT: %v", err)
		ctx.JSON(500, gin.H{"error": "Internal server error"})
		return
	}

	ctx.JSON(201, gin.H{
		"user": UserResponse{
			ID:    newUser.ID,
			Name:  newUser.Name,
			Email: newUser.Email,
		},
		"token": token,
	})
}

func LoginUser(ctx *gin.Context) {
	var user LoginUserRequest

	if err := ctx.BindJSON(&user); err != nil {
		log.Printf("Failed to bind JSON: %v", err)
		ctx.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	var existingUser models.User

	if err := db.DB.Where("email = ?", user.Email).First(&existingUser).Error; err != nil {
		ctx.JSON(400, gin.H{"error": "Invalid email or password"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(existingUser.PasswordHash), []byte(user.Password)); err != nil {
		ctx.JSON(400, gin.H{"error": "Invalid email or password"})
		return
	}

	token, err := generateJWT(existingUser.ID, existingUser.Email)

	if err != nil {
		log.Printf("Failed to generate JWT: %v", err)
		ctx.JSON(500, gin.H{"error": "Internal server error"})
		return
	}

	ctx.JSON(200, gin.H{
		"user": UserResponse{
			ID:    existingUser.ID,
			Name:  existingUser.Name,
			Email: existingUser.Email,
		},
		"token": token,
	})
}

func GetCurrentUser(ctx *gin.Context) {
	user, exists := ctx.Get("user")

	if !exists {
		ctx.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}

	currentUser, ok := user.(UserResponse)

	if !ok {
		ctx.JSON(500, gin.H{"error": "Internal server error"})
		return
	}

	ctx.JSON(200, gin.H{
		"user": currentUser,
	})
}
