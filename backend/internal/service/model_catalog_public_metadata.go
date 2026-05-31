package service

import "strings"

type publicModelCatalogMetadataSource struct {
	EntrySource      string
	CapabilitySource string
	LifecycleSource  string
	ContextSource    string
	Verified         bool
	LastCheckedAt    string
}

func enrichPublicModelCatalogItemMetadata(item PublicModelCatalogItem, source publicModelCatalogMetadataSource) PublicModelCatalogItem {
	item.CatalogEntrySource = normalizePublicModelCatalogEntrySource(firstNonEmptyTrimmed(item.CatalogEntrySource, source.EntrySource))
	item.IsDemo = item.IsDemo || item.CatalogEntrySource == PublicModelCatalogEntrySourceDemo
	item.Lifecycle = normalizePublicModelLifecycle(item.Lifecycle, item.LifecycleStatus, source)
	item.ContextWindow = normalizePublicModelContextWindow(item.ContextWindow, item.ContextWindowTokens, source)
	item.ProtocolEndpoints = normalizePublicModelProtocolEndpoints(item.ProtocolEndpoints, item.RequestProtocols, source)
	item.CapabilityMatrix = normalizePublicModelCapabilityMatrix(item.CapabilityMatrix, item.Capabilities, item.ProtocolEndpoints, source)
	item.ContextWindowTokens = item.ContextWindow.Tokens
	item.LifecycleStatus = firstNonEmptyTrimmed(item.Lifecycle.Status, item.LifecycleStatus, PublicModelLifecycleStable)
	item.RequestProtocols = publicModelRequestProtocolsFromEndpoints(item.ProtocolEndpoints, item.RequestProtocols)
	item.Capabilities = publicModelCapabilitiesFromMatrix(item.CapabilityMatrix, item.Capabilities)
	return item
}

func enrichPublicModelCatalogItemObservedMetadata(item PublicModelCatalogItem, source publicModelCatalogMetadataSource) PublicModelCatalogItem {
	item.CatalogEntrySource = normalizePublicModelCatalogEntrySource(firstNonEmptyTrimmed(item.CatalogEntrySource, source.EntrySource))
	item.IsDemo = item.IsDemo || item.CatalogEntrySource == PublicModelCatalogEntrySourceDemo
	item.Lifecycle = normalizePublicModelLifecycle(item.Lifecycle, item.LifecycleStatus, source)
	item.ContextWindow = normalizePublicModelContextWindow(item.ContextWindow, item.ContextWindowTokens, source)
	item.ProtocolEndpoints = normalizePublicModelProtocolEndpoints(item.ProtocolEndpoints, nil, source)
	item.CapabilityMatrix = normalizePublicModelCapabilityMatrix(item.CapabilityMatrix, nil, item.ProtocolEndpoints, source)
	item.ContextWindowTokens = item.ContextWindow.Tokens
	item.LifecycleStatus = firstNonEmptyTrimmed(item.Lifecycle.Status, item.LifecycleStatus, PublicModelLifecycleStable)
	item.RequestProtocols = publicModelRequestProtocolsFromEndpoints(item.ProtocolEndpoints, nil)
	item.Capabilities = publicModelCapabilitiesFromMatrix(item.CapabilityMatrix, nil)
	return item
}

func publicModelCatalogMetadataSourceVerified(source publicModelCatalogMetadataSource) bool {
	if !source.Verified {
		return false
	}
	switch strings.TrimSpace(source.CapabilitySource) {
	case PublicModelCapabilitySourceRuntimeObserved,
		PublicModelCapabilitySourceVerifiedProbe,
		PublicModelCapabilitySourceAccountProbe:
		return true
	default:
		return false
	}
}

func normalizePublicModelCatalogEntrySource(value string) string {
	switch strings.TrimSpace(value) {
	case PublicModelCatalogEntrySourceRealAccount,
		PublicModelCatalogEntrySourceLiveProjection,
		PublicModelCatalogEntrySourceDemo,
		PublicModelCatalogEntrySourceLegacySnapshot:
		return strings.TrimSpace(value)
	default:
		return PublicModelCatalogEntrySourceLegacySnapshot
	}
}

func normalizePublicModelLifecycle(lifecycle PublicModelLifecycle, legacy string, source publicModelCatalogMetadataSource) PublicModelLifecycle {
	status := normalizePublicModelLifecycleStatus(firstNonEmptyTrimmed(lifecycle.Status, legacy))
	lifecycleSource := firstNonEmptyTrimmed(lifecycle.Source, source.LifecycleSource)
	confidence := firstNonEmptyTrimmed(lifecycle.Confidence)
	if lifecycleSource == "" {
		lifecycleSource = PublicModelLifecycleSourceInferred
	}
	if confidence == "" {
		if lifecycleSource == PublicModelLifecycleSourceInferred {
			confidence = PublicModelLifecycleConfidenceInferred
		} else {
			confidence = PublicModelLifecycleConfidenceDeclared
		}
	}
	return PublicModelLifecycle{
		Status:     status,
		Source:     lifecycleSource,
		Confidence: confidence,
	}
}

