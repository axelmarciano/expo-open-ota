package store

import (
	"bytes"
	"context"
	"encoding/json"
	bucket2 "expo-open-ota/internal/bucket"
	"expo-open-ota/internal/crypto"
	"expo-open-ota/internal/database/postgres/pgdb"
	"expo-open-ota/internal/helpers"
	"expo-open-ota/internal/types"

	update2 "expo-open-ota/internal/update"
	"fmt"
	"sort"
	"strconv"
	"time"
)

type BucketUpdateStore struct {
	bucket bucket2.Bucket
}

func NewBucketUpdateStore(bucket bucket2.Bucket) *BucketUpdateStore {
	return &BucketUpdateStore{
		bucket: bucket,
	}
}

// GetLatestUpdate returns the newest complete update for the platform, or nil
// when the branch has none yet.
//
// Callers that need the cached answer must go through UpdateService, which owns
// the lastUpdate cache — this is the uncached read underneath it.
func (s *BucketUpdateStore) GetLatestUpdate(ctx context.Context, appId string, branchName string, runtimeVersion string, platform string) (*types.Update, error) {
	updates, err := s.allUpdatesForRuntimeVersion(appId, branchName, runtimeVersion, platform)
	if err != nil {
		return nil, err
	}
	for _, update := range updates {
		if update2.IsUpdateValid(update) {
			return &update, nil
		}
	}
	return nil, nil
}

// allUpdatesForRuntimeVersion lists the platform's updates, newest first.
//
// It resolves the bucket through the singleton rather than s.bucket on purpose:
// filterByPlatform below reads each update's metadata via internal/update, which
// is singleton-backed throughout. Mixing the two here would list updates from
// one bucket and read their metadata from another whenever they diverge.
// Untangling that means moving internal/update off the singleton wholesale.
func (s *BucketUpdateStore) allUpdatesForRuntimeVersion(appId, branch, runtimeVersion, platform string) ([]types.Update, error) {
	updates, err := bucket2.GetBucket().GetUpdates(appId, branch, runtimeVersion)
	if err != nil {
		return nil, err
	}
	return sortNewestFirst(filterByPlatform(updates, platform)), nil
}

func filterByPlatform(updates []types.Update, platform string) []types.Update {
	filtered := make([]types.Update, 0)
	for _, update := range updates {
		storedMetadata, err := update2.RetrieveUpdateStoredMetadata(update)
		if err == nil && storedMetadata != nil && storedMetadata.Platform == platform {
			filtered = append(filtered, update)
		}
	}
	return filtered
}

func sortNewestFirst(updates []types.Update) []types.Update {
	sort.Slice(updates, func(i, j int) bool {
		return updates[i].CreatedAt > updates[j].CreatedAt
	})
	return updates
}

func (s *BucketUpdateStore) GetUpdateType(ctx context.Context, update types.Update) (types.UpdateType, error) {
	return update2.GetUpdateType(update), nil
}

func (s *BucketUpdateStore) IsUpdateValid(ctx context.Context, update types.Update) (bool, error) {
	return update2.IsUpdateValid(update), nil
}

func (s *BucketUpdateStore) MarkUpdateAsChecked(ctx context.Context, update types.Update) error {
	return update2.MarkUpdateAsChecked(update)
}

func (s *BucketUpdateStore) CreateUpdate(ctx context.Context, appId string, updateId int64, branchName string, runtimeVersion string, platform string, commitHash string, message string) (*types.Update, error) {
	fileUpdateMetadata := map[string]interface{}{
		"platform":   platform,
		"commitHash": commitHash,
	}
	if message != "" {
		fileUpdateMetadata["message"] = message
	}
	marshalledMetadata, err := json.Marshal(fileUpdateMetadata)
	if err != nil {
		return nil, fmt.Errorf("Error marshalling file update metadata: %w", err)
	}
	metadataReader := bytes.NewReader(marshalledMetadata)
	createdAt := helpers.NormalizeTimestampToDuration(updateId)
	err = s.bucket.UploadFileIntoUpdate(types.Update{
		AppId:          appId,
		Branch:         branchName,
		RuntimeVersion: runtimeVersion,
		UpdateId:       update2.ConvertUpdateTimestampToString(updateId),
		CreatedAt:      createdAt,
	}, "update-metadata.json", metadataReader)
	if err != nil {
		return nil, fmt.Errorf("Error uploading file update metadata: %w", err)
	}
	return &types.Update{
		UpdateId:       strconv.FormatInt(updateId, 10),
		Branch:         branchName,
		RuntimeVersion: runtimeVersion,
		CreatedAt:      createdAt,
		AppId:          appId,
	}, nil
}

