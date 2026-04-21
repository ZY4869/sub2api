package service

import (
	"context"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/model"
	"github.com/Wei-Shaw/sub2api/internal/pkg/protocolruntime"
)

func recordPublicModelProjectionSource(source string) {
	protocolruntime.RecordPublicModelProjection(strings.TrimSpace(source))
}

func filterAPIKeyPublicEntriesByChannel(
	channel *model.Channel,
	platform string,
	entries []APIKeyPublicModelEntry,
) []APIKeyPublicModelEntry {
	if channel == nil || !channel.RestrictModels || len(entries) == 0 {
		return entries
	}
	filtered := make([]APIKeyPublicModelEntry, 0, len(entries))
	for _, entry := range entries {
		requestedModel := strings.TrimSpace(firstNonEmptyString(entry.AliasID, entry.PublicID, entry.SourceID))
		if requestedModel == "" {
			continue
		}
		selectionModel := resolveChannelMappingTarget(channel, platform, requestedModel)
		if selectionModel == "" {
			selectionModel = requestedModel
		}
		if !channelAllowsModel(channel, platform, requestedModel, selectionModel) {
			continue
		}
		filtered = append(filtered, entry)
	}
	return filtered
}

func (s *GatewayService) filterPublicEntriesByActiveChannel(
	ctx context.Context,
	groupID int64,
	platform string,
	entries []APIKeyPublicModelEntry,
) []APIKeyPublicModelEntry {
	if s == nil || s.channelService == nil || s.channelService.repo == nil || groupID <= 0 {
		return entries
	}
	channel, err := s.channelService.repo.GetActiveByGroupID(ctx, groupID)
	if err != nil || channel == nil {
		return entries
	}
	return filterAPIKeyPublicEntriesByChannel(channel, platform, entries)
}
