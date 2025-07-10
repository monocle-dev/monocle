package auth

import (
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret string

func InitJWTSecret() error {
	jwtSecret = os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return fmt.Errorf("JWT_SECRET environment variable is not set")
	}
	return nil
}

func GenerateJWT(userID uint, email string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"exp":     time.Now().Add(time.Hour * 168).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtSecret))
}

func VerifyJWT(tokenString string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(jwtSecret), nil
	})

	if err != nil || !token.Valid {
		return nil, fmt.Errorf("Invalid or expired token")
	}

	return token, nil
}
