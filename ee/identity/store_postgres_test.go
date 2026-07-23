// Copyright (c) 2026 Axel Marciano (Mercure Technologies). All rights reserved.
// This file is governed by the Mercure Technologies Enterprise Edition License
// (see ee/LICENSE); it is NOT covered by the MIT license of this repository.

// Integration tests for the identity store: the merge-under-lock transaction,
// the value-stat bookkeeping and the trigram search need a real Postgres.
// They skip unless TEST_DATABASE_URL is set, e.g.:
//
//	docker run -d --name eoo-pg -e POSTGRES_PASSWORD=test -p 55432:5432 postgres:16-alpine
//	TEST_DATABASE_URL="postgres://postgres:test@localhost:55432/postgres?sslmode=disable" go test ./ee/identity/
//
// Every test creates its own app row, so tests never observe each other's
// devices or stats even on a database reused across runs.

package identity

import (
	"context"
	"os"
	"sync"
	"testing"

	"expo-open-ota/internal/database"
	"expo-open-ota/internal/database/postgres"
	"expo-open-ota/internal/database/postgres/pgdb"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func setupIdentityStore(t *testing.T) (*PostgresIdentityStore, *pgxpool.Pool) {
	t.Helper()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		// Same guard as the audit and rbac store tests: a skip in CI is a
		// green job that ran none of these queries.
		if os.Getenv("CI") != "" {
			t.Fatal("TEST_DATABASE_URL must be set in CI: these tests cover SQL that unit tests cannot reach")
		}
		t.Skip("TEST_DATABASE_URL not set — start a Postgres and set it to run the identity store tests")
	}
	t.Setenv("ADMIN_EMAIL", "seed-admin@example.com")
	t.Setenv("ADMIN_PASSWORD", "Sup3rSecret!")
	postgres.RunDBMigrations(dbURL)

	pool, err := pgxpool.New(context.Background(), dbURL)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	return NewPostgresIdentityStore(&database.Engine{Queries: pgdb.New(pool), DB: pool}), pool
}

func seedApp(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	appID := uuid.NewString()
	_, err := pool.Exec(context.Background(), "INSERT INTO apps (id, name) VALUES ($1, $2)", appID, "identity-test-"+appID[:8])
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM apps WHERE id = $1", appID)
	})
	return appID
}

func declareKey(t *testing.T, store *PostgresIdentityStore, appID, key string, valueType ValueType) {
	t.Helper()
	_, err := store.UpsertSchemaKey(context.Background(), appID, KeySpec{Key: key, Type: valueType, MaxLength: DefaultMaxLength})
	require.NoError(t, err)
}

func TestSchemaCRUD(t *testing.T) {
	store, pool := setupIdentityStore(t)
	appID := seedApp(t, pool)
	ctx := context.Background()

	schema, err := store.GetSchema(ctx, appID)
	require.NoError(t, err)
	require.Empty(t, schema)

	_, err = store.UpsertSchemaKey(ctx, appID, KeySpec{Key: "userId", Type: ValueTypeString})
	require.NoError(t, err)
	_, err = store.UpsertSchemaKey(ctx, appID, KeySpec{Key: "seats", Type: ValueTypeNumber, MaxLength: 32})
	require.NoError(t, err)

	schema, err = store.GetSchema(ctx, appID)
	require.NoError(t, err)
	require.Len(t, schema, 2)
	// Omitted max length lands on the default, not on zero.
	require.Equal(t, DefaultMaxLength, schema["userId"].MaxLength)
	require.Equal(t, 32, schema["seats"].MaxLength)

	// Upsert re-types a key in place.
	_, err = store.UpsertSchemaKey(ctx, appID, KeySpec{Key: "seats", Type: ValueTypeString, MaxLength: 32})
	require.NoError(t, err)
	schema, err = store.GetSchema(ctx, appID)
	require.NoError(t, err)
	require.Equal(t, ValueTypeString, schema["seats"].Type)

	// Invalid specs are rejected before touching the database.
	_, err = store.UpsertSchemaKey(ctx, appID, KeySpec{Key: "bad key", Type: ValueTypeString})
	require.Error(t, err)

	deleted, err := store.DeleteSchemaKey(ctx, appID, "seats")
	require.NoError(t, err)
	require.True(t, deleted)
	deleted, err = store.DeleteSchemaKey(ctx, appID, "seats")
	require.NoError(t, err)
	require.False(t, deleted)
}

