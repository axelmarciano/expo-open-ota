package middleware

import (
	"expo-open-ota/config"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

// AppResolverMiddleware extracts APP_ID from the path vars and validates it
// (well-formed AND registered in the apps config). Downstream handlers read
// the id back from mux.Vars themselves — the middleware's job is just to
// short-circuit bad/unknown ids before they reach the handler.
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
		next.ServeHTTP(w, r)
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
