package service

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/model"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

var (
	ErrChannelNotFound      = infraerrors.NotFound("CHANNEL_NOT_FOUND", "channel not found")
	ErrChannelAlreadyExists = infraerrors.Conflict("CHANNEL_ALREADY_EXISTS", "channel already exists")
	ErrChannelGroupConflict = infraerrors.Conflict("CHANNEL_GROUP_CONFLICT", "one or more groups are already bound to another channel")
)

type ChannelListFilters struct {
	Status string
	Search string
}

type ChannelRepository interface {
	List(ctx context.Context, params pagination.PaginationParams, filters ChannelListFilters) ([]*model.Channel, *pagination.PaginationResult, error)
	GetByID(ctx context.Context, id int64) (*model.Channel, error)
	GetActiveByGroupID(ctx context.Context, groupID int64) (*model.Channel, error)
	Create(ctx context.Context, channel *model.Channel) (*model.Channel, error)
	Update(ctx context.Context, channel *model.Channel) (*model.Channel, error)
	Delete(ctx context.Context, id int64) error
}

type ChannelService struct {
	repo ChannelRepository
}

func NewChannelService(repo ChannelRepository) *ChannelService {
	return &ChannelService{repo: repo}
}

func (s *ChannelService) List(ctx context.Context, params pagination.PaginationParams, filters ChannelListFilters) ([]*model.Channel, *pagination.PaginationResult, error) {
	return s.repo.List(ctx, params, filters)
}

func (s *ChannelService) GetByID(ctx context.Context, id int64) (*model.Channel, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *ChannelService) Create(ctx context.Context, channel *model.Channel) (*model.Channel, error) {
	if err := channel.Validate(); err != nil {
		return nil, err
	}
	return s.repo.Create(ctx, channel)
}

func (s *ChannelService) Update(ctx context.Context, channel *model.Channel) (*model.Channel, error) {
	if err := channel.Validate(); err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, channel)
}

func (s *ChannelService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}
