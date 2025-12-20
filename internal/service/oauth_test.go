package service

import (
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// MockHTTPClient for testing OAuth providers
type MockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}

func (m *MockHTTPClient) Get(url string) (*http.Response, error) {
	req, _ := http.NewRequest("GET", url, nil)
	return m.DoFunc(req)
}

func (m *MockHTTPClient) PostForm(url string, data map[string][]string) (*http.Response, error) {
	req, _ := http.NewRequest("POST", url, nil)
	return m.DoFunc(req)
}

// ================== Facebook Provider Tests ==================

func TestFacebookProvider_Name(t *testing.T) {
	t.Parallel()
	provider := &FacebookProvider{}
	assert.Equal(t, "facebook", provider.Name())
}

func TestFacebookProvider_AuthURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		envSetup  func()
		envClean  func()
		wantEmpty bool
	}{
		{
			name: "with valid env vars",
			envSetup: func() {
				_ = os.Setenv("FACEBOOK_APP_ID", "test-app-id")
				_ = os.Setenv("FACEBOOK_REDIRECT_URI", "http://localhost/callback")
			},
			envClean: func() {
				_ = os.Unsetenv("FACEBOOK_APP_ID")
				_ = os.Unsetenv("FACEBOOK_REDIRECT_URI")
			},
			wantEmpty: false,
		},
		{
			name: "missing app id",
			envSetup: func() {
				_ = os.Unsetenv("FACEBOOK_APP_ID")
				_ = os.Setenv("FACEBOOK_REDIRECT_URI", "http://localhost/callback")
			},
			envClean: func() {
				_ = os.Unsetenv("FACEBOOK_REDIRECT_URI")
			},
			wantEmpty: true,
		},
		{
			name: "missing redirect uri",
			envSetup: func() {
				_ = os.Setenv("FACEBOOK_APP_ID", "test-app-id")
				_ = os.Unsetenv("FACEBOOK_REDIRECT_URI")
			},
			envClean: func() {
				_ = os.Unsetenv("FACEBOOK_APP_ID")
			},
			wantEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.envSetup()
			defer tt.envClean()

			provider := &FacebookProvider{}
			url := provider.AuthURL()

			if tt.wantEmpty {
				assert.Empty(t, url)
			} else {
				assert.NotEmpty(t, url)
				assert.Contains(t, url, "facebook.com")
				assert.Contains(t, url, "test-app-id")
			}
		})
	}
}

// ================== Google Provider Tests ==================

func TestGoogleProvider_Name(t *testing.T) {
	t.Parallel()
	provider := &GoogleProvider{}
	assert.Equal(t, "google", provider.Name())
}

func TestGoogleProvider_AuthURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		envSetup  func()
		envClean  func()
		wantEmpty bool
	}{
		{
			name: "with valid env vars",
			envSetup: func() {
				_ = os.Setenv("GOOGLE_CLIENT_ID", "test-client-id")
				_ = os.Setenv("GOOGLE_REDIRECT_URI", "http://localhost/callback")
			},
			envClean: func() {
				_ = os.Unsetenv("GOOGLE_CLIENT_ID")
				_ = os.Unsetenv("GOOGLE_REDIRECT_URI")
			},
			wantEmpty: false,
		},
		{
			name: "missing client id",
			envSetup: func() {
				_ = os.Unsetenv("GOOGLE_CLIENT_ID")
				_ = os.Setenv("GOOGLE_REDIRECT_URI", "http://localhost/callback")
			},
			envClean: func() {
				_ = os.Unsetenv("GOOGLE_REDIRECT_URI")
			},
			wantEmpty: true,
		},
		{
			name: "missing redirect uri",
			envSetup: func() {
				_ = os.Setenv("GOOGLE_CLIENT_ID", "test-client-id")
				_ = os.Unsetenv("GOOGLE_REDIRECT_URI")
			},
			envClean: func() {
				_ = os.Unsetenv("GOOGLE_CLIENT_ID")
			},
			wantEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.envSetup()
			defer tt.envClean()

			provider := &GoogleProvider{}
			url := provider.AuthURL()

			if tt.wantEmpty {
				assert.Empty(t, url)
			} else {
				assert.NotEmpty(t, url)
				assert.Contains(t, url, "accounts.google.com")
				assert.Contains(t, url, "test-client-id")
			}
		})
	}
}

// ================== OAuthProviders Registry Tests ==================

func TestOAuthProviders_Registry(t *testing.T) {
	t.Parallel()

	assert.NotNil(t, OAuthProviders)
	assert.NotNil(t, OAuthProviders["facebook"])
	assert.NotNil(t, OAuthProviders["google"])

	assert.Equal(t, "facebook", OAuthProviders["facebook"].Name())
	assert.Equal(t, "google", OAuthProviders["google"].Name())
}

// ================== OAuthUser Tests ==================

func TestOAuthUser_Struct(t *testing.T) {
	t.Parallel()

	user := &OAuthUser{
		ID:        "123",
		Email:     "test@example.com",
		Name:      "Test User",
		AvatarURL: "https://example.com/avatar.jpg",
	}

	assert.Equal(t, "123", user.ID)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, "Test User", user.Name)
	assert.Equal(t, "https://example.com/avatar.jpg", user.AvatarURL)
}
