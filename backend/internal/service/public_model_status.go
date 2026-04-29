package service

import "strings"

func normalizePublicModelLifecycleStatus(raw string, candidates ...string) string {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case PublicModelLifecycleDeprecated:
		return PublicModelLifecycleDeprecated
	case PublicModelLifecycleBeta:
		return PublicModelLifecycleBeta
	case "", PublicModelLifecycleStable:
		// fall through to semantic inference.
	default:
		return PublicModelLifecycleStable
	}

	for _, candidate := range candidates {
		value := strings.TrimSpace(strings.ToLower(candidate))
		if value == "" {
			continue
		}
		if strings.Contains(value, "deprecated") {
			return PublicModelLifecycleDeprecated
		}
		if strings.Contains(value, "beta") || strings.Contains(value, "preview") || strings.Contains(value, "experimental") {
			return PublicModelLifecycleBeta
		}
	}
	return PublicModelLifecycleStable
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
