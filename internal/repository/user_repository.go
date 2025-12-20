package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/wealthpath/backend/internal/model"
)

var ErrUserNotFound = errors.New("user not found")
var ErrEmailExists = errors.New("email already exists")

type UserRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	query := `
		INSERT INTO users (id, email, password_hash, name, currency, oauth_provider, oauth_id, avatar_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
		RETURNING created_at, updated_at`

	user.ID = uuid.New()
	return r.db.QueryRowxContext(ctx, query,
		user.ID, user.Email, user.PasswordHash, user.Name, user.Currency,
		user.OAuthProvider, user.OAuthID, user.AvatarURL,
	).Scan(&user.CreatedAt, &user.UpdatedAt)
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	query := `SELECT * FROM users WHERE email = $1`
	err := r.db.GetContext(ctx, &user, query, email)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	return &user, err
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	var user model.User
	query := `SELECT * FROM users WHERE id = $1`
	err := r.db.GetContext(ctx, &user, query, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	return &user, err
}

func (r *UserRepository) Update(ctx context.Context, user *model.User) error {
	query := `
		UPDATE users 
		SET name = $2, currency = $3, oauth_provider = $4, oauth_id = $5, avatar_url = $6, updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at`
	return r.db.QueryRowxContext(ctx, query,
		user.ID, user.Name, user.Currency,
		user.OAuthProvider, user.OAuthID, user.AvatarURL,
	).Scan(&user.UpdatedAt)
}

func (r *UserRepository) GetByOAuth(ctx context.Context, provider, oauthID string) (*model.User, error) {
	var user model.User
	query := `SELECT * FROM users WHERE oauth_provider = $1 AND oauth_id = $2`
	err := r.db.GetContext(ctx, &user, query, provider, oauthID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	return &user, err
}

func (r *UserRepository) EmailExists(ctx context.Context, email string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`
	err := r.db.GetContext(ctx, &exists, query, email)
	return exists, err
}

func (r *UserRepository) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE users SET updated_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *UserRepository) GetOrCreateByOAuth(ctx context.Context, user *model.User) (*model.User, error) {
	// Try to find existing user by OAuth provider
	existing, err := r.GetByOAuth(ctx, *user.OAuthProvider, *user.OAuthID)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, ErrUserNotFound) {
		return nil, err
	}

	// Create new user
	if err := r.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}
