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
				hash := "$2a$10$kBZ2G.6fNOhP0s8pKpD/euRqtMBSS.Q7y.lhsF2mMhTbh1J5yOi16"
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
