package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/antigravity"
	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/geminicli"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"go.uber.org/zap"
)

const (
	openAIModelsURL    = "https://api.openai.com/v1/models"
	anthropicModelsURL = "https://api.anthropic.com/v1/models"
	maxImportBodyBytes = 1 << 20
)

func (s *AccountModelImportService) detectModels(ctx context.Context, account *Account) (*accountModelProbeResult, error) {
	if IsProtocolGatewayAccount(account) && GetAccountGatewayProtocol(account) == GatewayProtocolMixed {
		return s.detectMixedProtocolGatewayModels(ctx, account)
	}
	switch RoutingPlatformForAccount(account) {
	case PlatformOpenAI:
		models, err := s.detectOpenAIModels(ctx, account)
		if err != nil {
			return nil, err
		}
		return newAccountModelProbeResult(models), nil
	case PlatformCopilot:
		models, err := s.detectCopilotModels(ctx, account)
		if err == nil {
			return newAccountModelProbeResult(models), nil
		}
		return &accountModelProbeResult{
			Models: copilotDefaultModelIDs(),
			Source: accountModelProbeSourceCopilotStaticFallback,
			Notice: "upstream /models detection failed; imported Copilot default models instead",
		}, nil
	case PlatformGemini:
		return s.detectGeminiModels(ctx, account)
	case PlatformSora:
		models, err := s.detectSoraModels(ctx, account)
		if err != nil {
			return nil, err
		}
		return newAccountModelProbeResult(models), nil
	case PlatformGrok:
		return s.detectGrokModels(ctx, account)
	case PlatformAntigravity:
		models, err := s.detectAntigravityModels(ctx, account)
		if err != nil {
			return nil, err
		}
		return newAccountModelProbeResult(models), nil
	case PlatformAnthropic:
		models, err := s.detectAnthropicModels(ctx, account)
		if err != nil {
			return nil, err
		}
		return newAccountModelProbeResult(models), nil
	case PlatformKiro:
		return &accountModelProbeResult{
			Models: KiroBuiltinModelIDs(),
			Source: KiroBuiltinCatalogSource,
			Notice: KiroBuiltinCatalogNotice,
		}, nil
	default:
		return nil, infraerrors.BadRequest("ACCOUNT_PLATFORM_UNSUPPORTED", "current account platform does not support model import")
	}
}

func (s *AccountModelImportService) detectMixedProtocolGatewayModels(ctx context.Context, account *Account) (*accountModelProbeResult, error) {
	acceptedProtocols := GetAccountGatewayAcceptedProtocols(account)
	if len(acceptedProtocols) == 0 {
		return nil, infraerrors.BadRequest("ACCOUNT_PLATFORM_UNSUPPORTED", "mixed protocol gateway account requires accepted protocols")
	}
	mergedModels := make([]string, 0)
	detailByID := make(map[string]AccountModelProbeModel)
	sources := make([]string, 0, len(acceptedProtocols))
	notices := make([]string, 0, len(acceptedProtocols))
	for _, protocol := range acceptedProtocols {
		protocolAccount := ResolveProtocolGatewayInboundAccount(account, protocol)
		probeResult, err := s.detectModels(ctx, protocolAccount)
		if err != nil {
			return nil, err
		}
		if probeResult == nil {
			continue
		}
		source := strings.TrimSpace(probeResult.Source)
		if source != "" {
			sources = append(sources, source)
		}
		if notice := strings.TrimSpace(probeResult.Notice); notice != "" {
			notices = append(notices, notice)
		}
		for _, modelID := range probeResult.Models {
			modelID = strings.TrimSpace(modelID)
			if modelID == "" {
				continue
			}
			if _, exists := detailByID[modelID]; !exists {
				mergedModels = append(mergedModels, modelID)
				detailByID[modelID] = AccountModelProbeModel{
					ID:             modelID,
					DisplayName:    FormatModelCatalogDisplayName(modelID),
					SourceProtocol: protocol,
				}
			}
		}
		for _, detail := range probeResult.Details {
			modelID := strings.TrimSpace(detail.ID)
			if modelID == "" {
				continue
			}
			if existing, ok := detailByID[modelID]; ok {
				if strings.TrimSpace(detail.DisplayName) != "" {
					existing.DisplayName = detail.DisplayName
				}
				if existing.SourceProtocol == "" {
					existing.SourceProtocol = protocol
				}
				detailByID[modelID] = existing
			}
		}
	}
	sort.Strings(mergedModels)
	details := make([]AccountModelProbeModel, 0, len(mergedModels))
	for _, modelID := range mergedModels {
		details = append(details, detailByID[modelID])
	}
	return &accountModelProbeResult{
		Models:  mergedModels,
		Details: details,
		Source:  strings.Join(uniqueStrings(sources), "+"),
		Notice:  strings.Join(uniqueStrings(notices), "; "),
	}, nil
}

