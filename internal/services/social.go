package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type SocialUser struct {
	Provider   string
	ProviderID string
	Email      string
	Name       string
	AvatarURL  string
}

type SocialProvider interface {
	Name() string
	GetAuthURL(state string) string
	ExchangeCode(code string) (*SocialUser, error)
}

type GoogleAdapter struct {
	config *oauth2.Config
}

func NewGoogleAdapter(redirectURL string) *GoogleAdapter {
	url := redirectURL
	if url == "" {
		url = os.Getenv("GOOGLE_REDIRECT_URL")
	}
	return &GoogleAdapter{
		config: &oauth2.Config{
			ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
			ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
			RedirectURL:  url,
			Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"},
			Endpoint:     google.Endpoint,
		},
	}
}

func (g *GoogleAdapter) Name() string {
	return "google"
}

func (g *GoogleAdapter) GetAuthURL(state string) string {
	return g.config.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

func (g *GoogleAdapter) ExchangeCode(code string) (*SocialUser, error) {
	token, err := g.config.Exchange(context.Background(), code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	client := g.config.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read user info: %w", err)
	}

	var googleUser struct {
		ID      string `json:"id"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}
	if err := json.Unmarshal(data, &googleUser); err != nil {
		return nil, fmt.Errorf("failed to parse user info: %w", err)
	}

	if googleUser.Email == "" {
		return nil, errors.New("email not provided by Google")
	}

	return &SocialUser{
		Provider:   "google",
		ProviderID: googleUser.ID,
		Email:      googleUser.Email,
		Name:       googleUser.Name,
		AvatarURL:  googleUser.Picture,
	}, nil
}
