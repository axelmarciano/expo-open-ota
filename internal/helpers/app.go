package helpers

import "net/http"

// appIDCtxKey is the typed context key used to carry the resolved APP_ID
// across handlers. Typed so it cannot collide with another package's
// string-keyed context value.
type appIDCtxKey struct{}

// AppIDContextKey is the sentinel used by AppResolverMiddleware and GetAppID.
var AppIDContextKey = appIDCtxKey{}

// GetAppID returns the APP_ID resolved by AppResolverMiddleware, or "" if
// the request did not pass through it.
func GetAppID(r *http.Request) string {
	v, _ := r.Context().Value(AppIDContextKey).(string)
	return v
}
