package service

import "strings"

type publicModelLifecycleResolution struct {
	Status   string
	Inferred bool
}

func publicModelCatalogItemConfirmedAvailable(item PublicModelCatalogItem) bool {
	return strings.EqualFold(item.AvailabilityState, AccountModelAvailabilityVerified) &&
		strings.EqualFold(item.StaleState, AccountModelStaleStateFresh)
}

func normalizePublicModelLifecycleStatus(raw string, candidates ...string) string {
	return resolvePublicModelLifecycleStatus(raw, candidates...).Status
}

func resolvePublicModelLifecycleStatus(raw string, candidates ...string) publicModelLifecycleResolution {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case PublicModelLifecycleDeprecated:
		return publicModelLifecycleResolution{Status: PublicModelLifecycleDeprecated}
	case PublicModelLifecycleBeta:
		return publicModelLifecycleResolution{Status: PublicModelLifecycleBeta}
	case "", PublicModelLifecycleStable:
		// fall through to semantic inference.
	default:
		return publicModelLifecycleResolution{Status: PublicModelLifecycleStable}
	}

	for _, candidate := range candidates {
		value := strings.TrimSpace(strings.ToLower(candidate))
		if value == "" {
			continue
		}
		if strings.Contains(value, "deprecated") {
			return publicModelLifecycleResolution{Status: PublicModelLifecycleDeprecated, Inferred: true}
		}
		if strings.Contains(value, "beta") || strings.Contains(value, "preview") || strings.Contains(value, "experimental") {
			return publicModelLifecycleResolution{Status: PublicModelLifecycleBeta, Inferred: true}
		}
	}
	return publicModelLifecycleResolution{Status: PublicModelLifecycleStable}
}

func publicModelLifecycleFromResolution(resolution publicModelLifecycleResolution, source string) PublicModelLifecycle {
	if resolution.Status == "" {
		return PublicModelLifecycle{}
	}
	if resolution.Inferred {
		return PublicModelLifecycle{
			Status:     resolution.Status,
			Source:     PublicModelLifecycleSourceInferred,
			Confidence: PublicModelLifecycleConfidenceInferred,
		}
	}
	if strings.TrimSpace(source) == "" {
		return PublicModelLifecycle{Status: resolution.Status}
	}
	return PublicModelLifecycle{
		Status: resolution.Status,
		Source: source,
	}
}

func publicModelRepresentativeRank(availabilityState string, staleState string, lifecycleStatus string) int {
	var base int
	switch {
	case strings.EqualFold(availabilityState, AccountModelAvailabilityVerified) && strings.EqualFold(staleState, AccountModelStaleStateFresh):
		base = 0
	case strings.EqualFold(availabilityState, AccountModelAvailabilityVerified) && strings.EqualFold(staleState, AccountModelStaleStateStale):
		base = 10
	case strings.EqualFold(availabilityState, AccountModelAvailabilityUnavailable):
		base = 30
	default:
		base = 20
	}
	return base + publicModelLifecycleRank(lifecycleStatus)
}

func publicModelLifecycleRank(value string) int {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case PublicModelLifecycleStable:
		return 0
	case PublicModelLifecycleBeta:
		return 1
	case PublicModelLifecycleDeprecated:
		return 2
	default:
		return 1
	}
}

func isBetterPublicModelRepresentative(
	leftAvailability string,
	leftStale string,
	leftLifecycle string,
	rightAvailability string,
	rightStale string,
	rightLifecycle string,
) bool {
	return publicModelRepresentativeRank(leftAvailability, leftStale, leftLifecycle) <
		publicModelRepresentativeRank(rightAvailability, rightStale, rightLifecycle)
}

func publicModelStatusFromProjection(
	availabilityState string,
	staleState string,
	lifecycleStatus string,
) string {
	switch {
	case strings.EqualFold(availabilityState, AccountModelAvailabilityUnavailable):
		return PublicModelStatusError
	case strings.EqualFold(availabilityState, AccountModelAvailabilityVerified) && strings.EqualFold(staleState, AccountModelStaleStateFresh):
		if strings.EqualFold(lifecycleStatus, PublicModelLifecycleStable) {
			return PublicModelStatusOK
		}
		return PublicModelStatusWarning
	case strings.EqualFold(availabilityState, AccountModelAvailabilityVerified) && strings.EqualFold(staleState, AccountModelStaleStateStale):
		return PublicModelStatusMaintenance
	default:
		return PublicModelStatusInfo
	}
}
