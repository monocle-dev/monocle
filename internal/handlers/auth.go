package handlers

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/monocle-dev/monocle/db"
	"github.com/monocle-dev/monocle/internal/auth"
	"github.com/monocle-dev/monocle/internal/models"
	"github.com/monocle-dev/monocle/internal/types"
	"github.com/monocle-dev/monocle/internal/utils"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
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

func CreateUser(ctx *gin.Context) {
	var user CreateUserRequest

	if err := ctx.BindJSON(&user); err != nil {
		log.Printf("Failed to bind JSON: %v", err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	user.Email = strings.ToLower(strings.TrimSpace(user.Email))

	var existingUser models.User

	err := db.DB.Where("email = ?", user.Email).First(&existingUser).Error

	if err == nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Email already exists"})
		return
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Printf("Database error when checking existing user: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)

	if err != nil {
		log.Printf("Failed to hash password: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	newUser := models.User{
		Name:         user.Name,
		Email:        user.Email,
		PasswordHash: string(passwordHash),
	}

	if err := db.DB.Create(&newUser).Error; err != nil {
		log.Printf("Failed to create user: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	token, err := auth.GenerateJWT(newUser.ID, newUser.Email)

	if err != nil {
		log.Printf("Failed to generate JWT: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	ctx.SetCookie(
		"token",    // cookie name
		token,      // value
		60*60*24*7, // maxAge (7 days in seconds)
		"/",        // path
		"",         // domain (empty = current domain)
		true,       // secure (set to true in production with HTTPS)
		true,       // httpOnly
	)

	ctx.JSON(http.StatusCreated, gin.H{
		"user": types.UserResponse{
			ID:    newUser.ID,
			Name:  newUser.Name,
			Email: newUser.Email,
		},
	})
}

func LoginUser(ctx *gin.Context) {
	var user LoginUserRequest

	if err := ctx.BindJSON(&user); err != nil {
		log.Printf("Failed to bind JSON: %v", err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	var existingUser models.User

	err := db.DB.Where("email = ?", user.Email).First(&existingUser).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid email or password"})
			return
		}
		log.Printf("Database error when fetching user: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(existingUser.PasswordHash), []byte(user.Password))

	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid email or password"})
		return
	}

	token, err := auth.GenerateJWT(existingUser.ID, existingUser.Email)

	if err != nil {
		log.Printf("Failed to generate JWT: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	ctx.SetCookie(
		"token",    // cookie name
		token,      // value
		60*60*24*7, // maxAge (7 days in seconds)
		"/",        // path
		"",         // domain (empty = current domain)
		true,       // secure (set to true in production with HTTPS)
		true,       // httpOnly
	)

	ctx.JSON(http.StatusOK, gin.H{
		"user": types.UserResponse{
			ID:    existingUser.ID,
			Name:  existingUser.Name,
			Email: existingUser.Email,
		},
	})
}

func Me(ctx *gin.Context) {
	currentUser, err := utils.GetCurrentUser(ctx)

	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"user": types.UserResponse{
			ID:    currentUser.ID,
			Name:  currentUser.Name,
			Email: currentUser.Email,
		},
	})
}

func LogoutUser(ctx *gin.Context) {
	ctx.SetCookie("token", "", -1, "/", "", true, true)
	ctx.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}