func (s *AccountModelImportService) detectOpenAIModels(ctx context.Context, account *Account) ([]string, error) {
	token := strings.TrimSpace(account.GetCredential("access_token"))
	if account.Type == AccountTypeAPIKey || account.Type == AccountTypeUpstream {
		token = strings.TrimSpace(account.GetCredential("api_key"))
	}
	if token == "" {
		return nil, infraerrors.BadRequest("ACCOUNT_CREDENTIAL_REQUIRED", "missing OpenAI credential for model import")
	}
	url := openAIModelsURL
	if account.Type == AccountTypeAPIKey || account.Type == AccountTypeUpstream {
		baseURL := strings.TrimSpace(account.GetOpenAIBaseURL())
		if baseURL == "" {
			baseURL = strings.TrimSpace(account.GetCredential("base_url"))
		}
		if baseURL != "" {
			url = strings.TrimRight(baseURL, "/") + "/v1/models"
		}
	}
	headers := map[string]string{
		"Authorization": "Bearer " + token,
		"Accept":        "application/json",
	}
	if v := strings.TrimSpace(account.GetOpenAIUserAgent()); v != "" {
		headers["User-Agent"] = v
	}
	if v := strings.TrimSpace(account.GetOpenAIOrganizationID()); v != "" {
		headers["OpenAI-Organization"] = v
	}
	body, err := s.doImportGET(ctx, account, url, headers, false)
	if err != nil {
		return nil, err
	}
	return parseOpenAIModelList(body)
}

func (s *AccountModelImportService) detectCopilotModels(ctx context.Context, account *Account) ([]string, error) {
	if account == nil || !isCopilotOAuthAccount(account) {
		return nil, infraerrors.BadRequest("ACCOUNT_TYPE_UNSUPPORTED", "current Copilot account type does not support model import")
	}
	if s.openAITokenProvider == nil {
		return nil, infraerrors.InternalServer("MODEL_IMPORT_OPENAI_TOKEN_PROVIDER_UNAVAILABLE", "copilot token provider is not configured")
	}
	token, err := s.openAITokenProvider.GetAccessToken(ctx, account)
	if err != nil {
		return nil, err
	}

	modelsURL := buildOpenAIModelsURLForPlatform(account.GetOpenAIBaseURL(), account.Platform)
	headers := map[string]string{
		"Authorization": "Bearer " + token,
	}
	applyCopilotDefaultHeadersMap(headers, account)

	body, err := s.doImportGET(ctx, account, modelsURL, headers, false)
	if err != nil {
		return nil, err
	}
	return parseOpenAIModelList(body)
}

