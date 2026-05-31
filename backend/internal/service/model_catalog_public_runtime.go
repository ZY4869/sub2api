package service

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/protocolruntime"
	"go.uber.org/zap"
)

const PublicCatalogModelUnavailableMessage = "Requested model is not published or not available for this API key"

var publicCatalogRuntimeFailureWritebackThrottle sync.Map

type publicModelCatalogRuntimeFailureWriteback struct {
	RequestID   string
	Entry       *PublishedPublicCatalogEntry
	Protocol    string
	EndpointKey string
	Capability  string
	Result      string
}

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
		if publicModelCatalogItemIsDemo(item) {
			continue
		}
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

func (s *ModelCatalogService) RecordPublicModelCatalogRuntimeCapabilityFailure(
	ctx context.Context,
	entry *PublishedPublicCatalogEntry,
	protocol string,
	endpointKey string,
	capability string,
	result string,
) {
	if s == nil || entry == nil {
		return
	}
	result = strings.TrimSpace(result)
	if result == "" {
		result = PublicModelSupportUnsupported
	}
	if !publicCatalogRuntimeFailureShouldWriteBack(entry, protocol, endpointKey, capability, result) {
		protocolruntime.RecordModelCapabilityVerification("skipped")
		return
	}
	requestID, _ := ctx.Value(ctxkey.RequestID).(string)
	go s.recordPublicModelCatalogRuntimeCapabilityFailure(context.Background(), publicModelCatalogRuntimeFailureWriteback{
		RequestID:   strings.TrimSpace(requestID),
		Entry:       clonePublishedPublicCatalogEntry(entry),
		Protocol:    protocol,
		EndpointKey: endpointKey,
		Capability:  capability,
		Result:      result,
	})
}

func RecordPublicModelCatalogRuntimeCapabilityFailureFromContext(
	ctx context.Context,
	modelCatalogService *ModelCatalogService,
	protocol string,
	endpointKey string,
	capability string,
	result string,
) {
	if modelCatalogService == nil {
		protocolruntime.RecordModelCapabilityVerification("skipped")
		return
	}
	entry, ok := PublishedPublicCatalogEntryFromContext(ctx)
	if !ok || entry == nil {
		protocolruntime.RecordModelCapabilityVerification("skipped")
		return
	}
	modelCatalogService.RecordPublicModelCatalogRuntimeCapabilityFailure(ctx, entry, protocol, endpointKey, capability, result)
}

func RecordPublicModelCatalogRuntimeFailureIfModelCapabilityError(
	ctx context.Context,
	modelCatalogService *ModelCatalogService,
	statusCode int,
	message string,
	protocol string,
	endpointKey string,
	capability string,
) {
	if !publicModelCatalogRuntimeErrorIsCapabilityFailure(statusCode, message) {
		protocolruntime.RecordModelCapabilityVerification("skipped")
		return
	}
	RecordPublicModelCatalogRuntimeCapabilityFailureFromContext(
		ctx,
		modelCatalogService,
		protocol,
		endpointKey,
		capability,
		PublicModelSupportUnsupported,
	)
}

func publicModelCatalogRuntimeErrorIsCapabilityFailure(statusCode int, message string) bool {
	switch statusCode {
	case http.StatusBadRequest, http.StatusNotFound, http.StatusUnprocessableEntity:
	default:
		return false
	}
	normalized := strings.ToLower(strings.TrimSpace(message))
	if normalized == "" {
		return false
	}
	blocked := []string{
		"rate limit",
		"quota",
		"unauthorized",
		"authentication",
		"permission",
		"forbidden",
		"billing",
		"insufficient",
	}
	for _, keyword := range blocked {
		if strings.Contains(normalized, keyword) {
			return false
		}
	}
	keywords := []string{
		"model not supported",
		"not supported for this model",
		"does not support",
		"unsupported model",
		"unsupported endpoint",
		"endpoint is not supported",
		"context limit",
		"context length",
		"maximum context",
	}
	for _, keyword := range keywords {
		if strings.Contains(normalized, keyword) {
			return true
		}
	}
	return false
}

func publicCatalogRuntimeFailureShouldWriteBack(entry *PublishedPublicCatalogEntry, protocol string, endpointKey string, capability string, result string) bool {
	key := strings.Join([]string{
		strings.TrimSpace(entry.EntryID),
		strings.TrimSpace(entry.PublicModelID),
		strings.TrimSpace(protocol),
		strings.TrimSpace(endpointKey),
		strings.TrimSpace(capability),
		strings.TrimSpace(result),
	}, "\x00")
	if strings.Trim(key, "\x00") == "" {
		return false
	}
	now := time.Now()
	if previous, ok := publicCatalogRuntimeFailureWritebackThrottle.Load(key); ok {
		if previousTime, ok := previous.(time.Time); ok && now.Sub(previousTime) < time.Hour {
			return false
		}
	}
	publicCatalogRuntimeFailureWritebackThrottle.Store(key, now)
	return true
}

