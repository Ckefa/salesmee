package ws

import (
	"errors"

	"salesmee/internal/config"
	"salesmee/internal/services"
)

type AuthResult struct {
	UserID     uint
	Email      string
	UserType   string
	BusinessID uint
}

func Authenticate(tokenString string) (*AuthResult, error) {
	claims, err := services.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	result := &AuthResult{
		UserID:   claims.UserID,
		Email:    claims.Email,
		UserType: "business",
	}

	if claims.Subject == "client" {
		result.UserType = "client"
		result.BusinessID = 0
		return result, nil
	}

	if config.C.BizID != "" {
		result.BusinessID = claims.UserID
	} else {
		result.BusinessID = claims.UserID
	}

	return result, nil
}

var (
	ErrNotAllowed = errors.New("not allowed")
	ErrNoToken    = errors.New("no token provided")
)
