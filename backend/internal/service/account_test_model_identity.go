package service

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/modelregistry"
)

func buildAvailableTestModelFromRegistryDetail(detail modelregistry.AdminModelDetail, sourceProtocol string) AvailableTestModel {
	publicID, normalizedAlias := normalizeAvailableTestModelPublicID(detail)
	status := strings.TrimSpace(detail.Status)
	deprecatedAt := strings.TrimSpace(detail.DeprecatedAt)
	replacedBy := strings.TrimSpace(detail.ReplacedBy)
	if normalizedAlias {
		status = "stable"
		deprecatedAt = ""
		replacedBy = ""
	}

	displayName := firstNonEmptyTestModelLabel(detail.DisplayName, FormatModelCatalogDisplayName(publicID), publicID)
	return AvailableTestModel{
		ID:             publicID,
		Type:           "model",
		DisplayName:    displayName,
		CreatedAt:      "",
		SourceProtocol: normalizeTestSourceProtocol(sourceProtocol),
		Status:         status,
		DeprecatedAt:   deprecatedAt,
		ReplacedBy:     replacedBy,
	}
}

func normalizeAvailableTestModelPublicID(detail modelregistry.AdminModelDetail) (string, bool) {
	rawID := strings.TrimSpace(detail.ID)
	replacedBy := strings.TrimSpace(detail.ReplacedBy)
	if !strings.EqualFold(strings.TrimSpace(detail.Status), "deprecated") || replacedBy == "" {
		return rawID, false
	}

	if normalized := NormalizeModelCatalogModelID(replacedBy); normalized != "" {
		return normalized, normalizeRegistryID(normalized) != normalizeRegistryID(rawID)
	}
	if normalized := NormalizeModelCatalogModelID(rawID); normalized != "" {
		return normalized, normalizeRegistryID(normalized) != normalizeRegistryID(rawID)
	}
	return rawID, false
}
