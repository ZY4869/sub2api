package service

import (
	"context"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/protocolruntime"
	"go.uber.org/zap"
)

const PublicCatalogModelUnavailableMessage = "Requested model is not published or not available for this API key"

type PublishedPublicCatalogEntry struct {
	EntryID          string
	PublicModelID    string
	SourceAccountID  int64
	BindingGroupID   int64
	SourceAlias      string
	SourceModelID    string
	SourceProtocol   string
	Currency         string
	RuntimePriceSpec PublicModelCatalogRuntimePriceSpec
	SalePriceDisplay PublicModelCatalogPriceDisplay
	Item             PublicModelCatalogItem
}

func (s *ModelCatalogService) ResolvePublishedPublicCatalogEntry(ctx context.Context, publicModelID string) (*PublishedPublicCatalogEntry, bool, error) {
	normalizedPublicID := NormalizeModelCatalogModelID(publicModelID)
	if s == nil || normalizedPublicID == "" {
		return nil, false, nil
	}
	published, active := s.activePublishedPublicModelCatalogSnapshot(ctx)
	if !active {
		return nil, false, nil
	}
	for _, item := range published.Snapshot.Items {
		if !publicModelCatalogItemMatchesPublicID(item, normalizedPublicID) {
			continue
		}
		return publishedPublicCatalogEntryFromItem(item), true, nil
	}
	for _, detail := range published.Details {
		if !publicModelCatalogItemMatchesPublicID(detail.Item, normalizedPublicID) {
			continue
		}
		return publishedPublicCatalogEntryFromItem(detail.Item), true, nil
	}
	return nil, false, nil
}

func (s *ModelCatalogService) PublishedPublicCatalogActive(ctx context.Context) bool {
	_, active := s.activePublishedPublicModelCatalogSnapshot(ctx)
	return active
}

func (s *ModelCatalogService) activePublishedPublicModelCatalogSnapshot(ctx context.Context) (*PublicModelCatalogPublishedSnapshot, bool) {
	if s == nil {
		return nil, false
	}
	published := s.loadPublishedPublicModelCatalogSnapshot(ctx)
	if published == nil {
		return nil, false
	}
	published = s.filterPublishedPublicModelCatalogSnapshot(ctx, published)
	if published == nil {
		return nil, false
	}
	return published, true
}

func (s *ModelCatalogService) filterPublishedPublicModelCatalogSnapshot(ctx context.Context, published *PublicModelCatalogPublishedSnapshot) *PublicModelCatalogPublishedSnapshot {
	cloned := clonePublicModelCatalogPublishedSnapshot(published)
	if cloned == nil {
		return nil
	}
	if len(cloned.Snapshot.Items) == 0 {
		return cloned
	}
	filteredItems := make([]PublicModelCatalogItem, 0, len(cloned.Snapshot.Items))
	filteredDetails := make(map[string]PublicModelCatalogDetail, len(cloned.Details))
	for _, item := range cloned.Snapshot.Items {
		if !s.publicModelCatalogItemRouteConfirmed(ctx, item) {
			continue
		}
		filteredItems = append(filteredItems, item)
		publicID := NormalizeModelCatalogModelID(firstNonEmptyTrimmed(item.PublicModelID, item.Model))
		if publicID == "" {
			continue
		}
		if detail, ok := cloned.Details[publicID]; ok {
			filteredDetails[publicID] = detail
		}
	}
	cloned.Snapshot.Items = filteredItems
	if len(filteredDetails) > 0 {
		cloned.Details = filteredDetails
	} else {
		cloned.Details = nil
	}
	return cloned
}

func publicModelCatalogItemMatchesPublicID(item PublicModelCatalogItem, publicModelID string) bool {
	normalized := NormalizeModelCatalogModelID(publicModelID)
	if normalized == "" {
		return false
	}
	for _, candidate := range []string{item.PublicModelID, item.Model} {
		if NormalizeModelCatalogModelID(candidate) == normalized {
			return true
		}
	}
	return false
}