func (s *AccountModelImportService) detectAnthropicModels(ctx context.Context, account *Account) ([]string, error) {
	headers := map[string]string{
		"Accept":            "application/json",
		"anthropic-version": "2023-06-01",
		"anthropic-beta":    claude.DefaultBetaHeader,
	}
	url := anthropicModelsURL
	switch account.Type {
	case AccountTypeOAuth, AccountTypeSetupToken:
		accessToken := strings.TrimSpace(account.GetCredential("access_token"))
		if accessToken == "" {
			return nil, infraerrors.BadRequest("ACCOUNT_CREDENTIAL_REQUIRED", "missing Anthropic access token for model import")
		}
		headers["Authorization"] = "Bearer " + accessToken
	case AccountTypeAPIKey, AccountTypeUpstream:
		apiKey := strings.TrimSpace(account.GetCredential("api_key"))
		if apiKey == "" {
			return nil, infraerrors.BadRequest("ACCOUNT_CREDENTIAL_REQUIRED", "missing Anthropic API key for model import")
		}
		headers["x-api-key"] = apiKey
		headers["anthropic-beta"] = claude.APIKeyBetaHeader
		baseURL := strings.TrimSpace(account.GetBaseURL())
		if baseURL != "" {
			url = strings.TrimRight(baseURL, "/") + "/v1/models"
		}
	default:
		return nil, infraerrors.BadRequest("ACCOUNT_TYPE_UNSUPPORTED", "current Anthropic account type does not support model import")
	}
	for key, value := range claude.DefaultHeaders {
		headers[key] = value
	}
	body, err := s.doImportGET(ctx, account, url, headers, account.IsTLSFingerprintEnabled())
	if err != nil {
		return nil, err
	}
	return parseAnthropicModelList(body)
}

func (s *AccountModelImportService) detectGeminiModels(ctx context.Context, account *Account) (*accountModelProbeResult, error) {
	if s.geminiCompatService == nil {
		return nil, infraerrors.InternalServer("MODEL_IMPORT_GEMINI_SERVICE_UNAVAILABLE", "gemini model import service is not configured")
	}
	log := logger.FromContext(ctx)
	if account != nil && account.IsGeminiVertexExpress() {
		if err := s.validateGeminiVertexExpressKey(ctx, account); err != nil {
			log.Warn("account model import: gemini vertex express validation failed",
				append(geminiImportProbeFields(account, http.StatusBadGateway, accountModelProbeSourceVertexExpressCatalog), zap.Error(err))...,
			)
			return nil, err
		}
		return newGeminiVertexCatalogProbeResult(
			accountModelProbeSourceVertexExpressCatalog,
			"Vertex AI Studio Key uses the official Express Mode countTokens validation and the built-in Vertex model catalog because Vertex Express does not expose a standard realtime /models endpoint",
		), nil
	}
	if account != nil && account.IsGeminiVertexAI() {
		if err := s.validateGeminiVertexServiceAccount(ctx, account); err != nil {
			log.Warn("account model import: gemini vertex service account validation failed",
				append(geminiImportProbeFields(account, http.StatusBadGateway, accountModelProbeSourceVertexServiceAccountCatalog), zap.Error(err))...,
			)
			return nil, err
		}
		return newGeminiVertexCatalogProbeResult(
			accountModelProbeSourceVertexServiceAccountCatalog,
			"Vertex AI Service Account uses the official countTokens validation and the built-in Vertex model catalog because Vertex publisher models does not expose a standard realtime GET /models endpoint",
		), nil
	}
	result, err := s.geminiCompatService.ForwardAIStudioGET(ctx, account, "/v1beta/models")
	if err != nil {
		return nil, infraerrors.ServiceUnavailable("MODEL_IMPORT_UPSTREAM_REQUEST_FAILED", "failed to request upstream model list").WithCause(err)
	}
	if result == nil {
		return nil, infraerrors.ServiceUnavailable("MODEL_IMPORT_UPSTREAM_EMPTY_RESPONSE", "upstream returned an empty response while listing models")
	}
	if result.StatusCode < http.StatusOK || result.StatusCode >= http.StatusMultipleChoices {
		if shouldFallbackGeminiCLIDefaultModels(account, result) {
			log.Info("account model import: gemini AI Studio listing scope insufficient, falling back to Gemini CLI default models",
				geminiImportProbeFields(account, result.StatusCode, accountModelProbeSourceGeminiCLIDefaultFallback)...,
			)
			return &accountModelProbeResult{
				Models: geminiCLIDefaultModelIDs(),
				Source: accountModelProbeSourceGeminiCLIDefaultFallback,
				Notice: "AI Studio model listing lacks required scopes; imported Gemini CLI default models instead",
			}, nil
		}
		statusErr := newAccountModelImportUpstreamStatusError(result.StatusCode, result.Body)
		log.Warn("account model import: gemini upstream model listing failed",
			append(geminiImportProbeFields(account, result.StatusCode, accountModelProbeSourceUpstream), zap.Error(statusErr))...,
		)
		return nil, statusErr
	}
	models, err := parseGeminiModelList(result.Body)
	if err != nil {
		return nil, err
	}
	return newAccountModelProbeResult(models), nil
}

