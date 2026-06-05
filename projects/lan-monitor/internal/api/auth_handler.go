package api

import (
	"database/sql"
	"net/http"
	"time"

	"lan-monitor/internal/auth"
	"lan-monitor/internal/config"
	"lan-monitor/internal/database"

	"github.com/gin-gonic/gin"
)

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token"`
	ExpiresIn    int64    `json:"expires_in"`
	User         UserInfo `json:"user"`
}

type UserInfo struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

func handleLogin(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入用户名和密码"})
		return
	}

	// Rate limiting check
	var recentFailures int64
	database.DB.QueryRow(
		"SELECT COUNT(*) FROM audit_logs WHERE action = ? AND ip = ? AND created_at > ?",
		"POST /api/v1/auth/login", c.ClientIP(), time.Now().Add(-5*time.Minute),
	).Scan(&recentFailures)

	if recentFailures >= 10 {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "登录尝试过于频繁，请5分钟后再试"})
		return
	}

	var userID uint
	var username, passwordHash, email, role, status string
	err := database.DB.QueryRow(
		"SELECT id, username, password_hash, email, role, status FROM users WHERE username = ?",
		req.Username,
	).Scan(&userID, &username, &passwordHash, &email, &role, &status)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	if status != "active" {
		c.JSON(http.StatusForbidden, gin.H{"error": "账户已被禁用"})
		return
	}

	if !auth.CheckPassword(req.Password, passwordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}

	accessToken, err := auth.GenerateAccessToken(userID, username, role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成令牌失败"})
		return
	}

	refreshToken, err := auth.GenerateRefreshToken(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成刷新令牌失败"})
		return
	}

	cfg := config.AppConfig
	c.JSON(http.StatusOK, LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(cfg.JWT.AccessTokenExpireMinutes * 60),
		User: UserInfo{
			ID:       userID,
			Username: username,
			Email:    email,
			Role:     role,
		},
	})
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func handleRefresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供刷新令牌"})
		return
	}

	claims, err := auth.ParseToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "刷新令牌无效或已过期"})
		return
	}

	var username, role string
	err = database.DB.QueryRow(
		"SELECT username, role FROM users WHERE id = ?",
		claims.UserID,
	).Scan(&username, &role)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户不存在"})
		return
	}

	accessToken, err := auth.GenerateAccessToken(claims.UserID, username, role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成令牌失败"})
		return
	}

	cfg := config.AppConfig
	c.JSON(http.StatusOK, gin.H{
		"access_token": accessToken,
		"expires_in":   int64(cfg.JWT.AccessTokenExpireMinutes * 60),
	})
}

func handleProfile(c *gin.Context) {
	userID := c.GetUint("user_id")
	var id uint
	var username, email, role string
	err := database.DB.QueryRow(
		"SELECT id, username, email, role FROM users WHERE id = ?", userID,
	).Scan(&id, &username, &email, &role)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	c.JSON(http.StatusOK, UserInfo{
		ID:       id,
		Username: username,
		Email:    email,
		Role:     role,
	})
}

func handleLogout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "已登出"})
}