func normalizePublicModelContextWindow(contextWindow PublicModelContextWindow, legacyTokens int64, source publicModelCatalogMetadataSource) PublicModelContextWindow {
	tokens := contextWindow.Tokens
	if tokens <= 0 {
		tokens = legacyTokens
	}
	if tokens < 0 {
		tokens = 0
	}
	contextWindow.Tokens = tokens
	contextWindow.Source = firstNonEmptyTrimmed(contextWindow.Source, source.ContextSource)
	if contextWindow.Source == "" && tokens > 0 {
		contextWindow.Source = PublicModelCapabilitySourcePricingCatalog
	}
	contextWindow.LastCheckedAt = firstNonEmptyTrimmed(contextWindow.LastCheckedAt, source.LastCheckedAt)
	contextWindow.LimitKind = firstNonEmptyTrimmed(contextWindow.LimitKind, PublicModelContextLimitKindInput)
	contextWindow.Verified = contextWindow.Verified || source.Verified && publicModelCapabilitySourceRank(source.ContextSource) <= publicModelCapabilitySourceRank(PublicModelCapabilitySourceAccountProbe)
	contextWindow.Notes = uniqueTrimmedStringsPreserveCase(contextWindow.Notes)
	return contextWindow
}

func mergePublicModelContextWindow(current PublicModelContextWindow, candidate PublicModelContextWindow) PublicModelContextWindow {
	current = normalizePublicModelContextWindow(current, 0, publicModelCatalogMetadataSource{})
	candidate = normalizePublicModelContextWindow(candidate, 0, publicModelCatalogMetadataSource{})
	if candidate.Tokens <= 0 && candidate.Source == "" {
		return current
	}
	if current.Tokens <= 0 && current.Source == "" {
		return candidate
	}
	if publicModelMetadataEntryPreferred(
		candidate.Source,
		candidate.Verified,
		PublicModelSupportSupported,
		candidate.LastCheckedAt,
		current.Source,
		current.Verified,
		PublicModelSupportSupported,
		current.LastCheckedAt,
	) {
		return candidate
	}
	return current
}

func publicModelCatalogMetadataSourceForRegistry() publicModelCatalogMetadataSource {
	return publicModelCatalogMetadataSource{
		EntrySource:      PublicModelCatalogEntrySourceLegacySnapshot,
		CapabilitySource: PublicModelCapabilitySourceOfficialRegistry,
		LifecycleSource:  PublicModelLifecycleSourceOfficialRegistry,
		ContextSource:    PublicModelCapabilitySourcePricingCatalog,
	}
}

func publicModelCatalogMetadataSourceForProjection() publicModelCatalogMetadataSource {
	return publicModelCatalogMetadataSource{
		EntrySource:      PublicModelCatalogEntrySourceLiveProjection,
		CapabilitySource: PublicModelCapabilitySourceManualConfig,
		LifecycleSource:  PublicModelLifecycleSourceManualConfig,
		ContextSource:    PublicModelCapabilitySourcePricingCatalog,
	}
}

func publicModelCatalogMetadataSourceForAccount(updatedAt string, source string, verified bool) publicModelCatalogMetadataSource {
	capabilitySource := PublicModelCapabilitySourceAccountProbe
	if strings.EqualFold(strings.TrimSpace(source), AccountModelProbeSnapshotSourceTestProbe) ||
		strings.EqualFold(strings.TrimSpace(source), AccountModelProbeSnapshotSourceManualProbe) {
		capabilitySource = PublicModelCapabilitySourceVerifiedProbe
	}
	return publicModelCatalogMetadataSource{
		EntrySource:      PublicModelCatalogEntrySourceRealAccount,
		CapabilitySource: capabilitySource,
		LifecycleSource:  PublicModelLifecycleSourceManualConfig,
		ContextSource:    PublicModelCapabilitySourceAccountProbe,
		Verified:         verified,
		LastCheckedAt:    strings.TrimSpace(updatedAt),
	}
}

func publicModelCatalogMetadataSourceForPublished(checkedAt string) publicModelCatalogMetadataSource {
	return publicModelCatalogMetadataSource{
		EntrySource:      PublicModelCatalogEntrySourceLegacySnapshot,
		CapabilitySource: PublicModelCapabilitySourcePublishedSnapshot,
		LifecycleSource:  PublicModelLifecycleSourcePublishedSnapshot,
		ContextSource:    PublicModelCapabilitySourcePublishedSnapshot,
		LastCheckedAt:    strings.TrimSpace(checkedAt),
	}
}
