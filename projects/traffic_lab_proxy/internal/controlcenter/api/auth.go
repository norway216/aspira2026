package api

import (
	"encoding/json"
	"net/http"

	"github.com/aspira2026/traffic_lab_proxy/internal/common"
)

type AuthHandler struct {
	AdminUser string
	AdminPass string
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req common.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, common.APIResponse{
			Success: false,
			Error:   "invalid request body",
		})
		return
	}

	if req.Username != h.AdminUser || req.Password != h.AdminPass {
		writeJSON(w, http.StatusUnauthorized, common.APIResponse{
			Success: false,
			Error:   "invalid credentials",
		})
		return
	}

	// Set session cookie
	sessionValue := encodeSessionCookie(req.Username)
	http.SetCookie(w, &http.Cookie{
		Name:     "admin_session",
		Value:    sessionValue,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400, // 24 hours
	})

	writeJSON(w, http.StatusOK, common.APIResponse{
		Success: true,
		Message: "login successful",
		Data: map[string]string{
			"username": req.Username,
		},
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "admin_session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
	writeJSON(w, http.StatusOK, common.APIResponse{
		Success: true,
		Message: "logout successful",
	})
}
