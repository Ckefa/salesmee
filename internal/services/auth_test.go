package services

import (
	"testing"
	"time"

	"salesmee/internal/config"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func TestGenerateToken(t *testing.T) {
	original := config.C.JWTSecret
	config.C.JWTSecret = "test-secret-for-testing"
	defer func() { config.C.JWTSecret = original }()

	token, err := GenerateToken(42, "test@example.com")
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestValidateToken(t *testing.T) {
	original := config.C.JWTSecret
	config.C.JWTSecret = "test-secret-for-testing"
	defer func() { config.C.JWTSecret = original }()

	tokenStr, err := GenerateToken(42, "test@example.com")
	assert.NoError(t, err)

	validated, err := ValidateToken(tokenStr)
	assert.NoError(t, err)
	assert.NotNil(t, validated)
	assert.Equal(t, uint(42), validated.UserID)
	assert.Equal(t, "test@example.com", validated.Email)
}

func TestValidateTokenInvalid(t *testing.T) {
	original := config.C.JWTSecret
	config.C.JWTSecret = "test-secret-for-testing"
	defer func() { config.C.JWTSecret = original }()

	_, err := ValidateToken("invalid-token-string")
	assert.Error(t, err)
}

func TestValidateTokenExpired(t *testing.T) {
	original := config.C.JWTSecret
	config.C.JWTSecret = "test-secret-for-testing"
	defer func() { config.C.JWTSecret = original }()

	claims := &Claims{
		UserID: 42,
		Email:  "test@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(config.C.JWTSecret))
	assert.NoError(t, err)

	_, err = ValidateToken(tokenStr)
	assert.Error(t, err)
}

func TestHash(t *testing.T) {
	hash := Hash("password123")
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, "password123", hash)
}

func TestCheck(t *testing.T) {
	hash := Hash("password123")
	assert.True(t, Check(hash, "password123"))
	assert.False(t, Check(hash, "wrongpassword"))
}
