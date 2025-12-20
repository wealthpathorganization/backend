package handler

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/wealthpath/backend/internal/service"
)

type OAuthHandler struct {
	userService *service.UserService
}

func NewOAuthHandler(userService *service.UserService) *OAuthHandler {
	return &OAuthHandler{userService: userService}
}

// OAuthLogin initiates OAuth flow for any provider
func (h *OAuthHandler) OAuthLogin(w http.ResponseWriter, r *http.Request) {
	providerName := chi.URLParam(r, "provider")
	provider, ok := service.OAuthProviders[providerName]
	if !ok {
		respondError(w, http.StatusBadRequest, "unsupported provider")
		return
	}

	authURL := provider.AuthURL()
	if authURL == "" {
		respondError(w, http.StatusServiceUnavailable, providerName+" OAuth is not configured. Please set "+providerName+" OAuth credentials in environment variables.")
		return
	}

	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

// OAuthCallback handles callback from any OAuth provider
func (h *OAuthHandler) OAuthCallback(w http.ResponseWriter, r *http.Request) {
	providerName := chi.URLParam(r, "provider")
	code := r.URL.Query().Get("code")

	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
	}

	if code == "" {
		http.Redirect(w, r, frontendURL+"/login?error=oauth_failed", http.StatusTemporaryRedirect)
		return
	}

	resp, err := h.userService.OAuthLogin(r.Context(), providerName, code)
	if err != nil {
		http.Redirect(w, r, frontendURL+"/login?error="+url.QueryEscape(err.Error()), http.StatusTemporaryRedirect)
		return
	}

	http.Redirect(w, r, frontendURL+"/login?token="+resp.Token, http.StatusTemporaryRedirect)
}

// OAuthToken handles token-based OAuth login (for frontend SDKs)
func (h *OAuthHandler) OAuthToken(w http.ResponseWriter, r *http.Request) {
	providerName := chi.URLParam(r, "provider")

	var input struct {
		AccessToken string `json:"accessToken"`
		IDToken     string `json:"idToken"` // For Google
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	token := input.AccessToken
	if token == "" {
		token = input.IDToken
	}
	if token == "" {
		respondError(w, http.StatusBadRequest, "access token is required")
		return
	}

	resp, err := h.userService.OAuthLoginWithToken(r.Context(), providerName, token)
	if err != nil {
		respondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, resp)
}
