package handlers

import (
	"errors"
	"log"
	"net/http"
	"os"
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

type UpdateUserRequest struct {
	Name            string `json:"name"`
	Email           string `json:"email" binding:"omitempty,email"`
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password" binding:"omitempty,min=8"`
}

var (
	Domain = os.Getenv("DOMAIN")
)

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

	http.SetCookie(ctx.Writer, &http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		Domain:   Domain,
		MaxAge:   60 * 60 * 24 * 7,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteNoneMode,
	})

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

	http.SetCookie(ctx.Writer, &http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		Domain:   Domain,
		MaxAge:   60 * 60 * 24 * 7,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteNoneMode,
	})

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
	http.SetCookie(ctx.Writer, &http.Cookie{
		Name:     "token",
		Value:    "",
		Path:     "/",
		Domain:   Domain,
		MaxAge:   -1,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteNoneMode,
	})

	ctx.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

func UpdateUser(ctx *gin.Context) {
	currentUser, err := utils.GetCurrentUser(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Fetch the full user record from database to access password hash
	var dbUser models.User
	if err := db.DB.First(&dbUser, currentUser.ID).Error; err != nil {
		log.Printf("Failed to fetch user: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	var updateReq UpdateUserRequest
	if err := ctx.BindJSON(&updateReq); err != nil {
		log.Printf("Failed to bind JSON: %v", err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Start building the update data
	updates := make(map[string]interface{})

	// Update name if provided
	if updateReq.Name != "" {
		updates["name"] = strings.TrimSpace(updateReq.Name)
	}

	// Update email if provided
	if updateReq.Email != "" {
		newEmail := strings.ToLower(strings.TrimSpace(updateReq.Email))

		// Check if email is already taken by another user
		if newEmail != dbUser.Email {
			var existingUser models.User
			err := db.DB.Where("email = ? AND id != ?", newEmail, dbUser.ID).First(&existingUser).Error
			if err == nil {
				ctx.JSON(http.StatusBadRequest, gin.H{"error": "Email already exists"})
				return
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				log.Printf("Database error when checking existing email: %v", err)
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
				return
			}
		}

		updates["email"] = newEmail
	}

	// Update password if provided
	if updateReq.NewPassword != "" {
		// Verify current password if changing password
		if updateReq.CurrentPassword == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "Current password is required to change password"})
			return
		}

		err = bcrypt.CompareHashAndPassword([]byte(dbUser.PasswordHash), []byte(updateReq.CurrentPassword))
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "Current password is incorrect"})
			return
		}

		// Hash new password
		passwordHash, err := bcrypt.GenerateFromPassword([]byte(updateReq.NewPassword), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("Failed to hash new password: %v", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}

		updates["password_hash"] = string(passwordHash)
	}

	// If no updates provided
	if len(updates) == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "No valid fields to update"})
		return
	}

	// Perform the update
	if err := db.DB.Model(&dbUser).Updates(updates).Error; err != nil {
		log.Printf("Failed to update user: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// Refresh user data from database
	if err := db.DB.First(&dbUser, dbUser.ID).Error; err != nil {
		log.Printf("Failed to refresh user data: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "User updated successfully",
		"user": types.UserResponse{
			ID:    dbUser.ID,
			Name:  dbUser.Name,
			Email: dbUser.Email,
		},
	})
}

func DeleteUser(ctx *gin.Context) {
	currentUser, err := utils.GetCurrentUser(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Fetch the full user record from database
	var dbUser models.User
	if err := db.DB.First(&dbUser, currentUser.ID).Error; err != nil {
		log.Printf("Failed to fetch user: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// Validate password for account deletion
	var deleteReq struct {
		Password string `json:"password" binding:"required"`
	}

	if err := ctx.BindJSON(&deleteReq); err != nil {
		log.Printf("Failed to bind JSON: %v", err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Password is required for account deletion"})
		return
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(dbUser.PasswordHash), []byte(deleteReq.Password))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Incorrect password"})
		return
	}

	// Delete user account (this will cascade delete related records due to foreign key constraints)
	if err := db.DB.Delete(&dbUser).Error; err != nil {
		log.Printf("Failed to delete user: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// Clear the authentication cookie
	http.SetCookie(ctx.Writer, &http.Cookie{
		Name:     "token",
		Value:    "",
		Path:     "/",
		Domain:   Domain,
		MaxAge:   -1,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteNoneMode,
	})

	ctx.JSON(http.StatusOK, gin.H{"message": "Account deleted successfully"})
}
