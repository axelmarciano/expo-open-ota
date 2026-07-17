package services

import (
	"context"
	"encoding/json"
	"expo-open-ota/internal/bucket"
	"expo-open-ota/internal/cache"
	"expo-open-ota/internal/database/postgres/pgdb"
	"expo-open-ota/internal/types"
	update2 "expo-open-ota/internal/update"
	"expo-open-ota/internal/validation"
	"fmt"
	"log"
)

type UpdateRepository interface {
	MarkUpdateAsChecked(ctx context.Context, update types.Update) error
	GetUpdateDetails(ctx context.Context, appId string, branchName string, runtimeVersion string, updateId string) (types.UpdateDetails, error)
	GetUpdate(ctx context.Context, appId string, branchName string, runtimeVersion string, updateId string) (*types.Update, error)
	GetLatestUpdate(ctx context.Context, appId string, branchName string, runtimeVersion string, platform string) (*types.Update, error)
	GetUpdateType(ctx context.Context, update types.Update) (types.UpdateType, error)
	IsUpdateValid(ctx context.Context, update types.Update) (bool, error)
	CreateUpdate(ctx context.Context, appId string, updateId int64, branchName string, runtimeVersion string, platform string, commitHash string, message string) (*types.Update, error)
	CreateRollback(ctx context.Context, appId string, updateId int64, branchName string, runtimeVersion string, platform string, commitHash string) (*types.Update, error)
	GetUpdateByBranchNameAndRuntime(ctx context.Context, appId string, updateId int64, branchName string, runtimeVersion string) (pgdb.GetUpdateByBranchNameAndRuntimeRow, error)
	GetUpdatesByRunTimeVersionAndBranchName(ctx context.Context, appId string, runtimeVersion string, branchName string) ([]types.UpdateItem, error)
	RetrieveUpdateStoredMetadata(ctx context.Context, update types.Update) (*types.UpdateStoredMetadata, error)
	StoreUpdateUUIDInMetadata(ctx context.Context, update types.Update, updateUUID string) error
}

type UpdateService struct {
	updateRepo UpdateRepository
	bucket     bucket.Bucket
}

func NewUpdateService(updateRepo UpdateRepository, bucket bucket.Bucket) *UpdateService {
	return &UpdateService{
		updateRepo: updateRepo,
		bucket:     bucket,
	}
}
func (s *UpdateService) GetLatestUpdate(ctx context.Context, appId string, branchName string, runtimeVersion string, platform string) (*types.Update, error) {
	cache := cache.GetCache()
	cacheKey := update2.ComputeLastUpdateCacheKey(appId, branchName, runtimeVersion, platform)
	if cachedValue := cache.Get(cacheKey); cachedValue != "" {
		var cachedUpdate types.Update
		if err := json.Unmarshal([]byte(cachedValue), &cachedUpdate); err == nil {
			return &cachedUpdate, nil
		}
		log.Printf("Warning: failed to unmarshal cached update for key %s", cacheKey)
	}
	latestUpdate, err := s.updateRepo.GetLatestUpdate(ctx, appId, branchName, runtimeVersion, platform)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve latest update from store: %w", err)
	}
	if latestUpdate == nil {
		return nil, nil
	}
	cacheValue, err := json.Marshal(latestUpdate)
	if err == nil {
		ttl := 1800
		if setErr := cache.Set(cacheKey, string(cacheValue), &ttl); setErr != nil {
			log.Printf("Warning: failed to write update to cache: %v", setErr)
		}
	} else {
		log.Printf("Warning: failed to marshal update for caching: %v", err)
	}
	return latestUpdate, nil
}

func (s *UpdateService) GetUpdateDetails(ctx context.Context, appId string, branchName string, runtimeVersion string, updateId string) (types.UpdateDetails, error) {
	if err := validation.Name("branchName", branchName); err != nil {
		return types.UpdateDetails{}, err
	}
	if err := validation.Name("runtimeVersion", runtimeVersion); err != nil {
		return types.UpdateDetails{}, err
	}
	if err := validation.Name("updateId", updateId); err != nil {
		return types.UpdateDetails{}, err
	}
	return s.updateRepo.GetUpdateDetails(ctx, appId, branchName, runtimeVersion, updateId)
}

func (s *UpdateService) GetUpdatesByRunTimeVersionAndBranchName(ctx context.Context, appId string, runtimeVersion string, branchName string) ([]types.UpdateItem, error) {
	if err := validation.Name("branchName", branchName); err != nil {
		return nil, err
	}
	if err := validation.Name("runtimeVersion", runtimeVersion); err != nil {
		return nil, err
	}
	return s.updateRepo.GetUpdatesByRunTimeVersionAndBranchName(ctx, appId, runtimeVersion, branchName)
}
