package store

import (
	"context"
	"expo-open-ota/internal/types"
)

// BucketRolloutStore is the stateless-mode implementation of the rollout repository.
// Progressive rollouts are a control-plane (Postgres) feature, so every method refuses
// with ErrNotSupportedInStatelessMode. Handlers translate this into a 400 telling the
// operator that rollouts require the database control plane.
type BucketRolloutStore struct{}

func NewBucketRolloutStore() *BucketRolloutStore {
	return &BucketRolloutStore{}
}

func (s *BucketRolloutStore) CreateChannelRollout(ctx context.Context, id string, appId string, channelName string, rolloutBranchName string, percentage int) (int64, error) {
	return 0, ErrNotSupportedInStatelessMode
}

func (s *BucketRolloutStore) GetChannelRollout(ctx context.Context, appId string, channelName string) (*types.ChannelRollout, error) {
	return nil, ErrNotSupportedInStatelessMode
}

func (s *BucketRolloutStore) UpdateChannelRolloutPercentage(ctx context.Context, appId string, channelName string, percentage int) (int64, error) {
	return 0, ErrNotSupportedInStatelessMode
}

func (s *BucketRolloutStore) DeleteChannelRollout(ctx context.Context, appId string, channelName string) (int64, error) {
	return 0, ErrNotSupportedInStatelessMode
}

func (s *BucketRolloutStore) PromoteChannelRollout(ctx context.Context, appId string, channelName string) (int64, error) {
	return 0, ErrNotSupportedInStatelessMode
}

func (s *BucketRolloutStore) GetChannelRolloutsByBranch(ctx context.Context, appId string, branchName string) ([]string, error) {
	return nil, ErrNotSupportedInStatelessMode
}

func (s *BucketRolloutStore) GetActiveRolloutUpdates(ctx context.Context, appId string, branchName string, runtimeVersion string) ([]types.RolloutUpdate, error) {
	return nil, ErrNotSupportedInStatelessMode
}

func (s *BucketRolloutStore) SetUpdateRolloutPercentage(ctx context.Context, appId string, branchName string, runtimeVersion string, percentage int) (int64, error) {
	return 0, ErrNotSupportedInStatelessMode
}

func (s *BucketRolloutStore) ClearUpdateRollout(ctx context.Context, appId string, branchName string, runtimeVersion string) (int64, error) {
	return 0, ErrNotSupportedInStatelessMode
}
