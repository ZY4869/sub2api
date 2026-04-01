package service

import (
	"context"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	googleBatchBindingMetadataSourceResourceNames = "source_resource_names"
	googleBatchBindingMetadataEstimatedTokens     = "estimated_batch_tokens"
	googleBatchBindingMetadataModelFamily         = "model_family"
	googleBatchBindingMetadataRequestedModel      = "requested_model"
	googleBatchBindingMetadataSourceProtocol      = "source_protocol"
	googleBatchBindingMetadataContentDigest       = "content_digest"
	googleBatchBindingMetadataUploadedAt          = "uploaded_at"
)

type googleBatchResolvedInputMetadata struct {
	requestedModel      string
	modelFamily         string
	estimatedTokens     int64
	sourceProtocol      string
	sourceResourceNames []string
}

var googleBatchModelFamilyRegistry = map[string]string{
	"gemini_pro":                    "gemini_pro",
	"gemini-pro":                    "gemini_pro",
	"gemini-1.5-pro":                "gemini_pro",
	"gemini-1.5-pro-latest":         "gemini_pro",
	"gemini-2.5-pro":                "gemini_pro",
	"gemini-2.5-pro-preview":        "gemini_pro",
	"gemini-2.5-pro-preview-tts":    "gemini_pro",
	"gemini-3-pro":                  "gemini_pro",
	"gemini-3-pro-preview":          "gemini_pro",
	"gemini_flash":                  "gemini_flash",
	"gemini-flash":                  "gemini_flash",
	"gemini-1.5-flash":              "gemini_flash",
	"gemini-1.5-flash-latest":       "gemini_flash",
	"gemini-2.5-flash":              "gemini_flash",
	"gemini-2.5-flash-preview":      "gemini_flash",
	"gemini-3.1-flash":              "gemini_flash",
	"gemini-3.1-flash-preview":      "gemini_flash",
	"gemini_flash_lite":             "gemini_flash_lite",
	"gemini-2.5-flash-lite":         "gemini_flash_lite",
	"gemini-2.5-flash-lite-preview": "gemini_flash_lite",
	"gemini-3.1-flash-lite":         "gemini_flash_lite",
	"gemini-3.1-flash-lite-preview": "gemini_flash_lite",
	"gemini_2_flash":                "gemini_2_flash",
	"gemini-2.0-flash":              "gemini_2_flash",
	"gemini-2.0-flash-001":          "gemini_2_flash",
	"gemini-2.0-flash-exp":          "gemini_2_flash",
}

var googleBatchModelFamilyRegistryPrefixes = []struct {
	prefix string
	family string
}{
	{prefix: "gemini-2.5-pro", family: "gemini_pro"},
	{prefix: "gemini-3-pro", family: "gemini_pro"},
	{prefix: "gemini-2.5-flash-lite", family: "gemini_flash_lite"},
	{prefix: "gemini-3.1-flash-lite", family: "gemini_flash_lite"},
	{prefix: "gemini-2.0-flash", family: "gemini_2_flash"},
	{prefix: "gemini-2.5-flash", family: "gemini_flash"},
	{prefix: "gemini-3.1-flash", family: "gemini_flash"},
}

func googleBatchModelRegistryFamily(model string) (string, bool) {
	normalized := normalizeGoogleBatchModelRegistryKey(model)
	if normalized == "" {
		return "", false
	}
	if family, ok := googleBatchModelFamilyRegistry[normalized]; ok {
		return family, true
	}
	for _, candidate := range googleBatchModelFamilyRegistryPrefixes {
		if strings.HasPrefix(normalized, candidate.prefix) {
			return candidate.family, true
		}
	}
	return "", false
}

func normalizeGoogleBatchModelRegistryKey(model string) string {
	value := strings.ToLower(strings.TrimSpace(model))
	if value == "" {
		return ""
	}
	for _, prefix := range []string{
		"projects/",
		"publishers/google/models/",
		"models/",
	} {
		if strings.HasPrefix(value, prefix) {
			if idx := strings.LastIndex(value, "/models/"); idx >= 0 {
				value = value[idx+len("/models/"):]
				break
			}
			value = strings.TrimPrefix(value, prefix)
		}
	}
	if idx := strings.Index(value, ":"); idx >= 0 {
		value = value[:idx]
	}
	return strings.TrimSpace(value)
}

func buildGoogleBatchSelectorFromResolvedMetadata(metadata googleBatchResolvedInputMetadata) *vertexBatchSelector {
	modelFamily := strings.TrimSpace(metadata.modelFamily)
	if modelFamily == "" {
		modelFamily = normalizeGoogleBatchModelFamily(metadata.requestedModel)
	}
	return &vertexBatchSelector{
		modelFamily:     modelFamily,
		estimatedTokens: metadata.estimatedTokens,
	}
}

