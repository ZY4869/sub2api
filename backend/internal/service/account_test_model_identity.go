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
	return applyAvailableTestModelProvider(AvailableTestModel{
		ID:             publicID,
		Type:           "model",
		DisplayName:    displayName,
		CreatedAt:      "",
		Mode:           inferAvailableTestModelMode(publicID, &detail.ModelEntry),
		SourceProtocol: normalizeTestSourceProtocol(sourceProtocol),
		Status:         status,
		DeprecatedAt:   deprecatedAt,
		ReplacedBy:     replacedBy,
	}, detail.Provider)
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

func inferAvailableTestModelMode(modelID string, entry *modelregistry.ModelEntry) string {
	if entry != nil {
		for _, capability := range entry.Capabilities {
			switch strings.TrimSpace(strings.ToLower(capability)) {
			case "image_generation":
				return "image"
			case "video_generation":
				return "video"
			case "embedding":
				return "embedding"
			}
		}
		for _, modality := range entry.Modalities {
			switch strings.TrimSpace(strings.ToLower(modality)) {
			case "image":
				return "image"
			case "video":
				return "video"
			case "embedding":
				return "embedding"
			case "text":
				return "text"
			}
		}
	}
	switch inferModelMode(modelID, "") {
	case "image":
		return "image"
	case "video":
		return "video"
	default:
		return "text"
	}
}
