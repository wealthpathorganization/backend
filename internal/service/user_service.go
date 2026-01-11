package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"

	"github.com/wealthpath/backend/internal/model"
	"github.com/wealthpath/backend/internal/repository"
	"github.com/wealthpath/backend/pkg/currency"
)

// Service-level errors for authentication and user management.
var (
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrEmailTaken          = errors.New("email already taken")
	ErrOAuthFailed         = errors.New("OAuth authentication failed")
	ErrUnsupportedCurrency = errors.New("unsupported currency")
	ErrTOTPRequired        = errors.New("2FA verification required")
)

// UserRepositoryInterface defines the contract for user data access.
// Implementations must be safe for concurrent use.
type UserRepositoryInterface interface {
	Create(ctx context.Context, user *model.User) error
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	Update(ctx context.Context, user *model.User) error
	EmailExists(ctx context.Context, email string) (bool, error)
	UpdateLastLogin(ctx context.Context, id uuid.UUID) error
	GetOrCreateByOAuth(ctx context.Context, user *model.User) (*model.User, error)
	GetByOAuth(ctx context.Context, provider, oauthID string) (*model.User, error)
}

// UserService handles business logic for user authentication and profile management.
type UserService struct {
	repo UserRepositoryInterface
}

// NewUserService creates a new UserService with the given repository.
func NewUserService(repo UserRepositoryInterface) *UserService {
	return &UserService{repo: repo}
}

type RegisterInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
	Currency string `json:"currency"`
}

type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token        string      `json:"token,omitempty"`
	User         *model.User `json:"user,omitempty"`
	RequiresTOTP bool        `json:"requiresTOTP,omitempty"`
	TempToken    string      `json:"tempToken,omitempty"` // Temporary token for 2FA flow
}

// Register creates a new user account with email and password.
// Returns ErrEmailTaken if the email is already registered.
func (s *UserService) Register(ctx context.Context, input RegisterInput) (*AuthResponse, error) {
	exists, err := s.repo.EmailExists(ctx, input.Email)
	if err != nil {
		return nil, fmt.Errorf("checking email existence: %w", err)
	}
	if exists {
		return nil, ErrEmailTaken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	curr := input.Currency
	if curr == "" {
		curr = string(currency.DefaultCurrency)
	}
	if !currency.IsValid(curr) {
		return nil, ErrUnsupportedCurrency
	}

	hashStr := string(hash)
	user := &model.User{
		Email:        input.Email,
		PasswordHash: &hashStr,
		Name:         input.Name,
		Currency:     curr,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("creating user: %w", err)
	}

	token, err := generateToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("generating token: %w", err)
	}

	return &AuthResponse{Token: token, User: user}, nil
}

// Login authenticates a user with email and password.
// Returns ErrInvalidCredentials if the credentials are incorrect.
// If 2FA is enabled, returns RequiresTOTP=true with a TempToken instead of the full token.
func (s *UserService) Login(ctx context.Context, input LoginInput) (*AuthResponse, error) {
	user, err := s.repo.GetByEmail(ctx, input.Email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("fetching user by email: %w", err)
	}

	if user.PasswordHash == nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(input.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// Check if 2FA is enabled
	if user.TOTPEnabled {
		tempToken, err := generateTempToken(user.ID)
		if err != nil {
			return nil, fmt.Errorf("generating temp token: %w", err)
		}
		return &AuthResponse{
			RequiresTOTP: true,
			TempToken:    tempToken,
		}, nil
	}

	token, err := generateToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("generating token: %w", err)
	}

	return &AuthResponse{Token: token, User: user}, nil
}

// LoginWithTOTP completes login for users with 2FA enabled.
// Requires a valid temp token (from Login) and a valid TOTP code.
func (s *UserService) LoginWithTOTP(ctx context.Context, tempToken, code string) (*AuthResponse, error) {
	userID, err := ValidateTempToken(tempToken)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("fetching user: %w", err)
	}

	if !user.TOTPEnabled || user.TOTPSecret == nil {
		return nil, ErrInvalidCredentials
	}

	// Import totp package for validation
	// Note: This creates a dependency on the totp package
	// For cleaner architecture, consider using the TOTPService
	if err := validateTOTPCode(*user.TOTPSecret, code); err != nil {
		return nil, ErrInvalidCredentials
	}

	token, err := generateToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("generating token: %w", err)
	}

	return &AuthResponse{Token: token, User: user}, nil
}

