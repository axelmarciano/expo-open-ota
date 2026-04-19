package middleware

import (
	"context"
	"expo-open-ota/internal/helpers"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

// AppResolverMiddleware extracts APP_ID from the path vars, runs format
// validation (non-empty, no "..", no path separators), and stores it in the
// request context for downstream handlers to pick up via helpers.GetAppID.
//
// Validation is permissive by design — any well-formed app id is accepted.
// Per-app authorization is expected to happen further down (via EAS token
// validation). Add an allowlist here if stricter isolation is needed.
func AppResolverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		appId := mux.Vars(r)["APP_ID"]
		if !isValidAppID(appId) {
			http.Error(w, "invalid app id", http.StatusBadRequest)
			return
		}
		ctx := context.WithValue(r.Context(), helpers.AppIDContextKey, appId)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func isValidAppID(id string) bool {
	if id == "" || id == "." || id == ".." {
		return false
	}
	if strings.ContainsAny(id, "/\\") {
		return false
	}
	return true
}