func (s *ModelCatalogService) recordPublicModelCatalogRuntimeCapabilityFailure(ctx context.Context, writeback publicModelCatalogRuntimeFailureWriteback) {
	entry := writeback.Entry
	if entry == nil {
		protocolruntime.RecordModelCapabilityVerification("failure")
		return
	}
	published := s.loadPublishedPublicModelCatalogSnapshot(ctx)
	if published == nil {
		protocolruntime.RecordModelCapabilityVerification("failure")
		return
	}
	updated := false
	checkedAt := time.Now().UTC().Format(time.RFC3339)
	for index, item := range published.Snapshot.Items {
		if !publicModelCatalogItemMatchesPublicID(item, entry.PublicModelID) {
			continue
		}
		item = applyPublicModelCatalogRuntimeFailure(item, writeback.Protocol, writeback.EndpointKey, writeback.Capability, writeback.Result, checkedAt)
		published.Snapshot.Items[index] = item
		publicID := NormalizeModelCatalogModelID(firstNonEmptyTrimmed(item.PublicModelID, item.Model))
		if publicID != "" {
			if detail, ok := published.Details[publicID]; ok {
				detail.Item = clonePublicModelCatalogItem(item)
				published.Details[publicID] = detail
			}
		}
		updated = true
		break
	}
	if !updated {
		protocolruntime.RecordModelCapabilityVerification("failure")
		return
	}
	published.Snapshot.LastRevalidatedAt = checkedAt
	if err := s.persistPublishedPublicModelCatalogSnapshot(ctx, published); err != nil {
		protocolruntime.RecordModelCapabilityVerification("failure")
		logger.FromContext(ctx).Warn("public model catalog runtime capability writeback failed", zap.Error(err))
		return
	}
	protocolruntime.RecordModelCapabilityVerification("failure")
	fields := []zap.Field{
		zap.String("component", "service.model_catalog"),
		zap.String("entry_id", entry.EntryID),
		zap.String("public_model_id", entry.PublicModelID),
		zap.String("source_model_id", entry.SourceModelID),
		zap.Int64("account_id", entry.SourceAccountID),
		zap.String("protocol", strings.TrimSpace(writeback.Protocol)),
		zap.String("endpoint", strings.TrimSpace(writeback.EndpointKey)),
		zap.String("capability", strings.TrimSpace(writeback.Capability)),
		zap.String("result", strings.TrimSpace(writeback.Result)),
	}
	if writeback.RequestID != "" {
		fields = append(fields, zap.String("request_id", writeback.RequestID))
	}
	logger.FromContext(ctx).Info("public model catalog runtime capability writeback", fields...)
}

func clonePublishedPublicCatalogEntry(entry *PublishedPublicCatalogEntry) *PublishedPublicCatalogEntry {
	if entry == nil {
		return nil
	}
	cloned := *entry
	cloned.RuntimePriceSpec = normalizePublicModelCatalogRuntimePriceSpec(entry.RuntimePriceSpec)
	cloned.SalePriceDisplay = clonePublicModelCatalogPriceDisplay(entry.SalePriceDisplay)
	cloned.Item = clonePublicModelCatalogItem(entry.Item)
	return &cloned
}

