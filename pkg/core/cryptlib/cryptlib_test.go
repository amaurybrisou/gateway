package cryptlib_test

import (
	"testing"

	"github.com/amaurybrisou/gateway/pkg/core/cryptlib"
	"golang.org/x/crypto/bcrypt"
)

func TestGenerateHash(t *testing.T) {
	password := "password123"
	cost := bcrypt.DefaultCost
	hash, err := cryptlib.GenerateHash(password, cost)
	if err != nil {
		t.Errorf("GenerateHash returned an error: %v", err)
	}

	if hash == "" {
		t.Errorf("GenerateHash returned an empty hash")
	}

	// Check if the generated hash is a valid bcrypt hash
	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		t.Errorf("Generated hash is not valid: %v", err)
	}
}

func TestValidateHash(t *testing.T) {
	password := "password123"
	cost := bcrypt.DefaultCost

	hash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		t.Errorf("Failed to generate bcrypt hash: %v", err)
	}

	match := cryptlib.ValidateHash(password, string(hash))
	if !match {
		t.Errorf("ValidateHash failed to validate the correct password")
	}

	incorrectPassword := "incorrectPassword"
	match = cryptlib.ValidateHash(incorrectPassword, string(hash))
	if match {
		t.Errorf("ValidateHash validated an incorrect password")
	}
}

func TestGenerateRandomPassword(t *testing.T) {
	length := 10

	randomPassword, err := cryptlib.GenerateRandomPassword(length)
	if err != nil {
		t.Errorf("GenerateRandomPassword returned an error: %v", err)
	}

	if len(randomPassword) != length {
		t.Errorf("GenerateRandomPassword generated a password with incorrect length")
	}
}
