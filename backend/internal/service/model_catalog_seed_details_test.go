//go:build unit

package service

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/modelregistry"
	"github.com/stretchr/testify/require"
)

func TestAppendBundledModelCatalogSeedDetails_LegacyGeminiProtocolIDsStayLegacyOnly(t *testing.T) {
	base := []modelregistry.AdminModelDetail{
		{
			ModelEntry: modelregistry.ModelEntry{
				ID:               "gemini-3-pro-preview",
				ProtocolIDs:      []string{"gemini-3-pro-preview"},
				PricingLookupIDs: []string{"gemini-3-pro-preview"},
			},
			Available: true,
		},
		{
			ModelEntry: modelregistry.ModelEntry{
				ID:               "gemini-3.1-pro-preview",
				ProtocolIDs:      []string{"gemini-3.1-pro-preview"},
				PricingLookupIDs: []string{"gemini-3.1-pro-preview"},
			},
			Available: true,
		},
	}

	details := appendBundledModelCatalogSeedDetails(base)

	legacyProtocolIDs := map[string][]string{
		"gemini-3-pro":   []string{"gemini-3-pro"},
		"gemini-3.1-pro": []string{"gemini-3.1-pro"},
	}
	for modelID, wantProtocolIDs := range legacyProtocolIDs {
		var detail *modelregistry.AdminModelDetail
		for i := range details {
			if details[i].ID == modelID {
				detail = &details[i]
				break
			}
		}
		require.NotNil(t, detail, "missing bundled detail for %s", modelID)
		require.ElementsMatch(t, wantProtocolIDs, detail.ProtocolIDs)
		require.Contains(t, detail.PricingLookupIDs, modelID)
	}

	collisions := collectBillingIdentifierCollisions(details)
	for _, collision := range collisions {
		if collision.Source != "protocol_ids" {
			continue
		}
		require.NotEqual(t, "gemini-3-pro-preview", collision.Identifier)
		require.NotEqual(t, "gemini-3.1-pro-preview", collision.Identifier)
	}
}
