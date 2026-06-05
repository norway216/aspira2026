package auth

import (
	"log/slog"
	"net/http"

	"portfolio-site/internal/middleware"
	"portfolio-site/internal/module/audit"
	"portfolio-site/internal/render"
)

// Handler handles authentication HTTP requests.
type Handler struct {
	svc      *Service
	auditSvc *audit.Service
	authMW   *middleware.Auth
	render   *render.Renderer
	logger   *slog.Logger
}

// NewHandler creates a new auth handler.
func NewHandler(svc *Service, auditSvc *audit.Service, authMW *middleware.Auth, render *render.Renderer, logger *slog.Logger) *Handler {
	return &Handler{
		svc:      svc,
		auditSvc: auditSvc,
		authMW:   authMW,
		render:   render,
		logger:   logger,
	}
}

// LoginPage displays the login form.
func (h *Handler) LoginPage(w http.ResponseWriter, r *http.Request) {
	if h.authMW.IsAuthenticated(r) {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}
	h.render.Render(w, "login", map[string]interface{}{
		"Title":       "Admin Login",
		"Description": "Login to admin panel",
	})
}

// Login processes the login form.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.render.ErrorPage(w, http.StatusBadRequest, "Invalid form data")
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	user, err := h.svc.Login(r.Context(), username, password)
	if err != nil {
		// Record failed login attempt.
		h.auditSvc.RecordFromRequest(r.Context(), r, audit.LogInput{
			ActorName:    username,
			Action:       audit.ActionLoginFailed,
			ResourceType: audit.ResourceUser,
			Status:       audit.StatusFailure,
			Detail:       audit.Logf("Failed login attempt for user: %s", username),
		})

		h.render.Render(w, "login", map[string]interface{}{
			"Title":       "Admin Login",
			"Description": "Login to admin panel",
			"Error":       "Invalid username or password",
		})
		return
	}

	// Set session.
	h.authMW.SetSession(w, user.ID, user.Username, string(user.Role), 86400, false, true)

	// Record successful login.
	h.auditSvc.RecordFromRequest(r.Context(), r, audit.LogInput{
		ActorID:      user.ID,
		ActorName:    user.Username,
		Action:       audit.ActionLogin,
		ResourceType: audit.ResourceUser,
		ResourceID:   user.ID,
		Status:       audit.StatusSuccess,
		Detail:       "User logged in successfully",
	})

	h.logger.Info("user logged in", "username", user.Username)

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

// Logout clears the session and redirects to login.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	h.authMW.ClearSession(w)

	h.logger.Info("user logged out")

	http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
}
