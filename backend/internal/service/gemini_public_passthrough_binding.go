package service

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

func (s *GeminiMessagesCompatService) persistGeminiPassthroughBinding(ctx context.Context, input GeminiPublicPassthroughInput, account *Account, binding *UpstreamResourceBinding, statusCode int, body []byte) error {
	if s.resourceBindingRepo == nil || strings.TrimSpace(input.ResourceKind) == "" {
		return nil
	}
	resourceName := extractGeminiPassthroughResourceName(input.ResourceKind, input.Path)
	if binding != nil && shouldSoftDeleteGeminiPassthroughBinding(input.Method) && resourceName != "" && isSuccessfulHTTPStatus(statusCode) {
		return s.resourceBindingRepo.SoftDelete(ctx, input.ResourceKind, resourceName)
	}
	if !shouldUpsertGeminiPassthroughBinding(input.Method) || !isSuccessfulHTTPStatus(statusCode) {
		return nil
	}
	resourceNames := extractGeminiPassthroughCreatedResourceNames(input.ResourceKind, body)
	for _, createdName := range resourceNames {
		if strings.TrimSpace(createdName) == "" {
			continue
		}
		apiKeyID := input.APIKeyID
		userID := input.UserID
		if err := s.resourceBindingRepo.Upsert(ctx, &UpstreamResourceBinding{
			ResourceKind:   input.ResourceKind,
			ResourceName:   createdName,
			ProviderFamily: UpstreamProviderAIStudio,
			AccountID:      account.ID,
			APIKeyID:       &apiKeyID,
			GroupID:        input.GroupID,
			UserID:         &userID,
			MetadataJSON: buildGoogleBatchBindingMetadata(map[string]any{
				"requested_model": strings.TrimSpace(firstNonEmptyString(input.RequestedModel, detectGeminiPassthroughRequestedModel(input.Path, input.Body))),
				"path":            strings.TrimSpace(input.Path),
			}),
		}); err != nil {
			return err
		}
	}
	return nil
}

func shouldSoftDeleteGeminiPassthroughBinding(method string) bool {
	return strings.EqualFold(strings.TrimSpace(method), http.MethodDelete)
}

func shouldUpsertGeminiPassthroughBinding(method string) bool {
	switch strings.ToUpper(strings.TrimSpace(method)) {
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		return true
	default:
		return false
	}
}

func extractGeminiPassthroughCreatedResourceNames(resourceKind string, body []byte) []string {
	switch resourceKind {
	case UpstreamResourceKindGeminiFile, UpstreamResourceKindGeminiBatch:
		return extractOpenAICompatObjectIDs(body)
	case UpstreamResourceKindGeminiCachedContent,
		UpstreamResourceKindGeminiFileSearchStore,
		UpstreamResourceKindGeminiDocument,
		UpstreamResourceKindGeminiOperation,
		UpstreamResourceKindGeminiUploadOperation,
		UpstreamResourceKindGeminiInteraction,
		UpstreamResourceKindGeminiCorpus,
		UpstreamResourceKindGeminiCorpusOperation,
		UpstreamResourceKindGeminiCorpusPermission,
		UpstreamResourceKindGeminiDynamic,
		UpstreamResourceKindGeminiGeneratedFile,
		UpstreamResourceKindGeminiGeneratedFileOperation,
		UpstreamResourceKindGeminiModelOperation,
		UpstreamResourceKindGeminiTunedModel,
		UpstreamResourceKindGeminiTunedModelPermission,
		UpstreamResourceKindGeminiTunedModelOperation:
		return extractTopLevelNames(body)
	default:
		return nil
	}
}

func extractGeminiPassthroughResourceName(resourceKind string, path string) string {
	trimmed := strings.TrimSpace(path)
	switch resourceKind {
	case UpstreamResourceKindGeminiFile:
		if id := extractAIStudioResourceName(trimmed, "/v1beta/files/"); id != "" {
			return id
		}
		return extractOpenAICompatResourceID(trimmed, "/v1beta/openai/files/")
	case UpstreamResourceKindGeminiBatch:
		if id := extractAIStudioResourceName(trimmed, "/v1beta/batches/"); id != "" {
			return id
		}
		return extractOpenAICompatResourceID(trimmed, "/v1beta/openai/batches/")
	case UpstreamResourceKindGeminiCachedContent:
		return extractAIStudioResourceName(trimmed, "/v1beta/cachedContents/")
	case UpstreamResourceKindGeminiFileSearchStore:
		return extractAIStudioResourceName(trimmed, "/v1beta/fileSearchStores/")
	case UpstreamResourceKindGeminiDocument:
		if name := extractAIStudioResourceName(trimmed, "/v1beta/documents/"); name != "" {
			return name
		}
		return extractGeminiNestedResourceName(trimmed, "documents")
	case UpstreamResourceKindGeminiOperation:
		if name := extractAIStudioResourceName(trimmed, "/v1beta/operations/"); name != "" {
			return name
		}
		return extractGeminiNestedResourceName(trimmed, "operations")
	case UpstreamResourceKindGeminiUploadOperation:
		return extractGeminiNestedUploadOperationName(trimmed)
	case UpstreamResourceKindGeminiInteraction:
		return extractAIStudioResourceName(trimmed, "/v1beta/interactions/")
	case UpstreamResourceKindGeminiCorpus:
		return extractAIStudioResourceName(trimmed, "/v1beta/corpora/")
	case UpstreamResourceKindGeminiCorpusOperation:
		return extractGeminiNestedCollectionResourceName(trimmed, "/v1beta/corpora/", "corpora", "/operations/")
	case UpstreamResourceKindGeminiCorpusPermission:
		return extractGeminiNestedCollectionResourceName(trimmed, "/v1beta/corpora/", "corpora", "/permissions/")
	case UpstreamResourceKindGeminiDynamic:
		return extractAIStudioResourceName(trimmed, "/v1beta/dynamic/")
	case UpstreamResourceKindGeminiGeneratedFile:
		return extractAIStudioResourceName(trimmed, "/v1beta/generatedFiles/")
	case UpstreamResourceKindGeminiGeneratedFileOperation:
		return extractGeminiNestedCollectionResourceName(trimmed, "/v1beta/generatedFiles/", "generatedFiles", "/operations/")
	case UpstreamResourceKindGeminiModelOperation:
		return extractGeminiNestedCollectionResourceName(trimmed, "/v1beta/models/", "models", "/operations/")
	case UpstreamResourceKindGeminiTunedModel:
		return extractAIStudioResourceName(trimmed, "/v1beta/tunedModels/")
	case UpstreamResourceKindGeminiTunedModelPermission:
		return extractGeminiNestedCollectionResourceName(trimmed, "/v1beta/tunedModels/", "tunedModels", "/permissions/")
	case UpstreamResourceKindGeminiTunedModelOperation:
		return extractGeminiNestedCollectionResourceName(trimmed, "/v1beta/tunedModels/", "tunedModels", "/operations/")
	}
	return ""
}

