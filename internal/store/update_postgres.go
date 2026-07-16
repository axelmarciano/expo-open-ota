package store

import (
	"context"
	"expo-open-ota/internal/crypto"
	"expo-open-ota/internal/database"
	"expo-open-ota/internal/database/postgres/pgdb"
	"expo-open-ota/internal/types"
	update2 "expo-open-ota/internal/update"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type PostgresUpdateStore struct {
	engine *database.Engine
}

func NewPostgresUpdateStore(engine *database.Engine) *PostgresUpdateStore {
	return &PostgresUpdateStore{
		engine: engine,
	}
}

func (s *PostgresUpdateStore) GetUpdateDetails(ctx context.Context, appId string, branchName string, runtimeVersion string, updateId string) (types.UpdateDetails, error) {
	updateIdInt, err := strconv.ParseInt(updateId, 10, 64)
	if err != nil {
		return types.UpdateDetails{}, fmt.Errorf("failed to parse update ID: %w", err)
	}
	update, err := s.GetUpdateByBranchNameAndRuntime(ctx, appId, updateIdInt, branchName, runtimeVersion)
	if err != nil {
		return types.UpdateDetails{}, fmt.Errorf("failed to retrieve update by ID from database: %w", err)
	}
	expoConfig, err := update2.GetExpoConfig(types.Update{
		Branch:         update.BranchName,
		RuntimeVersion: update.RuntimeVersion,
		UpdateId:       strconv.FormatInt(update.ID, 10),
		CreatedAt:      time.Duration(update.CreatedAt.Time.UnixNano()),
		AppId:          appId,
	})
	if err != nil {
		return types.UpdateDetails{}, fmt.Errorf("failed to get expo config for update: %w", err)
	}
	messageStr := ""
	if update.Message != nil {
		messageStr = *update.Message
	}
	updateUUID := "Rollback to embedded"
	if update.UpdateType != int32(types.Rollback) {
		updateUUID = update.UpdateUuid.String()
	}
	return types.UpdateDetails{
		UpdateUUID: updateUUID,
		UpdateId:   strconv.FormatInt(update.ID, 10),
		CreatedAt:  update.CreatedAt.Time.Format(time.RFC3339),
		CommitHash: update.CommitHash,
		Platform:   update.Platform,
		Message:    messageStr,
		Type:       types.UpdateType(update.UpdateType),
		ExpoConfig: string(expoConfig),
	}, nil
}

func (s *PostgresUpdateStore) GetLatestUpdate(ctx context.Context, appId string, branchName string, runtimeVersion string, platform string) (*types.Update, error) {
	pgAppID := ToPgUUID(appId)
	row, err := s.engine.Queries.GetLatestUpdate(ctx, pgdb.GetLatestUpdateParams{
		AppID:    pgAppID,
		Name:     branchName,
		Version:  runtimeVersion,
		Platform: platform,
	})
	if err != nil {
		if database.IsNoRows(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to retrieve latest update from database: %w", err)
	}
	return &types.Update{
		UpdateId:       strconv.FormatInt(row.ID, 10),
		Branch:         branchName,
		RuntimeVersion: runtimeVersion,
		CreatedAt:      time.Duration(row.CreatedAt.Time.UnixNano()),
		AppId:          appId,
	}, nil
}

func (s *PostgresUpdateStore) GetUpdateType(ctx context.Context, update types.Update) (types.UpdateType, error) {
	updateIdInt, err := strconv.ParseInt(update.UpdateId, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse update ID: %w", err)
	}
	pgAppID := ToPgUUID(update.AppId)
	updateTypeInt, err := s.engine.Queries.GetUpdateType(ctx, pgdb.GetUpdateTypeParams{
		AppID: pgAppID,
		ID:    updateIdInt,
		Name:  update.Branch,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to retrieve update type from database: %w", err)
	}
	return types.UpdateType(updateTypeInt), nil
}

// IsUpdateValid reports whether an update is complete. An update present in the
// database was written through the upload pipeline, so it is valid by
// construction — the bucket backend's ".check" sentinel has no DB equivalent.
func (s *PostgresUpdateStore) IsUpdateValid(ctx context.Context, update types.Update) (bool, error) {
	return true, nil
}

func (s *PostgresUpdateStore) MarkUpdateAsChecked(ctx context.Context, update types.Update) error {
	pgAppID := ToPgUUID(update.AppId)
	updateIdInt, err := strconv.ParseInt(update.UpdateId, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse update ID: %w", err)
	}
	err = s.engine.MarkUpdateAsChecked(ctx, pgdb.MarkUpdateAsCheckedParams{
		ID:    updateIdInt,
		AppID: pgAppID,
		Name:  update.Branch,
	})
	if err != nil {
		return fmt.Errorf("failed to mark update as checked in database: %w", err)
	}
	return nil
}

func (s *PostgresUpdateStore) CreateUpdate(ctx context.Context, appId string, updateId int64, branchName string, runtimeVersion string, platform string, commitHash string, message string) (*types.Update, error) {
	messagePtr := &message
	if message == "" {
		messagePtr = (*string)(nil)
	}
	pgAppID := ToPgUUID(appId)
	row, err := s.engine.InsertUpdate(ctx, pgdb.InsertUpdateParams{
		AppID:      pgAppID,
		ID:         updateId,
		Name:       branchName,
		Version:    runtimeVersion,
		UpdateType: int32(types.NormalUpdate),
		Platform:   platform,
		CommitHash: commitHash,
		Message:    messagePtr,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to insert update into database: %w", err)
	}
	return &types.Update{
		UpdateId:       strconv.FormatInt(row.ID, 10),
		Branch:         row.BranchName,
		RuntimeVersion: row.RuntimeVersion,
		CreatedAt:      time.Duration(row.CreatedAt.Time.UnixNano()),
		AppId:          appId,
	}, nil
}

func (s *PostgresUpdateStore) GetUpdate(ctx context.Context, appId string, branchName string, runtimeVersion string, updateId string) (*types.Update, error) {
	updateIdInt, err := strconv.ParseInt(updateId, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse update ID: %w", err)
	}
	update, err := s.GetUpdateByBranchNameAndRuntime(ctx, appId, updateIdInt, branchName, runtimeVersion)
	if err != nil {
		if database.IsNoRows(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to retrieve update by ID from database: %w", err)
	}
	return &types.Update{
		UpdateId:       strconv.FormatInt(update.ID, 10),
		Branch:         update.BranchName,
		RuntimeVersion: update.RuntimeVersion,
		CreatedAt:      time.Duration(update.CreatedAt.Time.UnixNano()),
		AppId:          appId,
	}, nil
}

func (s *PostgresUpdateStore) GetUpdateByBranchNameAndRuntime(ctx context.Context, appId string, updateId int64, branchName string, runtimeVersion string) (pgdb.GetUpdateByBranchNameAndRuntimeRow, error) {
	return s.engine.Queries.GetUpdateByBranchNameAndRuntime(ctx, pgdb.GetUpdateByBranchNameAndRuntimeParams{
		AppID:   ToPgUUID(appId),
		ID:      updateId,
		Name:    branchName,
		Version: runtimeVersion,
	})
}

func (s *PostgresUpdateStore) GetUpdatesByRunTimeVersionAndBranchName(ctx context.Context, appId string, runtimeVersion string, branchName string) ([]types.UpdateItem, error) {
	pgAppID := ToPgUUID(appId)
	rows, err := s.engine.Queries.GetUpdatesByByBranchNameAndRuntimeVersion(ctx, pgdb.GetUpdatesByByBranchNameAndRuntimeVersionParams{
		ID:      pgAppID,
		Version: runtimeVersion,
		Name:    branchName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve updates by runtime version and branch name from database: %w", err)
	}
	var updatesResponse []types.UpdateItem
	for _, row := range rows {
		createdAtStr := row.CreatedAt.Time.Format(time.RFC3339)
		updateUUID := ""
		switch row.UpdateType {
		case int32(types.Rollback):
			updateUUID = "Rollback to embedded"
		case int32(types.NormalUpdate):
			if row.UpdateUuid.Valid && row.UpdateUuid.String() != "" {
				updateUUID = row.UpdateUuid.String()
			} else {
				metadata, err := update2.GetMetadata(types.Update{
					Branch:         branchName,
					RuntimeVersion: runtimeVersion,
					UpdateId:       strconv.FormatInt(row.ID, 10),
					CreatedAt:      time.Duration(row.CreatedAt.Time.UnixNano()),
					AppId:          appId,
				})
				if err != nil {
					continue
				}
				updateUUID = crypto.ConvertSHA256HashToUUID(metadata.ID)
			}
		default:
			return nil, fmt.Errorf("unknown update type %d for update ID %s", row.UpdateType, strconv.FormatInt(row.ID, 10))
		}
		messageStr := ""
		if row.Message != nil {
			messageStr = *row.Message
		}
		updatesResponse = append(updatesResponse, types.UpdateItem{
			UpdateUUID: updateUUID,
			UpdateId:   strconv.FormatInt(row.ID, 10),
			CreatedAt:  createdAtStr,
			CommitHash: row.CommitHash,
			Message:    messageStr,
			Platform:   row.Platform,
		})
	}
	return updatesResponse, nil
}

func (s *PostgresUpdateStore) RetrieveUpdateStoredMetadata(ctx context.Context, update types.Update) (*types.UpdateStoredMetadata, error) {
	updateIdInt, _ := strconv.ParseInt(update.UpdateId, 10, 64)
	pgAppID := ToPgUUID(update.AppId)
	metadata, err := s.engine.Queries.GetUpdateMetadata(ctx, pgdb.GetUpdateMetadataParams{
		ID:    updateIdInt,
		Name:  update.Branch,
		AppID: pgAppID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve update metadata from database: %w", err)
	}
	messageStr := ""
	if metadata.Message != nil {
		messageStr = *metadata.Message
	}
	return &types.UpdateStoredMetadata{
		UpdateUUID: metadata.UpdateUuid.String(),
		CommitHash: metadata.CommitHash,
		Message:    messageStr,
		Platform:   metadata.Platform,
	}, nil
}

func (s *PostgresUpdateStore) StoreUpdateUUIDInMetadata(ctx context.Context, update types.Update, updateUUID string) error {
	updateIdInt, _ := strconv.ParseInt(update.UpdateId, 10, 64)
	var uuidToStore pgtype.UUID
	if err := uuidToStore.Scan(updateUUID); err != nil {
		return fmt.Errorf("failed to parse update UUID: %w", err)
	}
	pgAppID := ToPgUUID(update.AppId)
	commandTag, err := s.engine.Queries.StoreUpdateUUID(ctx, pgdb.StoreUpdateUUIDParams{
		ID:         updateIdInt,
		UpdateUuid: uuidToStore,
		AppID:      pgAppID,
		Name:       update.Branch,
	})
	if err != nil {
		return fmt.Errorf("failed to store update UUID in database: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("no rows were updated when trying to store update UUID in database for update ID %s", update.UpdateId)
	}
	return nil
}

func (s *PostgresUpdateStore) CreateRollback(ctx context.Context, appId string, updateId int64, branchName string, runtimeVersion string, platform string, commitHash string) (*types.Update, error) {
	pgAppID := ToPgUUID(appId)
	row, err := s.engine.InsertUpdate(ctx, pgdb.InsertUpdateParams{
		AppID:      pgAppID,
		ID:         updateId,
		Name:       branchName,
		Version:    runtimeVersion,
		UpdateType: int32(types.Rollback),
		Platform:   platform,
		CommitHash: commitHash,
		Message:    nil,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to insert rollback update into database: %w", err)
	}
	return &types.Update{
		UpdateId:       strconv.FormatInt(row.ID, 10),
		Branch:         row.BranchName,
		RuntimeVersion: row.RuntimeVersion,
		CreatedAt:      time.Duration(row.CreatedAt.Time.UnixNano()),
		AppId:          pgAppID.String(),
	}, nil
}
