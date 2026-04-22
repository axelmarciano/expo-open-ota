package middleware

import (
	"expo-open-ota/internal/auth"
	"expo-open-ota/internal/helpers"
	"expo-open-ota/internal/services"
	"net/http"

	"github.com/gorilla/mux"
)

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		useExpoAuth := r.Header.Get("Use-Expo-Auth")
		if useExpoAuth == "true" {
			// Expo-relayed session requires an appId to know which app's
			// EXPO_ACCESS_TOKEN to validate against. On /api/settings and
			// other app-agnostic routes there is no APP_ID in the path and
			// Use-Expo-Auth doesn't make sense.
			appId := mux.Vars(r)["APP_ID"]
			if appId == "" {
				http.Error(w, "Use-Expo-Auth requires an app-scoped route", http.StatusUnauthorized)
				return
			}
			expoAuth := helpers.GetExpoAuth(r)
			_, err := services.ValidateExpoAuth(appId, expoAuth)
			if err != nil {
				http.Error(w, "Invalid Expo auth", http.StatusUnauthorized)
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
		authService := auth.NewAuth()
		_, err = authService.ValidateToken(bearerToken)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)

	})
}