func TestApplyIdentifyMergesAndCounts(t *testing.T) {
	store, pool := setupIdentityStore(t)
	appID := seedApp(t, pool)
	ctx := context.Background()
	declareKey(t, store, appID, "userId", ValueTypeString)
	declareKey(t, store, appID, "tenant", ValueTypeString)

	clientID := uuid.NewString()
	result, err := store.ApplyIdentify(ctx, appID, clientID, map[string]any{
		"userId": "user_1",
		"junk":   "dropped by the allowlist",
	}, nil)
	require.NoError(t, err)
	require.Equal(t, map[string]any{"userId": "user_1"}, result.Device.Metadata)
	require.Equal(t, []string{"junk"}, result.DroppedKeys)

	// Second identify adds a key and keeps the first one (per-key merge).
	result, err = store.ApplyIdentify(ctx, appID, clientID, map[string]any{"tenant": "acme"}, nil)
	require.NoError(t, err)
	require.Equal(t, map[string]any{"userId": "user_1", "tenant": "acme"}, result.Device.Metadata)

	// Changing a value moves the device count from the old value to the new
	// one and prunes the emptied row.
	_, err = store.ApplyIdentify(ctx, appID, clientID, map[string]any{"tenant": "globex"}, nil)
	require.NoError(t, err)
	values, err := store.SearchMetadataValues(ctx, appID, "tenant", "", 10)
	require.NoError(t, err)
	require.Equal(t, []ValueCount{{Value: "globex", DeviceCount: 1}}, values)

	// Re-identifying the same value must not inflate the count.
	_, err = store.ApplyIdentify(ctx, appID, clientID, map[string]any{"tenant": "globex"}, nil)
	require.NoError(t, err)
	values, err = store.SearchMetadataValues(ctx, appID, "tenant", "", 10)
	require.NoError(t, err)
	require.Equal(t, []ValueCount{{Value: "globex", DeviceCount: 1}}, values)

	device, err := store.GetDevice(ctx, appID, clientID)
	require.NoError(t, err)
	require.NotNil(t, device)
	require.Equal(t, "globex", device.Metadata["tenant"])

	missing, err := store.GetDevice(ctx, appID, uuid.NewString())
	require.NoError(t, err)
	require.Nil(t, missing)

	_, err = store.ApplyIdentify(ctx, appID, "not-a-uuid", map[string]any{}, nil)
	require.Error(t, err)
}

func TestApplyIdentifyGeoCoalesce(t *testing.T) {
	store, pool := setupIdentityStore(t)
	appID := seedApp(t, pool)
	ctx := context.Background()
	clientID := uuid.NewString()

	result, err := store.ApplyIdentify(ctx, appID, clientID, nil, &Geo{CountryCode: "FR", City: "Paris", Lat: 48.85, Lng: 2.35})
	require.NoError(t, err)
	require.NotNil(t, result.Device.CountryCode)
	require.Equal(t, "FR", *result.Device.CountryCode)

	// An identify that resolves no geo keeps the previously known location.
	result, err = store.ApplyIdentify(ctx, appID, clientID, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, result.Device.CountryCode)
	require.Equal(t, "FR", *result.Device.CountryCode)
	require.NotNil(t, result.Device.Lat)
	require.InDelta(t, 48.85, *result.Device.Lat, 0.001)
}

