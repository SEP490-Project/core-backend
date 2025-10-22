package service

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/model"
	"errors"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type ChannelService struct {
	channelRepo irepository.GenericRepository[model.Channel]
}

// CreateChannel implements iservice.ChannelService.
func (c *ChannelService) CreateChannel(ctx context.Context, request *requests.CreateChannelRequest, uow irepository.UnitOfWork) (*responses.ChannelResponse, error) {
	zap.L().Info("CreateChannel called", zap.Any("request", request))

	channelRepo := uow.Channels()

	creatingModel := &model.Channel{
		ID:          uuid.New(),
		Name:        request.Name,
		Description: request.Description,
		HomePageURL: request.HomePageURL,
		IsActive:    request.IsActive,
	}

	if err := channelRepo.Add(ctx, creatingModel); err != nil {
		zap.L().Error("Failed to create channel", zap.Error(err))
		return nil, err
	}

	createdModel, err := channelRepo.GetByID(ctx, creatingModel.ID, nil)
	if err != nil {
		zap.L().Error("Failed to get created channel by ID", zap.Error(err))
		return nil, err
	}

	return responses.ChannelResponse{}.ToResponse(createdModel), nil
}

// DeleteChannel implements iservice.ChannelService.
func (c *ChannelService) DeleteChannel(ctx context.Context, channelID uuid.UUID, uow irepository.UnitOfWork) error {
	zap.L().Info("DeleteChannel called", zap.String("channel_id", channelID.String()))

	channelRepo := uow.Channels()

	channel, err := channelRepo.GetByID(ctx, channelID, nil)
	if err != nil {
		zap.L().Error("Failed to get channel by ID", zap.Error(err))
		return err
	} else if channel == nil {
		zap.L().Warn("Channel not found", zap.String("channel_id", channelID.String()))
		return errors.New("channel not found")
	}

	if err := channelRepo.Delete(ctx, channel); err != nil {
		zap.L().Error("Failed to delete channel", zap.Error(err))
		return err
	}

	return nil
}

// GetAllChannels implements iservice.ChannelService.
func (c *ChannelService) GetAllChannels(ctx context.Context) ([]responses.ChannelResponse, error) {
	zap.L().Info("GetAllChannels called")

	channels, _, err := c.channelRepo.GetAll(ctx, nil, nil, 0, 0)
	if err != nil {
		zap.L().Error("Failed to get all channels", zap.Error(err))
		return nil, err
	}

	return responses.ChannelResponse{}.ToListResponse(channels), nil
}

// GetChannelByID implements iservice.ChannelService.
func (c *ChannelService) GetChannelByID(ctx context.Context, channelID uuid.UUID) (*responses.ChannelResponse, error) {
	zap.L().Info("GetChannelByID called", zap.String("channel_id", channelID.String()))

	channel, err := c.channelRepo.GetByID(ctx, channelID, nil)
	if err != nil {
		zap.L().Error("Failed to get channel by ID", zap.Error(err))
		return nil, err
	}

	return responses.ChannelResponse{}.ToResponse(channel), nil
}

// UpdateChannel implements iservice.ChannelService.
func (c *ChannelService) UpdateChannel(
	ctx context.Context,
	channelID uuid.UUID,
	request *requests.UpdateChannelRequest,
	uow irepository.UnitOfWork,
) (*responses.ChannelResponse, error) {
	zap.L().Info("UpdateChannel called", zap.String("channel_id", channelID.String()), zap.Any("request", request))

	channelRepo := uow.Channels()

	channel, err := channelRepo.GetByID(ctx, channelID, nil)
	if err != nil {
		zap.L().Error("Failed to get channel by ID", zap.Error(err))
		return nil, err
	} else if channel == nil {
		zap.L().Warn("Channel not found", zap.String("channel_id", channelID.String()))
		return nil, errors.New("channel not found")
	}

	if request.Name != nil {
		channel.Name = *request.Name
	}
	if request.Description != nil {
		channel.Description = request.Description
	}
	if request.HomePageURL != nil {
		channel.HomePageURL = request.HomePageURL
	}
	if request.IsActive != nil && *request.IsActive != channel.IsActive {
		channel.IsActive = *request.IsActive
	}

	if err = channelRepo.Update(ctx, channel); err != nil {
		zap.L().Error("Failed to update channel", zap.Error(err))
		return nil, err
	}

	var updatedModel *model.Channel
	updatedModel, err = channelRepo.GetByID(ctx, channel.ID, nil)
	if err != nil {
		zap.L().Error("Failed to get updated channel by ID", zap.Error(err))
		return nil, err
	}

	return responses.ChannelResponse{}.ToResponse(updatedModel), nil
}

func NewChannelService(channelRepo irepository.GenericRepository[model.Channel]) iservice.ChannelService {
	return &ChannelService{
		channelRepo: channelRepo,
	}
}
