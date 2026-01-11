package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
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

// Token expiration constants
const (
	AccessTokenExpiry        = 15 * time.Minute      // Short-lived access token
	RefreshTokenExpiry       = 7 * 24 * time.Hour    // Default refresh token (7 days)
	RememberMeRefreshExpiry  = 30 * 24 * time.Hour   // Extended refresh token (30 days)
	RefreshTokenBytes        = 32                    // 256 bits of randomness
)

// Service-level errors for authentication and user management.
var (
	ErrInvalidCredentials    = errors.New("invalid credentials")
	ErrEmailTaken            = errors.New("email already taken")
	ErrOAuthFailed           = errors.New("OAuth authentication failed")
	ErrUnsupportedCurrency   = errors.New("unsupported currency")
	ErrTOTPRequired          = errors.New("2FA verification required")
	ErrRefreshTokenInvalid   = errors.New("refresh token invalid")
	ErrRefreshTokenExpired   = errors.New("refresh token expired")
	ErrRefreshTokenRevoked   = errors.New("refresh token revoked")
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

// RefreshTokenRepositoryInterface defines the contract for refresh token data access.
type RefreshTokenRepositoryInterface interface {
	Create(ctx context.Context, token *model.RefreshToken) error
	FindByTokenHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error)
	FindByID(ctx context.Context, id uuid.UUID) (*model.RefreshToken, error)
	FindActiveByUserID(ctx context.Context, userID uuid.UUID) ([]*model.RefreshToken, error)
	UpdateLastUsed(ctx context.Context, id uuid.UUID) error
	RevokeByID(ctx context.Context, id uuid.UUID, reason string) error
	RevokeByTokenHash(ctx context.Context, tokenHash, reason string) error
	RevokeByUserID(ctx context.Context, userID uuid.UUID, reason string) (int64, error)
	RevokeByUserIDExcept(ctx context.Context, userID uuid.UUID, exceptID uuid.UUID, reason string) (int64, error)
	DeleteExpired(ctx context.Context) (int64, error)
}

// UserService handles business logic for user authentication and profile management.
type UserService struct {
	repo         UserRepositoryInterface
	refreshRepo  RefreshTokenRepositoryInterface
}

// NewUserService creates a new UserService with the given repository.
func NewUserService(repo UserRepositoryInterface) *UserService {
	return &UserService{repo: repo}
}

// NewUserServiceWithRefreshTokens creates a UserService with refresh token support.
func NewUserServiceWithRefreshTokens(repo UserRepositoryInterface, refreshRepo RefreshTokenRepositoryInterface) *UserService {
	return &UserService{repo: repo, refreshRepo: refreshRepo}
}

// SetRefreshTokenRepo sets the refresh token repository (for backward compatibility).
func (s *UserService) SetRefreshTokenRepo(repo RefreshTokenRepositoryInterface) {
	s.refreshRepo = repo
}

type RegisterInput struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	Name       string `json:"name"`
	Currency   string `json:"currency"`
	RememberMe bool   `json:"rememberMe"`
}

type LoginInput struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	RememberMe bool   `json:"rememberMe"`
}

// LoginTOTPInput is used for completing 2FA login
type LoginTOTPInput struct {
	TempToken  string `json:"tempToken"`
	Code       string `json:"code"`
	RememberMe bool   `json:"rememberMe"`
}

// LoginBackupCodeInput is used for login with backup code
type LoginBackupCodeInput struct {
	TempToken  string `json:"tempToken"`
	BackupCode string `json:"backupCode"`
	RememberMe bool   `json:"rememberMe"`
}

type AuthResponse struct {
	Token        string      `json:"token,omitempty"`         // Access token (short-lived)
	RefreshToken string      `json:"refreshToken,omitempty"`  // Refresh token (for cookie, not sent in body usually)
	User         *model.User `json:"user,omitempty"`
	RequiresTOTP bool        `json:"requiresTOTP,omitempty"`
	TempToken    string      `json:"tempToken,omitempty"`     // Temporary token for 2FA flow
	ExpiresIn    int64       `json:"expiresIn,omitempty"`     // Access token expiry in seconds
}

// AuthContext holds information about the auth request context
type AuthContext struct {
	RememberMe bool
	DeviceInfo *model.DeviceInfo
}

// Register creates a new user account with email and password.
// Returns ErrEmailTaken if the email is already registered.
func (s *UserService) Register(ctx context.Context, input RegisterInput) (*AuthResponse, error) {
	return s.RegisterWithDeviceInfo(ctx, input, nil)
}

