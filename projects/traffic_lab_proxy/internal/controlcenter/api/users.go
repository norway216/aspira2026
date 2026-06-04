package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/aspira2026/traffic_lab_proxy/internal/common"
	"github.com/aspira2026/traffic_lab_proxy/internal/controlcenter/db"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type UserHandler struct {
	DB db.DB
}

func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req common.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, common.APIResponse{Success: false, Error: "invalid request body"})
		return
	}

	if req.Username == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, common.APIResponse{Success: false, Error: "username and password are required"})
		return
	}

	// Check existing
	existing, _ := h.DB.GetUserByUsername(r.Context(), req.Username)
	if existing != nil {
		writeJSON(w, http.StatusConflict, common.APIResponse{Success: false, Error: "username already exists"})
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, common.APIResponse{Success: false, Error: "failed to hash password"})
		return
	}

	token := uuid.New().String()
	if req.MaxRateMbps <= 0 {
		req.MaxRateMbps = 10
	}
	if req.MaxConnections <= 0 {
		req.MaxConnections = 5
	}

	user := &common.User{
		Username:       req.Username,
		PasswordHash:   string(hashed),
		Token:          token,
		Status:         common.UserStatusActive,
		TrafficTotal:   req.TrafficTotal,
		MaxRateMbps:    req.MaxRateMbps,
		MaxConnections: req.MaxConnections,
	}

	if err := h.DB.CreateUser(r.Context(), user); err != nil {
		writeJSON(w, http.StatusInternalServerError, common.APIResponse{Success: false, Error: "failed to create user: " + err.Error()})
		return
	}

	// Don't expose password hash
	user.PasswordHash = ""
	writeJSON(w, http.StatusCreated, common.APIResponse{
		Success: true,
		Message: "user created",
		Data:    user,
	})
}

func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	users, err := h.DB.ListUsers(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, common.APIResponse{Success: false, Error: err.Error()})
		return
	}

	// Strip password hashes
	for _, u := range users {
		u.PasswordHash = ""
	}

	writeJSON(w, http.StatusOK, common.APIResponse{
		Success: true,
		Data:    users,
	})
}

func (h *UserHandler) Get(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, common.APIResponse{Success: false, Error: "invalid user id"})
		return
	}

	user, err := h.DB.GetUser(r.Context(), id)
	if err != nil {
		if ae, ok := err.(*common.AppError); ok {
			writeJSON(w, ae.Code, common.APIResponse{Success: false, Error: ae.Message})
			return
		}
		writeJSON(w, http.StatusInternalServerError, common.APIResponse{Success: false, Error: err.Error()})
		return
	}

	user.PasswordHash = ""
	writeJSON(w, http.StatusOK, common.APIResponse{Success: true, Data: user})
}

func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, common.APIResponse{Success: false, Error: "invalid user id"})
		return
	}

	user, err := h.DB.GetUser(r.Context(), id)
	if err != nil {
		if ae, ok := err.(*common.AppError); ok {
			writeJSON(w, ae.Code, common.APIResponse{Success: false, Error: ae.Message})
			return
		}
		writeJSON(w, http.StatusInternalServerError, common.APIResponse{Success: false, Error: err.Error()})
		return
	}

	var req common.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, common.APIResponse{Success: false, Error: "invalid request body"})
		return
	}

	if req.Status != nil {
		user.Status = *req.Status
	}
	if req.MaxRateMbps != nil {
		user.MaxRateMbps = *req.MaxRateMbps
	}
	if req.MaxConnections != nil {
		user.MaxConnections = *req.MaxConnections
	}
	if req.TrafficTotal != nil {
		user.TrafficTotal = *req.TrafficTotal
	}
	if req.Password != nil && *req.Password != "" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, common.APIResponse{Success: false, Error: "failed to hash password"})
			return
		}
		user.PasswordHash = string(hashed)
	}

	if err := h.DB.UpdateUser(r.Context(), user); err != nil {
		writeJSON(w, http.StatusInternalServerError, common.APIResponse{Success: false, Error: err.Error()})
		return
	}

	user.PasswordHash = ""
	writeJSON(w, http.StatusOK, common.APIResponse{Success: true, Data: user})
}

func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, common.APIResponse{Success: false, Error: "invalid user id"})
		return
	}

	if err := h.DB.DeleteUser(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, common.APIResponse{Success: false, Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, common.APIResponse{Success: true, Message: "user deleted"})
}
