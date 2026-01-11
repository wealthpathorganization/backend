package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/wealthpath/backend/internal/model"
)

// MockUserRepo implements UserRepositoryInterface for testing
type MockUserRepo struct {
	mock.Mock
}

func (m *MockUserRepo) Create(ctx context.Context, user *model.User) error {
	args := m.Called(ctx, user)
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	return args.Error(0)
}

func (m *MockUserRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserRepo) Update(ctx context.Context, user *model.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepo) EmailExists(ctx context.Context, email string) (bool, error) {
	args := m.Called(ctx, email)
	return args.Bool(0), args.Error(1)
}

func (m *MockUserRepo) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserRepo) GetOrCreateByOAuth(ctx context.Context, user *model.User) (*model.User, error) {
	args := m.Called(ctx, user)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserRepo) GetByOAuth(ctx context.Context, provider, oauthID string) (*model.User, error) {
	args := m.Called(ctx, provider, oauthID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

// Table-driven tests with parallel execution (following Go rules)
func TestUserService_Register(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     RegisterInput
		setupMock func(*MockUserRepo)
		wantErr   bool
		errType   error
	}{
		{
			name: "success",
			input: RegisterInput{
				Email:    "test@example.com",
				Password: "password123",
				Name:     "Test User",
			},
			setupMock: func(m *MockUserRepo) {
				m.On("EmailExists", mock.Anything, "test@example.com").Return(false, nil)
				m.On("Create", mock.Anything, mock.AnythingOfType("*model.User")).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "email already taken",
			input: RegisterInput{
				Email:    "existing@example.com",
				Password: "password123",
			},
			setupMock: func(m *MockUserRepo) {
				m.On("EmailExists", mock.Anything, "existing@example.com").Return(true, nil)
			},
			wantErr: true,
			errType: ErrEmailTaken,
		},
		{
			name: "repository error on email check",
			input: RegisterInput{
				Email:    "test@example.com",
				Password: "password123",
			},
			setupMock: func(m *MockUserRepo) {
				m.On("EmailExists", mock.Anything, "test@example.com").Return(false, errors.New("db error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockRepo := new(MockUserRepo)
			service := NewUserService(mockRepo)
			tt.setupMock(mockRepo)

			resp, err := service.Register(context.Background(), tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, resp)
				if tt.errType != nil {
					assert.ErrorIs(t, err, tt.errType)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.NotEmpty(t, resp.Token)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestUserService_Login(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     LoginInput
		setupMock func(*MockUserRepo)
		wantErr   bool
	}{
		{
			name: "user not found",
			input: LoginInput{
				Email:    "notfound@example.com",
				Password: "password123",
			},
			setupMock: func(m *MockUserRepo) {
				m.On("GetByEmail", mock.Anything, "notfound@example.com").Return(nil, errors.New("not found"))
			},
			wantErr: true,
		},
		{
			name: "nil password hash (oauth user)",
			input: LoginInput{
				Email:    "oauth@example.com",
				Password: "password123",
			},
			setupMock: func(m *MockUserRepo) {
				m.On("GetByEmail", mock.Anything, "oauth@example.com").Return(&model.User{
					ID:           uuid.New(),
					Email:        "oauth@example.com",
					PasswordHash: nil, // OAuth user has no password
				}, nil)
			},
			wantErr: true,
		},
		{
			name: "wrong password",
			input: LoginInput{
				Email:    "test@example.com",
				Password: "wrongpassword",
			},
			setupMock: func(m *MockUserRepo) {
				// bcrypt hash for "password123"
				hash := "$2a$10$dseEMkGltX2F0l8M5kC.Y.Dkcb5BeVBjUx58w8KgSQbYGfgu0gRG."
				m.On("GetByEmail", mock.Anything, "test@example.com").Return(&model.User{
					ID:           uuid.New(),
					Email:        "test@example.com",
					PasswordHash: &hash,
				}, nil)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockRepo := new(MockUserRepo)
			service := NewUserService(mockRepo)
			tt.setupMock(mockRepo)

			resp, err := service.Login(context.Background(), tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestUserService_GetByID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setupMock func(*MockUserRepo, uuid.UUID)
		wantErr   bool
	}{
		{
			name: "success",
			setupMock: func(m *MockUserRepo, id uuid.UUID) {
				m.On("GetByID", mock.Anything, id).Return(&model.User{
					ID:    id,
					Email: "test@example.com",
				}, nil)
			},
			wantErr: false,
		},
		{
			name: "not found",
			setupMock: func(m *MockUserRepo, id uuid.UUID) {
				m.On("GetByID", mock.Anything, id).Return(nil, errors.New("not found"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockRepo := new(MockUserRepo)
			service := NewUserService(mockRepo)
			userID := uuid.New()
			tt.setupMock(mockRepo, userID)

			user, err := service.GetByID(context.Background(), userID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestUserService_UpdateSettings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     UpdateSettingsInput
		setupMock func(*MockUserRepo, uuid.UUID)
		wantErr   bool
	}{
		{
			name: "success",
			input: func() UpdateSettingsInput {
				name := "New Name"
				currency := "EUR"
				return UpdateSettingsInput{Name: &name, Currency: &currency}
			}(),
			setupMock: func(m *MockUserRepo, id uuid.UUID) {
				m.On("GetByID", mock.Anything, id).Return(&model.User{
					ID:       id,
					Name:     "Old Name",
					Currency: "USD",
				}, nil)
				m.On("Update", mock.Anything, mock.AnythingOfType("*model.User")).Return(nil)
			},
			wantErr: false,
		},
		{
			name:  "user not found",
			input: UpdateSettingsInput{},
			setupMock: func(m *MockUserRepo, id uuid.UUID) {
				m.On("GetByID", mock.Anything, id).Return(nil, errors.New("not found"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockRepo := new(MockUserRepo)
			service := NewUserService(mockRepo)
			userID := uuid.New()
			tt.setupMock(mockRepo, userID)

			user, err := service.UpdateSettings(context.Background(), userID, tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestGenerateToken(t *testing.T) {
	userID := uuid.New()

	token, err := generateToken(userID)

	assert.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestValidateToken_Valid(t *testing.T) {
	userID := uuid.New()

	token, _ := generateToken(userID)
	parsedID, err := ValidateToken(token)

	assert.NoError(t, err)
	assert.Equal(t, userID, parsedID)
}

func TestValidateToken_InvalidToken(t *testing.T) {
	parsedID, err := ValidateToken("invalid.token.here")

	assert.Error(t, err)
	assert.Equal(t, uuid.Nil, parsedID)
}

func TestValidateToken_EmptyToken(t *testing.T) {
	parsedID, err := ValidateToken("")

	assert.Error(t, err)
	assert.Equal(t, uuid.Nil, parsedID)
}

func TestRegisterInput_Validation(t *testing.T) {
	tests := []struct {
		name    string
		input   RegisterInput
		isValid bool
	}{
		{
			name: "valid input",
			input: RegisterInput{
				Email:    "test@example.com",
				Password: "password123",
				Name:     "Test User",
			},
			isValid: true,
		},
		{
			name: "missing email",
			input: RegisterInput{
				Password: "password123",
				Name:     "Test User",
			},
			isValid: false,
		},
		{
			name: "missing password",
			input: RegisterInput{
				Email: "test@example.com",
				Name:  "Test User",
			},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.input.Email != "" && tt.input.Password != ""
			assert.Equal(t, tt.isValid, isValid)
		})
	}
}

func TestLoginInput_Validation(t *testing.T) {
	tests := []struct {
		name    string
		input   LoginInput
		isValid bool
	}{
		{
			name: "valid input",
			input: LoginInput{
				Email:    "test@example.com",
				Password: "password123",
			},
			isValid: true,
		},
		{
			name: "missing email",
			input: LoginInput{
				Password: "password123",
			},
			isValid: false,
		},
		{
			name: "missing password",
			input: LoginInput{
				Email: "test@example.com",
			},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.input.Email != "" && tt.input.Password != ""
			assert.Equal(t, tt.isValid, isValid)
		})
	}
}

func TestUpdateSettingsInput(t *testing.T) {
	name := "New Name"
	currency := "EUR"

	input := UpdateSettingsInput{
		Name:     &name,
		Currency: &currency,
	}

	assert.Equal(t, "New Name", *input.Name)
	assert.Equal(t, "EUR", *input.Currency)
}

func TestUpdateSettingsInput_PartialUpdate(t *testing.T) {
	name := "New Name"

	input := UpdateSettingsInput{
		Name: &name,
	}

	assert.NotNil(t, input.Name)
	assert.Nil(t, input.Currency)
}

func TestAuthResponse_Structure(t *testing.T) {
	// Test that AuthResponse has the expected structure
	resp := &AuthResponse{
		Token: "test_token",
	}

	assert.Equal(t, "test_token", resp.Token)
}

func TestOAuthUser_Structure(t *testing.T) {
	user := OAuthUser{
		ID:        "123",
		Email:     "test@example.com",
		Name:      "Test User",
		AvatarURL: "https://example.com/avatar.png",
	}

	assert.Equal(t, "123", user.ID)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, "Test User", user.Name)
	assert.Equal(t, "https://example.com/avatar.png", user.AvatarURL)
}

// Test JWT token roundtrip
func TestTokenRoundtrip(t *testing.T) {
	userID := uuid.New()

	token, err := generateToken(userID)
	assert.NoError(t, err)

	parsedID, err := ValidateToken(token)
	assert.NoError(t, err)
	assert.Equal(t, userID, parsedID)
}

// ============ REFRESH TOKEN TESTS ============

// MockRefreshTokenRepo implements RefreshTokenRepositoryInterface for testing
type MockRefreshTokenRepo struct {
	mock.Mock
}

func (m *MockRefreshTokenRepo) Create(ctx context.Context, token *model.RefreshToken) error {
	args := m.Called(ctx, token)
	if token.ID == uuid.Nil {
		token.ID = uuid.New()
	}
	return args.Error(0)
}

func (m *MockRefreshTokenRepo) FindByTokenHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error) {
	args := m.Called(ctx, tokenHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.RefreshToken), args.Error(1)
}

func (m *MockRefreshTokenRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.RefreshToken, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.RefreshToken), args.Error(1)
}

func (m *MockRefreshTokenRepo) FindActiveByUserID(ctx context.Context, userID uuid.UUID) ([]*model.RefreshToken, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.RefreshToken), args.Error(1)
}

func (m *MockRefreshTokenRepo) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRefreshTokenRepo) RevokeByID(ctx context.Context, id uuid.UUID, reason string) error {
	args := m.Called(ctx, id, reason)
	return args.Error(0)
}

func (m *MockRefreshTokenRepo) RevokeByTokenHash(ctx context.Context, tokenHash, reason string) error {
	args := m.Called(ctx, tokenHash, reason)
	return args.Error(0)
}

func (m *MockRefreshTokenRepo) RevokeByUserID(ctx context.Context, userID uuid.UUID, reason string) (int64, error) {
	args := m.Called(ctx, userID, reason)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockRefreshTokenRepo) RevokeByUserIDExcept(ctx context.Context, userID uuid.UUID, exceptID uuid.UUID, reason string) (int64, error) {
	args := m.Called(ctx, userID, exceptID, reason)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockRefreshTokenRepo) DeleteExpired(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func TestGenerateRefreshTokenString(t *testing.T) {
	token1, err := generateRefreshTokenString()
	assert.NoError(t, err)
	assert.NotEmpty(t, token1)
	assert.Equal(t, 64, len(token1)) // 32 bytes = 64 hex characters

	// Tokens should be unique
	token2, err := generateRefreshTokenString()
	assert.NoError(t, err)
	assert.NotEqual(t, token1, token2)
}

func TestHashRefreshToken(t *testing.T) {
	token := "testtoken123"
	hash1 := hashRefreshToken(token)
	hash2 := hashRefreshToken(token)

	// Same input should produce same hash
	assert.Equal(t, hash1, hash2)

	// Hash should be 64 hex characters (SHA-256 = 32 bytes)
	assert.Equal(t, 64, len(hash1))

	// Different inputs should produce different hashes
	hash3 := hashRefreshToken("differenttoken")
	assert.NotEqual(t, hash1, hash3)
}

func TestUserService_RegisterWithRefreshTokens(t *testing.T) {
	t.Parallel()

	mockUserRepo := new(MockUserRepo)
	mockRefreshRepo := new(MockRefreshTokenRepo)
	service := NewUserServiceWithRefreshTokens(mockUserRepo, mockRefreshRepo)

	input := RegisterInput{
		Email:      "test@example.com",
		Password:   "password123",
		Name:       "Test User",
		RememberMe: true,
	}

	mockUserRepo.On("EmailExists", mock.Anything, "test@example.com").Return(false, nil)
	mockUserRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.User")).Return(nil)
	mockRefreshRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.RefreshToken")).Return(nil)

	resp, err := service.Register(context.Background(), input)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.Token)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.Equal(t, int64(AccessTokenExpiry.Seconds()), resp.ExpiresIn)
	mockUserRepo.AssertExpectations(t)
	mockRefreshRepo.AssertExpectations(t)
}

func TestUserService_LoginWithRefreshTokens(t *testing.T) {
	t.Parallel()

	mockUserRepo := new(MockUserRepo)
	mockRefreshRepo := new(MockRefreshTokenRepo)
	service := NewUserServiceWithRefreshTokens(mockUserRepo, mockRefreshRepo)

	// bcrypt hash for "password123"
	hash := "$2a$10$dseEMkGltX2F0l8M5kC.Y.Dkcb5BeVBjUx58w8KgSQbYGfgu0gRG."
	userID := uuid.New()

	input := LoginInput{
		Email:      "test@example.com",
		Password:   "password123",
		RememberMe: false,
	}

	mockUserRepo.On("GetByEmail", mock.Anything, "test@example.com").Return(&model.User{
		ID:           userID,
		Email:        "test@example.com",
		PasswordHash: &hash,
		TOTPEnabled:  false,
	}, nil)
	mockRefreshRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.RefreshToken")).Return(nil)

	resp, err := service.Login(context.Background(), input)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.Token)
	assert.NotEmpty(t, resp.RefreshToken)
	mockUserRepo.AssertExpectations(t)
	mockRefreshRepo.AssertExpectations(t)
}

func TestUserService_GetActiveSessions(t *testing.T) {
	t.Parallel()

	mockUserRepo := new(MockUserRepo)
	mockRefreshRepo := new(MockRefreshTokenRepo)
	service := NewUserServiceWithRefreshTokens(mockUserRepo, mockRefreshRepo)

	userID := uuid.New()
	tokens := []*model.RefreshToken{
		{ID: uuid.New(), UserID: userID, TokenHash: "hash1"},
		{ID: uuid.New(), UserID: userID, TokenHash: "hash2"},
	}

	mockRefreshRepo.On("FindActiveByUserID", mock.Anything, userID).Return(tokens, nil)

	sessions, err := service.GetActiveSessions(context.Background(), userID)

	assert.NoError(t, err)
	assert.Len(t, sessions, 2)
	mockRefreshRepo.AssertExpectations(t)
}

func TestUserService_RevokeAllUserTokens(t *testing.T) {
	t.Parallel()

	mockUserRepo := new(MockUserRepo)
	mockRefreshRepo := new(MockRefreshTokenRepo)
	service := NewUserServiceWithRefreshTokens(mockUserRepo, mockRefreshRepo)

	userID := uuid.New()

	mockRefreshRepo.On("RevokeByUserID", mock.Anything, userID, "sign_out_everywhere").Return(int64(3), nil)

	count, err := service.RevokeAllUserTokens(context.Background(), userID, "sign_out_everywhere")

	assert.NoError(t, err)
	assert.Equal(t, int64(3), count)
	mockRefreshRepo.AssertExpectations(t)
}

func TestUserService_RevokeSession(t *testing.T) {
	t.Parallel()

	mockUserRepo := new(MockUserRepo)
	mockRefreshRepo := new(MockRefreshTokenRepo)
	service := NewUserServiceWithRefreshTokens(mockUserRepo, mockRefreshRepo)

	userID := uuid.New()
	sessionID := uuid.New()

	mockRefreshRepo.On("FindByID", mock.Anything, sessionID).Return(&model.RefreshToken{
		ID:     sessionID,
		UserID: userID,
	}, nil)
	mockRefreshRepo.On("RevokeByID", mock.Anything, sessionID, "user_revoked").Return(nil)

	err := service.RevokeSession(context.Background(), userID, sessionID, "user_revoked")

	assert.NoError(t, err)
	mockRefreshRepo.AssertExpectations(t)
}

func TestUserService_RevokeSession_WrongUser(t *testing.T) {
	t.Parallel()

	mockUserRepo := new(MockUserRepo)
	mockRefreshRepo := new(MockRefreshTokenRepo)
	service := NewUserServiceWithRefreshTokens(mockUserRepo, mockRefreshRepo)

	userID := uuid.New()
	otherUserID := uuid.New()
	sessionID := uuid.New()

	mockRefreshRepo.On("FindByID", mock.Anything, sessionID).Return(&model.RefreshToken{
		ID:     sessionID,
		UserID: otherUserID, // Different user
	}, nil)

	err := service.RevokeSession(context.Background(), userID, sessionID, "user_revoked")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session does not belong to user")
	mockRefreshRepo.AssertExpectations(t)
}

func TestTempTokenWithRememberMe(t *testing.T) {
	userID := uuid.New()

	// Test with rememberMe = true
	token1, err := generateTempTokenWithRememberMe(userID, true)
	assert.NoError(t, err)
	assert.NotEmpty(t, token1)

	parsedID, rememberMe, err := ValidateTempTokenWithRememberMe(token1)
	assert.NoError(t, err)
	assert.Equal(t, userID, parsedID)
	assert.True(t, rememberMe)

	// Test with rememberMe = false
	token2, err := generateTempTokenWithRememberMe(userID, false)
	assert.NoError(t, err)

	parsedID, rememberMe, err = ValidateTempTokenWithRememberMe(token2)
	assert.NoError(t, err)
	assert.Equal(t, userID, parsedID)
	assert.False(t, rememberMe)
}

func TestValidateTempTokenWithRememberMe_InvalidToken(t *testing.T) {
	_, _, err := ValidateTempTokenWithRememberMe("invalid.token.here")
	assert.Error(t, err)
}

func TestAccessTokenExpiry(t *testing.T) {
	// Verify the access token expiry constant
	assert.Equal(t, 15*60, int(AccessTokenExpiry.Seconds()))
}

func TestRefreshTokenExpiry(t *testing.T) {
	// Verify the refresh token expiry constants
	assert.Equal(t, 7*24*60*60, int(RefreshTokenExpiry.Seconds()))
	assert.Equal(t, 30*24*60*60, int(RememberMeRefreshExpiry.Seconds()))
}
