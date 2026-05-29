package service

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/geminicli"
)

// validateTierID validates tier_id format and length
func validateTierID(tierID string) error {
	if tierID == "" {
		return nil // Empty is allowed
	}
	if len(tierID) > 64 {
		return fmt.Errorf("tier_id exceeds maximum length of 64 characters")
	}
	// Allow alphanumeric, underscore, hyphen, and slash (for tier paths)
	if !regexp.MustCompile(`^[a-zA-Z0-9_/-]+$`).MatchString(tierID) {
		return fmt.Errorf("tier_id contains invalid characters")
	}
	return nil
}

func canonicalGeminiTierID(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	lower := strings.ToLower(raw)
	switch lower {
	case GeminiTierGoogleOneFree,
		GeminiTierGoogleAIPro,
		GeminiTierGoogleAIUltra,
		GeminiTierGCPStandard,
		GeminiTierGCPEnterprise,
		GeminiTierAIStudioFree,
		GeminiTierAIStudioTier1,
		GeminiTierAIStudioTier2,
		GeminiTierAIStudioTier3,
		GeminiTierAIStudioPaid,
		GeminiTierGoogleOneUnknown:
		if lower == GeminiTierAIStudioPaid {
			return GeminiTierAIStudioTier1
		}
		return lower
	}

	upper := strings.ToUpper(raw)
	switch upper {
	// Google One legacy tiers
	case legacyTierAIPremium:
		return GeminiTierGoogleAIPro
	case legacyTierGoogleOneUnlimited:
		return GeminiTierGoogleAIUltra
	case legacyTierFree, legacyTierGoogleOneBasic, legacyTierGoogleOneStandard:
		return GeminiTierGoogleOneFree
	case legacyTierGoogleOneUnknown:
		return GeminiTierGoogleOneUnknown

	// Code Assist legacy tiers
	case "STANDARD", "PRO", "LEGACY":
		return GeminiTierGCPStandard
	case "ENTERPRISE", "ULTRA":
		return GeminiTierGCPEnterprise
	}

	// Some Code Assist responses use kebab-case tier identifiers.
	switch lower {
	case "standard-tier", "pro-tier":
		return GeminiTierGCPStandard
	case "ultra-tier":
		return GeminiTierGCPEnterprise
	}

	return ""
}

func canonicalGeminiTierIDForOAuthType(oauthType, tierID string) string {
	oauthType = strings.ToLower(strings.TrimSpace(oauthType))
	canonical := canonicalGeminiTierID(tierID)
	if canonical == "" {
		return ""
	}

	switch oauthType {
	case "google_one":
		switch canonical {
		case GeminiTierGoogleOneFree, GeminiTierGoogleAIPro, GeminiTierGoogleAIUltra:
			return canonical
		default:
			return ""
		}
	case "code_assist":
		switch canonical {
		case GeminiTierGCPStandard, GeminiTierGCPEnterprise:
			return canonical
		default:
			return ""
		}
	case "ai_studio":
		switch canonical {
		case GeminiTierAIStudioFree, GeminiTierAIStudioTier1, GeminiTierAIStudioTier2, GeminiTierAIStudioTier3:
			return canonical
		default:
			return ""
		}
	default:
		// Unknown oauth type: accept canonical tier.
		return canonical
	}
}

// extractTierIDFromAllowedTiers extracts tierID from LoadCodeAssist response
// Prioritizes IsDefault tier, falls back to first non-empty tier
func extractTierIDFromAllowedTiers(allowedTiers []geminicli.AllowedTier) string {
	tierID := "LEGACY"
	// First pass: look for default tier
	for _, tier := range allowedTiers {
		if tier.IsDefault && strings.TrimSpace(tier.ID) != "" {
			tierID = strings.TrimSpace(tier.ID)
			break
		}
	}
	// Second pass: if still LEGACY, take first non-empty tier
	if tierID == "LEGACY" {
		for _, tier := range allowedTiers {
			if strings.TrimSpace(tier.ID) != "" {
				tierID = strings.TrimSpace(tier.ID)
				break
			}
		}
	}
	return tierID
}