func publishedPublicCatalogEntryFromItem(item PublicModelCatalogItem) *PublishedPublicCatalogEntry {
	publicID := NormalizeModelCatalogModelID(firstNonEmptyTrimmed(item.PublicModelID, item.Model))
	sourceModelID := NormalizeModelCatalogModelID(firstNonEmptyTrimmed(item.SourceModelID, item.BaseModel, firstRegistryString(item.SourceIDs...), publicID))
	saleDisplay := item.SalePriceDisplay
	if len(saleDisplay.Primary) == 0 && len(saleDisplay.Secondary) == 0 {
		saleDisplay = item.PriceDisplay
	}
	return &PublishedPublicCatalogEntry{
		EntryID:          strings.TrimSpace(item.EntryID),
		PublicModelID:    publicID,
		SourceAccountID:  item.SourceAccountID,
		SourceAlias:      strings.TrimSpace(item.SourceAlias),
		SourceModelID:    sourceModelID,
		SourceProtocol:   strings.TrimSpace(item.SourceProtocol),
		Currency:         defaultModelPricingCurrency(item.Currency),
		RuntimePriceSpec: normalizePublicModelCatalogRuntimePriceSpec(item.RuntimePriceSpec),
		SalePriceDisplay: clonePublicModelCatalogPriceDisplay(saleDisplay),
		Item:             clonePublicModelCatalogItem(item),
	}
}

func clonePublicModelCatalogPriceDisplay(display PublicModelCatalogPriceDisplay) PublicModelCatalogPriceDisplay {
	return PublicModelCatalogPriceDisplay{
		Primary:   clonePublicModelCatalogPriceEntries(display.Primary),
		Secondary: clonePublicModelCatalogPriceEntries(display.Secondary),
	}
}

type publicCatalogRuntimeContextKey struct{}

func WithPublishedPublicCatalogEntry(ctx context.Context, entry *PublishedPublicCatalogEntry) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if entry == nil {
		return ctx
	}
	return context.WithValue(ctx, publicCatalogRuntimeContextKey{}, entry)
}

func PublishedPublicCatalogEntryFromContext(ctx context.Context) (*PublishedPublicCatalogEntry, bool) {
	if ctx == nil {
		return nil, false
	}
	entry, ok := ctx.Value(publicCatalogRuntimeContextKey{}).(*PublishedPublicCatalogEntry)
	return entry, ok && entry != nil
}

func AttachPublishedPublicCatalogEntry(ctx context.Context, entry *PublishedPublicCatalogEntry) context.Context {
	ctx = WithPublishedPublicCatalogEntry(ctx, entry)
	SetPublicCatalogRuntimeMetadata(ctx, entry)
	return ctx
}

func (s *GatewayService) ResolvePublishedPublicCatalogRuntime(ctx context.Context, publicModelID string) (*PublishedPublicCatalogEntry, bool, error) {
	if s == nil || s.modelCatalogService == nil {
		return nil, false, nil
	}
	entry, ok, err := s.modelCatalogService.ResolvePublishedPublicCatalogEntry(ctx, publicModelID)
	if err != nil || !ok {
		return entry, ok, err
	}
	protocolruntime.RecordBillingResolver("public_catalog_entry_route")
	logger.FromContext(ctx).Info(
		"public model catalog runtime entry resolved",
		zap.String("entry_id", entry.EntryID),
		zap.String("public_model_id", entry.PublicModelID),
		zap.String("source_model_id", entry.SourceModelID),
		zap.Int64("account_id", entry.SourceAccountID),
	)
	return entry, true, nil
}