func extractGeminiNestedCollectionResourceName(path string, parentPrefix string, parentCollection string, nestedMarker string) string {
	trimmed := strings.TrimSpace(path)
	if strings.HasPrefix(trimmed, parentPrefix) {
		trimmed = strings.TrimPrefix(trimmed, parentPrefix)
	}
	idx := strings.Index(trimmed, nestedMarker)
	if idx < 0 {
		return ""
	}
	parent := strings.Trim(trimmed[:idx], "/")
	nested := strings.Trim(strings.TrimPrefix(trimmed[idx:], nestedMarker), "/")
	for _, sep := range []string{"/", ":", "?"} {
		if cut := strings.Index(nested, sep); cut >= 0 {
			nested = nested[:cut]
		}
	}
	if parent == "" || nested == "" {
		return ""
	}
	collection := strings.Trim(strings.TrimSpace(nestedMarker), "/")
	parentCollection = strings.Trim(strings.TrimSpace(parentCollection), "/")
	if collection == "" || parentCollection == "" {
		return ""
	}
	return parentCollection + "/" + parent + "/" + collection + "/" + nested
}

func extractGeminiNestedResourceName(path string, collection string) string {
	trimmed := strings.TrimSpace(path)
	marker := "/" + strings.TrimSpace(collection) + "/"
	idx := strings.Index(trimmed, marker)
	if idx < 0 {
		return ""
	}
	prefix := strings.Trim(trimmed[:idx], "/")
	suffix := strings.Trim(trimmed[idx+1:], "/")
	for _, sep := range []string{":", "?"} {
		if cut := strings.Index(suffix, sep); cut >= 0 {
			suffix = suffix[:cut]
		}
	}
	if prefix == "" || suffix == "" {
		return ""
	}
	return prefix + "/" + suffix
}

func extractGeminiNestedUploadOperationName(path string) string {
	trimmed := strings.TrimSpace(path)
	marker := "/upload/operations/"
	idx := strings.Index(trimmed, marker)
	if idx < 0 {
		return ""
	}
	prefix := strings.Trim(trimmed[:idx], "/")
	suffix := strings.Trim(trimmed[idx+1:], "/")
	for _, sep := range []string{":", "?"} {
		if cut := strings.Index(suffix, sep); cut >= 0 {
			suffix = suffix[:cut]
		}
	}
	if prefix == "" || suffix == "" {
		return ""
	}
	return prefix + "/" + suffix
}

func extractOpenAICompatObjectIDs(body []byte) []string {
	ids := append(extractTopLevelIDs(body), extractListIDs(body, "data")...)
	return uniqueStrings(ids)
}

func extractTopLevelIDs(body []byte) []string {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil
	}
	id := strings.TrimSpace(stringMapValue(payload, "id"))
	if id == "" {
		return nil
	}
	return []string{id}
}

func extractListIDs(body []byte, listKey string) []string {
	items := extractNamedListItems(body, listKey)
	ids := make([]string, 0, len(items))
	for _, item := range items {
		if id := strings.TrimSpace(stringMapValue(item, "id")); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

func extractOpenAICompatResourceID(path string, prefix string) string {
	trimmed := strings.TrimSpace(path)
	if !strings.HasPrefix(trimmed, prefix) {
		return ""
	}
	resourceID := strings.TrimPrefix(trimmed, prefix)
	for _, sep := range []string{":", "/", "?"} {
		if idx := strings.Index(resourceID, sep); idx >= 0 {
			resourceID = resourceID[:idx]
		}
	}
	return strings.TrimSpace(resourceID)
}

func isSuccessfulHTTPStatus(status int) bool {
	return status >= 200 && status < 300
}
