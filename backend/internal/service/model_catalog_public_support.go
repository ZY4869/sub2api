package service

import "strings"

func normalizePublicModelSupport(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case PublicModelSupportSupported, "true", "yes":
		return PublicModelSupportSupported
	case PublicModelSupportPartial:
		return PublicModelSupportPartial
	case PublicModelSupportUnsupported, "false", "no":
		return PublicModelSupportUnsupported
	default:
		return PublicModelSupportUnknown
	}
}

func publicModelSupportAllowsSummary(value string) bool {
	switch normalizePublicModelSupport(value) {
	case PublicModelSupportSupported, PublicModelSupportPartial:
		return true
	default:
		return false
	}
}

func publicModelSupportRank(value string) int {
	switch normalizePublicModelSupport(value) {
	case PublicModelSupportSupported:
		return 0
	case PublicModelSupportPartial:
		return 1
	case PublicModelSupportUnknown:
		return 2
	default:
		return 3
	}
}

func publicModelCapabilitySourceRank(value string) int {
	switch strings.TrimSpace(value) {
	case PublicModelCapabilitySourceRuntimeObserved:
		return 0
	case PublicModelCapabilitySourceVerifiedProbe:
		return 1
	case PublicModelCapabilitySourceAccountProbe:
		return 2
	case PublicModelCapabilitySourceOfficialRegistry, PublicModelCapabilitySourceManualConfig:
		return 3
	case PublicModelCapabilitySourcePublishedSnapshot:
		return 4
	case PublicModelCapabilitySourcePricingCatalog:
		return 5
	case PublicModelCapabilitySourceInferred:
		return 6
	default:
		return 7
	}
}

func publicModelMetadataEntryRank(source string, verified bool, support string, lastCheckedAt string) []int {
	verifiedRank := 1
	if verified {
		verifiedRank = 0
	}
	checkedRank := 1
	if strings.TrimSpace(lastCheckedAt) != "" {
		checkedRank = 0
	}
	return []int{
		publicModelCapabilitySourceRank(source),
		verifiedRank,
		publicModelSupportRank(support),
		checkedRank,
	}
}

func publicModelMetadataEntryPreferred(leftSource string, leftVerified bool, leftSupport string, leftChecked string, rightSource string, rightVerified bool, rightSupport string, rightChecked string) bool {
	leftRank := publicModelMetadataEntryRank(leftSource, leftVerified, leftSupport, leftChecked)
	rightRank := publicModelMetadataEntryRank(rightSource, rightVerified, rightSupport, rightChecked)
	for index := range leftRank {
		if leftRank[index] != rightRank[index] {
			return leftRank[index] < rightRank[index]
		}
	}
	return false
}

func maxInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}
