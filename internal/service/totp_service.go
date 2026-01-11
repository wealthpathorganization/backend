package service

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

var (
	ErrTOTPAlreadyEnabled  = errors.New("2FA is already enabled")
	ErrTOTPNotEnabled      = errors.New("2FA is not enabled")
	ErrInvalidTOTPCode     = errors.New("invalid 2FA code")
	ErrBackupCodeNotFound  = errors.New("invalid backup code")
	ErrTOTPNotSetup        = errors.New("2FA has not been set up")
	ErrTOTPRequiresVerify  = errors.New("2FA requires verification")
)

// TOTPUserRepository defines the interface for user data access needed by TOTP service.
type TOTPUserRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*UserEntity, error)
	UpdateTOTPSecret(ctx context.Context, userID uuid.UUID, secret *string) error
	EnableTOTP(ctx context.Context, userID uuid.UUID, backupCodes []string) error
	DisableTOTP(ctx context.Context, userID uuid.UUID) error
	UpdateBackupCodes(ctx context.Context, userID uuid.UUID, codes []string) error
}

// UserEntity represents a user with TOTP fields for the TOTP service.
// This avoids circular imports with the model package.
type UserEntity struct {
	ID              uuid.UUID
	Email           string
	Name            string
	TOTPSecret      *string
	TOTPEnabled     bool
	TOTPBackupCodes []string
	TOTPVerifiedAt  *time.Time
}

// TOTPService handles two-factor authentication operations.
type TOTPService struct {
	userRepo   TOTPUserRepository
	issuerName string
}

// NewTOTPService creates a new TOTP service.
func NewTOTPService(userRepo TOTPUserRepository, issuerName string) *TOTPService {
	if issuerName == "" {
		issuerName = "WealthPath"
	}
	return &TOTPService{
		userRepo:   userRepo,
		issuerName: issuerName,
	}
}

// TOTPSetupResponse contains the data needed for the user to set up 2FA.
type TOTPSetupResponse struct {
	Secret      string `json:"secret"`
	QRCodeURL   string `json:"qrCodeUrl"`
	ManualEntry string `json:"manualEntry"`
}

// Setup generates a new TOTP secret for the user without enabling 2FA.
// The user must verify the code using Verify before 2FA is enabled.
func (s *TOTPService) Setup(ctx context.Context, userID uuid.UUID) (*TOTPSetupResponse, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("getting user: %w", err)
	}

	if user.TOTPEnabled {
		return nil, ErrTOTPAlreadyEnabled
	}

	// Generate a new TOTP key
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      s.issuerName,
		AccountName: user.Email,
		SecretSize:  32,
		Algorithm:   otp.AlgorithmSHA1,
		Digits:      otp.DigitsSix,
		Period:      30,
	})
	if err != nil {
		return nil, fmt.Errorf("generating TOTP key: %w", err)
	}

	// Store the secret temporarily (not enabled yet)
	secret := key.Secret()
	if err := s.userRepo.UpdateTOTPSecret(ctx, userID, &secret); err != nil {
		return nil, fmt.Errorf("storing TOTP secret: %w", err)
	}

	return &TOTPSetupResponse{
		Secret:      key.Secret(),
		QRCodeURL:   key.URL(),
		ManualEntry: fmt.Sprintf("%s (%s)", key.Secret(), user.Email),
	}, nil
}

// Verify validates a TOTP code and enables 2FA if successful.
// This should be called after Setup to complete the 2FA enrollment.
func (s *TOTPService) Verify(ctx context.Context, userID uuid.UUID, code string) ([]string, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("getting user: %w", err)
	}

	if user.TOTPEnabled {
		return nil, ErrTOTPAlreadyEnabled
	}

	if user.TOTPSecret == nil {
		return nil, ErrTOTPNotSetup
	}

	// Validate the code
	if !totp.Validate(code, *user.TOTPSecret) {
		return nil, ErrInvalidTOTPCode
	}

	// Generate backup codes
	backupCodes, err := generateBackupCodes(10)
	if err != nil {
		return nil, fmt.Errorf("generating backup codes: %w", err)
	}

	// Enable 2FA
	if err := s.userRepo.EnableTOTP(ctx, userID, backupCodes); err != nil {
		return nil, fmt.Errorf("enabling 2FA: %w", err)
	}

	return backupCodes, nil
}

// ValidateCode validates a TOTP code for an already-enabled 2FA user.
func (s *TOTPService) ValidateCode(ctx context.Context, userID uuid.UUID, code string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("getting user: %w", err)
	}

	if !user.TOTPEnabled || user.TOTPSecret == nil {
		return ErrTOTPNotEnabled
	}

	if !totp.Validate(code, *user.TOTPSecret) {
		return ErrInvalidTOTPCode
	}

	return nil
}

// ValidateBackupCode validates and consumes a backup code.
func (s *TOTPService) ValidateBackupCode(ctx context.Context, userID uuid.UUID, code string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("getting user: %w", err)
	}

	if !user.TOTPEnabled {
		return ErrTOTPNotEnabled
	}

	// Find and remove the backup code
	normalizedCode := strings.ToUpper(strings.TrimSpace(code))
	var newCodes []string
	found := false

	for _, c := range user.TOTPBackupCodes {
		if strings.ToUpper(c) == normalizedCode {
			found = true
			continue // Don't include this code in new list (consume it)
		}
		newCodes = append(newCodes, c)
	}

	if !found {
		return ErrBackupCodeNotFound
	}

	// Update backup codes
	if err := s.userRepo.UpdateBackupCodes(ctx, userID, newCodes); err != nil {
		return fmt.Errorf("updating backup codes: %w", err)
	}

	return nil
}

// Disable disables 2FA for the user after verifying the code.
func (s *TOTPService) Disable(ctx context.Context, userID uuid.UUID, code string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("getting user: %w", err)
	}

	if !user.TOTPEnabled || user.TOTPSecret == nil {
		return ErrTOTPNotEnabled
	}

	// Validate the code before disabling
	if !totp.Validate(code, *user.TOTPSecret) {
		return ErrInvalidTOTPCode
	}

	if err := s.userRepo.DisableTOTP(ctx, userID); err != nil {
		return fmt.Errorf("disabling 2FA: %w", err)
	}

	return nil
}

// RegenerateBackupCodes generates new backup codes for the user.
func (s *TOTPService) RegenerateBackupCodes(ctx context.Context, userID uuid.UUID, code string) ([]string, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("getting user: %w", err)
	}

	if !user.TOTPEnabled || user.TOTPSecret == nil {
		return nil, ErrTOTPNotEnabled
	}

	// Validate the code before regenerating
	if !totp.Validate(code, *user.TOTPSecret) {
		return nil, ErrInvalidTOTPCode
	}

	// Generate new backup codes
	backupCodes, err := generateBackupCodes(10)
	if err != nil {
		return nil, fmt.Errorf("generating backup codes: %w", err)
	}

	if err := s.userRepo.UpdateBackupCodes(ctx, userID, backupCodes); err != nil {
		return nil, fmt.Errorf("updating backup codes: %w", err)
	}

	return backupCodes, nil
}

// generateBackupCodes generates n random backup codes.
func generateBackupCodes(n int) ([]string, error) {
	codes := make([]string, n)
	for i := 0; i < n; i++ {
		bytes := make([]byte, 5) // 8 characters in base32
		if _, err := rand.Read(bytes); err != nil {
			return nil, err
		}
		// Format as XXXX-XXXX for readability
		code := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(bytes)
		codes[i] = code[:4] + "-" + code[4:8]
	}
	return codes, nil
}