func (s *GeminiMessagesCompatService) buildGoogleBatchSelector(ctx context.Context, input GoogleBatchForwardInput) (*vertexBatchSelector, error) {
	metadata, err := s.resolveGoogleBatchInputMetadata(ctx, input)
	if err != nil {
		return nil, err
	}
	return buildGoogleBatchSelectorFromResolvedMetadata(metadata), nil
}

func (s *GeminiMessagesCompatService) buildGoogleBatchSelectorBestEffort(ctx context.Context, input GoogleBatchForwardInput) *vertexBatchSelector {
	metadata, err := s.resolveGoogleBatchInputMetadata(ctx, input)
	if err != nil {
		return buildGoogleBatchSelectorFromInput(input)
	}
	return buildGoogleBatchSelectorFromResolvedMetadata(metadata)
}

func (s *GeminiMessagesCompatService) resolveGoogleBatchInputMetadata(ctx context.Context, input GoogleBatchForwardInput) (googleBatchResolvedInputMetadata, error) {
	metadata := googleBatchResolvedInputMetadata{
		sourceProtocol:      publicGoogleBatchProtocol(input.Path),
		sourceResourceNames: uniqueStrings(collectStringFieldsByKey(input.Body, "fileName")),
	}
	metadata.requestedModel = strings.TrimSpace(extractGoogleBatchModelID(input.Path, input.Body))
	bindings, err := s.resolveGoogleBatchReferencedFileBindings(ctx, metadata.sourceResourceNames)
	if err != nil {
		return metadata, err
	}
	if len(bindings) > 0 {
		mergeGoogleBatchResolvedInputMetadataFromBindings(&metadata, bindings)
	} else {
		metadata.estimatedTokens = estimateGoogleBatchTokensFromPayload(input.Body)
	}
	if metadata.modelFamily == "" {
		metadata.modelFamily = normalizeGoogleBatchModelFamily(metadata.requestedModel)
	}
	return metadata, nil
}

func (s *GeminiMessagesCompatService) resolveGoogleBatchReferencedFileBindings(ctx context.Context, fileNames []string) ([]*UpstreamResourceBinding, error) {
	if len(fileNames) == 0 || s == nil || s.resourceBindingRepo == nil {
		return nil, nil
	}
	bindings, err := s.resourceBindingRepo.GetByNames(ctx, UpstreamResourceKindGeminiFile, fileNames)
	if err != nil {
		return nil, err
	}
	indexed := make(map[string]*UpstreamResourceBinding, len(bindings))
	for _, binding := range bindings {
		if binding == nil {
			continue
		}
		indexed[strings.TrimSpace(strings.ToLower(binding.ResourceName))] = binding
	}
	ordered := make([]*UpstreamResourceBinding, 0, len(fileNames))
	for _, fileName := range fileNames {
		binding, ok := indexed[strings.TrimSpace(strings.ToLower(fileName))]
		if !ok || binding == nil {
			return nil, infraerrors.Conflict("GEMINI_BATCH_FILE_BINDING_MISSING", "Gemini batch file binding not found")
		}
		ordered = append(ordered, binding)
	}
	return ordered, nil
}

func mergeGoogleBatchResolvedInputMetadataFromBindings(metadata *googleBatchResolvedInputMetadata, bindings []*UpstreamResourceBinding) {
	if metadata == nil || len(bindings) == 0 {
		return
	}
	if requestedModel := stableGoogleBatchBindingMetadataString(bindings, googleBatchBindingMetadataRequestedModel); metadata.requestedModel == "" && requestedModel != "" {
		metadata.requestedModel = requestedModel
	}
	if modelFamily := stableGoogleBatchBindingMetadataString(bindings, googleBatchBindingMetadataModelFamily); modelFamily != "" {
		metadata.modelFamily = modelFamily
	}
	if estimatedTokens := sumGoogleBatchBindingEstimatedTokens(bindings); estimatedTokens > 0 {
		metadata.estimatedTokens = estimatedTokens
	}
	if sourceProtocol := stableGoogleBatchBindingMetadataString(bindings, googleBatchBindingMetadataSourceProtocol); sourceProtocol != "" {
		metadata.sourceProtocol = sourceProtocol
	}
}

func stableGoogleBatchBindingMetadataString(bindings []*UpstreamResourceBinding, key string) string {
	var stable string
	for _, binding := range bindings {
		current, ok := metadataString(bindingMetadata(binding), key)
		if !ok || current == "" {
			continue
		}
		if stable == "" {
			stable = current
			continue
		}
		if !strings.EqualFold(stable, current) {
			return ""
		}
	}
	return stable
}

func sumGoogleBatchBindingEstimatedTokens(bindings []*UpstreamResourceBinding) int64 {
	var total int64
	for _, binding := range bindings {
		value, ok := metadataInt64(bindingMetadata(binding), googleBatchBindingMetadataEstimatedTokens)
		if !ok || value <= 0 {
			continue
		}
		total += value
	}
	return total
}
