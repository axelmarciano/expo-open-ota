package handlers

import (
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
	password := r.FormValue("password")
	if password == "" {
		handlers.RenderError(w, http.StatusBadRequest, "Password is empty")
		return
	}
	session, err := ah.dashboardAuthService.LoginWithPassword(password)
	if err != nil {
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
	session, err := ah.dashboardAuthService.RefreshSession(refreshToken)
	if err != nil {
		handlers.RenderError(w, http.StatusInternalServerError, "Error refreshing token")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"token":"` + session.Token + `","refreshToken":"` + session.RefreshToken + `"}`))
	w.WriteHeader(http.StatusOK)
}
