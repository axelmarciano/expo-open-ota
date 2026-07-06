package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"expo-open-ota/internal/auth"
)

func runAuthMiddleware(t *testing.T, configure func(r *http.Request)) *httptest.ResponseRecorder {
	t.Helper()
	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	r := httptest.NewRequest("POST", "/requestUploadUrl/branch", nil)
	configure(r)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	return w
}

func TestAuthMiddleware(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("ADMIN_PASSWORD", "test-password")

	t.Run("expo auth rejected when EXPO_ACCESS_TOKEN is not configured", func(t *testing.T) {
		t.Setenv("EXPO_ACCESS_TOKEN", "")
		w := runAuthMiddleware(t, func(r *http.Request) {
			r.Header.Set("Use-Expo-Auth", "true")
			r.Header.Set("Authorization", "Bearer some-expo-token")
		})
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 when expo auth is not enabled, got %d", w.Code)
		}
	})

	t.Run("valid admin JWT is accepted", func(t *testing.T) {
		resp, err := auth.NewAuth().LoginWithPassword("test-password")
		if err != nil {
			t.Fatalf("login failed: %v", err)
		}
		w := runAuthMiddleware(t, func(r *http.Request) {
			r.Header.Set("Authorization", "Bearer "+resp.Token)
		})
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 with valid JWT, got %d", w.Code)
		}
	})

	t.Run("invalid bearer token is rejected", func(t *testing.T) {
		w := runAuthMiddleware(t, func(r *http.Request) {
			r.Header.Set("Authorization", "Bearer not-a-jwt")
		})
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 with invalid token, got %d", w.Code)
		}
	})

	t.Run("missing Authorization header is rejected", func(t *testing.T) {
		w := runAuthMiddleware(t, func(r *http.Request) {})
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 with no Authorization header, got %d", w.Code)
		}
	})
}
