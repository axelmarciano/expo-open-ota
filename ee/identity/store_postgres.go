// Copyright (c) 2026 Axel Marciano (Mercure Technologies). All rights reserved.
// This file is governed by the Mercure Technologies Enterprise Edition License
// (see ee/LICENSE); it is NOT covered by the MIT license of this repository.

package identity

import (
	"context"
	"encoding/json"
	"expo-open-ota/internal/database"
	"expo-open-ota/internal/database/postgres/pgdb"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type PostgresIdentityStore struct {
	engine *database.Engine
}

func NewPostgresIdentityStore(engine *database.Engine) *PostgresIdentityStore {
	return &PostgresIdentityStore{engine: engine}
}

func toPgUUID(id string) (pgtype.UUID, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("invalid uuid %q: %w", id, err)
	}
	return pgtype.UUID{Bytes: parsed, Valid: true}, nil
}

func specFromRow(row pgdb.IdentitySchema) KeySpec {
	return KeySpec{Key: row.Key, Type: ValueType(row.ValueType), MaxLength: int(row.MaxLength)}
}

func deviceFromRow(row pgdb.DeviceIdentity) (Device, error) {
	metadata := map[string]any{}
	if len(row.Metadata) > 0 {
		if err := json.Unmarshal(row.Metadata, &metadata); err != nil {
			return Device{}, fmt.Errorf("corrupt device metadata: %w", err)
		}
	}
	return Device{
		AppID:       uuid.UUID(row.AppID.Bytes).String(),
		EASClientID: uuid.UUID(row.EasClientID.Bytes).String(),
		Metadata:    metadata,
		CountryCode: row.CountryCode,
		City:        row.City,
		Lat:         row.Lat,
		Lng:         row.Lng,
		FirstSeenAt: row.FirstSeenAt.Time.UTC().Format("2006-01-02T15:04:05.000Z"),
		LastSeenAt:  row.LastSeenAt.Time.UTC().Format("2006-01-02T15:04:05.000Z"),
	}, nil
}

// GetSchema returns the app's allowlist. An app with no declared keys gets an
// empty schema, under which Sanitize drops everything: identity is opt-in per
// app by declaring keys, there is no implicit passthrough.
func (s *PostgresIdentityStore) GetSchema(ctx context.Context, appID string) (Schema, error) {
	appUUID, err := toPgUUID(appID)
	if err != nil {
		return nil, err
	}
	rows, err := s.engine.Queries.ListIdentitySchemaKeys(ctx, appUUID)
	if err != nil {
		return nil, fmt.Errorf("listing identity schema: %w", err)
	}
	schema := make(Schema, len(rows))
	for _, row := range rows {
		schema[row.Key] = specFromRow(row)
	}
	return schema, nil
}

func (s *PostgresIdentityStore) UpsertSchemaKey(ctx context.Context, appID string, spec KeySpec) (KeySpec, error) {
	if spec.MaxLength == 0 {
		spec.MaxLength = DefaultMaxLength
	}
	if err := ValidateKeySpec(spec); err != nil {
		return KeySpec{}, err
	}
	appUUID, err := toPgUUID(appID)
	if err != nil {
		return KeySpec{}, err
	}
	row, err := s.engine.Queries.UpsertIdentitySchemaKey(ctx, pgdb.UpsertIdentitySchemaKeyParams{
		AppID:     appUUID,
		Key:       spec.Key,
		ValueType: string(spec.Type),
		MaxLength: int32(spec.MaxLength),
	})
	if err != nil {
		return KeySpec{}, fmt.Errorf("upserting identity schema key: %w", err)
	}
	return specFromRow(row), nil
}

