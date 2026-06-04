package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/embedded-ops-platform/backend/internal/model"
	"github.com/embedded-ops-platform/backend/internal/repository"
	"github.com/embedded-ops-platform/backend/pkg/logger"
	"github.com/embedded-ops-platform/backend/pkg/token"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// AuthAPI handles authentication endpoints.
type AuthAPI struct {
	db     *repository.PostgresRepo
	tokens *token.Manager
}

// NewAuthAPI creates a new auth API handler.
func NewAuthAPI(db *repository.PostgresRepo, tokens *token.Manager) *AuthAPI {
	return &AuthAPI{db: db, tokens: tokens}
}

// Login handles user login.
func (a *AuthAPI) Login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	user, err := a.db.GetUserByUsername(ctx, req.Username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	if !user.Enabled {
		c.JSON(http.StatusForbidden, gin.H{"error": "account disabled"})
		return
	}

	// Verify password using bcrypt
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	userIDStr := fmt.Sprintf("%d", user.ID)

	accessToken, err := a.tokens.GenerateAccessToken(
		userIDStr,
		user.Username,
		user.Role,
	)
	if err != nil {
		logger.Error("Failed to generate access token", logger.ErrField(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	refreshToken, err := a.tokens.GenerateRefreshToken(userIDStr)
	if err != nil {
		logger.Error("Failed to generate refresh token", logger.ErrField(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, model.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         *user,
	})
}

// Me returns the current user's info.
func (a *AuthAPI) Me(c *gin.Context) {
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")
	role, _ := c.Get("user_role")

	c.JSON(http.StatusOK, gin.H{
		"user_id":  userID,
		"username": username,
		"role":     role,
	})
}

// RegisterRoutes
func (a *AuthAPI) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/auth/login", a.Login)
	rg.GET("/auth/me", a.Me)
}