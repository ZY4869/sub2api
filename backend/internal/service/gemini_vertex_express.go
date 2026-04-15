package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"

	pkggemini "github.com/Wei-Shaw/sub2api/internal/pkg/gemini"
	"github.com/Wei-Shaw/sub2api/internal/pkg/geminicli"
)

const (
	GeminiAPIKeyVariantAIStudio      = "ai_studio"
	GeminiAPIKeyVariantVertexExpress = "vertex_express"
	GeminiVertexDefaultAliasPrefix   = "Vertex-"
)

func NormalizeGeminiAPIKeyVariant(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case GeminiAPIKeyVariantVertexExpress:
		return GeminiAPIKeyVariantVertexExpress
	default:
		return GeminiAPIKeyVariantAIStudio
	}
}

func defaultGeminiAPIKeyBaseURLForVariant(variant string) string {
	if NormalizeGeminiAPIKeyVariant(variant) == GeminiAPIKeyVariantVertexExpress {
		return geminicli.VertexAIBaseURL
	}
	return geminicli.AIStudioBaseURL
}

func NormalizeGeminiCredentialsForStorage(accountType string, credentials map[string]any) map[string]any {
	if len(credentials) == 0 {
		credentials = map[string]any{}
	}
	normalized := make(map[string]any, len(credentials)+2)
	for key, value := range credentials {
		normalized[key] = value
	}

	switch strings.TrimSpace(strings.ToLower(accountType)) {
	case AccountTypeAPIKey:
		normalized["gemini_api_variant"] = NormalizeGeminiAPIKeyVariant(stringCredentialValue(normalized["gemini_api_variant"]))
		normalized["api_key"] = strings.TrimSpace(stringCredentialValue(normalized["api_key"]))
		baseURL := strings.TrimSpace(stringCredentialValue(normalized["base_url"]))
		if baseURL == "" {
			baseURL = defaultGeminiAPIKeyBaseURLForVariant(stringCredentialValue(normalized["gemini_api_variant"]))
		}
		normalized["base_url"] = strings.TrimRight(baseURL, "/")
	case AccountTypeOAuth:
		if strings.EqualFold(strings.TrimSpace(stringCredentialValue(normalized["oauth_type"])), "vertex_ai") {
			normalized["oauth_type"] = "vertex_ai"
			normalized["vertex_project_id"] = strings.TrimSpace(stringCredentialValue(normalized["vertex_project_id"]))
			location := normalizeVertexLocation(stringCredentialValue(normalized["vertex_location"]))
			normalized["vertex_location"] = location
			normalized["vertex_service_account_json"] = strings.TrimSpace(stringCredentialValue(normalized["vertex_service_account_json"]))
			normalized["access_token"] = strings.TrimSpace(stringCredentialValue(normalized["access_token"]))
			baseURL := strings.TrimSpace(stringCredentialValue(normalized["base_url"]))
			if baseURL == "" {
				baseURL = DefaultGeminiVertexBaseURL(location)
			}
			normalized["base_url"] = strings.TrimRight(baseURL, "/")
		}
	}
	if canonicalTierID := canonicalGeminiTierID(stringCredentialValue(normalized["tier_id"])); canonicalTierID != "" {
		normalized["tier_id"] = canonicalTierID
	}
	return normalized
}

func stringCredentialValue(value any) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	case json.Number:
		return v.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

func normalizeVertexLocation(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "global"
	}
	return value
}

func (a *Account) GeminiAPIKeyVariant() string {
	if a == nil || EffectiveProtocol(a) != PlatformGemini || a.Type != AccountTypeAPIKey {
		return ""
	}
	return NormalizeGeminiAPIKeyVariant(a.GetCredential("gemini_api_variant"))
}

func (a *Account) IsGeminiVertexExpress() bool {
	if a == nil {
		return false
	}
	return a.GeminiAPIKeyVariant() == GeminiAPIKeyVariantVertexExpress
}

func (a *Account) IsGeminiVertexSource() bool {
	if a == nil {
		return false
	}
	return a.IsGeminiVertexAI() || a.IsGeminiVertexExpress()
}

func DefaultVertexPublicModelAlias(modelID string) string {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return GeminiVertexDefaultAliasPrefix
	}
	return GeminiVertexDefaultAliasPrefix + modelID
}

func (a *Account) GetGeminiVertexExpressBaseURL(defaultBaseURL string) string {
	if a == nil {
		return defaultBaseURL
	}
	baseURL := strings.TrimSpace(a.GetCredential("base_url"))
	if baseURL == "" {
		return defaultBaseURL
	}
	return baseURL
}

func (a *Account) GeminiVertexExpressModelActionPath(model, action string) (string, error) {
	modelID := strings.TrimSpace(model)
	if modelID == "" {
		return "", fmt.Errorf("missing model")
	}
	action = strings.TrimSpace(action)
	if action == "" {
		return "", fmt.Errorf("missing action")
	}
	return fmt.Sprintf(
		"/v1/publishers/google/models/%s:%s",
		url.PathEscape(modelID),
		action,
	), nil
}

