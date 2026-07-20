// Copyright (c) 2026 Axel Marciano (Mercure Technologies). All rights reserved.
// This file is governed by the Mercure Technologies Enterprise Edition License
// (see ee/LICENSE); it is NOT covered by the MIT license of this repository.

package rbac

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type fakeRepo struct {
	roles map[string]Role
	// grants is keyed by userID; the slice order mirrors what the store
	// would return.
	grants map[string][]AppGrant
	// replaced records the last ReplaceUserGrants call for assertions.
	replaced map[string][]GrantInput
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		roles:    map[string]Role{},
		grants:   map[string][]AppGrant{},
		replaced: map[string][]GrantInput{},
	}
}

func (f *fakeRepo) ListRoles(_ context.Context) ([]Role, error) {
	roles := make([]Role, 0, len(f.roles))
	for _, role := range f.roles {
		roles = append(roles, role)
	}
	return roles, nil
}

func (f *fakeRepo) GetRoleByID(_ context.Context, id string) (Role, error) {
	role, ok := f.roles[id]
	if !ok {
		return Role{}, ErrRoleNotFound
	}
	return role, nil
}

func (f *fakeRepo) InsertRole(_ context.Context, role Role) (Role, error) {
	f.roles[role.ID] = role
	return role, nil
}

func (f *fakeRepo) UpdateRole(_ context.Context, id string, name string, permissions []Permission) error {
	role, ok := f.roles[id]
	if !ok {
		return ErrRoleNotFound
	}
	role.Name = name
	role.Permissions = permissions
	f.roles[id] = role
	return nil
}

func (f *fakeRepo) DeleteRole(_ context.Context, id string) error {
	if _, ok := f.roles[id]; !ok {
		return ErrRoleNotFound
	}
	delete(f.roles, id)
	return nil
}

func (f *fakeRepo) ListUserGrants(_ context.Context, userID string) ([]AppGrant, error) {
	return f.grants[userID], nil
}

func (f *fakeRepo) GetUserAppGrant(_ context.Context, userID string, appID string) (*AppGrant, error) {
	for _, grant := range f.grants[userID] {
		if grant.AppID == appID {
			return &grant, nil
		}
	}
	return nil, nil
}

func (f *fakeRepo) ReplaceUserGrants(_ context.Context, userID string, grants []GrantInput) error {
	f.replaced[userID] = grants
	return nil
}

func (f *fakeRepo) ListAccessibleAppIDs(_ context.Context, userID string) ([]string, error) {
	ids := make([]string, 0, len(f.grants[userID]))
	for _, grant := range f.grants[userID] {
		ids = append(ids, grant.AppID)
	}
	return ids, nil
}

func licensedService(repo RBACRepository) *RBACService {
	service := NewRBACService(repo)
	service.licenseValid = func() bool { return true }
	return service
}

func unlicensedService(repo RBACRepository) *RBACService {
	service := NewRBACService(repo)
	service.licenseValid = func() bool { return false }
	return service
}

func TestEnabledRequiresRepoAndLicense(t *testing.T) {
	require.False(t, licensedService(nil).Enabled(), "stateless mode can never enforce roles")
	require.False(t, unlicensedService(newFakeRepo()).Enabled(), "no license, community rules")
	require.True(t, licensedService(newFakeRepo()).Enabled())
}

func TestManagementRequiresControlPlane(t *testing.T) {
	ctx := context.Background()
	service := licensedService(nil)

	_, err := service.ListRoles(ctx)
	require.ErrorIs(t, err, ErrRequiresControlPlane)
	_, err = service.CreateRole(ctx, "Release manager", []Permission{PermChannelRolloutManage})
	require.ErrorIs(t, err, ErrRequiresControlPlane)
	_, err = service.GetUserGrants(ctx, "user-1")
	require.ErrorIs(t, err, ErrRequiresControlPlane)
	require.ErrorIs(t, service.SetUserGrants(ctx, "user-1", nil), ErrRequiresControlPlane)
}

func TestWritesRequireLicenseButReadsStayOpen(t *testing.T) {
	ctx := context.Background()
	repo := newFakeRepo()
	repo.roles["role-1"] = Role{ID: "role-1", Name: "Release manager"}
	service := unlicensedService(repo)

	_, err := service.CreateRole(ctx, "Ops", []Permission{PermBranchProtect})
	require.ErrorIs(t, err, ErrRequiresValidLicense)
	require.ErrorIs(t, service.UpdateRole(ctx, "role-1", "Ops", nil), ErrRequiresValidLicense)
	require.ErrorIs(t, service.DeleteRole(ctx, "role-1"), ErrRequiresValidLicense)
	require.ErrorIs(t, service.SetUserGrants(ctx, "user-1", nil), ErrRequiresValidLicense)

	// Reads keep working so the dashboard can show what exists (dormant).
	roles, err := service.ListRoles(ctx)
	require.NoError(t, err)
	require.Len(t, roles, 1)
	_, err = service.GetUserGrants(ctx, "user-1")
	require.NoError(t, err)
}

func TestCreateRoleValidatesInput(t *testing.T) {
	ctx := context.Background()
	service := licensedService(newFakeRepo())

	validationErr := (*ValidationError)(nil)
	_, err := service.CreateRole(ctx, "   ", []Permission{PermAppDelete})
	require.True(t, errors.As(err, &validationErr), "empty name must be refused, got %v", err)
	_, err = service.CreateRole(ctx, "Ops", []Permission{"branch:invalid"})
	require.True(t, errors.As(err, &validationErr), "unknown permission must be refused, got %v", err)

	role, err := service.CreateRole(ctx, "  Release manager  ", []Permission{PermChannelRolloutManage})
	require.NoError(t, err)
	require.Equal(t, "Release manager", role.Name, "name must be trimmed")
	require.NotEmpty(t, role.ID)
}