func newGeminiVertexCatalogProbeResult(source string, notice string) *accountModelProbeResult {
	models := GeminiVertexCatalogModelIDs()
	details := make([]AccountModelProbeModel, 0, len(models))
	for _, model := range GeminiVertexCatalogModels() {
		details = append(details, AccountModelProbeModel{
			ID:          strings.TrimPrefix(model.Name, "models/"),
			DisplayName: model.DisplayName,
		})
	}
	return &accountModelProbeResult{
		Models:  models,
		Details: details,
		Source:  source,
		Notice:  notice,
	}
}

func (s *AccountModelImportService) validateGeminiVertexServiceAccount(ctx context.Context, account *Account) error {
	if account == nil {
		return infraerrors.BadRequest("ACCOUNT_REQUIRED", "account is required")
	}
	if s.geminiCompatService == nil || s.geminiCompatService.tokenProvider == nil {
		return infraerrors.InternalServer("MODEL_IMPORT_GEMINI_TOKEN_PROVIDER_UNAVAILABLE", "gemini token provider is not configured")
	}
	if s.httpUpstream == nil {
		return infraerrors.InternalServer("MODEL_IMPORT_HTTP_UPSTREAM_UNAVAILABLE", "model import http upstream is not configured")
	}
	accessToken, err := s.geminiCompatService.tokenProvider.GetAccessToken(ctx, account)
	if err != nil {
		return err
	}
	baseURL := account.GetGeminiVertexBaseURL(geminicli.VertexAIBaseURL)
	normalizedBaseURL, err := s.geminiCompatService.validateUpstreamBaseURL(baseURL)
	if err != nil {
		return err
	}
	actionPath, err := account.GeminiVertexModelActionPath(geminicli.DefaultTestModel, "countTokens")
	if err != nil {
		return err
	}
	reqBody := []byte(`{"contents":[{"role":"user","parts":[{"text":"ping"}]}]}`)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(normalizedBaseURL, "/")+actionPath, bytes.NewReader(reqBody))
	if err != nil {
		return infraerrors.InternalServer("MODEL_IMPORT_REQUEST_BUILD_FAILED", "failed to build Vertex service account probe request").WithCause(err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	proxyURL, err := s.resolveImportProxyURL(ctx, account)
	if err != nil {
		return infraerrors.BadRequest("MODEL_IMPORT_PROXY_RESOLVE_FAILED", "failed to resolve account proxy for model import").WithCause(err)
	}
	resp, err := s.httpUpstream.Do(req, proxyURL, account.ID, account.Concurrency)
	if err != nil {
		return infraerrors.ServiceUnavailable("MODEL_IMPORT_UPSTREAM_REQUEST_FAILED", "failed to validate Vertex service account access").WithCause(err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxImportBodyBytes))
	if readErr != nil {
		return infraerrors.ServiceUnavailable("MODEL_IMPORT_UPSTREAM_READ_FAILED", "failed to read Vertex service account validation response").WithCause(readErr)
	}
	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		return nil
	}
	return newAccountModelImportUpstreamStatusErrorForOperation("vertex service account validation failed", resp.StatusCode, body)
}

