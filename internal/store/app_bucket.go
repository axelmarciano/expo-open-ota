package store

import (
	"context"
	"expo-open-ota/config"
	"expo-open-ota/internal/bucket"

	"fmt"
)

type BucketAppStore struct {
	bucket bucket.Bucket
}

func NewBucketAppStore(bucket bucket.Bucket) *BucketAppStore {
	return &BucketAppStore{
		bucket: bucket,
	}
}

func (s *BucketAppStore) GetApps(ctx context.Context) ([]config.AppDescriptor, error) {
	return config.ListApps(), nil
}

func (s *BucketAppStore) GetAppByID(ctx context.Context, id string) (config.AppConfig, error) {
	app, err := config.GetAppConfig(id)
	if err != nil {
		return config.AppConfig{}, fmt.Errorf("app not found: %w", err)
	}
	if app == nil {
		return config.AppConfig{}, fmt.Errorf("app not found")
	}
	return *app, nil
}

func (s *BucketAppStore) InsertApp(ctx context.Context, app InsertAppParameters) (string, error) {
	return "", ErrNotSupportedInStatelessMode
}

func (s *BucketAppStore) DeleteAppByID(ctx context.Context, id string) error {
	return ErrNotSupportedInStatelessMode
}

func (s *BucketAppStore) UpdateAppNameByID(ctx context.Context, id string, newName string) error {
	return ErrNotSupportedInStatelessMode
}
