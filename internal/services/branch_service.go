package services

import (
	"context"
	"expo-open-ota/internal/bucket"
	"expo-open-ota/internal/database/postgres/pgdb"
	"expo-open-ota/internal/store"
	"expo-open-ota/internal/types"
	"fmt"
	"strconv"
)

type BranchService struct {
	branchRepo  BranchRepository
	channelRepo ChannelRepository
	updateRepo  UpdateRepository
	bucket      bucket.Bucket
}

type BranchRepository interface {
	InsertBranch(ctx context.Context, branch pgdb.InsertBranchParams) (int64, error)
	UpsertBranchAndRuntimeVersion(ctx context.Context, appId string, branchName string, runtimeVersion string) error
	GetUpdatedMetadataByBranchName(ctx context.Context, appId string, branchName string) ([]pgdb.GetUpdatesMetadataByBranchNameRow, error)
	DeleteBranchByName(ctx context.Context, appId string, branchName string) error
	GetBranches(ctx context.Context, appId string) ([]types.BranchMapping, error)
	GetRuntimeVersionsWithUpdateStats(ctx context.Context, appId string, branchName string) ([]types.RuntimeVersionWithStats, error)
	UpdateChannelBranchMapping(ctx context.Context, appId string, channelId string, branchId string) error
	CreateRuntimeVersion(ctx context.Context, appId string, version string) (int64, error)
	GetBranchByName(ctx context.Context, appId string, branchName string) (int64, error)
}

func NewBranchService(branchRepo BranchRepository, channelRepo ChannelRepository, updateRepo UpdateRepository, bucket bucket.Bucket) *BranchService {
	return &BranchService{
		branchRepo:  branchRepo,
		channelRepo: channelRepo,
		updateRepo:  updateRepo,
		bucket:      bucket,
	}
}

func (s *BranchService) CreateBranch(ctx context.Context, appId string, branchName string) (int64, error) {
	pgAppID := store.ToPgUUID(appId)
	if branchName == "" {
		return 0, fmt.Errorf("branch name is required")
	}
	branchId, err := s.branchRepo.InsertBranch(ctx, pgdb.InsertBranchParams{
		AppID: pgAppID,
		Name:  branchName,
	})
	if err != nil {
		return 0, err
	}
	return branchId, nil
}

func (s *BranchService) DeleteBranch(ctx context.Context, branchName string, appId string) error {
	channels, err := s.channelRepo.GetChannelNameByBranchName(ctx, appId, branchName)
	if err != nil {
		return fmt.Errorf("failed to validate branch dependencies: %w", err)
	}
	if len(channels) > 0 {
		return &store.ErrBranchHasActiveChannels{
			BranchName:   branchName,
			ChannelNames: channels,
		}
	}
	rows, err := s.branchRepo.GetUpdatedMetadataByBranchName(ctx, appId, branchName)
	if err != nil {
		return fmt.Errorf("failed to retrieve updates linked to the branch from database: %w", err)
	}
	err = s.branchRepo.DeleteBranchByName(ctx, appId, branchName)
	if err != nil {
		return err
	}
	go func(bucketRows []pgdb.GetUpdatesMetadataByBranchNameRow) {
		for _, row := range bucketRows {
			err := s.bucket.DeleteUpdateFolder(appId, branchName, row.RuntimeVersion, strconv.FormatInt(row.ID, 10))
			if err != nil {
				fmt.Printf("failed to delete update files for update %d: %v\n", row.ID, err)
			}
		}
	}(rows)
	return nil
}

func (s *BranchService) GetBranches(ctx context.Context, appId string) ([]types.BranchMapping, error) {
	return s.branchRepo.GetBranches(ctx, appId)
}

func (s *BranchService) GetRuntimeVersionsWithUpdateStats(ctx context.Context, appId string, branchName string) ([]types.RuntimeVersionWithStats, error) {
	return s.branchRepo.GetRuntimeVersionsWithUpdateStats(ctx, appId, branchName)
}

func (s *BranchService) UpdateChannelBranchMapping(ctx context.Context, appId string, channelId string, branchId string) error {
	return s.branchRepo.UpdateChannelBranchMapping(ctx, appId, channelId, branchId)
}

func (s *BranchService) UpsertBranchAndRuntimeVersion(ctx context.Context, appId string, branchName string, runtimeVersion string) error {
	return s.branchRepo.UpsertBranchAndRuntimeVersion(ctx, appId, branchName, runtimeVersion)
}
