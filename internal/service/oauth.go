package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

// OAuthProvider defines the interface for OAuth authentication providers.
// Implementations handle provider-specific OAuth flows.
type OAuthProvider interface {
	Name() string
	AuthURL() string
	ExchangeCode(code string) (string, error)
	GetUser(accessToken string) (*OAuthUser, error)
}

// OAuthUser represents normalized user data from any OAuth provider.
type OAuthUser struct {
	ID        string
	Email     string
	Name      string
	AvatarURL string
}

// ============ FACEBOOK ============

// FacebookProvider implements OAuth authentication with Facebook.
type FacebookProvider struct{}

// Name returns the provider identifier.
func (p *FacebookProvider) Name() string { return "facebook" }

// AuthURL returns the Facebook OAuth authorization URL.
func (p *FacebookProvider) AuthURL() string {
	clientID := os.Getenv("FACEBOOK_APP_ID")
	redirectURI := os.Getenv("FACEBOOK_REDIRECT_URI")

	if clientID == "" || redirectURI == "" {
		return ""
	}

	return "https://www.facebook.com/v18.0/dialog/oauth?" +
		"client_id=" + clientID +
		"&redirect_uri=" + url.QueryEscape(redirectURI) +
		"&scope=email,public_profile" +
		"&response_type=code"
}

// ExchangeCode exchanges an authorization code for an access token.
func (p *FacebookProvider) ExchangeCode(code string) (string, error) {
	clientID := os.Getenv("FACEBOOK_APP_ID")
	clientSecret := os.Getenv("FACEBOOK_APP_SECRET")
	redirectURI := os.Getenv("FACEBOOK_REDIRECT_URI")

	tokenURL := "https://graph.facebook.com/v18.0/oauth/access_token?" +
		"client_id=" + clientID +
		"&redirect_uri=" + url.QueryEscape(redirectURI) +
		"&client_secret=" + clientSecret +
		"&code=" + code

	resp, err := http.Get(tokenURL)
	if err != nil {
		return "", fmt.Errorf("facebook token exchange: %w", ErrOAuthFailed)
	}
	defer func() { _ = resp.Body.Close() }()

	var tokenResp struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil || tokenResp.AccessToken == "" {
		return "", fmt.Errorf("parsing facebook token response: %w", ErrOAuthFailed)
	}
	return tokenResp.AccessToken, nil
}

// GetUser retrieves user information from Facebook using an access token.
func (p *FacebookProvider) GetUser(accessToken string) (*OAuthUser, error) {
	userURL := "https://graph.facebook.com/me?fields=id,name,email,picture.type(large)&access_token=" + accessToken

	resp, err := http.Get(userURL)
	if err != nil {
		return nil, fmt.Errorf("fetching facebook user: %w", ErrOAuthFailed)
	}
	defer func() { _ = resp.Body.Close() }()

	var fbUser struct {
		ID      string `json:"id"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture struct {
			Data struct {
				URL string `json:"url"`
			} `json:"data"`
		} `json:"picture"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&fbUser); err != nil || fbUser.ID == "" {
		return nil, fmt.Errorf("parsing facebook user response: %w", ErrOAuthFailed)
	}

	return &OAuthUser{
		ID:        fbUser.ID,
		Email:     fbUser.Email,
		Name:      fbUser.Name,
		AvatarURL: fbUser.Picture.Data.URL,
	}, nil
}

// ============ GOOGLE ============

// GoogleProvider implements OAuth authentication with Google.
type GoogleProvider struct{}

// Name returns the provider identifier.
func (p *GoogleProvider) Name() string { return "google" }

// AuthURL returns the Google OAuth authorization URL.
func (p *GoogleProvider) AuthURL() string {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	redirectURI := os.Getenv("GOOGLE_REDIRECT_URI")

	if clientID == "" || redirectURI == "" {
		return ""
	}

	return "https://accounts.google.com/o/oauth2/v2/auth?" +
		"client_id=" + clientID +
		"&redirect_uri=" + url.QueryEscape(redirectURI) +
		"&scope=" + url.QueryEscape("openid email profile") +
		"&response_type=code" +
		"&access_type=offline"
}

// ExchangeCode exchanges an authorization code for an access token.
func (p *GoogleProvider) ExchangeCode(code string) (string, error) {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	redirectURI := os.Getenv("GOOGLE_REDIRECT_URI")

	data := url.Values{}
	data.Set("code", code)
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("redirect_uri", redirectURI)
	data.Set("grant_type", "authorization_code")

	resp, err := http.PostForm("https://oauth2.googleapis.com/token", data)
	if err != nil {
		return "", fmt.Errorf("google token exchange: %w", ErrOAuthFailed)
	}
	defer func() { _ = resp.Body.Close() }()

	var tokenResp struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil || tokenResp.AccessToken == "" {
		return "", fmt.Errorf("parsing google token response: %w", ErrOAuthFailed)
	}
	return tokenResp.AccessToken, nil
}

// GetUser retrieves user information from Google using an access token.
func (p *GoogleProvider) GetUser(accessToken string) (*OAuthUser, error) {
	req, err := http.NewRequest("GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, fmt.Errorf("creating google user request: %w", ErrOAuthFailed)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching google user: %w", ErrOAuthFailed)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading google user response: %w", ErrOAuthFailed)
	}

	var gUser struct {
		ID      string `json:"id"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}

	if err := json.Unmarshal(body, &gUser); err != nil || gUser.ID == "" {
		return nil, fmt.Errorf("parsing google user response: %w", ErrOAuthFailed)
	}

	return &OAuthUser{
		ID:        gUser.ID,
		Email:     gUser.Email,
		Name:      gUser.Name,
		AvatarURL: gUser.Picture,
	}, nil
}

// OAuthProviders is the registry of available OAuth providers.
var OAuthProviders = map[string]OAuthProvider{
	"facebook": &FacebookProvider{},
	"google":   &GoogleProvider{},
}
