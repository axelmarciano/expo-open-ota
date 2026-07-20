// Copyright (c) 2026 Axel Marciano (Mercure Technologies). All rights reserved.
// This file is governed by the Mercure Technologies Enterprise Edition License
// (see ee/LICENSE); it is NOT covered by the MIT license of this repository.

package licensing

import (
	"context"
	"errors"
	"testing"
	"time"
)

// fakeLicenseRepo is an in-memory LicenseRepository mirroring the singleton
// row of the enterprise_license table.
type fakeLicenseRepo struct {
	stored *StoredLicense
	err    error
}

func (r *fakeLicenseRepo) GetLicense(ctx context.Context) (*StoredLicense, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.stored, nil
}

func (r *fakeLicenseRepo) UpsertLicense(ctx context.Context, key string) (StoredLicense, error) {
	if r.err != nil {
		return StoredLicense{}, r.err
	}
	r.stored = &StoredLicense{Key: key, UpdatedAt: time.Now().UTC()}
	return *r.stored, nil
}

func (r *fakeLicenseRepo) DeleteLicense(ctx context.Context) error {
	if r.err != nil {
		return r.err
	}
	r.stored = nil
	return nil
}

func TestServiceStatelessModeAnswersControlPlaneError(t *testing.T) {
	service := NewLicenseService(nil)
	if _, err := service.Status(context.Background()); !errors.Is(err, ErrLicenseRequiresControlPlane) {
		t.Fatalf("expected ErrLicenseRequiresControlPlane, got %v", err)
	}
	if _, err := service.Activate(context.Background(), "key/whatever"); !errors.Is(err, ErrLicenseRequiresControlPlane) {
		t.Fatalf("expected ErrLicenseRequiresControlPlane, got %v", err)
	}
	if err := service.Remove(context.Background()); !errors.Is(err, ErrLicenseRequiresControlPlane) {
		t.Fatalf("expected ErrLicenseRequiresControlPlane, got %v", err)
	}
	if err := service.ActivateFromStore(context.Background()); err != nil {
		t.Fatalf("expected boot activation to be a no-op in stateless mode, got %v", err)
	}
}

func TestServiceActivatePersistsAndEnablesEnterprise(t *testing.T) {
	priv := setupTestKeypair(t)
	repo := &fakeLicenseRepo{}
	service := NewLicenseService(repo)

	status, err := service.Activate(context.Background(), signTestKey(t, priv, in(24*time.Hour)))
	if err != nil {
		t.Fatalf("expected activation to succeed, got %v", err)
	}
	if !status.Valid() {
		t.Fatalf("expected a valid status, got %+v", status)
	}
	if repo.stored == nil {
		t.Fatal("expected the key to be persisted")
	}
	if !IsEnterprise() {
		t.Fatal("expected IsEnterprise after activation")
	}
}

func TestServiceActivateRejectsBadKeysWithoutPersisting(t *testing.T) {
	priv := setupTestKeypair(t)
	repo := &fakeLicenseRepo{}
	service := NewLicenseService(repo)

	if _, err := service.Activate(context.Background(), "not-a-key"); !errors.Is(err, ErrMalformedKey) {
		t.Fatalf("expected ErrMalformedKey, got %v", err)
	}
	if _, err := service.Activate(context.Background(), signTestKey(t, priv, in(-time.Hour))); !errors.Is(err, ErrExpired) {
		t.Fatalf("expected ErrExpired, got %v", err)
	}
	if repo.stored != nil {
		t.Fatal("expected no key to be persisted after rejected activations")
	}
	if IsEnterprise() {
		t.Fatal("expected community edition after rejected activations")
	}
}

func TestServiceRemoveDropsToCommunity(t *testing.T) {
	priv := setupTestKeypair(t)
	repo := &fakeLicenseRepo{}
	service := NewLicenseService(repo)

	if _, err := service.Activate(context.Background(), signTestKey(t, priv, in(24*time.Hour))); err != nil {
		t.Fatalf("expected activation to succeed, got %v", err)
	}
	if err := service.Remove(context.Background()); err != nil {
		t.Fatalf("expected removal to succeed, got %v", err)
	}
	if repo.stored != nil {
		t.Fatal("expected the stored key to be deleted")
	}
	if IsEnterprise() {
		t.Fatal("expected community edition after removal")
	}
	status, err := service.Status(context.Background())
	if err != nil {
		t.Fatalf("expected status to succeed, got %v", err)
	}
	if status.HasKey || status.Valid() {
		t.Fatalf("expected an empty status, got %+v", status)
	}
}

