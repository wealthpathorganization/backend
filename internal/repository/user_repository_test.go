package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/wealthpath/backend/internal/model"
)

func TestNewUserRepository(t *testing.T) {
	t.Parallel()

	mockDB, _, _ := sqlmock.New()
	defer func() { _ = mockDB.Close() }()
	db := sqlx.NewDb(mockDB, "sqlmock")

	repo := NewUserRepository(db)
	assert.NotNil(t, repo)
}

func TestUserRepository_Create(t *testing.T) {
	t.Parallel()

	mockDB, mock, _ := sqlmock.New()
	defer func() { _ = mockDB.Close() }()
	db := sqlx.NewDb(mockDB, "sqlmock")
	repo := NewUserRepository(db)

	ctx := context.Background()
	hash := "$2a$10$abc123"
	user := &model.User{
		Email:        "test@example.com",
		PasswordHash: &hash,
		Name:         "Test User",
		Currency:     "USD",
	}

	now := time.Now()
	rows := sqlmock.NewRows([]string{"created_at", "updated_at"}).AddRow(now, now)

	mock.ExpectQuery(`INSERT INTO users`).
		WithArgs(sqlmock.AnyArg(), user.Email, user.PasswordHash, user.Name, user.Currency, nil, nil, nil).
		WillReturnRows(rows)

	err := repo.Create(ctx, user)

	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, user.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_GetByEmail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		email     string
		setupMock func(sqlmock.Sqlmock, string)
		wantErr   bool
		errType   error
	}{
		{
			name:  "success",
			email: "test@example.com",
			setupMock: func(mock sqlmock.Sqlmock, email string) {
				hash := "$2a$10$abc"
				rows := sqlmock.NewRows([]string{"id", "email", "password_hash", "name", "currency", "oauth_provider", "oauth_id", "avatar_url", "created_at", "updated_at"}).
					AddRow(uuid.New(), email, &hash, "Test", "USD", nil, nil, nil, time.Now(), time.Now())
				mock.ExpectQuery(`SELECT \* FROM users WHERE email = \$1`).
					WithArgs(email).
					WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name:  "not found",
			email: "notfound@example.com",
			setupMock: func(mock sqlmock.Sqlmock, email string) {
				mock.ExpectQuery(`SELECT \* FROM users WHERE email = \$1`).
					WithArgs(email).
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
			errType: ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockDB, mock, _ := sqlmock.New()
			defer func() { _ = mockDB.Close() }()
			db := sqlx.NewDb(mockDB, "sqlmock")
			repo := NewUserRepository(db)

			ctx := context.Background()
			tt.setupMock(mock, tt.email)

			user, err := repo.GetByEmail(ctx, tt.email)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != nil {
					assert.ErrorIs(t, err, tt.errType)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, tt.email, user.Email)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepository_GetByID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setupMock func(sqlmock.Sqlmock, uuid.UUID)
		wantErr   bool
		errType   error
	}{
		{
			name: "success",
			setupMock: func(mock sqlmock.Sqlmock, id uuid.UUID) {
				hash := "$2a$10$abc"
				rows := sqlmock.NewRows([]string{"id", "email", "password_hash", "name", "currency", "oauth_provider", "oauth_id", "avatar_url", "created_at", "updated_at"}).
					AddRow(id, "test@example.com", &hash, "Test", "USD", nil, nil, nil, time.Now(), time.Now())
				mock.ExpectQuery(`SELECT \* FROM users WHERE id = \$1`).
					WithArgs(id).
					WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name: "not found",
			setupMock: func(mock sqlmock.Sqlmock, id uuid.UUID) {
				mock.ExpectQuery(`SELECT \* FROM users WHERE id = \$1`).
					WithArgs(id).
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
			errType: ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockDB, mock, _ := sqlmock.New()
			defer func() { _ = mockDB.Close() }()
			db := sqlx.NewDb(mockDB, "sqlmock")
			repo := NewUserRepository(db)

			ctx := context.Background()
			userID := uuid.New()
			tt.setupMock(mock, userID)

			user, err := repo.GetByID(ctx, userID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != nil {
					assert.ErrorIs(t, err, tt.errType)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, userID, user.ID)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepository_Update(t *testing.T) {
	t.Parallel()

	mockDB, mock, _ := sqlmock.New()
	defer func() { _ = mockDB.Close() }()
	db := sqlx.NewDb(mockDB, "sqlmock")
	repo := NewUserRepository(db)

	ctx := context.Background()
	user := &model.User{
		ID:       uuid.New(),
		Name:     "Updated Name",
		Currency: "EUR",
	}

	now := time.Now()
	rows := sqlmock.NewRows([]string{"updated_at"}).AddRow(now)

	mock.ExpectQuery(`UPDATE users`).
		WithArgs(user.ID, user.Name, user.Currency, nil, nil, nil).
		WillReturnRows(rows)

	err := repo.Update(ctx, user)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_GetByOAuth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		provider  string
		oauthID   string
		setupMock func(sqlmock.Sqlmock, string, string)
		wantErr   bool
		errType   error
	}{
		{
			name:     "success",
			provider: "google",
			oauthID:  "123456",
			setupMock: func(mock sqlmock.Sqlmock, provider, oauthID string) {
				rows := sqlmock.NewRows([]string{"id", "email", "password_hash", "name", "currency", "oauth_provider", "oauth_id", "avatar_url", "created_at", "updated_at"}).
					AddRow(uuid.New(), "test@example.com", nil, "Test", "USD", &provider, &oauthID, nil, time.Now(), time.Now())
				mock.ExpectQuery(`SELECT \* FROM users WHERE oauth_provider = \$1 AND oauth_id = \$2`).
					WithArgs(provider, oauthID).
					WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name:     "not found",
			provider: "facebook",
			oauthID:  "unknown",
			setupMock: func(mock sqlmock.Sqlmock, provider, oauthID string) {
				mock.ExpectQuery(`SELECT \* FROM users WHERE oauth_provider = \$1 AND oauth_id = \$2`).
					WithArgs(provider, oauthID).
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
			errType: ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockDB, mock, _ := sqlmock.New()
			defer func() { _ = mockDB.Close() }()
			db := sqlx.NewDb(mockDB, "sqlmock")
			repo := NewUserRepository(db)

			ctx := context.Background()
			tt.setupMock(mock, tt.provider, tt.oauthID)

			user, err := repo.GetByOAuth(ctx, tt.provider, tt.oauthID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != nil {
					assert.ErrorIs(t, err, tt.errType)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepository_EmailExists(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		email      string
		setupMock  func(sqlmock.Sqlmock, string)
		wantExists bool
		wantErr    bool
	}{
		{
			name:  "exists",
			email: "existing@example.com",
			setupMock: func(mock sqlmock.Sqlmock, email string) {
				rows := sqlmock.NewRows([]string{"exists"}).AddRow(true)
				mock.ExpectQuery(`SELECT EXISTS`).
					WithArgs(email).
					WillReturnRows(rows)
			},
			wantExists: true,
			wantErr:    false,
		},
		{
			name:  "not exists",
			email: "new@example.com",
			setupMock: func(mock sqlmock.Sqlmock, email string) {
				rows := sqlmock.NewRows([]string{"exists"}).AddRow(false)
				mock.ExpectQuery(`SELECT EXISTS`).
					WithArgs(email).
					WillReturnRows(rows)
			},
			wantExists: false,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockDB, mock, _ := sqlmock.New()
			defer func() { _ = mockDB.Close() }()
			db := sqlx.NewDb(mockDB, "sqlmock")
			repo := NewUserRepository(db)

			ctx := context.Background()
			tt.setupMock(mock, tt.email)

			exists, err := repo.EmailExists(ctx, tt.email)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantExists, exists)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepository_UpdateLastLogin(t *testing.T) {
	t.Parallel()

	mockDB, mock, _ := sqlmock.New()
	defer func() { _ = mockDB.Close() }()
	db := sqlx.NewDb(mockDB, "sqlmock")
	repo := NewUserRepository(db)

	ctx := context.Background()
	userID := uuid.New()

	mock.ExpectExec(`UPDATE users SET updated_at = NOW\(\) WHERE id = \$1`).
		WithArgs(userID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.UpdateLastLogin(ctx, userID)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_GetOrCreateByOAuth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setupMock func(sqlmock.Sqlmock, *model.User)
		wantErr   bool
	}{
		{
			name: "existing user found",
			setupMock: func(mock sqlmock.Sqlmock, user *model.User) {
				rows := sqlmock.NewRows([]string{"id", "email", "password_hash", "name", "currency", "oauth_provider", "oauth_id", "avatar_url", "created_at", "updated_at"}).
					AddRow(uuid.New(), "existing@example.com", nil, "Existing", "USD", user.OAuthProvider, user.OAuthID, nil, time.Now(), time.Now())
				mock.ExpectQuery(`SELECT \* FROM users WHERE oauth_provider = \$1 AND oauth_id = \$2`).
					WithArgs(*user.OAuthProvider, *user.OAuthID).
					WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name: "create new user",
			setupMock: func(mock sqlmock.Sqlmock, user *model.User) {
				// First, GetByOAuth returns not found
				mock.ExpectQuery(`SELECT \* FROM users WHERE oauth_provider = \$1 AND oauth_id = \$2`).
					WithArgs(*user.OAuthProvider, *user.OAuthID).
					WillReturnError(sql.ErrNoRows)

				// Then Create is called
				now := time.Now()
				rows := sqlmock.NewRows([]string{"created_at", "updated_at"}).AddRow(now, now)
				mock.ExpectQuery(`INSERT INTO users`).
					WithArgs(sqlmock.AnyArg(), user.Email, nil, user.Name, user.Currency, user.OAuthProvider, user.OAuthID, nil).
					WillReturnRows(rows)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockDB, mock, _ := sqlmock.New()
			defer func() { _ = mockDB.Close() }()
			db := sqlx.NewDb(mockDB, "sqlmock")
			repo := NewUserRepository(db)

			ctx := context.Background()
			provider := "google"
			oauthID := "123456"
			user := &model.User{
				Email:         "test@example.com",
				Name:          "Test",
				Currency:      "USD",
				OAuthProvider: &provider,
				OAuthID:       &oauthID,
			}

			tt.setupMock(mock, user)

			result, err := repo.GetOrCreateByOAuth(ctx, user)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepository_Errors(t *testing.T) {
	t.Parallel()

	assert.Error(t, ErrUserNotFound)
	assert.Equal(t, "user not found", ErrUserNotFound.Error())

	assert.Error(t, ErrEmailExists)
	assert.Equal(t, "email already exists", ErrEmailExists.Error())
}
