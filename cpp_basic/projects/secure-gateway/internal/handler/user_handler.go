package handler

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/secure-gateway/internal/model"
	"github.com/secure-gateway/pkg/logger"
	"golang.org/x/crypto/bcrypt"
)

// UserHandler handles user management endpoints.
type UserHandler struct {
	db *sql.DB
}

// NewUserHandler creates a new user handler.
func NewUserHandler(db *sql.DB) *UserHandler {
	return &UserHandler{db: db}
}

// ListUsers returns all users.
func (h *UserHandler) ListUsers(c *gin.Context) {
	rows, err := h.db.Query(
		"SELECT id, username, role, status, traffic_quota_bytes, traffic_used_bytes, created_at, updated_at FROM users ORDER BY id",
	)
	if err != nil {
		logger.Error("Failed to list users", "error", err)
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: "Internal server error"})
		return
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.Username, &u.Role, &u.Status, &u.TrafficQuotaBytes, &u.TrafficUsedBytes, &u.CreatedAt, &u.UpdatedAt); err != nil {
			logger.Error("Failed to scan user row", "error", err)
			continue
		}
		users = append(users, u)
	}

	c.JSON(http.StatusOK, model.APIResponse{Code: 200, Message: "Success", Data: users})
}

// CreateUser creates a new user.
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req model.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: "Invalid request: " + err.Error()})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: "Internal server error"})
		return
	}

	role := req.Role
	if role == "" {
		role = model.RoleUser
	}

	var u model.User
	err = h.db.QueryRow(
		`INSERT INTO users (username, password_hash, role, traffic_quota_bytes)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, username, role, status, traffic_quota_bytes, traffic_used_bytes, created_at, updated_at`,
		req.Username, string(hash), role, req.Quota,
	).Scan(&u.ID, &u.Username, &u.Role, &u.Status, &u.TrafficQuotaBytes, &u.TrafficUsedBytes, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		logger.Error("Failed to create user", "error", err)
		c.JSON(http.StatusConflict, model.APIResponse{Code: 409, Message: "Username already exists"})
		return
	}

	logger.Info("User created", "user", u.Username, "role", u.Role)
	c.JSON(http.StatusCreated, model.APIResponse{Code: 201, Message: "User created", Data: u})
}

// UpdateUser updates an existing user.
func (h *UserHandler) UpdateUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: "Invalid user ID"})
		return
	}

	var req model.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: "Invalid request"})
		return
	}

	// Build update query dynamically
	if req.Password != nil && *req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: "Internal server error"})
			return
		}
		_, err = h.db.Exec("UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2", string(hash), id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: "Update failed"})
			return
		}
	}
	if req.Role != nil {
		_, err = h.db.Exec("UPDATE users SET role = $1, updated_at = NOW() WHERE id = $2", *req.Role, id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: "Update failed"})
			return
		}
	}
	if req.Status != nil {
		_, err = h.db.Exec("UPDATE users SET status = $1, updated_at = NOW() WHERE id = $2", *req.Status, id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: "Update failed"})
			return
		}
	}
	if req.Quota != nil {
		_, err = h.db.Exec("UPDATE users SET traffic_quota_bytes = $1, updated_at = NOW() WHERE id = $2", *req.Quota, id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: "Update failed"})
			return
		}
	}

	logger.Info("User updated", "user_id", id)
	c.JSON(http.StatusOK, model.APIResponse{Code: 200, Message: "User updated"})
}

// DeleteUser deletes a user.
func (h *UserHandler) DeleteUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: "Invalid user ID"})
		return
	}

	result, err := h.db.Exec("DELETE FROM users WHERE id = $1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: "Delete failed"})
		return
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		c.JSON(http.StatusNotFound, model.APIResponse{Code: 404, Message: "User not found"})
		return
	}

	logger.Info("User deleted", "user_id", id)
	c.JSON(http.StatusOK, model.APIResponse{Code: 200, Message: "User deleted"})
}