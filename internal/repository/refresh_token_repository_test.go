package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wealthpath/backend/internal/model"
)

func setupMockDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	db := sqlx.NewDb(mockDB, "postgres")
	return db, mock
}

func TestRefreshTokenRepository_Create(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewRefreshTokenRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	token := &model.RefreshToken{
		UserID:    userID,
		TokenHash: "hashedtoken123",
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	rows := sqlmock.NewRows([]string{"created_at", "last_used_at"}).
		AddRow(time.Now(), time.Now())

	mock.ExpectQuery(`INSERT INTO refresh_tokens`).
		WithArgs(sqlmock.AnyArg(), userID, "hashedtoken123", nil, token.ExpiresAt).
		WillReturnRows(rows)

	err := repo.Create(ctx, token)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, token.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRefreshTokenRepository_Create_WithDeviceInfo(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewRefreshTokenRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	token := &model.RefreshToken{
		UserID:    userID,
		TokenHash: "hashedtoken123",
		DeviceInfo: &model.DeviceInfo{
			Browser:    "Chrome",
			OS:         "macOS",
			DeviceType: "desktop",
			IP:         "192.168.1.1",
		},
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	rows := sqlmock.NewRows([]string{"created_at", "last_used_at"}).
		AddRow(time.Now(), time.Now())

	mock.ExpectQuery(`INSERT INTO refresh_tokens`).
		WithArgs(sqlmock.AnyArg(), userID, "hashedtoken123", sqlmock.AnyArg(), token.ExpiresAt).
		WillReturnRows(rows)

	err := repo.Create(ctx, token)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRefreshTokenRepository_FindByTokenHash(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewRefreshTokenRepository(db)
	ctx := context.Background()

	tokenID := uuid.New()
	userID := uuid.New()
	tokenHash := "hashedtoken123"
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "token_hash", "device_info",
		"created_at", "expires_at", "last_used_at", "revoked_at", "revoked_reason",
	}).AddRow(
		tokenID, userID, tokenHash, nil,
		now, now.Add(7*24*time.Hour), now, nil, nil,
	)

	mock.ExpectQuery(`SELECT \* FROM refresh_tokens WHERE token_hash = \$1`).
		WithArgs(tokenHash).
		WillReturnRows(rows)

	token, err := repo.FindByTokenHash(ctx, tokenHash)
	assert.NoError(t, err)
	assert.NotNil(t, token)
	assert.Equal(t, tokenID, token.ID)
	assert.Equal(t, userID, token.UserID)
	assert.Equal(t, tokenHash, token.TokenHash)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRefreshTokenRepository_FindByTokenHash_NotFound(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewRefreshTokenRepository(db)
	ctx := context.Background()

	mock.ExpectQuery(`SELECT \* FROM refresh_tokens WHERE token_hash = \$1`).
		WithArgs("nonexistent").
		WillReturnRows(sqlmock.NewRows(nil))

	token, err := repo.FindByTokenHash(ctx, "nonexistent")
	assert.ErrorIs(t, err, ErrRefreshTokenNotFound)
	assert.Nil(t, token)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRefreshTokenRepository_FindActiveByUserID(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewRefreshTokenRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "token_hash", "device_info",
		"created_at", "expires_at", "last_used_at", "revoked_at", "revoked_reason",
	}).
		AddRow(uuid.New(), userID, "hash1", nil, now, now.Add(7*24*time.Hour), now, nil, nil).
		AddRow(uuid.New(), userID, "hash2", nil, now, now.Add(7*24*time.Hour), now, nil, nil)

	mock.ExpectQuery(`SELECT \* FROM refresh_tokens`).
		WithArgs(userID).
		WillReturnRows(rows)

	tokens, err := repo.FindActiveByUserID(ctx, userID)
	assert.NoError(t, err)
	assert.Len(t, tokens, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRefreshTokenRepository_RevokeByID(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewRefreshTokenRepository(db)
	ctx := context.Background()

	tokenID := uuid.New()

	mock.ExpectExec(`UPDATE refresh_tokens`).
		WithArgs(tokenID, "logout").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.RevokeByID(ctx, tokenID, "logout")
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRefreshTokenRepository_RevokeByID_NotFound(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewRefreshTokenRepository(db)
	ctx := context.Background()

	tokenID := uuid.New()

	mock.ExpectExec(`UPDATE refresh_tokens`).
		WithArgs(tokenID, "logout").
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.RevokeByID(ctx, tokenID, "logout")
	assert.ErrorIs(t, err, ErrRefreshTokenNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRefreshTokenRepository_RevokeByUserID(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewRefreshTokenRepository(db)
	ctx := context.Background()

	userID := uuid.New()

	mock.ExpectExec(`UPDATE refresh_tokens`).
		WithArgs(userID, "sign_out_everywhere").
		WillReturnResult(sqlmock.NewResult(0, 3))

	count, err := repo.RevokeByUserID(ctx, userID, "sign_out_everywhere")
	assert.NoError(t, err)
	assert.Equal(t, int64(3), count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRefreshTokenRepository_RevokeByUserIDExcept(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewRefreshTokenRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	exceptID := uuid.New()

	mock.ExpectExec(`UPDATE refresh_tokens`).
		WithArgs(userID, exceptID, "revoke_others").
		WillReturnResult(sqlmock.NewResult(0, 2))

	count, err := repo.RevokeByUserIDExcept(ctx, userID, exceptID, "revoke_others")
	assert.NoError(t, err)
	assert.Equal(t, int64(2), count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRefreshTokenRepository_UpdateLastUsed(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewRefreshTokenRepository(db)
	ctx := context.Background()

	tokenID := uuid.New()

	mock.ExpectExec(`UPDATE refresh_tokens SET last_used_at = NOW\(\) WHERE id = \$1`).
		WithArgs(tokenID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.UpdateLastUsed(ctx, tokenID)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRefreshTokenRepository_DeleteExpired(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewRefreshTokenRepository(db)
	ctx := context.Background()

	mock.ExpectExec(`DELETE FROM refresh_tokens WHERE expires_at < NOW\(\)`).
		WillReturnResult(sqlmock.NewResult(0, 5))

	count, err := repo.DeleteExpired(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(5), count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRefreshToken_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		expected  bool
	}{
		{
			name:      "not expired",
			expiresAt: time.Now().Add(time.Hour),
			expected:  false,
		},
		{
			name:      "expired",
			expiresAt: time.Now().Add(-time.Hour),
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := &model.RefreshToken{ExpiresAt: tt.expiresAt}
			assert.Equal(t, tt.expected, token.IsExpired())
		})
	}
}

func TestRefreshToken_IsRevoked(t *testing.T) {
	tests := []struct {
		name      string
		revokedAt *time.Time
		expected  bool
	}{
		{
			name:      "not revoked",
			revokedAt: nil,
			expected:  false,
		},
		{
			name:      "revoked",
			revokedAt: func() *time.Time { t := time.Now(); return &t }(),
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := &model.RefreshToken{RevokedAt: tt.revokedAt}
			assert.Equal(t, tt.expected, token.IsRevoked())
		})
	}
}

func TestRefreshToken_IsValid(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name      string
		expiresAt time.Time
		revokedAt *time.Time
		expected  bool
	}{
		{
			name:      "valid",
			expiresAt: now.Add(time.Hour),
			revokedAt: nil,
			expected:  true,
		},
		{
			name:      "expired",
			expiresAt: now.Add(-time.Hour),
			revokedAt: nil,
			expected:  false,
		},
		{
			name:      "revoked",
			expiresAt: now.Add(time.Hour),
			revokedAt: &now,
			expected:  false,
		},
		{
			name:      "expired and revoked",
			expiresAt: now.Add(-time.Hour),
			revokedAt: &now,
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := &model.RefreshToken{
				ExpiresAt: tt.expiresAt,
				RevokedAt: tt.revokedAt,
			}
			assert.Equal(t, tt.expected, token.IsValid())
		})
	}
}
