package service

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

func (s *OpenAIGatewayService) publicCatalogPinnedAccount(ctx context.Context, groupID *int64, requestedModel string, excludedIDs map[int64]struct{}) *Account {
	if s == nil || s.accountRepo == nil {
		return nil
	}
	entry, ok := PublishedPublicCatalogEntryFromContext(ctx)
	if !ok || entry.SourceAccountID <= 0 {
		return nil
	}
	if excludedIDs != nil {
		if _, excluded := excludedIDs[entry.SourceAccountID]; excluded {
			return nil
		}
	}
	account, err := s.getSchedulableAccount(ctx, entry.SourceAccountID)
	if err != nil || account == nil || !account.IsSchedulable() {
		recordPublicCatalogPinnedUnavailable(ctx, entry, groupID, "account_not_schedulable", err)
		return nil
	}
	if !accountBoundToGroupID(account, groupID) {
		recordPublicCatalogPinnedUnavailable(ctx, entry, groupID, "account_not_in_group", nil)
		return nil
	}
	protocol := firstNonEmptyTrimmed(entry.SourceProtocol, OpenAIPlatformFromContext(ctx))
	resolved := ResolveProtocolGatewayInboundAccount(account, protocol)
	if resolved == nil || !resolved.IsSchedulable() || !isOpenAITextRuntimeAccount(resolved) {
		recordPublicCatalogPinnedUnavailable(ctx, entry, groupID, "resolved_account_not_schedulable", nil)
		return nil
	}
	sourceModel := firstNonEmptyTrimmed(entry.SourceModelID, requestedModel)
	if sourceModel != "" && !s.isModelSupportedByAccountWithContext(ctx, resolved, sourceModel) {
		recordPublicCatalogPinnedUnavailable(ctx, entry, groupID, "source_model_not_supported", nil)
		return nil
	}
	logger.FromContext(ctx).Info(
		"public model catalog openai pinned account selected",
		publicCatalogLogFields(ctx, entry, groupID, nil)...,
	)
	return resolved
}