// RegisterWithDeviceInfo creates a new user account with device info for session tracking.
func (s *UserService) RegisterWithDeviceInfo(ctx context.Context, input RegisterInput, deviceInfo *model.DeviceInfo) (*AuthResponse, error) {
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

	// Generate access token (short-lived)
	accessToken, err := s.generateAccessToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("generating access token: %w", err)
	}

	// If refresh token repository is available, create refresh token
	var refreshToken string
	if s.refreshRepo != nil {
		refreshToken, err = s.createAndStoreRefreshToken(ctx, user.ID, input.RememberMe, deviceInfo)
		if err != nil {
			return nil, fmt.Errorf("creating refresh token: %w", err)
		}
	}

	return &AuthResponse{
		Token:        accessToken,
		RefreshToken: refreshToken,
		User:         user,
		ExpiresIn:    int64(AccessTokenExpiry.Seconds()),
	}, nil
}

// Login authenticates a user with email and password.
// Returns ErrInvalidCredentials if the credentials are incorrect.
// If 2FA is enabled, returns RequiresTOTP=true with a TempToken instead of the full token.
func (s *UserService) Login(ctx context.Context, input LoginInput) (*AuthResponse, error) {
	return s.LoginWithDeviceInfo(ctx, input, nil)
}

// LoginWithDeviceInfo authenticates a user with device info for session tracking.
func (s *UserService) LoginWithDeviceInfo(ctx context.Context, input LoginInput, deviceInfo *model.DeviceInfo) (*AuthResponse, error) {
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
		// Generate temp token that includes rememberMe preference for 2FA flow
		tempToken, err := generateTempTokenWithRememberMe(user.ID, input.RememberMe)
		if err != nil {
			return nil, fmt.Errorf("generating temp token: %w", err)
		}
		return &AuthResponse{
			RequiresTOTP: true,
			TempToken:    tempToken,
		}, nil
	}

	// Generate access token (short-lived)
	accessToken, err := s.generateAccessToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("generating access token: %w", err)
	}

	// If refresh token repository is available, create refresh token
	var refreshToken string
	if s.refreshRepo != nil {
		refreshToken, err = s.createAndStoreRefreshToken(ctx, user.ID, input.RememberMe, deviceInfo)
		if err != nil {
			return nil, fmt.Errorf("creating refresh token: %w", err)
		}
	}

	return &AuthResponse{
		Token:        accessToken,
		RefreshToken: refreshToken,
		User:         user,
		ExpiresIn:    int64(AccessTokenExpiry.Seconds()),
	}, nil
}

// LoginWithTOTP completes login for users with 2FA enabled.
// Requires a valid temp token (from Login) and a valid TOTP code.
func (s *UserService) LoginWithTOTP(ctx context.Context, tempToken, code string) (*AuthResponse, error) {
	return s.LoginWithTOTPAndDeviceInfo(ctx, tempToken, code, nil)
}

// LoginWithTOTPAndDeviceInfo completes login with 2FA and device info for session tracking.
func (s *UserService) LoginWithTOTPAndDeviceInfo(ctx context.Context, tempToken, code string, deviceInfo *model.DeviceInfo) (*AuthResponse, error) {
	userID, rememberMe, err := ValidateTempTokenWithRememberMe(tempToken)
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

	// Generate access token (short-lived)
	accessToken, err := s.generateAccessToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("generating access token: %w", err)
	}

	// If refresh token repository is available, create refresh token
	var refreshToken string
	if s.refreshRepo != nil {
		refreshToken, err = s.createAndStoreRefreshToken(ctx, user.ID, rememberMe, deviceInfo)
		if err != nil {
			return nil, fmt.Errorf("creating refresh token: %w", err)
		}
	}

	return &AuthResponse{
		Token:        accessToken,
		RefreshToken: refreshToken,
		User:         user,
		ExpiresIn:    int64(AccessTokenExpiry.Seconds()),
	}, nil
}

// LoginWithBackupCode completes login using a backup code.
func (s *UserService) LoginWithBackupCode(ctx context.Context, tempToken, backupCode string) (*AuthResponse, error) {
	return s.LoginWithBackupCodeAndDeviceInfo(ctx, tempToken, backupCode, nil)
}

