package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

type FacebookAdapter struct {
	config *oauth2.Config
}

func NewFacebookAdapter(redirectURL string) *FacebookAdapter {
	return &FacebookAdapter{
		config: &oauth2.Config{
			ClientID:     os.Getenv("FB_APP_ID"),
			ClientSecret: os.Getenv("FB_SECRET"),
			RedirectURL:  redirectURL,
			Scopes:       []string{"email", "public_profile"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://www.facebook.com/v12.0/dialog/oauth",
				TokenURL: "https://graph.facebook.com/v12.0/oauth/access_token",
			},
		},
	}
}

func (f *FacebookAdapter) Name() string {
	return "facebook"
}

func (f *FacebookAdapter) GetAuthURL(state string) string {
	return f.config.AuthCodeURL(state)
}

func (f *FacebookAdapter) ExchangeCode(code string) (*SocialUser, error) {
	resp, err := http.PostForm(f.config.Endpoint.TokenURL, url.Values{
		"client_id":     {f.config.ClientID},
		"client_secret": {f.config.ClientSecret},
		"redirect_uri":  {f.config.RedirectURL},
		"code":          {code},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to exchange facebook code: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read facebook token response: %w", err)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(data, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse facebook token response: %w", err)
	}
	if tokenResp.AccessToken == "" {
		return nil, errors.New("empty access token from Facebook")
	}

	userResp, err := http.Get("https://graph.facebook.com/me?fields=id,name,email,picture&access_token=" + tokenResp.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get facebook user info: %w", err)
	}
	defer userResp.Body.Close()

	userData, err := io.ReadAll(userResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read facebook user info: %w", err)
	}

	var fbUser struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Email   string `json:"email"`
		Picture struct {
			Data struct {
				URL string `json:"url"`
			} `json:"data"`
		} `json:"picture"`
	}
	if err := json.Unmarshal(userData, &fbUser); err != nil {
		return nil, fmt.Errorf("failed to parse facebook user info: %w", err)
	}

	return &SocialUser{
		Provider:   "facebook",
		ProviderID: fbUser.ID,
		Email:      fbUser.Email,
		Name:       fbUser.Name,
		AvatarURL:  fbUser.Picture.Data.URL,
	}, nil
}