func (s *AccountModelImportService) validateGeminiVertexExpressKey(ctx context.Context, account *Account) error {
	if account == nil {
		return infraerrors.BadRequest("ACCOUNT_REQUIRED", "account is required")
	}
	apiKey := strings.TrimSpace(account.GetCredential("api_key"))
	if apiKey == "" {
		return infraerrors.BadRequest("ACCOUNT_CREDENTIAL_REQUIRED", "missing Gemini API key for model import")
	}
	if s.httpUpstream == nil {
		return infraerrors.InternalServer("MODEL_IMPORT_HTTP_UPSTREAM_UNAVAILABLE", "model import http upstream is not configured")
	}
	baseURL := account.GetGeminiVertexExpressBaseURL(geminicli.VertexAIBaseURL)
	normalizedBaseURL, err := s.geminiCompatService.validateUpstreamBaseURL(baseURL)
	if err != nil {
		return err
	}
	actionPath, err := account.GeminiVertexExpressModelActionPath(geminicli.DefaultTestModel, "countTokens")
	if err != nil {
		return err
	}
	reqBody := []byte(`{"contents":[{"role":"user","parts":[{"text":"ping"}]}]}`)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(normalizedBaseURL, "/")+actionPath, bytes.NewReader(reqBody))
	if err != nil {
		return infraerrors.InternalServer("MODEL_IMPORT_REQUEST_BUILD_FAILED", "failed to build Vertex Express probe request").WithCause(err)
	}
	query := req.URL.Query()
	query.Set("key", apiKey)
	req.URL.RawQuery = query.Encode()
	req.Header.Set("Content-Type", "application/json")
	proxyURL, err := s.resolveImportProxyURL(ctx, account)
	if err != nil {
		return infraerrors.BadRequest("MODEL_IMPORT_PROXY_RESOLVE_FAILED", "failed to resolve account proxy for model import").WithCause(err)
	}
	resp, err := s.httpUpstream.Do(req, proxyURL, account.ID, account.Concurrency)
	if err != nil {
		return infraerrors.ServiceUnavailable("MODEL_IMPORT_UPSTREAM_REQUEST_FAILED", "failed to validate Vertex Express API key").WithCause(err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxImportBodyBytes))
	if readErr != nil {
		return infraerrors.ServiceUnavailable("MODEL_IMPORT_UPSTREAM_READ_FAILED", "failed to read Vertex Express validation response").WithCause(readErr)
	}
	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		return nil
	}
	if resp.StatusCode == http.StatusNotFound {
		lowerBody := strings.ToLower(string(body))
		if strings.Contains(lowerBody, "not found") || strings.Contains(lowerBody, "requested entity was not found") {
			return nil
		}
	}
	return newAccountModelImportUpstreamStatusErrorForOperation("vertex express validation failed", resp.StatusCode, body)
}

func (s *AccountModelImportService) detectSoraModels(ctx context.Context, account *Account) ([]string, error) {
	switch account.Type {
	case AccountTypeAPIKey, AccountTypeUpstream:
		apiKey := strings.TrimSpace(account.GetCredential("api_key"))
		if apiKey == "" {
			return nil, infraerrors.BadRequest("ACCOUNT_CREDENTIAL_REQUIRED", "missing Sora API key for model import")
		}
		baseURL := strings.TrimSpace(account.GetCredential("base_url"))
		if baseURL == "" {
			return nil, infraerrors.BadRequest("ACCOUNT_CREDENTIAL_REQUIRED", "missing Sora base_url for model import")
		}
		headers := map[string]string{
			"Authorization": "Bearer " + apiKey,
			"Accept":        "application/json",
		}
		body, err := s.doImportGET(ctx, account, strings.TrimRight(baseURL, "/")+"/sora/v1/models", headers, false)
		if err != nil {
			return nil, err
		}
		return parseOpenAIModelList(body)
	case AccountTypeOAuth:
		return nil, infraerrors.BadRequest("ACCOUNT_TYPE_UNSUPPORTED", "current Sora OAuth account type does not support real model probing")
	default:
		return nil, infraerrors.BadRequest("ACCOUNT_TYPE_UNSUPPORTED", "current Sora account type does not support model import")
	}
}

