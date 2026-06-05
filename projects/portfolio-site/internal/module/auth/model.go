package auth

import "time"

// Role represents a user role.
type Role string

const (
	RoleAdmin  Role = "admin"
	RoleEditor Role = "editor"
)

// User represents an administrator user.
type User struct {
	ID           int64      `json:"id"`
	Username     string     `json:"username"`
	Email        string     `json:"email"`
	PasswordHash string     `json:"-"`
	Role         Role       `json:"role"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// LoginInput represents a login request.
type LoginInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// CreateUserInput is used for creating a new user.
type CreateUserInput struct {
	Username string
	Email    string
	Password string
	Role     Role
}