func applyPublicModelCatalogRuntimeFailure(item PublicModelCatalogItem, protocol string, endpointKey string, capability string, result string, checkedAt string) PublicModelCatalogItem {
	item = enrichPublicModelCatalogItemObservedMetadata(item, publicModelCatalogMetadataSourceForPublished(checkedAt))
	protocol = publicModelCatalogProtocolFamily(protocol)
	endpointKey = strings.TrimSpace(endpointKey)
	capability = strings.TrimSpace(capability)
	normalizedResult := normalizePublicModelSupport(result)
	limitation := publicModelCatalogRuntimeFailureLimitation(result)
	if endpointKey != "" && !publicModelCatalogHasEndpoint(item.ProtocolEndpoints, protocol, endpointKey) {
		item.ProtocolEndpoints = append(item.ProtocolEndpoints, PublicModelProtocolEndpoint{
			Key:           endpointKey,
			Protocol:      protocol,
			Support:       normalizedResult,
			Source:        PublicModelCapabilitySourceRuntimeObserved,
			Verified:      true,
			LastCheckedAt: checkedAt,
			Limitations:   publicModelCatalogRuntimeFailureLimitations(nil, limitation),
		})
	}
	if capability != "" && !publicModelCatalogHasCapability(item.CapabilityMatrix, protocol, endpointKey, capability) {
		item.CapabilityMatrix = append(item.CapabilityMatrix, PublicModelCapabilityMatrixEntry{
			Capability:    capability,
			Protocol:      protocol,
			Endpoint:      endpointKey,
			Support:       normalizedResult,
			Source:        PublicModelCapabilitySourceRuntimeObserved,
			Verified:      true,
			LastCheckedAt: checkedAt,
			Limitations:   publicModelCatalogRuntimeFailureLimitations(nil, limitation),
		})
	}
	for index := range item.ProtocolEndpoints {
		if endpointKey != "" && item.ProtocolEndpoints[index].Key != endpointKey {
			continue
		}
		if protocol != "" && item.ProtocolEndpoints[index].Protocol != protocol {
			continue
		}
		if publicModelCatalogPreserveVerifiedSuccess(
			item.ProtocolEndpoints[index].Source,
			item.ProtocolEndpoints[index].Verified,
			item.ProtocolEndpoints[index].Support,
		) {
			item.ProtocolEndpoints[index].LastCheckedAt = checkedAt
			item.ProtocolEndpoints[index].Limitations = publicModelCatalogRuntimeFailureLimitations(item.ProtocolEndpoints[index].Limitations, limitation)
			continue
		}
		item.ProtocolEndpoints[index].Support = normalizedResult
		item.ProtocolEndpoints[index].Source = PublicModelCapabilitySourceRuntimeObserved
		item.ProtocolEndpoints[index].Verified = true
		item.ProtocolEndpoints[index].LastCheckedAt = checkedAt
		item.ProtocolEndpoints[index].Limitations = publicModelCatalogRuntimeFailureLimitations(item.ProtocolEndpoints[index].Limitations, limitation)
	}
	for index := range item.CapabilityMatrix {
		if capability != "" && item.CapabilityMatrix[index].Capability != capability {
			continue
		}
		if endpointKey != "" && item.CapabilityMatrix[index].Endpoint != endpointKey {
			continue
		}
		if protocol != "" && item.CapabilityMatrix[index].Protocol != protocol {
			continue
		}
		if publicModelCatalogPreserveVerifiedSuccess(
			item.CapabilityMatrix[index].Source,
			item.CapabilityMatrix[index].Verified,
			item.CapabilityMatrix[index].Support,
		) {
			item.CapabilityMatrix[index].LastCheckedAt = checkedAt
			item.CapabilityMatrix[index].Limitations = publicModelCatalogRuntimeFailureLimitations(item.CapabilityMatrix[index].Limitations, limitation)
			continue
		}
		item.CapabilityMatrix[index].Support = normalizedResult
		item.CapabilityMatrix[index].Source = PublicModelCapabilitySourceRuntimeObserved
		item.CapabilityMatrix[index].Verified = true
		item.CapabilityMatrix[index].LastCheckedAt = checkedAt
		item.CapabilityMatrix[index].Limitations = publicModelCatalogRuntimeFailureLimitations(item.CapabilityMatrix[index].Limitations, limitation)
	}
	item.ProtocolEndpoints = dedupePublicModelProtocolEndpoints(item.ProtocolEndpoints)
	item.CapabilityMatrix = dedupePublicModelCapabilityMatrix(item.CapabilityMatrix)
	item.RequestProtocols = publicModelRequestProtocolsFromEndpoints(item.ProtocolEndpoints, item.RequestProtocols)
	item.Capabilities = publicModelCapabilitiesFromMatrix(item.CapabilityMatrix, item.Capabilities)
	return item
}

func publicModelCatalogPreserveVerifiedSuccess(source string, verified bool, support string) bool {
	if !verified || !publicModelSupportAllowsSummary(support) {
		return false
	}
	return publicModelCapabilitySourceRank(source) <= publicModelCapabilitySourceRank(PublicModelCapabilitySourceAccountProbe)
}

func publicModelCatalogRuntimeFailureLimitation(result string) string {
	result = strings.TrimSpace(result)
	if result == "" {
		return "runtime_failure_observed"
	}
	return "runtime_failure_observed:" + result
}

func publicModelCatalogRuntimeFailureLimitations(current []string, limitation string) []string {
	if strings.TrimSpace(limitation) == "" {
		return uniqueTrimmedStringsPreserveCase(current)
	}
	return uniqueTrimmedStringsPreserveCase(append(append([]string(nil), current...), limitation))
}

func publicModelCatalogHasEndpoint(endpoints []PublicModelProtocolEndpoint, protocol string, endpointKey string) bool {
	protocol = publicModelCatalogProtocolFamily(protocol)
	endpointKey = strings.TrimSpace(endpointKey)
	for _, endpoint := range endpoints {
		if endpointKey != "" && endpoint.Key != endpointKey {
			continue
		}
		if protocol != "" && endpoint.Protocol != protocol {
			continue
		}
		return true
	}
	return false
}

func publicModelCatalogHasCapability(matrix []PublicModelCapabilityMatrixEntry, protocol string, endpointKey string, capability string) bool {
	protocol = publicModelCatalogProtocolFamily(protocol)
	endpointKey = strings.TrimSpace(endpointKey)
	capability = strings.TrimSpace(capability)
	for _, entry := range matrix {
		if capability != "" && entry.Capability != capability {
			continue
		}
		if endpointKey != "" && entry.Endpoint != endpointKey {
			continue
		}
		if protocol != "" && entry.Protocol != protocol {
			continue
		}
		return true
	}
	return false
}