func (s *AccountModelImportService) detectAntigravityModels(ctx context.Context, account *Account) ([]string, error) {
	switch account.Type {
	case AccountTypeOAuth:
		accessToken := strings.TrimSpace(account.GetCredential("access_token"))
		if accessToken == "" {
			return nil, infraerrors.BadRequest("ACCOUNT_CREDENTIAL_REQUIRED", "missing Antigravity access token for model import")
		}
		projectID := strings.TrimSpace(account.GetCredential("project_id"))
		proxyURL, err := s.resolveImportProxyURL(ctx, account)
		if err != nil {
			return nil, err
		}
		client, err := antigravity.NewClient(proxyURL)
		if err != nil {
			return nil, infraerrors.InternalServer("MODEL_IMPORT_ANTIGRAVITY_CLIENT_INIT_FAILED", "failed to initialize Antigravity model probe").WithCause(err)
		}
		resp, _, err := client.FetchAvailableModels(ctx, accessToken, projectID)
		if err != nil {
			return nil, infraerrors.ServiceUnavailable("MODEL_IMPORT_UPSTREAM_REQUEST_FAILED", "failed to request upstream model list").WithCause(err)
		}
		if resp == nil || len(resp.Models) == 0 {
			return nil, infraerrors.BadRequest("MODEL_IMPORT_EMPTY_UPSTREAM", "upstream returned an empty model list")
		}
		models := make([]string, 0, len(resp.Models))
		for modelID := range resp.Models {
			models = append(models, modelID)
		}
		sort.Strings(models)
		return models, nil
	case AccountTypeAPIKey, AccountTypeUpstream:
		return s.detectAnthropicModels(ctx, account)
	default:
		return nil, infraerrors.BadRequest("ACCOUNT_TYPE_UNSUPPORTED", "current Antigravity account type does not support model import")
	}
}

func (s *AccountModelImportService) resolveImportProxyURL(ctx context.Context, account *Account) (string, error) {
	if account == nil || account.ProxyID == nil {
		return "", nil
	}
	if account.Proxy != nil {
		return strings.TrimSpace(account.Proxy.URL()), nil
	}
	if s.proxyRepo == nil {
		return "", nil
	}
	proxy, err := s.proxyRepo.GetByID(ctx, *account.ProxyID)
	if err != nil || proxy == nil {
		return "", err
	}
	return strings.TrimSpace(proxy.URL()), nil
}

func (s *AccountModelImportService) doImportGET(ctx context.Context, account *Account, url string, headers map[string]string, enableTLSFingerprint bool) ([]byte, error) {
	if s.httpUpstream == nil {
		return nil, infraerrors.InternalServer("MODEL_IMPORT_HTTP_UPSTREAM_UNAVAILABLE", "model import http upstream is not configured")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, infraerrors.InternalServer("MODEL_IMPORT_REQUEST_BUILD_FAILED", "failed to build upstream model list request").WithCause(err)
	}
	for key, value := range headers {
		if strings.TrimSpace(value) != "" {
			req.Header.Set(key, value)
		}
	}
	proxyURL, err := s.resolveImportProxyURL(ctx, account)
	if err != nil {
		return nil, infraerrors.BadRequest("MODEL_IMPORT_PROXY_RESOLVE_FAILED", "failed to resolve account proxy for model import").WithCause(err)
	}
	var tlsProfile *tlsfingerprint.Profile
	if enableTLSFingerprint {
		tlsProfile = resolveAccountTLSFingerprintProfile(account, s.tlsFingerprintProfileService)
	}
	resp, err := s.httpUpstream.DoWithTLS(req, proxyURL, account.ID, account.Concurrency, tlsProfile)
	if err != nil {
		return nil, infraerrors.ServiceUnavailable("MODEL_IMPORT_UPSTREAM_REQUEST_FAILED", "failed to request upstream model list").WithCause(err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxImportBodyBytes))
	if readErr != nil {
		return nil, infraerrors.ServiceUnavailable("MODEL_IMPORT_UPSTREAM_READ_FAILED", "failed to read upstream model list response").WithCause(readErr)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, newAccountModelImportUpstreamStatusError(resp.StatusCode, body)
	}
	return body, nil
}

func parseOpenAIModelList(body []byte) ([]string, error) {
	var payload struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, infraerrors.BadRequest("MODEL_IMPORT_INVALID_RESPONSE", "upstream returned invalid model list JSON").WithCause(err)
	}
	ids := make([]string, 0, len(payload.Data))
	for _, model := range payload.Data {
		if v := strings.TrimSpace(model.ID); v != "" {
			ids = append(ids, v)
		}
	}
	if len(ids) == 0 {
		return nil, infraerrors.BadRequest("MODEL_IMPORT_EMPTY_UPSTREAM", "upstream returned an empty model list")
	}
	return ids, nil
}

func parseAnthropicModelList(body []byte) ([]string, error) {
	return parseOpenAIModelList(body)
}

