package utils

import (
	"log"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword - hash password using bcrypt algorithm.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Failed to hash password: %v", err)
		return "", err
	}
	return string(hash), nil
}

// CheckPassword - compare hashed password with plain password.
func CheckPassword(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