func buildGeminiVertexCatalogModelsResponseFromCatalog(models []VertexCatalogModel) (*UpstreamHTTPResult, error) {
	items := make([]pkggemini.Model, 0, len(models))
	for _, model := range models {
		modelID := strings.TrimSpace(model.ID)
		if modelID == "" {
			continue
		}
		displayName := strings.TrimSpace(model.DisplayName)
		if displayName == "" {
			displayName = FormatModelCatalogDisplayName(modelID)
		}
		items = append(items, pkggemini.Model{
			Name:                       "models/" + modelID,
			DisplayName:                displayName,
			SupportedGenerationMethods: pkggemini.SupportedGenerationMethodsForModel(modelID),
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})
	return buildGeminiUpstreamJSONResult(http.StatusOK, pkggemini.ModelsListResponse{Models: items})
}

func buildGeminiVertexCatalogModelResponseFromCatalog(modelID string, models []VertexCatalogModel) (*UpstreamHTTPResult, error) {
	modelID = strings.TrimSpace(strings.TrimPrefix(modelID, "models/"))
	for _, item := range models {
		if strings.TrimSpace(item.ID) != modelID {
			continue
		}
		displayName := strings.TrimSpace(item.DisplayName)
		if displayName == "" {
			displayName = FormatModelCatalogDisplayName(modelID)
		}
		return buildGeminiUpstreamJSONResult(http.StatusOK, pkggemini.Model{
			Name:                       "models/" + modelID,
			DisplayName:                displayName,
			SupportedGenerationMethods: pkggemini.SupportedGenerationMethodsForModel(modelID),
		})
	}
	return buildGeminiUpstreamJSONResult(http.StatusNotFound, map[string]any{
		"error": map[string]any{
			"code":    http.StatusNotFound,
			"message": "Model not found",
			"status":  "NOT_FOUND",
		},
	})
}

func buildGeminiUpstreamJSONResult(statusCode int, payload any) (*UpstreamHTTPResult, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	headers := make(http.Header)
	headers.Set("Content-Type", "application/json")
	return &UpstreamHTTPResult{
		StatusCode: statusCode,
		Headers:    headers,
		Body:       body,
	}, nil
}

func (s *GeminiMessagesCompatService) buildGeminiAPIKeyUpstreamRequest(
	ctx context.Context,
	account *Account,
	mappedModel string,
	action string,
	body []byte,
	mimicGeminiCLI bool,
) (*http.Request, string, error) {
	if account == nil {
		return nil, "", fmt.Errorf("account is nil")
	}
	apiKey := strings.TrimSpace(account.GetCredential("api_key"))
	if apiKey == "" {
		return nil, "", fmt.Errorf("gemini api_key not configured")
	}
	if account.IsGeminiVertexExpress() {
		mappedModel = normalizeVertexUpstreamModelID(mappedModel)
		baseURL := account.GetGeminiVertexExpressBaseURL(geminicli.VertexAIBaseURL)
		normalizedBaseURL, err := s.validateUpstreamBaseURL(baseURL)
		if err != nil {
			return nil, "", err
		}
		actionPath, err := account.GeminiVertexExpressModelActionPath(mappedModel, action)
		if err != nil {
			return nil, "", err
		}
		upstreamReq, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(normalizedBaseURL, "/")+actionPath, bytes.NewReader(body))
		if err != nil {
			return nil, "", err
		}
		query := upstreamReq.URL.Query()
		query.Set("key", apiKey)
		if action == "streamGenerateContent" {
			query.Set("alt", "sse")
		}
		upstreamReq.URL.RawQuery = query.Encode()
		upstreamReq.Header.Set("Content-Type", "application/json")
		if mimicGeminiCLI {
			upstreamReq.Header.Set("User-Agent", geminicli.GeminiCLIUserAgent)
		}
		return upstreamReq, "x-request-id", nil
	}
	baseURL := account.GetGeminiBaseURL(geminicli.AIStudioBaseURL)
	normalizedBaseURL, err := s.validateUpstreamBaseURL(baseURL)
	if err != nil {
		return nil, "", err
	}
	fullURL := fmt.Sprintf("%s/v1beta/models/%s:%s", strings.TrimRight(normalizedBaseURL, "/"), mappedModel, action)
	if action == "streamGenerateContent" {
		fullURL += "?alt=sse"
	}
	restBody := normalizeGeminiRequestForAIStudio(body)
	upstreamReq, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, bytes.NewReader(restBody))
	if err != nil {
		return nil, "", err
	}
	upstreamReq.Header.Set("Content-Type", "application/json")
	upstreamReq.Header.Set("x-goog-api-key", apiKey)
	if mimicGeminiCLI {
		upstreamReq.Header.Set("User-Agent", geminicli.GeminiCLIUserAgent)
	}
	return upstreamReq, "x-request-id", nil
}
