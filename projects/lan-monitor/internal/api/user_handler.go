package api

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"lan-monitor/internal/auth"
	"lan-monitor/internal/database"

	"github.com/gin-gonic/gin"
)

func handleUserList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	var total int64
	database.DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&total)

	offset := (page - 1) * size
	rows, _ := database.DB.Query(
		"SELECT id, username, email, role, status, created_at, updated_at FROM users ORDER BY id ASC LIMIT ? OFFSET ?",
		size, offset)
	if rows != nil {
		defer rows.Close()
	}

	type UR struct {
		ID        uint      `json:"id"`
		Username  string    `json:"username"`
		Email     string    `json:"email"`
		Role      string    `json:"role"`
		Status    string    `json:"status"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
	}
	var users []UR
	if rows != nil {
		for rows.Next() {
			var u UR
			rows.Scan(&u.ID, &u.Username, &u.Email, &u.Role, &u.Status, &u.CreatedAt, &u.UpdatedAt)
			users = append(users, u)
		}
	}
	if users == nil {
		users = []UR{}
	}
	c.JSON(http.StatusOK, gin.H{"data": users, "total": total, "page": page})
}

type CreateUserRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

func handleUserCreate(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入用户名和密码"})
		return
	}
	if len(req.Username) < 2 || len(req.Username) > 64 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户名长度应为2-64个字符"})
		return
	}
	if len(req.Password) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "密码长度至少6位"})
		return
	}

	var existing int64
	database.DB.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", req.Username).Scan(&existing)
	if existing > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "用户名已存在"})
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
		return
	}

	role := req.Role
	if role == "" {
		role = "viewer"
	}

	result, err := database.DB.Exec(
		"INSERT INTO users (username, password_hash, email, role, status) VALUES (?, ?, ?, ?, 'active')",
		req.Username, hash, req.Email, role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建用户失败"})
		return
	}
	uid, _ := result.LastInsertId()

	c.JSON(http.StatusCreated, gin.H{
		"message": "用户创建成功",
		"user":    gin.H{"id": uid, "username": req.Username, "email": req.Email, "role": role},
	})
}

type UpdateUserRequest struct {
	Email    *string `json:"email"`
	Role     *string `json:"role"`
	Status   *string `json:"status"`
	Password *string `json:"password"`
}

func handleUserUpdate(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var role, status string
	err := database.DB.QueryRow("SELECT role, status FROM users WHERE id = ?", id).Scan(&role, &status)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	if role == "super_admin" && c.GetString("role") != "super_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权限修改超级管理员"})
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求数据格式错误"})
		return
	}

	if req.Email != nil {
		database.DB.Exec("UPDATE users SET email = ? WHERE id = ?", *req.Email, id)
	}
	if req.Role != nil {
		database.DB.Exec("UPDATE users SET role = ? WHERE id = ?", *req.Role, id)
	}
	if req.Status != nil {
		database.DB.Exec("UPDATE users SET status = ? WHERE id = ?", *req.Status, id)
	}
	if req.Password != nil {
		if len(*req.Password) < 6 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "密码长度至少6位"})
			return
		}
		hash, _ := auth.HashPassword(*req.Password)
		database.DB.Exec("UPDATE users SET password_hash = ? WHERE id = ?", hash, id)
	}

	c.JSON(http.StatusOK, gin.H{"message": "用户已更新"})
}

func handleUserDelete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if uint(id) == c.GetUint("user_id") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不能删除自己"})
		return
	}
	database.DB.Exec("DELETE FROM users WHERE id = ?", id)
	c.JSON(http.StatusOK, gin.H{"message": "用户已删除"})
}

func handleRoleList(c *gin.Context) {
	rows, _ := database.DB.Query("SELECT id, name, description, created_at FROM roles")
	if rows != nil {
		defer rows.Close()
	}
	type Role struct {
		ID          uint      `json:"id"`
		Name        string    `json:"name"`
		Description string    `json:"description"`
		CreatedAt   time.Time `json:"created_at"`
	}
	var roles []Role
	if rows != nil {
		for rows.Next() {
			var r Role
			rows.Scan(&r.ID, &r.Name, &r.Description, &r.CreatedAt)
			roles = append(roles, r)
		}
	}
	if roles == nil {
		roles = []Role{}
	}
	c.JSON(http.StatusOK, gin.H{"data": roles})
}
