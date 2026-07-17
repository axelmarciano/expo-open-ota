package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"expo-open-ota/internal/services"

	"github.com/gorilla/mux"
)

// runAuthMiddleware sends a request through NewAuthMiddleware on a
// non-app-scoped route (no APP_ID path variable), so the CLI-auth branch can
// only ever reject — cliAuthService is never reached and stays nil-backed.
func runAuthMiddleware(t *testing.T, configure func(r *http.Request)) *httptest.ResponseRecorder {
	t.Helper()
	router := mux.NewRouter()
	router.Use(NewAuthMiddleware(services.NewDashboardAuthService(), services.NewCliAuthService(nil)))
	router.HandleFunc("/settings", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	r := httptest.NewRequest("GET", "/settings", nil)
	configure(r)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	return w
}

func TestAuthMiddleware(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("ADMIN_PASSWORD", "test-password")

	t.Run("cli auth rejected on a non-app-scoped route", func(t *testing.T) {
		w := runAuthMiddleware(t, func(r *http.Request) {
			r.Header.Set("Use-Cli-Auth", "true")
			r.Header.Set("Authorization", "Bearer some-cli-credential")
		})
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 for cli auth without app context, got %d", w.Code)
		}
	})

	t.Run("valid dashboard session is accepted", func(t *testing.T) {
		session, err := services.NewDashboardAuthService().LoginWithPassword("test-password")
		if err != nil {
			t.Fatalf("login failed: %v", err)
		}
		w := runAuthMiddleware(t, func(r *http.Request) {
			r.Header.Set("Authorization", "Bearer "+session.Token)
		})
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 with valid session token, got %d", w.Code)
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
