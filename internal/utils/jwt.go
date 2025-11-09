package utils

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// GenerateJWT создает JWT токен с user_id и сроком жизни 24 часа.
func GenerateJWT(userID int, secret string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