func TestServiceActivateFromStore(t *testing.T) {
	priv := setupTestKeypair(t)
	repo := &fakeLicenseRepo{stored: &StoredLicense{Key: signTestKey(t, priv, in(24*time.Hour)), UpdatedAt: time.Now()}}
	service := NewLicenseService(repo)

	if err := service.ActivateFromStore(context.Background()); err != nil {
		t.Fatalf("expected boot activation to succeed, got %v", err)
	}
	if !IsEnterprise() {
		t.Fatal("expected IsEnterprise after boot activation")
	}
}

func TestServiceActivateFromStoreWithExpiredKeyStaysCommunity(t *testing.T) {
	priv := setupTestKeypair(t)
	repo := &fakeLicenseRepo{stored: &StoredLicense{Key: signTestKey(t, priv, in(-time.Hour)), UpdatedAt: time.Now()}}
	service := NewLicenseService(repo)

	if err := service.ActivateFromStore(context.Background()); err != nil {
		t.Fatalf("expected an expired stored key to be non-fatal, got %v", err)
	}
	if IsEnterprise() {
		t.Fatal("expected community edition with an expired stored key")
	}
	// The dashboard still sees the key, the parsed payload and the reason it
	// is not usable.
	status, err := service.Status(context.Background())
	if err != nil {
		t.Fatalf("expected status to succeed, got %v", err)
	}
	if !status.HasKey || status.Valid() || !errors.Is(status.Err, ErrExpired) || status.License == nil {
		t.Fatalf("expected an expired-key status, got %+v", status)
	}
}

func TestServiceActivateFromStoreSurfacesInfrastructureErrors(t *testing.T) {
	infraErr := errors.New("connection refused")
	service := NewLicenseService(&fakeLicenseRepo{err: infraErr})
	if err := service.ActivateFromStore(context.Background()); !errors.Is(err, infraErr) {
		t.Fatalf("expected the infrastructure error to surface, got %v", err)
	}
}

// The sync loop scenarios below call syncFromStore directly: they model what
// a replica that did NOT serve the dashboard request observes when its
// reconciliation fires.

func TestSyncFromStorePicksUpActivationFromAnotherReplica(t *testing.T) {
	priv := setupTestKeypair(t)
	repo := &fakeLicenseRepo{}
	service := NewLicenseService(repo)

	service.syncFromStore(context.Background())
	if IsEnterprise() {
		t.Fatal("expected community edition while no key is stored")
	}
	// Another replica stores a key: only the database row moves.
	repo.stored = &StoredLicense{Key: signTestKey(t, priv, in(24*time.Hour)), UpdatedAt: time.Now()}
	service.syncFromStore(context.Background())
	if !IsEnterprise() {
		t.Fatal("expected IsEnterprise after the sync picked up the stored key")
	}
}

func TestSyncFromStoreDropsRemovalFromAnotherReplica(t *testing.T) {
	priv := setupTestKeypair(t)
	repo := &fakeLicenseRepo{}
	service := NewLicenseService(repo)

	if _, err := service.Activate(context.Background(), signTestKey(t, priv, in(24*time.Hour))); err != nil {
		t.Fatalf("expected activation to succeed, got %v", err)
	}
	// Another replica removes the license: only the database row moves.
	repo.stored = nil
	service.syncFromStore(context.Background())
	if IsEnterprise() {
		t.Fatal("expected community edition after the sync noticed the removal")
	}
}

func TestSyncFromStorePropagatesRenewal(t *testing.T) {
	priv := setupTestKeypair(t)
	repo := &fakeLicenseRepo{stored: &StoredLicense{Key: signTestKey(t, priv, in(time.Hour)), UpdatedAt: time.Now()}}
	service := NewLicenseService(repo)
	if err := service.ActivateFromStore(context.Background()); err != nil {
		t.Fatalf("expected boot activation to succeed, got %v", err)
	}
	// A renewed key carries the same license id with a later expiry; the sync
	// must refresh the in-memory expiry, not treat it as the same state.
	repo.stored = &StoredLicense{Key: signTestKey(t, priv, in(48*time.Hour)), UpdatedAt: time.Now()}
	service.syncFromStore(context.Background())
	active := Current()
	if active == nil || active.Expiry == nil || time.Until(*active.Expiry) < 24*time.Hour {
		t.Fatalf("expected the renewed expiry to be active, got %+v", active)
	}
}

func TestSyncFromStoreKeepsStateOnInfrastructureError(t *testing.T) {
	priv := setupTestKeypair(t)
	repo := &fakeLicenseRepo{}
	service := NewLicenseService(repo)
	if _, err := service.Activate(context.Background(), signTestKey(t, priv, in(24*time.Hour))); err != nil {
		t.Fatalf("expected activation to succeed, got %v", err)
	}
	// A transient database failure must not drop a valid license.
	repo.err = errors.New("connection refused")
	service.syncFromStore(context.Background())
	if !IsEnterprise() {
		t.Fatal("expected the active license to survive a transient database error")
	}
}
