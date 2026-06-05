package auth

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Repository handles database operations for users.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new auth repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// FindByUsername finds a user by username.
func (r *Repository) FindByUsername(ctx context.Context, username string) (*User, error) {
	query := `SELECT id, username, email, password_hash, role, last_login_at, created_at, updated_at
		FROM users WHERE username = ?`

	var u User
	var lastLogin sql.NullTime
	err := r.db.QueryRowContext(ctx, query, username).Scan(
		&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Role,
		&lastLogin, &u.CreatedAt, &u.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, fmt.Errorf("find user by username: %w", err)
	}
	if lastLogin.Valid {
		u.LastLoginAt = &lastLogin.Time
	}
	return &u, nil
}

// FindByID finds a user by ID.
func (r *Repository) FindByID(ctx context.Context, id int64) (*User, error) {
	query := `SELECT id, username, email, password_hash, role, last_login_at, created_at, updated_at
		FROM users WHERE id = ?`

	var u User
	var lastLogin sql.NullTime
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Role,
		&lastLogin, &u.CreatedAt, &u.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}
	if lastLogin.Valid {
		u.LastLoginAt = &lastLogin.Time
	}
	return &u, nil
}

// Create inserts a new user.
func (r *Repository) Create(ctx context.Context, u *User) error {
	now := time.Now()
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO users (username, email, password_hash, role, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		u.Username, u.Email, u.PasswordHash, u.Role, now, now,
	)
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	id, _ := result.LastInsertId()
	u.ID = id
	u.CreatedAt = now
	u.UpdatedAt = now
	return nil
}

// UpdateLastLogin updates the last login time.
func (r *Repository) UpdateLastLogin(ctx context.Context, id int64) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx, "UPDATE users SET last_login_at = ?, updated_at = ? WHERE id = ?", now, now, id)
	return err
}

// HasUsers checks if any users exist in the database.
func (r *Repository) HasUsers(ctx context.Context) (bool, error) {
	var count int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

// ErrInvalidCredentials is returned for invalid login attempts.
var ErrInvalidCredentials = fmt.Errorf("invalid credentials")