func TestSetUserGrantsValidatesInput(t *testing.T) {
	ctx := context.Background()
	repo := newFakeRepo()
	service := licensedService(repo)

	validationErr := (*ValidationError)(nil)
	err := service.SetUserGrants(ctx, "user-1", []GrantInput{
		{AppID: "app-1"},
		{AppID: "app-1"},
	})
	require.True(t, errors.As(err, &validationErr), "duplicate app must be refused, got %v", err)

	err = service.SetUserGrants(ctx, "user-1", []GrantInput{
		{AppID: "app-1", ExtraPermissions: []Permission{"nope"}},
	})
	require.True(t, errors.As(err, &validationErr), "unknown permission must be refused, got %v", err)

	grants := []GrantInput{{AppID: "app-1", ExtraPermissions: []Permission{PermBranchCreate}}}
	require.NoError(t, service.SetUserGrants(ctx, "user-1", grants))
	require.Equal(t, grants, repo.replaced["user-1"])
}

func TestAuthorizeMember(t *testing.T) {
	ctx := context.Background()
	repo := newFakeRepo()
	roleID := "role-1"
	repo.grants["user-1"] = []AppGrant{{
		AppID:            "app-1",
		RoleID:           &roleID,
		RolePermissions:  []Permission{PermChannelRolloutManage},
		ExtraPermissions: []Permission{PermBranchCreate},
	}}
	service := licensedService(repo)

	// Through the role and through the direct grant.
	require.NoError(t, service.AuthorizeMember(ctx, "user-1", "app-1", PermChannelRolloutManage))
	require.NoError(t, service.AuthorizeMember(ctx, "user-1", "app-1", PermBranchCreate))

	// Granted app, missing permission: a 403 naming the permission.
	err := service.AuthorizeMember(ctx, "user-1", "app-1", PermAppDelete)
	deniedErr := (*ErrPermissionDenied)(nil)
	require.True(t, errors.As(err, &deniedErr), "expected ErrPermissionDenied, got %v", err)
	require.Equal(t, PermAppDelete, deniedErr.Permission)

	// No grant on the app: it does not exist for this member.
	require.ErrorIs(t, service.AuthorizeMember(ctx, "user-1", "app-2", PermBranchCreate), ErrNoAppAccess)
	require.ErrorIs(t, service.AuthorizeMember(ctx, "user-2", "app-1", PermBranchCreate), ErrNoAppAccess)
}

func TestAuthorizeMemberFallsBackWithoutLicense(t *testing.T) {
	ctx := context.Background()
	repo := newFakeRepo()
	repo.grants["user-1"] = []AppGrant{{AppID: "app-1", ExtraPermissions: []Permission{PermAppDelete}}}

	err := unlicensedService(repo).AuthorizeMember(ctx, "user-1", "app-1", PermAppDelete)
	require.ErrorIs(t, err, ErrRequiresValidLicense,
		"an expired license must never widen member access beyond community rules")
}

func TestMemberVisibility(t *testing.T) {
	ctx := context.Background()
	repo := newFakeRepo()
	repo.grants["user-1"] = []AppGrant{{AppID: "app-1"}, {AppID: "app-2"}}

	// Community fallback: nothing is restricted.
	restricted, _, err := unlicensedService(repo).MemberVisibleApps(ctx, "user-1")
	require.NoError(t, err)
	require.False(t, restricted)
	visible, err := unlicensedService(repo).MemberCanSeeApp(ctx, "user-1", "app-3")
	require.NoError(t, err)
	require.True(t, visible)

	// Enforced: only granted apps, and a grant-less member sees nothing.
	service := licensedService(repo)
	restricted, ids, err := service.MemberVisibleApps(ctx, "user-1")
	require.NoError(t, err)
	require.True(t, restricted)
	require.ElementsMatch(t, []string{"app-1", "app-2"}, ids)

	restricted, ids, err = service.MemberVisibleApps(ctx, "user-2")
	require.NoError(t, err)
	require.True(t, restricted)
	require.Empty(t, ids)

	visible, err = service.MemberCanSeeApp(ctx, "user-1", "app-2")
	require.NoError(t, err)
	require.True(t, visible)
	visible, err = service.MemberCanSeeApp(ctx, "user-1", "app-3")
	require.NoError(t, err)
	require.False(t, visible)
}

func TestEffectivePermissionsDeduplicateInCatalogOrder(t *testing.T) {
	grant := AppGrant{
		AppID:            "app-1",
		RolePermissions:  []Permission{PermChannelRolloutManage, PermBranchCreate},
		ExtraPermissions: []Permission{PermBranchCreate, PermAppDelete},
	}
	require.Equal(t,
		[]Permission{PermAppDelete, PermBranchCreate, PermChannelRolloutManage},
		grant.Effective())

	ctx := context.Background()
	repo := newFakeRepo()
	repo.grants["user-1"] = []AppGrant{grant, {AppID: "app-2"}}
	byApp, err := licensedService(repo).EffectivePermissionsByApp(ctx, "user-1")
	require.NoError(t, err)
	require.Len(t, byApp, 2)
	require.Equal(t, []Permission{PermAppDelete, PermBranchCreate, PermChannelRolloutManage}, byApp["app-1"])
	require.Empty(t, byApp["app-2"], "a role-less, permission-less grant still lists the app (visibility)")
}