// LoginWithBackupCodeAndDeviceInfo completes login with backup code and device info for session tracking.
func (s *UserService) LoginWithBackupCodeAndDeviceInfo(ctx context.Context, tempToken, backupCode string, deviceInfo *model.DeviceInfo) (*AuthResponse, error) {
	userID, rememberMe, err := ValidateTempTokenWithRememberMe(tempToken)
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

	// Generate access token (short-lived)
	accessToken, err := s.generateAccessToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("generating access token: %w", err)
	}

	// If refresh token repository is available, create refresh token
	var refreshToken string
	if s.refreshRepo != nil {
		refreshToken, err = s.createAndStoreRefreshToken(ctx, user.ID, rememberMe, deviceInfo)
		if err != nil {
			return nil, fmt.Errorf("creating refresh token: %w", err)
		}
	}

	return &AuthResponse{
		Token:        accessToken,
		RefreshToken: refreshToken,
		User:         user,
		ExpiresIn:    int64(AccessTokenExpiry.Seconds()),
	}, nil
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

// ============ REFRESH TOKEN METHODS ============

// generateAccessToken creates a short-lived JWT access token for the given user ID.
// Access tokens expire in 15 minutes.
func (s *UserService) generateAccessToken(userID uuid.UUID) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "dev-secret-change-in-production"
	}

	claims := jwt.MapClaims{
		"sub":  userID.String(),
		"exp":  time.Now().Add(AccessTokenExpiry).Unix(),
		"iat":  time.Now().Unix(),
		"type": "access",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// generateRefreshTokenString generates a cryptographically secure random refresh token.
// Returns the raw token string (32 bytes = 256 bits of randomness).
func generateRefreshTokenString() (string, error) {
	bytes := make([]byte, RefreshTokenBytes)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generating random bytes: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// hashRefreshToken hashes a refresh token using SHA-256.
// The hash is stored in the database, never the raw token.
func hashRefreshToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// createAndStoreRefreshToken creates a new refresh token and stores it in the database.
// Returns the raw token string (to be sent in cookie) or an error.
func (s *UserService) createAndStoreRefreshToken(ctx context.Context, userID uuid.UUID, rememberMe bool, deviceInfo *model.DeviceInfo) (string, error) {
	if s.refreshRepo == nil {
		return "", errors.New("refresh token repository not configured")
	}

	rawToken, err := generateRefreshTokenString()
	if err != nil {
		return "", err
	}

	tokenHash := hashRefreshToken(rawToken)

	// Determine expiry based on rememberMe
	expiry := RefreshTokenExpiry
	if rememberMe {
		expiry = RememberMeRefreshExpiry
	}

	refreshToken := &model.RefreshToken{
		UserID:     userID,
		TokenHash:  tokenHash,
		DeviceInfo: deviceInfo,
		ExpiresAt:  time.Now().Add(expiry),
	}

	if err := s.refreshRepo.Create(ctx, refreshToken); err != nil {
		return "", fmt.Errorf("storing refresh token: %w", err)
	}

	return rawToken, nil
}

// RefreshAccessToken validates a refresh token and issues a new access token.
// Implements token rotation: the old refresh token is revoked and a new one is issued.
func (s *UserService) RefreshAccessToken(ctx context.Context, refreshTokenString string, deviceInfo *model.DeviceInfo) (*AuthResponse, error) {
	if s.refreshRepo == nil {
		return nil, errors.New("refresh token repository not configured")
	}

	tokenHash := hashRefreshToken(refreshTokenString)

	// Find the refresh token
	storedToken, err := s.refreshRepo.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, ErrRefreshTokenInvalid
	}

	// Check if token is revoked
	if storedToken.IsRevoked() {
		// Potential token reuse attack - revoke all user tokens
		_, _ = s.refreshRepo.RevokeByUserID(ctx, storedToken.UserID, "security_token_reuse")
		return nil, ErrRefreshTokenRevoked
	}

	// Check if token is expired
	if storedToken.IsExpired() {
		return nil, ErrRefreshTokenExpired
	}

	// Get user
	user, err := s.repo.GetByID(ctx, storedToken.UserID)
	if err != nil {
		return nil, fmt.Errorf("fetching user: %w", err)
	}

	// Calculate if this was a "remember me" token based on original expiry
	// If original expiry was > 14 days from creation, it was remember me
	originalDuration := storedToken.ExpiresAt.Sub(storedToken.CreatedAt)
	rememberMe := originalDuration > 14*24*time.Hour

	// Revoke the old refresh token (token rotation)
	if err := s.refreshRepo.RevokeByID(ctx, storedToken.ID, "rotated"); err != nil {
		return nil, fmt.Errorf("revoking old refresh token: %w", err)
	}

	// Generate new access token
	accessToken, err := s.generateAccessToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("generating access token: %w", err)
	}

	// Create new refresh token (rotation)
	newRefreshToken, err := s.createAndStoreRefreshToken(ctx, user.ID, rememberMe, deviceInfo)
	if err != nil {
		return nil, fmt.Errorf("creating new refresh token: %w", err)
	}

	return &AuthResponse{
		Token:        accessToken,
		RefreshToken: newRefreshToken,
		User:         user,
		ExpiresIn:    int64(AccessTokenExpiry.Seconds()),
	}, nil
}

