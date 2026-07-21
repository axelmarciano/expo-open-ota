// Copyright (c) 2026 Axel Marciano (Mercure Technologies). All rights reserved.
// This file is governed by the Mercure Technologies Enterprise Edition License
// (see ee/LICENSE); it is NOT covered by the MIT license of this repository.

package licensing

import (
	"context"
	"expo-open-ota/ee/audit"
	"expo-open-ota/internal/auditlog"
	"expo-open-ota/internal/middleware"
	"expo-open-ota/internal/services"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeAuditStore lets these tests run the REAL AuditService gated by the real
// IsEnterprise, so the emit-before-Deactivate ordering in Remove is actually
// exercised: an event emitted after the gate closed lands nowhere and the
// assertions fail.
type fakeAuditStore struct{ inserted []auditlog.Event }

func (f *fakeAuditStore) Insert(_ context.Context, event auditlog.Event) (auditlog.Event, error) {
	f.inserted = append(f.inserted, event)
	return event, nil
}
func (f *fakeAuditStore) List(_ context.Context, _ audit.ListParams) ([]auditlog.Event, error) {
	return nil, nil
}
func (f *fakeAuditStore) Count(_ context.Context, _ audit.ListFilters) (int64, error) {
	return 0, nil
}

func TestActivateAndRemoveEmitAuditEvents(t *testing.T) {
	priv := setupTestKeypair(t)
	service := NewLicenseService(&fakeLicenseRepo{})
	auditStore := &fakeAuditStore{}
	service.SetOnAuditEvent(audit.NewAuditService(auditStore, IsEnterprise).Record)
	ctx := middleware.WithPrincipal(context.Background(),
		&services.DashboardPrincipal{UserId: "admin-1", Email: "admin@example.com"})

	expiry := time.Now().Add(365 * 24 * time.Hour).UTC()
	_, err := service.Activate(ctx, signTestKey(t, priv, &expiry))
	require.NoError(t, err)

	// The activation itself is recorded through the gate it just opened.
	require.Len(t, auditStore.inserted, 1)
	activated := auditStore.inserted[0]
	assert.Equal(t, auditlog.ActionLicenseActivated, activated.Action)
	assert.Equal(t, "admin-1", activated.ActorID)
	assert.Equal(t, "admin@example.com", activated.ActorDisplay)
	assert.NotEmpty(t, activated.Metadata["license_id"])
	assert.NotEmpty(t, activated.Metadata["expires_at"])

	require.NoError(t, service.Remove(ctx))

	// Emitted before Deactivate: the removal is the last entry the license
	// gate lets through. A regression emitting after would drop it here.
	require.Len(t, auditStore.inserted, 2)
	assert.Equal(t, auditlog.ActionLicenseRemoved, auditStore.inserted[1].Action)
	assert.False(t, IsEnterprise())
}
