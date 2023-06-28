package cryptlib

import (
	"crypto/rand"
	"encoding/base64"

	"golang.org/x/crypto/bcrypt"
)

// GenerateHash generates a bcrypt hash of the given password with the specified cost factor.
func GenerateHash(password string, cost int) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

// ValidateHash compares a plain-text password with a bcrypt hashed password and returns true if they match.
func ValidateHash(password, hashedPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

// GenerateRandomPassword generates a random password with the specified length.
func GenerateRandomPassword(length int) (string, error) {
	randomBytes := make([]byte, length)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}
	randomPassword := base64.URLEncoding.EncodeToString(randomBytes)
	return randomPassword[:length], nil
}