// LoginWithBackupCode completes login using a backup code.
func (s *UserService) LoginWithBackupCode(ctx context.Context, tempToken, backupCode string) (*AuthResponse, error) {
	userID, err := ValidateTempToken(tempToken)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("fetching user: %w", err)
	}

	if !user.TOTPEnabled {
		return nil, ErrInvalidCredentials
	}

	// Find and remove the backup code
	normalizedCode := normalizeBackupCode(backupCode)
	var newCodes []string
	found := false

	for _, c := range user.TOTPBackupCodes {
		if normalizeBackupCode(c) == normalizedCode {
			found = true
			continue // Don't include this code in new list (consume it)
		}
		newCodes = append(newCodes, c)
	}

	if !found {
		return nil, ErrInvalidCredentials
	}

	// Update backup codes in the repository
	if repo, ok := s.repo.(interface {
		UpdateBackupCodes(ctx context.Context, userID uuid.UUID, codes []string) error
	}); ok {
		if err := repo.UpdateBackupCodes(ctx, userID, newCodes); err != nil {
			return nil, fmt.Errorf("updating backup codes: %w", err)
		}
	}

	token, err := generateToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("generating token: %w", err)
	}

	return &AuthResponse{Token: token, User: user}, nil
}

// GetByID retrieves a user by their ID.
func (s *UserService) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting user %s: %w", id, err)
	}
	return user, nil
}

// SupportedCurrencies returns the list of supported currency codes.
// Deprecated: Use currency.SupportedCurrencyCodes() instead.
var SupportedCurrencies = currency.SupportedCurrencyCodes()

type UpdateSettingsInput struct {
	Name     *string `json:"name"`
	Currency *string `json:"currency"`
}

// UpdateSettings updates user profile settings (name, currency).
// Returns ErrUnsupportedCurrency if the currency is not supported.
func (s *UserService) UpdateSettings(ctx context.Context, userID uuid.UUID, input UpdateSettingsInput) (*model.User, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("fetching user %s for update: %w", userID, err)
	}

	if input.Name != nil && *input.Name != "" {
		user.Name = *input.Name
	}

	if input.Currency != nil && *input.Currency != "" {
		if !currency.IsValid(*input.Currency) {
			return nil, ErrUnsupportedCurrency
		}
		user.Currency = *input.Currency
	}

	if err := s.repo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("updating user %s: %w", userID, err)
	}

	return user, nil
}

// RefreshToken generates a new JWT token for an authenticated user.
// This allows mobile apps to refresh their tokens without re-authenticating.
func (s *UserService) RefreshToken(ctx context.Context, userID uuid.UUID) (*AuthResponse, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("fetching user %s for token refresh: %w", userID, err)
	}

	token, err := generateToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("generating new token: %w", err)
	}

	return &AuthResponse{Token: token, User: user}, nil
}

// GenerateTokenForTest generates a JWT token for testing purposes.
func GenerateTokenForTest() (string, error) {
	return generateToken(uuid.New())
}

// generateToken creates a signed JWT token for the given user ID.
// Token expires in 7 days.
func generateToken(userID uuid.UUID) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "dev-secret-change-in-production"
	}

	claims := jwt.MapClaims{
		"sub": userID.String(),
		"exp": time.Now().Add(time.Hour * 24 * 7).Unix(),
		"iat": time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ValidateToken parses and validates a JWT token string.
// Returns the user ID if valid, or an error if invalid.
func ValidateToken(tokenString string) (uuid.UUID, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "dev-secret-change-in-production"
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})

	if err != nil || !token.Valid {
		return uuid.Nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return uuid.Nil, errors.New("invalid claims")
	}

	userID, err := uuid.Parse(claims["sub"].(string))
	if err != nil {
		return uuid.Nil, errors.New("invalid user id in token")
	}

	return userID, nil
}

// ============ GENERIC OAUTH ============

// OAuthLogin handles OAuth login for any provider using authorization code
func (s *UserService) OAuthLogin(ctx context.Context, providerName, code string) (*AuthResponse, error) {
	provider, ok := OAuthProviders[providerName]
	if !ok {
		return nil, ErrOAuthFailed
	}

	accessToken, err := provider.ExchangeCode(code)
	if err != nil {
		return nil, err
	}

	return s.OAuthLoginWithToken(ctx, providerName, accessToken)
}