// DeleteSchemaKey removes a key from the allowlist and wipes its autocomplete
// stats in the same transaction, so searchMetadata never suggests values of a
// removed key. Values already merged into device metadata are left in place;
// they stop being accepted and stop being suggested.
func (s *PostgresIdentityStore) DeleteSchemaKey(ctx context.Context, appID string, key string) (bool, error) {
	appUUID, err := toPgUUID(appID)
	if err != nil {
		return false, err
	}
	tx, err := s.engine.DB.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("beginning delete schema key tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := s.engine.Queries.WithTx(tx)

	tag, err := qtx.DeleteIdentitySchemaKey(ctx, pgdb.DeleteIdentitySchemaKeyParams{AppID: appUUID, Key: key})
	if err != nil {
		return false, fmt.Errorf("deleting identity schema key: %w", err)
	}
	if err := qtx.DeleteIdentityValueStatsForKey(ctx, pgdb.DeleteIdentityValueStatsForKeyParams{AppID: appUUID, Key: key}); err != nil {
		return false, fmt.Errorf("deleting identity value stats: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("committing delete schema key tx: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

// ApplyIdentify runs one identify against the store: sanitize the raw wire
// metadata against the allowlist, merge it into the device row (per-key merge,
// incoming keys win), refresh geo when provided, and keep the per-value
// device counts in sync. Everything happens in one transaction with the
// device row locked, so concurrent identifies of the same install serialize
// and the counts never drift from the merges that produced them.
func (s *PostgresIdentityStore) ApplyIdentify(ctx context.Context, appID string, easClientID string, raw map[string]any, geo *Geo) (ApplyResult, error) {
	appUUID, err := toPgUUID(appID)
	if err != nil {
		return ApplyResult{}, err
	}
	clientUUID, err := toPgUUID(easClientID)
	if err != nil {
		return ApplyResult{}, err
	}

	tx, err := s.engine.DB.Begin(ctx)
	if err != nil {
		return ApplyResult{}, fmt.Errorf("beginning identify tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := s.engine.Queries.WithTx(tx)

	// The schema read shares the transaction so a concurrent allowlist change
	// cannot produce a merge that mixes two versions of the schema.
	schemaRows, err := qtx.ListIdentitySchemaKeys(ctx, appUUID)
	if err != nil {
		return ApplyResult{}, fmt.Errorf("listing identity schema: %w", err)
	}
	schema := make(Schema, len(schemaRows))
	for _, row := range schemaRows {
		schema[row.Key] = specFromRow(row)
	}
	sanitized, dropped := schema.Sanitize(raw)

	if err := qtx.EnsureDeviceIdentity(ctx, pgdb.EnsureDeviceIdentityParams{AppID: appUUID, EasClientID: clientUUID}); err != nil {
		return ApplyResult{}, fmt.Errorf("ensuring device row: %w", err)
	}
	current, err := qtx.GetDeviceIdentityForUpdate(ctx, pgdb.GetDeviceIdentityForUpdateParams{AppID: appUUID, EasClientID: clientUUID})
	if err != nil {
		return ApplyResult{}, fmt.Errorf("locking device row: %w", err)
	}
	previous := map[string]any{}
	if len(current.Metadata) > 0 {
		if err := json.Unmarshal(current.Metadata, &previous); err != nil {
			return ApplyResult{}, fmt.Errorf("corrupt device metadata: %w", err)
		}
	}

	merged := make(map[string]any, len(previous)+len(sanitized))
	for key, value := range previous {
		merged[key] = value
	}
	for key, value := range sanitized {
		merged[key] = value
	}
	mergedJSON, err := json.Marshal(merged)
	if err != nil {
		return ApplyResult{}, fmt.Errorf("marshalling merged metadata: %w", err)
	}

	params := pgdb.UpdateDeviceIdentityParams{
		AppID:       appUUID,
		EasClientID: clientUUID,
		Metadata:    mergedJSON,
	}
	if geo != nil {
		params.CountryCode = &geo.CountryCode
		params.City = &geo.City
		params.Lat = &geo.Lat
		params.Lng = &geo.Lng
	}
	updated, err := qtx.UpdateDeviceIdentity(ctx, params)
	if err != nil {
		return ApplyResult{}, fmt.Errorf("updating device row: %w", err)
	}

	// Value stats: a key newly set counts its value in; a changed value moves
	// the device from the old value's count to the new one. Comparison happens
	// on the rendered form, the same normalization the stats table stores.
	for key, value := range sanitized {
		newRendered := RenderValue(value)
		oldValue, existed := previous[key]
		if existed {
			oldRendered := RenderValue(oldValue)
			if oldRendered == newRendered {
				continue
			}
			decParams := pgdb.DecrementIdentityValueStatParams{AppID: appUUID, Key: key, Value: oldRendered}
			if err := qtx.DecrementIdentityValueStat(ctx, decParams); err != nil {
				return ApplyResult{}, fmt.Errorf("decrementing value stat: %w", err)
			}
			delParams := pgdb.DeleteZeroIdentityValueStatsParams{AppID: appUUID, Key: key, Value: oldRendered}
			if err := qtx.DeleteZeroIdentityValueStats(ctx, delParams); err != nil {
				return ApplyResult{}, fmt.Errorf("pruning zero value stat: %w", err)
			}
		}
		incParams := pgdb.IncrementIdentityValueStatParams{AppID: appUUID, Key: key, Value: newRendered}
		if err := qtx.IncrementIdentityValueStat(ctx, incParams); err != nil {
			return ApplyResult{}, fmt.Errorf("incrementing value stat: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return ApplyResult{}, fmt.Errorf("committing identify tx: %w", err)
	}
	device, err := deviceFromRow(updated)
	if err != nil {
		return ApplyResult{}, err
	}
	return ApplyResult{Device: device, DroppedKeys: dropped}, nil
}

// GetDevice returns nil when the install was never seen.
func (s *PostgresIdentityStore) GetDevice(ctx context.Context, appID string, easClientID string) (*Device, error) {
	appUUID, err := toPgUUID(appID)
	if err != nil {
		return nil, err
	}
	clientUUID, err := toPgUUID(easClientID)
	if err != nil {
		return nil, err
	}
	row, err := s.engine.Queries.GetDeviceIdentity(ctx, pgdb.GetDeviceIdentityParams{AppID: appUUID, EasClientID: clientUUID})
	if err != nil {
		if database.IsNoRows(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("getting device identity: %w", err)
	}
	device, err := deviceFromRow(row)
	if err != nil {
		return nil, err
	}
	return &device, nil
}

// SearchMetadataValues is the autocomplete behind searchMetadata: top values
// of one key ranked by device count, optionally narrowed by a substring.
func (s *PostgresIdentityStore) SearchMetadataValues(ctx context.Context, appID string, key string, search string, limit int) ([]ValueCount, error) {
	appUUID, err := toPgUUID(appID)
	if err != nil {
		return nil, err
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	rows, err := s.engine.Queries.SearchIdentityValues(ctx, pgdb.SearchIdentityValuesParams{
		AppID:      appUUID,
		Key:        key,
		Search:     search,
		MaxResults: int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("searching identity values: %w", err)
	}
	values := make([]ValueCount, 0, len(rows))
	for _, row := range rows {
		values = append(values, ValueCount{Value: row.Value, DeviceCount: row.DeviceCount})
	}
	return values, nil
}
