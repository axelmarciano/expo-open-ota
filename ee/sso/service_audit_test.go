// Copyright (c) 2026 Axel Marciano (Mercure Technologies). All rights reserved.
// This file is governed by the Mercure Technologies Enterprise Edition License
// (see ee/LICENSE); it is NOT covered by the MIT license of this repository.

package sso

import (
	"context"
	"expo-open-ota/internal/auditlog"
	"expo-open-ota/internal/middleware"
	"expo-open-ota/internal/services"
	"expo-open-ota/internal/store"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeAuditRecorder struct{ events []auditlog.Event }

func (f *fakeAuditRecorder) Record(_ context.Context, event auditlog.Event) {
	f.events = append(f.events, event)
}

func TestCompleteLoginEmitsAuditEvent(t *testing.T) {
	idp := newFakeIdP(t)
	users := newFakeUserRepo()
	repo := newFakeSSORepo(users, testConfigFor(idp))
	service, _ := newTestService(t, repo, users)
	recorder := &fakeAuditRecorder{}
	service.SetOnAuditEvent(recorder.Record)

	_, err := completeFlow(t, service, idp, nil)
	require.NoError(t, err)

	user, err := users.GetUserByEmail(context.Background(), testEmail)
	require.NoError(t, err)
	// A first sign-in JIT-provisions the account then signs it in: two
	// events, provisioning first, each with its own actor.
	require.Len(t, recorder.events, 2)

	provisioned := recorder.events[0]
	assert.Equal(t, auditlog.ActionUserSSOProvisioned, provisioned.Action)
	assert.Equal(t, auditlog.ActorSystem, provisioned.ActorType)
	assert.Equal(t, user.Id, provisioned.TargetID)
	assert.Equal(t, testEmail, provisioned.TargetDisplay)
	assert.Equal(t, idp.issuer, provisioned.Metadata["issuer"])
	assert.Equal(t, testSubject, provisioned.Metadata["subject"])
	assert.Equal(t, false, provisioned.Metadata["pending_approval"])

	login := recorder.events[1]
	assert.Equal(t, auditlog.ActionUserSSOLogin, login.Action)
	assert.Equal(t, auditlog.OutcomeSuccess, login.Outcome)
	assert.Equal(t, auditlog.ActorUser, login.ActorType)
	assert.Equal(t, user.Id, login.ActorID)
	assert.Equal(t, testEmail, login.ActorDisplay)
	assert.Equal(t, user.Id, login.TargetID)
}

func TestPendingProvisioningEmitsProvisionedOnly(t *testing.T) {
	idp := newFakeIdP(t)
	users := newFakeUserRepo()
	cfg := testConfigFor(idp)
	cfg.ManualUserValidation = true
	service, _ := newTestService(t, newFakeSSORepo(users, cfg), users)
	recorder := &fakeAuditRecorder{}
	service.SetOnAuditEvent(recorder.Record)

	_, err := completeFlow(t, service, idp, nil)
	require.ErrorIs(t, err, ErrSSOAccountPendingApproval)

	// The disabled account was created: exactly the trail an admin approval
	// workflow relies on. No sso_login, the sign-in itself was refused.
	require.Len(t, recorder.events, 1)
	assert.Equal(t, auditlog.ActionUserSSOProvisioned, recorder.events[0].Action)
	assert.Equal(t, true, recorder.events[0].Metadata["pending_approval"])
}

func TestLinkingExistingAccountEmitsAuditEvent(t *testing.T) {
	idp := newFakeIdP(t)
	users := newFakeUserRepo()
	existing, err := users.InsertUser(context.Background(), store.InsertUserParameters{
		ID: "existing-user", Email: testEmail, PasswordHash: "some-bcrypt-hash", IsAdmin: true, Enabled: true,
	})
	require.NoError(t, err)
	service, _ := newTestService(t, newFakeSSORepo(users, testConfigFor(idp)), users)
	recorder := &fakeAuditRecorder{}
	service.SetOnAuditEvent(recorder.Record)

	_, err = completeFlow(t, service, idp, nil)
	require.NoError(t, err)

	// Binding an IdP identity to a pre-existing (here admin) account is its
	// own security event, then the sign-in records as usual. No provisioning
	// event: no account was created.
	require.Len(t, recorder.events, 2)
	linked := recorder.events[0]
	assert.Equal(t, auditlog.ActionUserSSOLinked, linked.Action)
	assert.Equal(t, auditlog.ActorSystem, linked.ActorType)
	assert.Equal(t, existing.Id, linked.TargetID)
	assert.Equal(t, testEmail, linked.TargetDisplay)
	assert.Equal(t, idp.issuer, linked.Metadata["issuer"])
	assert.Equal(t, testSubject, linked.Metadata["subject"])
	assert.Equal(t, auditlog.ActionUserSSOLogin, recorder.events[1].Action)
}

func TestSSOConfigChangesEmitAuditEvents(t *testing.T) {
	idp := newFakeIdP(t)
	users := newFakeUserRepo()
	service, _ := newTestService(t, newFakeSSORepo(users, nil), users)
	recorder := &fakeAuditRecorder{}
	service.SetOnAuditEvent(recorder.Record)
	ctx := middleware.WithPrincipal(context.Background(),
		&services.DashboardPrincipal{UserId: "admin-1", Email: "admin@example.com"})

	// Disabled configuration: saving skips OIDC discovery, no IdP needed.
	_, err := service.SaveConfig(ctx, SaveConfigInput{
		Issuer:               idp.issuer,
		ClientID:             "client-1",
		ClientSecret:         "s3cret",
		Enabled:              false,
		TrustUnverifiedEmail: true,
	})
	require.NoError(t, err)

	require.Len(t, recorder.events, 1)
	saved := recorder.events[0]
	assert.Equal(t, auditlog.ActionSSOConfigSaved, saved.Action)
	assert.Equal(t, "admin-1", saved.ActorID)
	// The security toggles are recorded; the secret never is.
	assert.Equal(t, true, saved.Metadata["trust_unverified_email"])
	assert.Equal(t, false, saved.Metadata["manual_user_validation"])
	assert.NotContains(t, saved.Metadata, "client_secret")

	require.NoError(t, service.DeleteConfig(ctx))
	require.Len(t, recorder.events, 2)
	assert.Equal(t, auditlog.ActionSSOConfigDeleted, recorder.events[1].Action)
}

func TestCompleteLoginFailureEmitsNothing(t *testing.T) {
	idp := newFakeIdP(t)
	users := newFakeUserRepo()
	repo := newFakeSSORepo(users, testConfigFor(idp))
	service, _ := newTestService(t, repo, users)
	recorder := &fakeAuditRecorder{}
	service.SetOnAuditEvent(recorder.Record)

	// A rejected sign-in (unverified email) mints no session and must leave
	// no sso_login event.
	_, err := completeFlow(t, service, idp, func(claims jwt.MapClaims) {
		claims["email_verified"] = false
	})
	require.Error(t, err)
	require.Empty(t, recorder.events)
}
