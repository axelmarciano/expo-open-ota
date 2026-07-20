// Copyright (c) 2026 Axel Marciano (Mercure Technologies). All rights reserved.
// This file is governed by the Mercure Technologies Enterprise Edition License
// (see ee/LICENSE); it is NOT covered by the MIT license of this repository.

package rbac

import (
	"context"
	"errors"
	"expo-open-ota/internal/handlers"
	"expo-open-ota/internal/middleware"
	"expo-open-ota/internal/store"
	"net/http"

	"github.com/gorilla/mux"
)

// UserLookup is the one read the middlewares need from the users store.
// services.UserRepository satisfies it; keeping it narrow lets tests fake a
// single method instead of the whole repository. Nil in stateless mode, where
// the session claim is authoritative (the single ADMIN_EMAIL account).
type UserLookup interface {
	GetUserByID(ctx context.Context, id string) (store.User, error)
}

// resolveSubject authenticates the request as a dashboard account and
// resolves its admin flag from a fresh users-table read, exactly like the
// community admin gate: a session token lives 2 hours, and a revoked admin
// (or deleted user) must lose access immediately, not at the next refresh.
// On failure it writes the response and returns ok=false.
func resolveSubject(w http.ResponseWriter, r *http.Request, userLookup UserLookup) (Subject, bool) {
	principal := middleware.PrincipalFromContext(r.Context())
	if principal == nil {
		handlers.RenderError(w, http.StatusForbidden, "This action requires a dashboard session")
		return Subject{}, false
	}
	if userLookup == nil {
		// Stateless mode: the single ADMIN_EMAIL account is always an admin
		return Subject{UserID: principal.UserId, IsAdmin: principal.IsAdmin}, true
	}
	user, err := userLookup.GetUserByID(r.Context(), principal.UserId)
	if err != nil {
		// Only a missing row means the account is gone; an infrastructure
		// failure must not read as a dead session.
		if notFoundErr := (*store.ErrResourceNotFound)(nil); errors.As(err, &notFoundErr) {
			handlers.RenderError(w, http.StatusUnauthorized, "Invalid token")
		} else {
			handlers.RenderError(w, http.StatusInternalServerError, "Could not verify the account")
		}
		return Subject{}, false
	}
	return Subject{UserID: principal.UserId, IsAdmin: user.IsAdmin}, true
}

// RequirePermission guards one app-scoped dashboard mutation: admins pass,
// members need the permission on the route's APP_ID. It replaces the
// community adminOnly gate on these routes, and degrades to exactly its
// behavior when roles are not enforced (no control plane, no valid license):
// members get the same 403 an admin-only route gives them today.
func RequirePermission(service *RBACService, userLookup UserLookup, perm Permission) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			subject, ok := resolveSubject(w, r, userLookup)
			if !ok {
				return
			}
			appId := mux.Vars(r)["APP_ID"]
			if appId == "" {
				handlers.RenderError(w, http.StatusBadRequest, "invalid app id")
				return
			}
			if err := service.Authorize(r.Context(), subject, appId, perm); err != nil {
				renderAuthorizeError(w, err)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func renderAuthorizeError(w http.ResponseWriter, err error) {
	deniedErr := (*ErrPermissionDenied)(nil)
	switch {
	case errors.Is(err, ErrRequiresControlPlane), errors.Is(err, ErrRequiresValidLicense):
		// Community fallback: members are read-only, same refusal as the
		// admin-only gate so an expired license reads identically to today.
		handlers.RenderError(w, http.StatusForbidden, "This action requires an admin account")
	case errors.Is(err, ErrNoAppAccess):
		// Same body as the app resolver's 404: an app the member has no
		// grant on does not exist for them.
		handlers.RenderError(w, http.StatusNotFound, "app not found")
	case errors.As(err, &deniedErr):
		handlers.RenderError(w, http.StatusForbidden, deniedErr.Error())
	default:
		handlers.RenderError(w, http.StatusInternalServerError, "Could not verify permissions")
	}
}

// RequireAppVisible guards the app-scoped dashboard reads: while roles are
// enforced, members only see the apps they hold a grant on — anything else
// 404s like an app that does not exist. Admins and the community fallback see
// everything. CLI credentials pass through on the explicit marker the auth
// middleware stamped after validating their app scope — asserted, not
// inferred from a missing principal, so a wiring mistake fails closed.
func RequireAppVisible(service *RBACService, userLookup UserLookup) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if middleware.PrincipalFromContext(r.Context()) == nil {
				if middleware.CliAuthAppFromContext(r.Context()) != "" {
					next.ServeHTTP(w, r)
					return
				}
				handlers.RenderError(w, http.StatusForbidden, "This action requires a dashboard session")
				return
			}
			subject, ok := resolveSubject(w, r, userLookup)
			if !ok {
				return
			}
			visible, err := service.CanSeeApp(r.Context(), subject, mux.Vars(r)["APP_ID"])
			if err != nil {
				handlers.RenderError(w, http.StatusInternalServerError, "Could not verify permissions")
				return
			}
			if !visible {
				handlers.RenderError(w, http.StatusNotFound, "app not found")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
