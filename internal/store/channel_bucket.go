package store

import (
	"context"
	"expo-open-ota/internal/bucket"
	"expo-open-ota/internal/providers"
	"expo-open-ota/internal/types"
	"fmt"
)

type BucketChannelStore struct {
	bucket bucket.Bucket
}

func NewBucketChannelStore(bucket bucket.Bucket) *BucketChannelStore {
	return &BucketChannelStore{
		bucket: bucket,
	}
}

func (s *BucketChannelStore) InsertChannel(ctx context.Context, appId string, branchId *int64, channelName string) (int64, error) {
	return 0, ErrNotSupportedInStatelessMode
}

func (s *BucketChannelStore) DeleteChannel(ctx context.Context, channelName string, appId string) error {
	return ErrNotSupportedInStatelessMode
}

func (s *BucketChannelStore) GetChannelNameByBranchName(ctx context.Context, appId string, branchName string) ([]string, error) {
	return nil, ErrNotSupportedInStatelessMode
}

func (s *BucketChannelStore) GetChannels(ctx context.Context, appId string) ([]types.ChannelMapping, error) {
	allChannels, err := providers.FetchExpoChannels(appId)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch expo channels: %w", err)
	}
	branchesMapping, err := providers.FetchExpoBranchesMapping(appId)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch expo branches mapping: %w", err)
	}
	var channels []types.ChannelMapping
	for _, channel := range allChannels {
		var branchName *string
		var branchId *string
		for _, mapping := range branchesMapping {
			if mapping.ChannelName != nil && *mapping.ChannelName == channel.Name {
				branchName = &mapping.BranchName
				branchId = &mapping.BranchId
				break
			}
		}
		channels = append(channels, types.ChannelMapping{
			ReleaseChannelId:   channel.Id,
			ReleaseChannelName: channel.Name,
			BranchName:         branchName,
			BranchId:           branchId,
		})
	}
	return channels, nil
}

func (s *BucketChannelStore) GetChannelBranchMapping(ctx context.Context, appId string, channelName string) (*providers.ExpoChannelMapping, error) {
	return providers.FetchExpoChannelMapping(appId, channelName)
}
