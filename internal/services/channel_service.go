package services

import (
	"context"
	"expo-open-ota/internal/providers"
	"expo-open-ota/internal/types"
	"fmt"
)

type ChannelService struct {
	branchRepo  BranchRepository
	channelRepo ChannelRepository
}

type ChannelRepository interface {
	InsertChannel(ctx context.Context, appId string, branchId *int64, channelName string) (int64, error)
	DeleteChannel(ctx context.Context, channelName string, appId string) error
	GetChannelNameByBranchName(ctx context.Context, appId string, branchName string) ([]string, error)
	GetChannels(ctx context.Context, appId string) ([]types.ChannelMapping, error)
	GetChannelBranchMapping(ctx context.Context, appId string, channelName string) (*providers.ExpoChannelMapping, error)
}

func NewChannelService(branchRepo BranchRepository, channelRepo ChannelRepository) *ChannelService {
	return &ChannelService{
		branchRepo:  branchRepo,
		channelRepo: channelRepo,
	}
}

func (s *ChannelService) CreateChannel(ctx context.Context, appId string, branchName *string, channelName string) (int64, error) {
	var branchIdPtr *int64
	if branchName != nil {
		branchId, err := s.branchRepo.GetBranchByName(ctx, appId, *branchName)
		if err != nil {
			return 0, fmt.Errorf("failed to map channel: target branch '%s' does not exist: %w", *branchName, err)
		}
		branchIdPtr = &branchId
	}
	channelId, err := s.channelRepo.InsertChannel(ctx, appId, branchIdPtr, channelName)
	if err != nil {
		return 0, err
	}
	return channelId, nil
}

func (s *ChannelService) DeleteChannel(ctx context.Context, channelName string, appId string) error {
	err := s.channelRepo.DeleteChannel(ctx, channelName, appId)
	if err != nil {
		return err
	}
	return nil
}

func (s *ChannelService) GetChannels(ctx context.Context, appId string) ([]types.ChannelMapping, error) {
	return s.channelRepo.GetChannels(ctx, appId)
}
