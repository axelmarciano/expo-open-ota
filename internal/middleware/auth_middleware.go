package middleware

import (
	"expo-open-ota/internal/helpers"
	"expo-open-ota/internal/services"
	"net/http"

	"github.com/gorilla/mux"
)

func NewAuthMiddleware(authService *services.AuthService) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			useCliAuth := r.Header.Get("Use-Cli-Auth")
			if useCliAuth == "true" {
				// CLI-driven external authentication requires an APP_ID path variable
				// to locate the correct tenant boundary.
				// - In DB Mode: Used to check the api_keys table for app-scoped access.
				// - In Stateless Mode: Relayed to select the correct EXPO_ACCESS_TOKEN.
				// On global or app-agnostic routes (like /api/settings or /api/apps),
				// there is no app context anchor, making Use-Cli-Auth invalid.
				appId := mux.Vars(r)["APP_ID"]
				if appId == "" {
					http.Error(w, "Use-Cli-Auth requires an app-scoped route", http.StatusUnauthorized)
					return
				}

				auth := helpers.GetAuth(r)
				err := authService.ValidateAuth(r.Context(), appId, auth)
				if err != nil {
					http.Error(w, "Error validating auth", http.StatusUnauthorized)
					return
				}

				next.ServeHTTP(w, r)
				return
			}
			bearerToken, err := helpers.GetBearerToken(r)
			if err != nil {
				http.Error(w, "Invalid Authorization header format", http.StatusUnauthorized)
				return
			}
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "No Authorization header provided", http.StatusUnauthorized)
				return
			}
			_, err = authService.ValidateToken(bearerToken)
			if err != nil {
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)

		})
	}
}