func TestSearchMetadataValuesRankingAndFilter(t *testing.T) {
	store, pool := setupIdentityStore(t)
	appID := seedApp(t, pool)
	ctx := context.Background()
	declareKey(t, store, appID, "tenant", ValueTypeString)

	seed := map[string]int{"acme": 3, "acme-eu": 2, "globex": 1}
	for tenant, devices := range seed {
		for i := 0; i < devices; i++ {
			_, err := store.ApplyIdentify(ctx, appID, uuid.NewString(), map[string]any{"tenant": tenant}, nil)
			require.NoError(t, err)
		}
	}

	// Empty search: top values by device count.
	values, err := store.SearchMetadataValues(ctx, appID, "tenant", "", 10)
	require.NoError(t, err)
	require.Equal(t, []ValueCount{{Value: "acme", DeviceCount: 3}, {Value: "acme-eu", DeviceCount: 2}, {Value: "globex", DeviceCount: 1}}, values)

	// Case-insensitive substring narrows, ranking is preserved.
	values, err = store.SearchMetadataValues(ctx, appID, "tenant", "ACME", 10)
	require.NoError(t, err)
	require.Equal(t, []ValueCount{{Value: "acme", DeviceCount: 3}, {Value: "acme-eu", DeviceCount: 2}}, values)

	// Limit applies after ranking.
	values, err = store.SearchMetadataValues(ctx, appID, "tenant", "", 1)
	require.NoError(t, err)
	require.Equal(t, []ValueCount{{Value: "acme", DeviceCount: 3}}, values)

	// Unknown key: no rows, no error.
	values, err = store.SearchMetadataValues(ctx, appID, "nope", "", 10)
	require.NoError(t, err)
	require.Empty(t, values)
}

func TestDeleteSchemaKeyWipesItsStats(t *testing.T) {
	store, pool := setupIdentityStore(t)
	appID := seedApp(t, pool)
	ctx := context.Background()
	declareKey(t, store, appID, "tenant", ValueTypeString)
	declareKey(t, store, appID, "plan", ValueTypeString)

	_, err := store.ApplyIdentify(ctx, appID, uuid.NewString(), map[string]any{"tenant": "acme", "plan": "pro"}, nil)
	require.NoError(t, err)

	deleted, err := store.DeleteSchemaKey(ctx, appID, "tenant")
	require.NoError(t, err)
	require.True(t, deleted)

	// The removed key stops being suggested; the surviving key is untouched.
	values, err := store.SearchMetadataValues(ctx, appID, "tenant", "", 10)
	require.NoError(t, err)
	require.Empty(t, values)
	values, err = store.SearchMetadataValues(ctx, appID, "plan", "", 10)
	require.NoError(t, err)
	require.Equal(t, []ValueCount{{Value: "pro", DeviceCount: 1}}, values)

	// And its values are no longer accepted on the next identify.
	result, err := store.ApplyIdentify(ctx, appID, uuid.NewString(), map[string]any{"tenant": "acme"}, nil)
	require.NoError(t, err)
	require.Empty(t, result.Device.Metadata)
	require.Equal(t, []string{"tenant"}, result.DroppedKeys)
}

// Concurrent first identifies of the same install must both land: the
// insert-then-lock sequence serializes the merges, so neither metadata write
// nor stat increment is lost.
func TestApplyIdentifyConcurrentFirstWrite(t *testing.T) {
	store, pool := setupIdentityStore(t)
	appID := seedApp(t, pool)
	ctx := context.Background()
	declareKey(t, store, appID, "userId", ValueTypeString)
	declareKey(t, store, appID, "tenant", ValueTypeString)

	clientID := uuid.NewString()
	var wg sync.WaitGroup
	errs := make([]error, 2)
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, errs[0] = store.ApplyIdentify(ctx, appID, clientID, map[string]any{"userId": "user_1"}, nil)
	}()
	go func() {
		defer wg.Done()
		_, errs[1] = store.ApplyIdentify(ctx, appID, clientID, map[string]any{"tenant": "acme"}, nil)
	}()
	wg.Wait()
	require.NoError(t, errs[0])
	require.NoError(t, errs[1])

	device, err := store.GetDevice(ctx, appID, clientID)
	require.NoError(t, err)
	require.NotNil(t, device)
	require.Equal(t, map[string]any{"userId": "user_1", "tenant": "acme"}, device.Metadata)
}
