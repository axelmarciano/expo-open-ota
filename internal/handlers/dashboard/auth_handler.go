package handlers

import (
	"errors"
	"expo-open-ota/internal/dashboard"
	"expo-open-ota/internal/handlers"
	"expo-open-ota/internal/services"
	"net/http"
)

type AuthHandler struct {
	dashboardAuthService *services.DashboardAuthService
}

func NewAuthHandler(dashboardAuthService *services.DashboardAuthService) *AuthHandler {
	return &AuthHandler{
		dashboardAuthService: dashboardAuthService,
	}
}

func (ah *AuthHandler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	dashboardEnabled := dashboard.IsDashboardEnabled()
	if !dashboardEnabled {
		handlers.RenderError(w, http.StatusNotFound, "Dashboard is disabled")
		return
	}
	email := r.FormValue("email")
	if email == "" {
		handlers.RenderError(w, http.StatusBadRequest, "Email is empty")
		return
	}
	password := r.FormValue("password")
	if password == "" {
		handlers.RenderError(w, http.StatusBadRequest, "Password is empty")
		return
	}
	session, err := ah.dashboardAuthService.LoginWithEmailPassword(r.Context(), email, password)
	if err != nil {
		// A missing ADMIN_EMAIL is the operator's misconfiguration, not the
		// user's bad credential — surface the instruction instead of a 401.
		if errors.Is(err, services.ErrAdminEmailNotSet) {
			handlers.RenderError(w, http.StatusInternalServerError, err.Error())
			return
		}
		handlers.RenderError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"token":"` + session.Token + `","refreshToken":"` + session.RefreshToken + `"}`))
	w.WriteHeader(http.StatusOK)
}

func (ah *AuthHandler) RefreshTokenHandler(w http.ResponseWriter, r *http.Request) {
	dashboardEnabled := dashboard.IsDashboardEnabled()
	if !dashboardEnabled {
		handlers.RenderError(w, http.StatusNotFound, "Dashboard is disabled")
		return
	}
	refreshToken := r.FormValue("refreshToken")
	if refreshToken == "" {
		handlers.RenderError(w, http.StatusBadRequest, "Refresh token is empty")
		return
	}
	session, err := ah.dashboardAuthService.RefreshSession(r.Context(), refreshToken)
	if err != nil {
		handlers.RenderError(w, http.StatusUnauthorized, "Error refreshing token")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"token":"` + session.Token + `","refreshToken":"` + session.RefreshToken + `"}`))
	w.WriteHeader(http.StatusOK)
}