func (s *GatewayService) ResolveAPIKeyPublishedPublicCatalogRuntime(ctx context.Context, apiKey *APIKey, platform string, publicModelID string) (*PublishedPublicCatalogEntry, bool, bool, error) {
	if s == nil || s.modelCatalogService == nil {
		return nil, false, false, nil
	}
	matches, active, err := s.apiKeyPublishedPublicCatalogVisibleMatches(ctx, apiKey, platform, publicModelID)
	if err != nil || !active {
		return nil, false, active, err
	}
	if len(matches) == 0 {
		recordPublicCatalogRouteMiss(ctx, apiKey, nil, publicModelID, platform)
		return nil, false, true, nil
	}
	match := matches[0]
	entry := match.Catalog
	if entry == nil {
		entry = publishedPublicCatalogEntryFromItem(match.SourceItem)
	}
	entry = entry.WithBindingGroupID(match.GroupID)
	protocolruntime.RecordBillingResolver("public_catalog_entry_route")
	logger.FromContext(ctx).Info(
		"public model catalog runtime entry resolved",
		publicCatalogLogFields(ctx, entry, match.GroupID, apiKey)...,
	)
	return entry, true, true, nil
}

func (s *OpenAIGatewayService) ResolvePublishedPublicCatalogRuntime(ctx context.Context, publicModelID string) (*PublishedPublicCatalogEntry, bool, error) {
	if s == nil || s.modelCatalogService == nil {
		return nil, false, nil
	}
	entry, ok, err := s.modelCatalogService.ResolvePublishedPublicCatalogEntry(ctx, publicModelID)
	if err != nil || !ok {
		return entry, ok, err
	}
	protocolruntime.RecordBillingResolver("public_catalog_entry_route")
	logger.FromContext(ctx).Info(
		"public model catalog openai runtime entry resolved",
		zap.String("entry_id", entry.EntryID),
		zap.String("public_model_id", entry.PublicModelID),
		zap.String("source_model_id", entry.SourceModelID),
		zap.Int64("account_id", entry.SourceAccountID),
	)
	return entry, true, nil
}

func (s *OpenAIGatewayService) ResolveAPIKeyPublishedPublicCatalogRuntime(ctx context.Context, apiKey *APIKey, platform string, publicModelID string) (*PublishedPublicCatalogEntry, bool, bool, error) {
	if s == nil || s.modelCatalogService == nil {
		return nil, false, false, nil
	}
	matches, active, err := s.apiKeyPublishedPublicCatalogVisibleMatches(ctx, apiKey, platform, publicModelID)
	if err != nil || !active {
		return nil, false, active, err
	}
	if len(matches) == 0 {
		recordPublicCatalogRouteMiss(ctx, apiKey, nil, publicModelID, platform)
		return nil, false, true, nil
	}
	match := matches[0]
	entry := match.Catalog
	if entry == nil {
		entry = publishedPublicCatalogEntryFromItem(match.SourceItem)
	}
	entry = entry.WithBindingGroupID(match.GroupID)
	protocolruntime.RecordBillingResolver("public_catalog_entry_route")
	logger.FromContext(ctx).Info(
		"public model catalog openai runtime entry resolved",
		publicCatalogLogFields(ctx, entry, match.GroupID, apiKey)...,
	)
	return entry, true, true, nil
}

func (entry *PublishedPublicCatalogEntry) WithBindingGroupID(groupID *int64) *PublishedPublicCatalogEntry {
	if entry == nil || groupID == nil || *groupID <= 0 {
		return entry
	}
	copied := *entry
	copied.BindingGroupID = *groupID
	return &copied
}

func publicCatalogLogFields(ctx context.Context, entry *PublishedPublicCatalogEntry, groupID *int64, apiKey *APIKey) []zap.Field {
	fields := make([]zap.Field, 0, 9)
	if requestID, _ := ctx.Value(ctxkey.RequestID).(string); strings.TrimSpace(requestID) != "" {
		fields = append(fields, zap.String("request_id", strings.TrimSpace(requestID)))
	}
	if entry != nil {
		fields = append(fields,
			zap.String("entry_id", entry.EntryID),
			zap.String("public_model_id", entry.PublicModelID),
			zap.String("source_model_id", entry.SourceModelID),
			zap.Int64("account_id", entry.SourceAccountID),
		)
	}
	if groupID != nil {
		fields = append(fields, zap.Int64("group_id", *groupID))
	}
	if apiKey != nil && apiKey.ID > 0 {
		fields = append(fields, zap.Int64("api_key_id", apiKey.ID))
	}
	fields = append(fields, zap.String("pricing_source", "public_catalog_entry"))
	return fields
}
