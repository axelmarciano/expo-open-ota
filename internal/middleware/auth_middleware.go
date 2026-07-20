package middleware

import (
	"context"
	"errors"
	"expo-open-ota/internal/handlers"
	"expo-open-ota/internal/helpers"
	"expo-open-ota/internal/services"
	"expo-open-ota/internal/store"
	"net/http"

	"github.com/gorilla/mux"
)

type principalContextKey struct{}

// PrincipalFromContext returns the dashboard principal stored by
// NewAuthMiddleware, or nil when the request was authenticated another way
// (CLI credential) or not at all.
func PrincipalFromContext(ctx context.Context) *services.DashboardPrincipal {
	principal, _ := ctx.Value(principalContextKey{}).(*services.DashboardPrincipal)
	return principal
}

// WithPrincipal stores a dashboard principal on the context.
func WithPrincipal(ctx context.Context, principal *services.DashboardPrincipal) context.Context {
	return context.WithValue(ctx, principalContextKey{}, principal)
}

type cliAuthContextKey struct{}

// WithCliAuth marks the request as authenticated by an app-scoped CLI
// credential. The marker exists so downstream gates can assert "validated CLI
// request" as a fact instead of inferring it from the absence of a dashboard
// principal, which would silently fail open on a route someone mounts without
// the auth middleware.
func WithCliAuth(ctx context.Context, appId string) context.Context {
	return context.WithValue(ctx, cliAuthContextKey{}, appId)
}

// CliAuthAppFromContext returns the app the CLI credential was validated for,
// or "" when the request did not authenticate through the CLI path.
func CliAuthAppFromContext(ctx context.Context) string {
	appId, _ := ctx.Value(cliAuthContextKey{}).(string)
	return appId
}

// NewAuthMiddleware guards a route with one of two unrelated credentials,
// picked by the Use-Cli-Auth header:
//   - "true": a CLI credential scoped to an app (an eoo_ API key in DB mode, an
//     Expo token/session in stateless mode) -> cliAuthService.
//   - otherwise: the dashboard's own session JWT -> dashboardAuthService. The
//     resolved principal is stored on the request context for downstream
//     handlers and the admin gate.
//
// Both travel as `Authorization: Bearer …`, which is why the header decides
// which one to expect rather than the credential's shape.
func NewAuthMiddleware(dashboardAuthService *services.DashboardAuthService, cliAuthService *services.CliAuthService) mux.MiddlewareFunc {
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
				// These routes are branch-less reads: only the IP allowlist
				// applies here, branch protection is enforced on the publish
				// routes that carry a BRANCH.
				err := cliAuthService.ValidateCliCredential(r.Context(), appId, auth, "", helpers.ClientIP(r))
				if err != nil {
					handlers.RenderCliAuthError(w, err)
					return
				}

				next.ServeHTTP(w, r.WithContext(WithCliAuth(r.Context(), appId)))
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
			principal, err := dashboardAuthService.ValidateSession(bearerToken)
			if err != nil {
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r.WithContext(WithPrincipal(r.Context(), principal)))
		})
	}
}

// NewAdminMiddleware guards a route behind the account-level admin flag. It
// only accepts dashboard sessions — a CLI credential is app-scoped publishing
// access, not an account, so it never reaches admin-gated routes.
//
// The flag is re-read from the users table on every call rather than trusted
// from the JWT: a session token lives 2 hours, and a revoked admin (or deleted
// user) must lose these routes immediately, not at the next refresh. userRepo
// is nil in stateless mode, where the single ADMIN_EMAIL account is always an
// admin and the claim alone is authoritative.
func NewAdminMiddleware(userRepo services.UserRepository) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal := PrincipalFromContext(r.Context())
			if principal == nil {
				http.Error(w, "This action requires a dashboard session", http.StatusForbidden)
				return
			}
			if userRepo != nil {
				user, err := userRepo.GetUserByID(r.Context(), principal.UserId)
				if err != nil {
					// Only a missing row means the account is gone; an
					// infrastructure failure must not read as a dead session.
					if notFoundErr := (*store.ErrResourceNotFound)(nil); errors.As(err, &notFoundErr) {
						http.Error(w, "Invalid token", http.StatusUnauthorized)
					} else {
						http.Error(w, "Could not verify the account", http.StatusInternalServerError)
					}
					return
				}
				if !user.IsAdmin {
					http.Error(w, "This action requires an admin account", http.StatusForbidden)
					return
				}
				next.ServeHTTP(w, r)
				return
			}
			if !principal.IsAdmin {
				http.Error(w, "This action requires an admin account", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