// RevokeRefreshToken revokes a specific refresh token by its hash.
func (s *UserService) RevokeRefreshToken(ctx context.Context, tokenHash, reason string) error {
	if s.refreshRepo == nil {
		return errors.New("refresh token repository not configured")
	}
	return s.refreshRepo.RevokeByTokenHash(ctx, tokenHash, reason)
}

// RevokeRefreshTokenByString revokes a refresh token given the raw token string.
func (s *UserService) RevokeRefreshTokenByString(ctx context.Context, refreshTokenString, reason string) error {
	tokenHash := hashRefreshToken(refreshTokenString)
	return s.RevokeRefreshToken(ctx, tokenHash, reason)
}

// RevokeAllUserTokens revokes all refresh tokens for a user.
func (s *UserService) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID, reason string) (int64, error) {
	if s.refreshRepo == nil {
		return 0, errors.New("refresh token repository not configured")
	}
	return s.refreshRepo.RevokeByUserID(ctx, userID, reason)
}

// GetActiveSessions returns all active sessions for a user.
func (s *UserService) GetActiveSessions(ctx context.Context, userID uuid.UUID) ([]*model.Session, error) {
	if s.refreshRepo == nil {
		return nil, errors.New("refresh token repository not configured")
	}

	tokens, err := s.refreshRepo.FindActiveByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("fetching active sessions: %w", err)
	}

	sessions := make([]*model.Session, len(tokens))
	for i, token := range tokens {
		sessions[i] = &model.Session{
			ID:         token.ID,
			DeviceInfo: token.DeviceInfo,
			CreatedAt:  token.CreatedAt,
			LastUsedAt: token.LastUsedAt,
			IsCurrent:  false, // Handler will set this based on current session
		}
	}

	return sessions, nil
}

// RevokeSession revokes a specific session by its ID.
func (s *UserService) RevokeSession(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID, reason string) error {
	if s.refreshRepo == nil {
		return errors.New("refresh token repository not configured")
	}

	// First verify the session belongs to this user
	token, err := s.refreshRepo.FindByID(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("session not found: %w", err)
	}

	if token.UserID != userID {
		return errors.New("session does not belong to user")
	}

	return s.refreshRepo.RevokeByID(ctx, sessionID, reason)
}

// RevokeOtherSessions revokes all sessions except the current one.
func (s *UserService) RevokeOtherSessions(ctx context.Context, userID uuid.UUID, currentSessionID uuid.UUID, reason string) (int64, error) {
	if s.refreshRepo == nil {
		return 0, errors.New("refresh token repository not configured")
	}
	return s.refreshRepo.RevokeByUserIDExcept(ctx, userID, currentSessionID, reason)
}

// GetSessionIDFromRefreshToken returns the session ID for a given refresh token.
func (s *UserService) GetSessionIDFromRefreshToken(ctx context.Context, refreshTokenString string) (uuid.UUID, error) {
	if s.refreshRepo == nil {
		return uuid.Nil, errors.New("refresh token repository not configured")
	}

	tokenHash := hashRefreshToken(refreshTokenString)
	token, err := s.refreshRepo.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		return uuid.Nil, err
	}
	return token.ID, nil
}

// generateTempTokenWithRememberMe creates a temp token that also stores the rememberMe preference.
func generateTempTokenWithRememberMe(userID uuid.UUID, rememberMe bool) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "dev-secret-change-in-production"
	}

	claims := jwt.MapClaims{
		"sub":        userID.String(),
		"exp":        time.Now().Add(time.Minute * 5).Unix(),
		"iat":        time.Now().Unix(),
		"type":       "2fa_temp",
		"rememberMe": rememberMe,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ValidateTempTokenWithRememberMe validates a temp token and extracts the rememberMe preference.
func ValidateTempTokenWithRememberMe(tokenString string) (uuid.UUID, bool, error) {
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
		return uuid.Nil, false, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return uuid.Nil, false, errors.New("invalid claims")
	}

	// Verify this is a temp token
	tokenType, ok := claims["type"].(string)
	if !ok || tokenType != "2fa_temp" {
		return uuid.Nil, false, errors.New("not a temporary token")
	}

	userID, err := uuid.Parse(claims["sub"].(string))
	if err != nil {
		return uuid.Nil, false, errors.New("invalid user id in token")
	}

	// Extract rememberMe preference (default to false if not present)
	rememberMe, _ := claims["rememberMe"].(bool)

	return userID, rememberMe, nil
}
