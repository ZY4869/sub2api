package service

import (
	"sort"
	"strings"
)

func normalizePublicModelCapabilityMatrix(
	existing []PublicModelCapabilityMatrixEntry,
	legacyCapabilities []string,
	endpoints []PublicModelProtocolEndpoint,
	source publicModelCatalogMetadataSource,
) []PublicModelCapabilityMatrixEntry {
	entries := make([]PublicModelCapabilityMatrixEntry, 0, len(existing)+len(legacyCapabilities)*maxInt(1, len(endpoints)))
	for _, entry := range existing {
		if normalized, ok := normalizePublicModelCapabilityMatrixEntry(entry, source); ok {
			entries = append(entries, normalized)
		}
	}
	if len(endpoints) == 0 {
		endpoints = []PublicModelProtocolEndpoint{{Support: PublicModelSupportUnknown, Source: source.CapabilitySource}}
	}
	for _, capability := range legacyCapabilities {
		capability = strings.TrimSpace(capability)
		if capability == "" {
			continue
		}
		for _, endpoint := range endpoints {
			if !publicModelEndpointMatchesCapability(endpoint, capability) {
				continue
			}
			entries = append(entries, PublicModelCapabilityMatrixEntry{
				Capability:    capability,
				Protocol:      endpoint.Protocol,
				Endpoint:      endpoint.Key,
				Support:       endpoint.Support,
				Mode:          firstNonEmptyTrimmed(endpoint.Source, source.CapabilitySource),
				Source:        firstNonEmptyTrimmed(source.CapabilitySource, endpoint.Source, PublicModelCapabilitySourceInferred),
				Verified:      publicModelCatalogMetadataSourceVerified(source) && publicModelSupportAllowsSummary(endpoint.Support),
				LastCheckedAt: firstNonEmptyTrimmed(source.LastCheckedAt, endpoint.LastCheckedAt),
			})
		}
	}
	return dedupePublicModelCapabilityMatrix(entries)
}

func publicModelEndpointMatchesCapability(endpoint PublicModelProtocolEndpoint, capability string) bool {
	capability = strings.TrimSpace(strings.ToLower(capability))
	key := strings.TrimSpace(strings.ToLower(endpoint.Key))
	if capability == "" {
		return false
	}
	if strings.Contains(capability, "image") {
		return strings.Contains(key, "image")
	}
	if strings.Contains(capability, "video") {
		return strings.Contains(key, "video")
	}
	if strings.Contains(capability, "embedding") || strings.Contains(capability, "embed") {
		return strings.Contains(key, "embedding") || strings.Contains(key, "embed")
	}
	if capability == "text" || capability == "tools" || capability == "vision" || capability == "reasoning" || capability == "json" {
		return !strings.Contains(key, "image") && !strings.Contains(key, "video") && !strings.Contains(key, "embedding")
	}
	return true
}

func normalizePublicModelCapabilityMatrixEntry(entry PublicModelCapabilityMatrixEntry, source publicModelCatalogMetadataSource) (PublicModelCapabilityMatrixEntry, bool) {
	entry.Capability = strings.TrimSpace(entry.Capability)
	if entry.Capability == "" {
		return PublicModelCapabilityMatrixEntry{}, false
	}
	entry.Protocol = publicModelCatalogProtocolFamily(entry.Protocol)
	entry.Endpoint = strings.TrimSpace(entry.Endpoint)
	entry.Support = normalizePublicModelSupport(entry.Support)
	entry.Mode = strings.TrimSpace(entry.Mode)
	entry.Source = firstNonEmptyTrimmed(entry.Source, source.CapabilitySource, PublicModelCapabilitySourceInferred)
	entry.Verified = entry.Verified || publicModelCatalogMetadataSourceVerified(source) && publicModelSupportAllowsSummary(entry.Support)
	entry.LastCheckedAt = firstNonEmptyTrimmed(entry.LastCheckedAt, source.LastCheckedAt)
	entry.Limitations = uniqueTrimmedStringsPreserveCase(entry.Limitations)
	return entry, true
}

func dedupePublicModelCapabilityMatrix(entries []PublicModelCapabilityMatrixEntry) []PublicModelCapabilityMatrixEntry {
	if len(entries) == 0 {
		return nil
	}
	byKey := map[string]PublicModelCapabilityMatrixEntry{}
	for _, entry := range entries {
		normalized, ok := normalizePublicModelCapabilityMatrixEntry(entry, publicModelCatalogMetadataSource{})
		if !ok {
			continue
		}
		key := normalized.Capability + "\x00" + normalized.Protocol + "\x00" + normalized.Endpoint
		existing, exists := byKey[key]
		if !exists || publicModelCapabilityEntryPreferred(normalized, existing) {
			byKey[key] = normalized
		}
	}
	keys := make([]string, 0, len(byKey))
	for key := range byKey {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]PublicModelCapabilityMatrixEntry, 0, len(keys))
	for _, key := range keys {
		result = append(result, byKey[key])
	}
	return result
}

func publicModelCapabilityEntryPreferred(left PublicModelCapabilityMatrixEntry, right PublicModelCapabilityMatrixEntry) bool {
	return publicModelMetadataEntryPreferred(
		left.Source,
		left.Verified,
		left.Support,
		left.LastCheckedAt,
		right.Source,
		right.Verified,
		right.Support,
		right.LastCheckedAt,
	)
}

func publicModelCapabilitiesFromMatrix(matrix []PublicModelCapabilityMatrixEntry, fallback []string) []string {
	values := make([]string, 0, len(matrix)+len(fallback))
	for _, entry := range matrix {
		if publicModelSupportAllowsSummary(entry.Support) {
			values = append(values, entry.Capability)
		}
	}
	if len(values) == 0 {
		values = append(values, fallback...)
	}
	return uniqueTrimmedStringsPreserveCase(values)
}