func parseGeminiModelList(body []byte) ([]string, error) {
	var payload struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, infraerrors.BadRequest("MODEL_IMPORT_INVALID_RESPONSE", "upstream returned invalid model list JSON").WithCause(err)
	}
	ids := make([]string, 0, len(payload.Models))
	for _, model := range payload.Models {
		name := strings.TrimSpace(model.Name)
		if name == "" {
			continue
		}
		if strings.Contains(name, "/") {
			parts := strings.Split(name, "/")
			name = parts[len(parts)-1]
		}
		ids = append(ids, strings.TrimPrefix(name, "models/"))
	}
	if len(ids) == 0 {
		return nil, infraerrors.BadRequest("MODEL_IMPORT_EMPTY_UPSTREAM", "upstream returned an empty model list")
	}
	return ids, nil
}

func newAccountModelImportUpstreamStatusError(statusCode int, body []byte) error {
	return newAccountModelImportUpstreamStatusErrorForOperation("upstream model listing failed", statusCode, body)
}

func newAccountModelImportUpstreamStatusErrorForOperation(operation string, statusCode int, body []byte) error {
	message := fmt.Sprintf("%s with status %d", strings.TrimSpace(operation), statusCode)
	if truncated := truncateImportBody(body); truncated != "" {
		message = fmt.Sprintf("%s: %s", message, truncated)
	}
	if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
		return infraerrors.BadRequest("MODEL_IMPORT_UPSTREAM_UNAUTHORIZED", message)
	}
	if statusCode >= http.StatusInternalServerError {
		return infraerrors.ServiceUnavailable("MODEL_IMPORT_UPSTREAM_SERVER_ERROR", message)
	}
	if statusCode == http.StatusTooManyRequests {
		return infraerrors.TooManyRequests("MODEL_IMPORT_UPSTREAM_RATE_LIMITED", message)
	}
	return infraerrors.BadRequest("MODEL_IMPORT_UPSTREAM_FAILED", message)
}

func shouldFallbackGeminiCLIDefaultModels(account *Account, result *UpstreamHTTPResult) bool {
	if account == nil || result == nil {
		return false
	}
	if result.StatusCode != http.StatusForbidden {
		return false
	}
	if !account.IsGeminiCodeAssist() {
		return false
	}
	return isGeminiInsufficientScope(result.Headers, result.Body)
}

func geminiCLIDefaultModelIDs() []string {
	ids := make([]string, 0, len(geminicli.DefaultModels))
	for _, model := range geminicli.DefaultModels {
		if id := strings.TrimSpace(model.ID); id != "" {
			ids = append(ids, id)
		}
	}
	normalized, _ := normalizeImportedModelIDs(ids)
	return normalized
}

func geminiImportProbeFields(account *Account, statusCode int, probeSource string) []zap.Field {
	if account == nil {
		return []zap.Field{
			zap.Int("status", statusCode),
			zap.String("probe_source", probeSource),
		}
	}
	baseURL := geminiBaseURLForLogging(account)
	return []zap.Field{
		zap.Int64("account_id", account.ID),
		zap.String("platform", RoutingPlatformForAccount(account)),
		zap.String("type", account.Type),
		zap.String("oauth_type", account.GeminiOAuthType()),
		zap.String("gemini_api_variant", account.GeminiAPIKeyVariant()),
		zap.String("base_host", extractImportBaseHost(baseURL)),
		zap.String("vertex_location", account.GetGeminiVertexLocation()),
		zap.Int("status", statusCode),
		zap.String("probe_source", probeSource),
	}
}

func extractImportBaseHost(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	if host := strings.TrimSpace(parsed.Host); host != "" {
		return host
	}
	return rawURL
}

func truncateImportBody(body []byte) string {
	message := strings.TrimSpace(string(body))
	if len(message) <= 256 {
		return message
	}
	return message[:256] + "..."
}

func copilotDefaultModelIDs() []string {
	ids := make([]string, 0, len(openai.DefaultModels))
	for _, model := range openai.DefaultModels {
		if id := strings.TrimSpace(model.ID); id != "" {
			ids = append(ids, id)
		}
	}
	normalized, _ := normalizeImportedModelIDs(ids)
	return normalized
}