// OAuthLoginWithToken handles OAuth login using an access token
func (s *UserService) OAuthLoginWithToken(ctx context.Context, providerName, accessToken string) (*AuthResponse, error) {
	provider, ok := OAuthProviders[providerName]
	if !ok {
		return nil, ErrOAuthFailed
	}

	oauthUser, err := provider.GetUser(accessToken)
	if err != nil {
		return nil, err
	}

	// Check if user exists with this OAuth ID
	user, err := s.repo.GetByOAuth(ctx, providerName, oauthUser.ID)
	if err == nil {
		token, err := generateToken(user.ID)
		if err != nil {
			return nil, err
		}
		return &AuthResponse{Token: token, User: user}, nil
	}

	// Check if email exists (link accounts)
	if oauthUser.Email != "" {
		existingUser, err := s.repo.GetByEmail(ctx, oauthUser.Email)
		if err == nil {
			existingUser.OAuthProvider = &providerName
			existingUser.OAuthID = &oauthUser.ID
			if oauthUser.AvatarURL != "" {
				existingUser.AvatarURL = &oauthUser.AvatarURL
			}
			if err := s.repo.Update(ctx, existingUser); err != nil {
				return nil, err
			}
			token, err := generateToken(existingUser.ID)
			if err != nil {
				return nil, err
			}
			return &AuthResponse{Token: token, User: existingUser}, nil
		}
	}

	// Create new user
	user = &model.User{
		Email:         oauthUser.Email,
		Name:          oauthUser.Name,
		Currency:      "USD",
		OAuthProvider: &providerName,
		OAuthID:       &oauthUser.ID,
	}
	if oauthUser.AvatarURL != "" {
		user.AvatarURL = &oauthUser.AvatarURL
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	token, err := generateToken(user.ID)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{Token: token, User: user}, nil
}

// Legacy methods for backward compatibility
func (s *UserService) FacebookLogin(ctx context.Context, code string) (*AuthResponse, error) {
	return s.OAuthLogin(ctx, "facebook", code)
}

func (s *UserService) FacebookLoginWithToken(ctx context.Context, accessToken string) (*AuthResponse, error) {
	return s.OAuthLoginWithToken(ctx, "facebook", accessToken)
}

func (s *UserService) GoogleLogin(ctx context.Context, code string) (*AuthResponse, error) {
	return s.OAuthLogin(ctx, "google", code)
}

func (s *UserService) GoogleLoginWithToken(ctx context.Context, accessToken string) (*AuthResponse, error) {
	return s.OAuthLoginWithToken(ctx, "google", accessToken)
}

// ============ TOTP HELPERS ============

// generateTempToken creates a short-lived token for 2FA flow.
// This token is valid for 5 minutes and can only be used to complete 2FA login.
func generateTempToken(userID uuid.UUID) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "dev-secret-change-in-production"
	}

	claims := jwt.MapClaims{
		"sub":  userID.String(),
		"exp":  time.Now().Add(time.Minute * 5).Unix(),
		"iat":  time.Now().Unix(),
		"type": "2fa_temp",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ValidateTempToken validates a temporary token from the 2FA flow.
func ValidateTempToken(tokenString string) (uuid.UUID, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "dev-secret-change-in-production"
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})

	if err != nil || !token.Valid {
		return uuid.Nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return uuid.Nil, errors.New("invalid claims")
	}

	// Verify this is a temp token
	tokenType, ok := claims["type"].(string)
	if !ok || tokenType != "2fa_temp" {
		return uuid.Nil, errors.New("not a temporary token")
	}

	userID, err := uuid.Parse(claims["sub"].(string))
	if err != nil {
		return uuid.Nil, errors.New("invalid user id in token")
	}

	return userID, nil
}

// validateTOTPCode validates a TOTP code against the secret.
func validateTOTPCode(secret, code string) error {
	if !totp.Validate(code, secret) {
		return errors.New("invalid code")
	}
	return nil
}

// normalizeBackupCode normalizes a backup code for comparison.
func normalizeBackupCode(code string) string {
	// Remove dashes and convert to uppercase
	normalized := ""
	for _, c := range code {
		if c != '-' && c != ' ' {
			normalized += string(c)
		}
	}
	return strings.ToUpper(normalized)
}