func (s *BucketUpdateStore) GetUpdateDetails(ctx context.Context, appId string, branchName string, runtimeVersion string, updateId string) (types.UpdateDetails, error) {
	update, err := s.GetUpdate(ctx, appId, branchName, runtimeVersion, updateId)
	if err != nil {
		return types.UpdateDetails{}, fmt.Errorf("failed to fetch update: %w", err)
	}
	metadata, err := update2.GetMetadata(*update)
	if err != nil {
		return types.UpdateDetails{}, fmt.Errorf("failed to get update metadata: %w", err)
	}
	expoConfig, err := update2.GetExpoConfig(*update)
	if err != nil {
		return types.UpdateDetails{}, fmt.Errorf("failed to get expo config for update: %w", err)
	}
	numberUpdate, _ := strconv.ParseInt(update.UpdateId, 10, 64)
	storedMetadata, _ := update2.RetrieveUpdateStoredMetadata(*update)
	updateUUID := "Rollback to embedded"
	if update2.GetUpdateType(*update) != types.Rollback {
		updateUUID = storedMetadata.UpdateUUID
		if updateUUID == "" {
			updateUUID = crypto.ConvertSHA256HashToUUID(metadata.ID)
		}
	}
	return types.UpdateDetails{
		UpdateUUID: updateUUID,
		UpdateId:   update.UpdateId,
		CreatedAt:  helpers.NormalizeTimestamp(numberUpdate).Format(time.RFC3339),
		CommitHash: storedMetadata.CommitHash,
		Platform:   storedMetadata.Platform,
		Message:    storedMetadata.Message,
		Type:       update2.GetUpdateType(*update),
		ExpoConfig: string(expoConfig),
	}, nil
}

func (s *BucketUpdateStore) GetUpdatesByRunTimeVersionAndBranchName(ctx context.Context, appId string, runtimeVersion string, branchName string) ([]types.UpdateItem, error) {

	updates, err := s.bucket.GetUpdates(appId, branchName, runtimeVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch updates: %w", err)
	}

	var updatesResponse []types.UpdateItem
	for _, update := range updates {
		isValid := update2.IsUpdateValid(update)
		if !isValid {
			continue
		}
		numberUpdate, _ := strconv.ParseInt(update.UpdateId, 10, 64)
		storedMetadata, _ := update2.RetrieveUpdateStoredMetadata(update)
		updateType := update2.GetUpdateType(update)
		if updateType == types.Rollback {
			updatesResponse = append(updatesResponse, types.UpdateItem{
				UpdateUUID: "Rollback to embedded",
				UpdateId:   update.UpdateId,
				CreatedAt:  helpers.NormalizeTimestamp(numberUpdate).Format(time.RFC3339),
				CommitHash: storedMetadata.CommitHash,
				Platform:   storedMetadata.Platform,
				Message:    storedMetadata.Message,
			})
			continue
		}

		metadata, err := update2.GetMetadata(update)
		if err != nil {
			continue
		}
		updateUUID := storedMetadata.UpdateUUID
		if updateUUID == "" {
			updateUUID = crypto.ConvertSHA256HashToUUID(metadata.ID)
		}
		updatesResponse = append(updatesResponse, types.UpdateItem{
			UpdateUUID: updateUUID,
			UpdateId:   update.UpdateId,
			CreatedAt:  helpers.NormalizeTimestamp(numberUpdate).Format(time.RFC3339),
			CommitHash: storedMetadata.CommitHash,
			Platform:   storedMetadata.Platform,
			Message:    storedMetadata.Message,
		})
	}
	sort.Slice(updatesResponse, func(i, j int) bool {
		timeI, _ := time.Parse(time.RFC3339, updatesResponse[i].CreatedAt)
		timeJ, _ := time.Parse(time.RFC3339, updatesResponse[j].CreatedAt)
		return timeI.After(timeJ)
	})
	return updatesResponse, nil
}

func (s *BucketUpdateStore) GetUpdate(ctx context.Context, appId string, branchName string, runtimeVersion string, updateId string) (*types.Update, error) {
	return update2.GetUpdate(appId, branchName, runtimeVersion, updateId)
}

func (s *BucketUpdateStore) RetrieveUpdateStoredMetadata(ctx context.Context, update types.Update) (*types.UpdateStoredMetadata, error) {
	return update2.RetrieveUpdateStoredMetadata(update)
}

func (s *BucketUpdateStore) StoreUpdateUUIDInMetadata(ctx context.Context, update types.Update, updateUUID string) error {
	return update2.StoreUpdateUUIDInMetadata(update, updateUUID)
}

func (s *BucketUpdateStore) CreateRollback(ctx context.Context, appId string, updateId int64, branchName string, runtimeVersion string, platform string, commitHash string) (*types.Update, error) {
	return update2.CreateRollback(appId, updateId, platform, commitHash, runtimeVersion, branchName)
}

func (s *BucketUpdateStore) GetUpdateByBranchNameAndRuntime(ctx context.Context, appId string, updateId int64, branchName string, runtimeVersion string) (pgdb.GetUpdateByBranchNameAndRuntimeRow, error) {
	return pgdb.GetUpdateByBranchNameAndRuntimeRow{}, ErrNotSupportedInStatelessMode
}
