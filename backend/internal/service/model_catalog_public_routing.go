package service

import (
	"context"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/protocolruntime"
	"go.uber.org/zap"
)

func (s *GatewayService) publicCatalogPinnedAccount(ctx context.Context, groupID *int64, requestedModel string, excludedIDs map[int64]struct{}) *Account {
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
	account, err := s.accountRepo.GetByID(ctx, entry.SourceAccountID)
	if err != nil || account == nil {
		recordPublicCatalogPinnedUnavailable(ctx, entry, groupID, "account_not_found", err)
		return nil
	}
	if !accountBoundToGroupID(account, groupID) {
		recordPublicCatalogPinnedUnavailable(ctx, entry, groupID, "account_not_in_group", nil)
		return nil
	}
	if !account.IsSchedulable() {
		recordPublicCatalogPinnedUnavailable(ctx, entry, groupID, "account_not_schedulable", nil)
		return nil
	}
	protocol := firstNonEmptyTrimmed(entry.SourceProtocol, RoutingPlatformForAccount(account))
	resolved := ResolveProtocolGatewayInboundAccount(account, protocol)
	if resolved == nil || !resolved.IsSchedulable() {
		recordPublicCatalogPinnedUnavailable(ctx, entry, groupID, "resolved_account_not_schedulable", nil)
		return nil
	}
	sourceModel := firstNonEmptyTrimmed(entry.SourceModelID, requestedModel)
	if sourceModel != "" && !s.isModelSupportedByAccountWithContext(ctx, resolved, sourceModel) {
		recordPublicCatalogPinnedUnavailable(ctx, entry, groupID, "source_model_not_supported", nil)
		return nil
	}
	logger.FromContext(ctx).Info(
		"public model catalog pinned account selected",
		publicCatalogLogFields(ctx, entry, groupID, nil)...,
	)
	return resolved
}

func recordPublicCatalogPinnedUnavailable(ctx context.Context, entry *PublishedPublicCatalogEntry, groupID *int64, reason string, err error) {
	protocolruntime.RecordBillingResolverFallback("public_catalog_pinned_unavailable")
	fields := []zap.Field{
		zap.String("reason", reason),
	}
	fields = append(fields, publicCatalogLogFields(ctx, entry, groupID, nil)...)
	if err != nil {
		fields = append(fields, zap.Error(err))
	}
	logger.FromContext(ctx).Warn("public model catalog pinned account unavailable", fields...)
}

func ApplyPublicCatalogEntryToParsedRequest(ctx context.Context, parsed *ParsedRequest, entry *PublishedPublicCatalogEntry) context.Context {
	if parsed == nil || entry == nil {
		return ctx
	}
	ctx = AttachPublishedPublicCatalogEntry(ctx, entry)
	sourceModel := strings.TrimSpace(entry.SourceModelID)
	if sourceModel == "" {
		return ctx
	}
	if parsed.RawModel == "" {
		parsed.RawModel = parsed.Model
	}
	parsed.Model = sourceModel
	parsed.Capability.RequestedModelNormalized = sourceModel
	return ctx
}
