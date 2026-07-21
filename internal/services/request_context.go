package services

import "context"

// The request-identity context keys live here, next to the types and services
// that produce them, not in the HTTP middleware that stamps them: the
// middleware is plumbing, the identity is this package's domain. It also
// keeps the keys readable from any service (the audit emissions resolve
// their actor this way); middleware imports services, so the reverse import
// these helpers would otherwise force is a cycle.

type principalContextKey struct{}

// PrincipalFromContext returns the dashboard principal stored by the auth
// middleware, or nil when the request was authenticated another way (CLI
// credential) or not at all.
func PrincipalFromContext(ctx context.Context) *DashboardPrincipal {
	principal, _ := ctx.Value(principalContextKey{}).(*DashboardPrincipal)
	return principal
}

// WithPrincipal stores a dashboard principal on the context.
func WithPrincipal(ctx context.Context, principal *DashboardPrincipal) context.Context {
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
