// Copyright (c) 2026 Mercure Technologies. All rights reserved.
// This file is governed by the Expo Open OTA Enterprise Edition license
// (see ee/LICENSE); it is NOT covered by the MIT license of this repository.

package apikeyscopes

import (
	"context"
	"expo-open-ota/internal/database"
	"expo-open-ota/internal/database/postgres/pgdb"
	"expo-open-ota/internal/store"
	"fmt"
	"net/netip"
)

type PostgresApiKeyScopeStore struct {
	engine *database.Engine
}

func NewPostgresApiKeyScopeStore(engine *database.Engine) *PostgresApiKeyScopeStore {
	return &PostgresApiKeyScopeStore{engine: engine}
}

// GetScopesByAppID returns one entry per API key that has at least one
// restriction set; unrestricted keys are simply absent.
func (s *PostgresApiKeyScopeStore) GetScopesByAppID(ctx context.Context, appID string) ([]ApiKeyScopes, error) {
	pgAppID := store.ToPgUUID(appID)
	ipRows, err := s.engine.Queries.GetApiKeysAllowedIpsByAppID(ctx, pgAppID)
	if err != nil {
		return nil, fmt.Errorf("failed to read api key ip allowlists: %w", err)
	}
	channelRows, err := s.engine.Queries.GetApiKeyChannelsByAppID(ctx, pgAppID)
	if err != nil {
		return nil, fmt.Errorf("failed to read api key channel scopes: %w", err)
	}

	byKey := make(map[int64]*ApiKeyScopes)
	var order []int64
	scopesOf := func(apiKeyID int64) *ApiKeyScopes {
		if existing := byKey[apiKeyID]; existing != nil {
			return existing
		}
		created := &ApiKeyScopes{ApiKeyID: apiKeyID}
		byKey[apiKeyID] = created
		order = append(order, apiKeyID)
		return created
	}
	for _, row := range ipRows {
		scopesOf(row.ID).AllowedIps = row.AllowedIps
	}
	for _, row := range channelRows {
		scopes := scopesOf(row.ApiKeyID)
		scopes.ChannelIDs = append(scopes.ChannelIDs, row.ChannelID)
	}

	result := make([]ApiKeyScopes, 0, len(order))
	for _, apiKeyID := range order {
		result = append(result, *byKey[apiKeyID])
	}
	return result, nil
}

// ReplaceScopes swaps the whole restriction set of one key in a single
// transaction. The initial UPDATE doubles as the ownership check: zero rows
// means the key does not exist for this app (or is revoked), so nothing else
// runs.
func (s *PostgresApiKeyScopeStore) ReplaceScopes(ctx context.Context, appID string, apiKeyID int64, channelIDs []int64, allowedIps []netip.Prefix) error {
	pgAppID := store.ToPgUUID(appID)
	return s.engine.WithTx(ctx, func(q *pgdb.Queries) error {
		updated, err := q.UpdateApiKeyAllowedIps(ctx, pgdb.UpdateApiKeyAllowedIpsParams{
			AllowedIps: allowedIps,
			ID:         apiKeyID,
			AppID:      pgAppID,
		})
		if err != nil {
			return fmt.Errorf("failed to update api key ip allowlist: %w", err)
		}
		if updated == 0 {
			return ErrApiKeyNotFound
		}
		if err := q.DeleteApiKeyChannelsByKey(ctx, pgdb.DeleteApiKeyChannelsByKeyParams{
			ID:    apiKeyID,
			AppID: pgAppID,
		}); err != nil {
			return fmt.Errorf("failed to clear api key channel scopes: %w", err)
		}
		if len(channelIDs) == 0 {
			return nil
		}
		inserted, err := q.InsertApiKeyChannels(ctx, pgdb.InsertApiKeyChannelsParams{
			ApiKeyID:   apiKeyID,
			AppID:      pgAppID,
			ChannelIds: channelIDs,
		})
		if err != nil {
			return fmt.Errorf("failed to insert api key channel scopes: %w", err)
		}
		if inserted != int64(len(channelIDs)) {
			return ErrChannelNotFound
		}
		return nil
	})
}
