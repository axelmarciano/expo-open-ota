package store

import (
	"context"
	"expo-open-ota/internal/bucket"
	"expo-open-ota/internal/database/postgres/pgdb"
	"expo-open-ota/internal/providers"
	"expo-open-ota/internal/types"
	"fmt"
)

type BucketAuthStore struct {
	bucket bucket.Bucket
}

func NewBucketAuthStore(bucket bucket.Bucket) *BucketAuthStore {
	return &BucketAuthStore{
		bucket: bucket,
	}
}

func (s *BucketAuthStore) ValidateAuth(ctx context.Context, appId string, auth types.Auth) error {
	// ValidateExpoAuth(appId, ...) enforces that the caller's Expo session
	// matches the app identified by APP_ID — without the appId check,
	// FetchExpoUserAccountInformations alone would accept any authenticated
	// Expo user against any app (cross-tenant authz bypass).
	expoAccount, err := providers.ValidateExpoAuth(appId, auth)
	if err != nil || expoAccount == nil {
		return fmt.Errorf("Error validating expo auth: %w", err)
	}
	return nil
}

func (s *BucketAuthStore) InsertApiKey(ctx context.Context, appId string, name string, hint string, hashedKey string) error {
	return ErrNotSupportedInStatelessMode
}

func (s *BucketAuthStore) GetApiKeysMetadataByAppID(ctx context.Context, appId string) ([]pgdb.GetApiKeysMetadataByAppIDRow, error) {
	return []pgdb.GetApiKeysMetadataByAppIDRow{}, ErrNotSupportedInStatelessMode
}

func (s *BucketAuthStore) RevokeApiKeyByID(ctx context.Context, apiKeyId int64, appId string) error {
	return ErrNotSupportedInStatelessMode
}
