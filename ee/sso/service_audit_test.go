// Copyright (c) 2026 Axel Marciano (Mercure Technologies). All rights reserved.
// This file is governed by the Mercure Technologies Enterprise Edition License
// (see ee/LICENSE); it is NOT covered by the MIT license of this repository.

package sso

import (
	"context"
	"expo-open-ota/internal/auditlog"
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
	require.Len(t, recorder.events, 1)
	event := recorder.events[0]
	assert.Equal(t, auditlog.ActionUserSSOLogin, event.Action)
	assert.Equal(t, auditlog.OutcomeSuccess, event.Outcome)
	assert.Equal(t, auditlog.ActorUser, event.ActorType)
	assert.Equal(t, user.Id, event.ActorID)
	assert.Equal(t, testEmail, event.ActorDisplay)
	assert.Equal(t, user.Id, event.TargetID)
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
