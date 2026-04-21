package handler

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/modelregistry"
	"github.com/Wei-Shaw/sub2api/internal/pkg/gemini"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const (
	geminiModelMetadataSourceProjectedEmpty = "projected_empty"
	geminiModelMetadataSourceUpstream       = "upstream"
	geminiFallbackPageTokenPrefix           = "fallback:"
)

func apiKeyPublicEntriesToGeminiModelsWithRegistry(
	entries []service.APIKeyPublicModelEntry,
	nextPageToken string,
	modelRegistryService *service.ModelRegistryService,
) gemini.ModelsListResponse {
	models := make([]gemini.Model, 0, len(entries))
	for _, entry := range entries {
		models = append(models, apiKeyPublicEntryToGeminiModelWithRegistry(entry, modelRegistryService))
	}
	return gemini.ModelsListResponse{Models: models, NextPageToken: strings.TrimSpace(nextPageToken)}
}

func apiKeyPublicEntryToGeminiModelWithRegistry(
	entry service.APIKeyPublicModelEntry,
	modelRegistryService *service.ModelRegistryService,
) gemini.Model {
	displayName := entry.DisplayName
	if displayName == "" {
		displayName = entry.PublicID
	}
	description := "Gemini model metadata fallback entry; upstream metadata is unavailable and uncertain fields are omitted."
	if platform := strings.TrimSpace(entry.Platform); platform != "" {
		description = "Gemini model metadata fallback entry projected from " + platform + " availability; uncertain upstream fields are omitted."
	}
	return gemini.ProjectMinimalModel(
		entry.PublicID,
		displayName,
		description,
		supportedGenerationMethodsForPublicEntry(entry, modelRegistryService),
	)
}

func supportedGenerationMethodsForPublicEntry(
	entry service.APIKeyPublicModelEntry,
	modelRegistryService *service.ModelRegistryService,
) []string {
	if modelRegistryService == nil {
		return nil
	}
	registryEntry := findRegistryModelEntry(modelRegistryService, entry)
	if registryEntry == nil {
		return nil
	}
	return gemini.SupportedGenerationMethodsForModelWithOptions(entry.PublicID, gemini.SupportedGenerationMethodsOptions{
		Modalities:   append([]string(nil), registryEntry.Modalities...),
		Capabilities: append([]string(nil), registryEntry.Capabilities...),
	})
}

func findRegistryModelEntry(
	modelRegistryService *service.ModelRegistryService,
	entry service.APIKeyPublicModelEntry,
) *modelregistry.ModelEntry {
	for _, candidate := range []string{entry.SourceID, entry.PublicID, entry.AliasID} {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		registryEntry, err := modelRegistryService.GetModel(context.Background(), candidate)
		if err == nil && registryEntry != nil {
			return registryEntry
		}
	}
	return nil
}

func encodeGeminiFallbackPageToken(offset int) string {
	if offset <= 0 {
		return ""
	}
	payload := geminiFallbackPageTokenPrefix + strconv.Itoa(offset)
	return base64.RawURLEncoding.EncodeToString([]byte(payload))
}

func parseGeminiModelsPageToken(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	if value, err := strconv.Atoi(raw); err == nil && value >= 0 {
		return value, nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid pageToken")
	}
	payload := strings.TrimSpace(string(decoded))
	if !strings.HasPrefix(payload, geminiFallbackPageTokenPrefix) {
		return 0, fmt.Errorf("invalid pageToken")
	}
	value, err := strconv.Atoi(strings.TrimPrefix(payload, geminiFallbackPageTokenPrefix))
	if err != nil || value < 0 {
		return 0, fmt.Errorf("invalid pageToken")
	}
	return value, nil
}

func applyGeminiModelMetadataSource(c *gin.Context, source string) {
	if c == nil || c.Request == nil {
		return
	}
	ctx := service.EnsureRequestMetadata(c.Request.Context())
	service.SetGeminiModelMetadataSourceMetadata(ctx, source)
	c.Request = c.Request.WithContext(ctx)
}
