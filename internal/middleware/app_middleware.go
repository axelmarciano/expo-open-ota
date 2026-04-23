package middleware

import (
	"context"
	"expo-open-ota/config"
	"expo-open-ota/internal/helpers"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

// AppResolverMiddleware extracts APP_ID from the path vars, validates it
// (well-formed AND registered in the apps config), and stores it in the
// request context for downstream handlers to pick up via helpers.GetAppID.
//
// The registry check matches the manifest/assets handlers' edge behavior:
// unknown app ids return 404 here, instead of falling through to handlers
// that try to validate the request against api.expo.dev with no token and
// surface that as a misleading 401.
func AppResolverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		appId := mux.Vars(r)["APP_ID"]
		if !isValidAppID(appId) {
			http.Error(w, "invalid app id", http.StatusBadRequest)
			return
		}
		if _, err := config.GetAppConfig(appId); err != nil {
			http.Error(w, "Unknown app id", http.StatusNotFound)
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
