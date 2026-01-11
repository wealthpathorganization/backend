package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/wealthpath/backend/internal/model"
)

var ErrRefreshTokenNotFound = errors.New("refresh token not found")

// RefreshTokenRepository handles persistence of refresh tokens.
type RefreshTokenRepository struct {
	db *sqlx.DB
}

// NewRefreshTokenRepository creates a new refresh token repository.
func NewRefreshTokenRepository(db *sqlx.DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

// refreshTokenRow is an internal struct for database scanning with JSONB support.
type refreshTokenRow struct {
	ID            uuid.UUID      `db:"id"`
	UserID        uuid.UUID      `db:"user_id"`
	TokenHash     string         `db:"token_hash"`
	DeviceInfo    sql.NullString `db:"device_info"`
	CreatedAt     time.Time      `db:"created_at"`
	ExpiresAt     time.Time      `db:"expires_at"`
	LastUsedAt    time.Time      `db:"last_used_at"`
	RevokedAt     sql.NullTime   `db:"revoked_at"`
	RevokedReason sql.NullString `db:"revoked_reason"`
}

// toModel converts a database row to a model.RefreshToken.
func (r *refreshTokenRow) toModel() *model.RefreshToken {
	token := &model.RefreshToken{
		ID:         r.ID,
		UserID:     r.UserID,
		TokenHash:  r.TokenHash,
		CreatedAt:  r.CreatedAt,
		ExpiresAt:  r.ExpiresAt,
		LastUsedAt: r.LastUsedAt,
	}

	if r.DeviceInfo.Valid {
		var deviceInfo model.DeviceInfo
		if err := json.Unmarshal([]byte(r.DeviceInfo.String), &deviceInfo); err == nil {
			token.DeviceInfo = &deviceInfo
		}
	}

	if r.RevokedAt.Valid {
		token.RevokedAt = &r.RevokedAt.Time
	}

	if r.RevokedReason.Valid {
		token.RevokedReason = &r.RevokedReason.String
	}

	return token
}

// Create stores a new refresh token in the database.
func (r *RefreshTokenRepository) Create(ctx context.Context, token *model.RefreshToken) error {
	var deviceInfoJSON interface{}
	if token.DeviceInfo != nil {
		jsonBytes, err := json.Marshal(token.DeviceInfo)
		if err != nil {
			return err
		}
		deviceInfoJSON = string(jsonBytes)
	}

	query := `
		INSERT INTO refresh_tokens (id, user_id, token_hash, device_info, expires_at, created_at, last_used_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		RETURNING created_at, last_used_at`

	token.ID = uuid.New()
	return r.db.QueryRowxContext(ctx, query,
		token.ID, token.UserID, token.TokenHash, deviceInfoJSON, token.ExpiresAt,
	).Scan(&token.CreatedAt, &token.LastUsedAt)
}

// FindByTokenHash retrieves a refresh token by its hash.
func (r *RefreshTokenRepository) FindByTokenHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error) {
	var row refreshTokenRow
	query := `SELECT * FROM refresh_tokens WHERE token_hash = $1`
	err := r.db.GetContext(ctx, &row, query, tokenHash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrRefreshTokenNotFound
	}
	if err != nil {
		return nil, err
	}
	return row.toModel(), nil
}

// FindByID retrieves a refresh token by its ID.
func (r *RefreshTokenRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.RefreshToken, error) {
	var row refreshTokenRow
	query := `SELECT * FROM refresh_tokens WHERE id = $1`
	err := r.db.GetContext(ctx, &row, query, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrRefreshTokenNotFound
	}
	if err != nil {
		return nil, err
	}
	return row.toModel(), nil
}

// FindActiveByUserID retrieves all active (non-revoked, non-expired) refresh tokens for a user.
func (r *RefreshTokenRepository) FindActiveByUserID(ctx context.Context, userID uuid.UUID) ([]*model.RefreshToken, error) {
	var rows []refreshTokenRow
	query := `
		SELECT * FROM refresh_tokens
		WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > NOW()
		ORDER BY last_used_at DESC`
	err := r.db.SelectContext(ctx, &rows, query, userID)
	if err != nil {
		return nil, err
	}

	tokens := make([]*model.RefreshToken, len(rows))
	for i, row := range rows {
		tokens[i] = row.toModel()
	}
	return tokens, nil
}

// UpdateLastUsed updates the last_used_at timestamp for a token.
func (r *RefreshTokenRepository) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE refresh_tokens SET last_used_at = NOW() WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrRefreshTokenNotFound
	}
	return nil
}

// RevokeByID revokes a specific refresh token.
func (r *RefreshTokenRepository) RevokeByID(ctx context.Context, id uuid.UUID, reason string) error {
	query := `
		UPDATE refresh_tokens
		SET revoked_at = NOW(), revoked_reason = $2
		WHERE id = $1 AND revoked_at IS NULL`
	result, err := r.db.ExecContext(ctx, query, id, reason)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrRefreshTokenNotFound
	}
	return nil
}

// RevokeByTokenHash revokes a refresh token by its hash.
func (r *RefreshTokenRepository) RevokeByTokenHash(ctx context.Context, tokenHash, reason string) error {
	query := `
		UPDATE refresh_tokens
		SET revoked_at = NOW(), revoked_reason = $2
		WHERE token_hash = $1 AND revoked_at IS NULL`
	result, err := r.db.ExecContext(ctx, query, tokenHash, reason)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrRefreshTokenNotFound
	}
	return nil
}

// RevokeByUserID revokes all refresh tokens for a user.
func (r *RefreshTokenRepository) RevokeByUserID(ctx context.Context, userID uuid.UUID, reason string) (int64, error) {
	query := `
		UPDATE refresh_tokens
		SET revoked_at = NOW(), revoked_reason = $2
		WHERE user_id = $1 AND revoked_at IS NULL`
	result, err := r.db.ExecContext(ctx, query, userID, reason)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// RevokeByUserIDExcept revokes all refresh tokens for a user except the specified one.
// Useful for "sign out other sessions" functionality.
func (r *RefreshTokenRepository) RevokeByUserIDExcept(ctx context.Context, userID uuid.UUID, exceptID uuid.UUID, reason string) (int64, error) {
	query := `
		UPDATE refresh_tokens
		SET revoked_at = NOW(), revoked_reason = $3
		WHERE user_id = $1 AND id != $2 AND revoked_at IS NULL`
	result, err := r.db.ExecContext(ctx, query, userID, exceptID, reason)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// DeleteExpired removes all expired tokens from the database.
// Should be called periodically by a cleanup job.
func (r *RefreshTokenRepository) DeleteExpired(ctx context.Context) (int64, error) {
	query := `DELETE FROM refresh_tokens WHERE expires_at < NOW()`
	result, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// CountActiveByUserID returns the number of active sessions for a user.
func (r *RefreshTokenRepository) CountActiveByUserID(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM refresh_tokens WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > NOW()`
	err := r.db.GetContext(ctx, &count, query, userID)
	return count, err
}
