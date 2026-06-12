package store

import (
	"context"
	"expo-open-ota/internal/crypto"
	"expo-open-ota/internal/database"
	"expo-open-ota/internal/database/postgres/pgdb"
	"expo-open-ota/internal/types"
	"fmt"
)

type PostgresAuthStore struct {
	engine *database.Engine
}

func NewPostgresAuthStore(engine *database.Engine) *PostgresAuthStore {
	return &PostgresAuthStore{
		engine: engine,
	}
}

func (s *PostgresAuthStore) ValidateAuth(ctx context.Context, appId string, auth types.Auth) error {
	pgAppID := ToPgUUID(appId)
	token := auth.Token
	if token == nil {
		return fmt.Errorf("no token provided in auth")
	}
	hashedToken, err := crypto.HashPlaintextAPIKey(*token)
	if err != nil {
		return fmt.Errorf("failed to hash API key: %w", err)
	}
	isValid, err := s.engine.Queries.ValidateAndTouchAuth(ctx, pgdb.ValidateAndTouchAuthParams{
		AppID:     pgAppID,
		HashedKey: hashedToken,
	})
	if err != nil {
		return err
	}
	if !isValid {
		return fmt.Errorf("invalid API key")
	}
	return nil
}

func (s *PostgresAuthStore) InsertApiKey(ctx context.Context, appId string, name string, hint string, hashedKey string) error {
	pgAppID := ToPgUUID(appId)
	return s.engine.Queries.InsertApiKey(ctx, pgdb.InsertApiKeyParams{
		AppID:     pgAppID,
		Name:      name,
		Hint:      hint,
		HashedKey: hashedKey,
	})
}

func (s *PostgresAuthStore) GetApiKeysMetadataByAppID(ctx context.Context, appId string) ([]pgdb.GetApiKeysMetadataByAppIDRow, error) {
	pgAppID := ToPgUUID(appId)
	return s.engine.Queries.GetApiKeysMetadataByAppID(ctx, pgAppID)
}

func (s *PostgresAuthStore) RevokeApiKeyByID(ctx context.Context, apiKeyId int64, appId string) error {
	pgAppID := ToPgUUID(appId)
	commandTag, err := s.engine.Queries.RevokeApiKeyByID(ctx, pgdb.RevokeApiKeyByIDParams{
		ID:    apiKeyId,
		AppID: pgAppID,
	})
	if err != nil {
		return fmt.Errorf("failed to execute revoke query: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return &ErrResourceNotFound{
			Resource:   "api_key",
			Identifier: fmt.Sprintf("id: %d, appId: %s", apiKeyId, appId),
		}
	}
	return nil
}
