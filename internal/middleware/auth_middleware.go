package middleware

import (
	"expo-open-ota/internal/auth"
	"expo-open-ota/internal/helpers"
	"net/http"
)

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try EOAS API key auth first
		eoasAuth := helpers.GetEoasAuth(r)
		err := auth.ValidateEOASAuth(&eoasAuth)
		if err == nil {
			next.ServeHTTP(w, r)
			return
		}

		// Fall back to JWT dashboard auth
		bearerToken, err := helpers.GetBearerToken(r)
		if err != nil {
			http.Error(w, "Invalid Authorization header format", http.StatusUnauthorized)
			return
		}
		if bearerToken == "" {
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
