package auth

import (
	"context"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// Service handles authentication business logic.
type Service struct {
	repo *Repository
}

// NewService creates a new auth service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// Login authenticates a user with username and password.
func (s *Service) Login(ctx context.Context, username, password string) (*User, error) {
	user, err := s.repo.FindByUsername(ctx, username)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// Update last login time.
	if err := s.repo.UpdateLastLogin(ctx, user.ID); err != nil {
		return nil, fmt.Errorf("update last login: %w", err)
	}

	return user, nil
}

// CreateUser creates a new admin user with hashed password.
func (s *Service) CreateUser(ctx context.Context, input CreateUserInput) (*User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	role := input.Role
	if role == "" {
		role = RoleAdmin
	}

	user := &User{
		Username:     input.Username,
		Email:        input.Email,
		PasswordHash: string(hashedPassword),
		Role:         role,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

// GetByID returns a user by ID.
func (s *Service) GetByID(ctx context.Context, id int64) (*User, error) {
	return s.repo.FindByID(ctx, id)
}

// HasUsers checks if any admin users exist.
func (s *Service) HasUsers(ctx context.Context) (bool, error) {
	return s.repo.HasUsers(ctx)
}

// EnsureDefaultAdmin creates a default admin user if none exist.
func (s *Service) EnsureDefaultAdmin(ctx context.Context) error {
	hasUsers, err := s.repo.HasUsers(ctx)
	if err != nil {
		return err
	}
	if hasUsers {
		return nil
	}

	_, err = s.CreateUser(ctx, CreateUserInput{
		Username: "admin",
		Email:    "admin@aspira.dev",
		Password: "admin123",
		Role:     RoleAdmin,
	})
	return err
}
