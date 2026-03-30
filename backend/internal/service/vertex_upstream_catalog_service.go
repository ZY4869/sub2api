package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/geminicli"
	"github.com/Wei-Shaw/sub2api/internal/pkg/googleapi"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/util/urlvalidator"
	gocache "github.com/patrickmn/go-cache"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
)

const (
	vertexCatalogCacheTTL             = 5 * time.Minute
	vertexCatalogListPageSize         = 200
	vertexCatalogListPath             = "/v1beta1/publishers/google/models"
	vertexCatalogOfficialSource       = "official"
	vertexCatalogVerifiedExtraSource  = "verified_extra"
	vertexCatalogCallableAvailability = "callable"
	vertexCatalogFailedAvailability   = "uncallable"
	vertexCatalogCallableReason       = "countTokens ok"
)

var vertexLegacyUpstreamModelAliases = map[string]string{
	"gemini-2.5-flash-image": "gemini-2.5-flash-image-preview",
	"gemini-3-flash":         "gemini-3-flash-preview",
	"gemini-3-pro":           "gemini-3-pro-preview",
	"gemini-3-pro-image":     "gemini-3-pro-image-preview",
	"gemini-3.1-pro":         "gemini-3.1-pro-preview",
	"gemini-3.1-flash-lite":  "gemini-3.1-flash-lite-preview",
	"gemini-3.1-flash-image": "gemini-3.1-flash-image-preview",
}

type VertexCatalogProvider interface {
	GetCatalog(ctx context.Context, account *Account, forceRefresh bool) (*VertexCatalogResult, error)
}

type VertexCatalogModel struct {
	ID                 string
	DisplayName        string
	LaunchStage        string
	OfficialResource   string
	UpstreamSource     string
	Availability       string
	AvailabilityReason string
}

type VertexCatalogResult struct {
	OfficialModels []VertexCatalogModel
	VerifiedExtras []VertexCatalogModel
	CallableUnion  []VertexCatalogModel
}

type VertexUpstreamCatalogService struct {
	httpUpstream  HTTPUpstream
	tokenProvider *GeminiTokenProvider
	proxyRepo     ProxyRepository
	cfg           *config.Config
	cache         *gocache.Cache
	sf            singleflight.Group
}

type vertexPublisherModelsResponse struct {
	PublisherModels []struct {
		Name        string `json:"name"`
		DisplayName string `json:"displayName"`
		LaunchStage string `json:"launchStage"`
	} `json:"publisherModels"`
	NextPageToken string `json:"nextPageToken"`
}

func NewVertexUpstreamCatalogService(
	httpUpstream HTTPUpstream,
	tokenProvider *GeminiTokenProvider,
	proxyRepo ProxyRepository,
	cfg *config.Config,
) *VertexUpstreamCatalogService {
	return &VertexUpstreamCatalogService{
		httpUpstream:  httpUpstream,
		tokenProvider: tokenProvider,
		proxyRepo:     proxyRepo,
		cfg:           cfg,
		cache:         gocache.New(vertexCatalogCacheTTL, time.Minute),
	}
}

func normalizeVertexUpstreamModelID(modelID string) string {
	normalized := normalizeRegistryID(modelID)
	if replacement, ok := vertexLegacyUpstreamModelAliases[normalized]; ok {
		return replacement
	}
	return normalized
}

func (s *VertexUpstreamCatalogService) GetCatalog(
	ctx context.Context,
	account *Account,
	forceRefresh bool,
) (*VertexCatalogResult, error) {
	if account == nil {
		return nil, fmt.Errorf("account is nil")
	}
	if !account.IsGeminiVertexSource() {
		return nil, fmt.Errorf("account is not a Vertex Gemini account")
	}
	if s == nil || s.httpUpstream == nil {
		return nil, fmt.Errorf("vertex catalog service is not configured")
	}

	cacheKey := s.cacheKey(account)
	if !forceRefresh {
		if cached, ok := s.cache.Get(cacheKey); ok {
			if result, castOK := cached.(*VertexCatalogResult); castOK && result != nil {
				logger.FromContext(ctx).Debug("vertex_catalog_cache_hit",
					s.vertexLogFields(account,
						zap.Int("official_count", len(result.OfficialModels)),
						zap.Int("callable_count", len(result.CallableUnion)),
						zap.Int("verified_extra_count", len(result.VerifiedExtras)),
					)...,
				)
				return cloneVertexCatalogResult(result), nil
			}
		}
	}
	logger.FromContext(ctx).Debug("vertex_catalog_cache_miss",
		s.vertexLogFields(account, zap.Bool("force_refresh", forceRefresh))...,
	)

	if forceRefresh {
		result, err := s.fetchCatalog(ctx, account)
		if err != nil {
			return nil, err
		}
		s.cache.Set(cacheKey, result, vertexCatalogCacheTTL)
		return cloneVertexCatalogResult(result), nil
	}

	value, err, _ := s.sf.Do(cacheKey, func() (any, error) {
		if cached, ok := s.cache.Get(cacheKey); ok {
			if result, castOK := cached.(*VertexCatalogResult); castOK && result != nil {
				return cloneVertexCatalogResult(result), nil
			}
		}
		result, fetchErr := s.fetchCatalog(ctx, account)
		if fetchErr != nil {
			return nil, fetchErr
		}
		s.cache.Set(cacheKey, result, vertexCatalogCacheTTL)
		return cloneVertexCatalogResult(result), nil
	})
	if err != nil {
		return nil, err
	}
	result, _ := value.(*VertexCatalogResult)
	return cloneVertexCatalogResult(result), nil
}

func (s *VertexUpstreamCatalogService) fetchCatalog(
	ctx context.Context,
	account *Account,
) (*VertexCatalogResult, error) {
	log := logger.FromContext(ctx)
	log.Info("vertex_catalog_fetch_start", s.vertexLogFields(account)...)

	proxyURL, err := s.resolveProxyURL(ctx, account)
	if err != nil {
		return nil, err
	}

	officialModels, err := s.listOfficialModels(ctx, account, proxyURL)
	if err != nil {
		log.Warn("vertex_catalog_official_fetch_failed",
			s.vertexLogFields(account, zap.String("error_code", "official_fetch_failed"), zap.Error(err))...,
		)
		return nil, err
	}

	officialSet := make(map[string]struct{}, len(officialModels))
	callableUnionSet := make(map[string]struct{}, len(officialModels))
	officialResults := make([]VertexCatalogModel, 0, len(officialModels))
	callableUnion := make([]VertexCatalogModel, 0, len(officialModels))

	for _, model := range officialModels {
		officialSet[model.ID] = struct{}{}
		callable, reason := s.checkCallable(ctx, account, proxyURL, model.ID)
		model.Availability = vertexCatalogFailedAvailability
		model.AvailabilityReason = reason
		if callable {
			model.Availability = vertexCatalogCallableAvailability
			model.AvailabilityReason = vertexCatalogCallableReason
			if _, exists := callableUnionSet[model.ID]; !exists {
				callableUnionSet[model.ID] = struct{}{}
				callableUnion = append(callableUnion, model)
			}
		}
		officialResults = append(officialResults, model)
	}
	log.Info("vertex_catalog_validation_summary",
		s.vertexLogFields(account,
			zap.Int("official_count", len(officialResults)),
			zap.Int("callable_count", len(callableUnion)),
		)...,
	)

	extraCandidates := s.extraCandidateModels(account, officialSet)
	verifiedExtras := make([]VertexCatalogModel, 0, len(extraCandidates))
	for _, modelID := range extraCandidates {
		callable, reason := s.checkCallable(ctx, account, proxyURL, modelID)
		if !callable {
			log.Debug("vertex_catalog_extra_validation_failed",
				s.vertexLogFields(account,
					zap.String("model_id", modelID),
					zap.String("availability_reason", reason),
				)...,
			)
			continue
		}
		model := VertexCatalogModel{
			ID:                 modelID,
			DisplayName:        FormatModelCatalogDisplayName(modelID),
			UpstreamSource:     vertexCatalogVerifiedExtraSource,
			Availability:       vertexCatalogCallableAvailability,
			AvailabilityReason: vertexCatalogCallableReason,
		}
		verifiedExtras = append(verifiedExtras, model)
		if _, exists := callableUnionSet[model.ID]; !exists {
			callableUnionSet[model.ID] = struct{}{}
			callableUnion = append(callableUnion, model)
		}
	}
	log.Info("vertex_catalog_extra_hit_summary",
		s.vertexLogFields(account,
			zap.Int("verified_extra_count", len(verifiedExtras)),
		)...,
	)

	sortVertexCatalogModels(officialResults)
	sortVertexCatalogModels(verifiedExtras)
	sortVertexCatalogModels(callableUnion)

	result := &VertexCatalogResult{
		OfficialModels: officialResults,
		VerifiedExtras: verifiedExtras,
		CallableUnion:  callableUnion,
	}

	log.Info("vertex_catalog_fetch_success",
		s.vertexLogFields(account,
			zap.Int("official_count", len(result.OfficialModels)),
			zap.Int("callable_count", len(result.CallableUnion)),
			zap.Int("verified_extra_count", len(result.VerifiedExtras)),
		)...,
	)
	return result, nil
}

func (s *VertexUpstreamCatalogService) listOfficialModels(
	ctx context.Context,
	account *Account,
	proxyURL string,
) ([]VertexCatalogModel, error) {
	var accessToken string
	var err error

	if account.IsGeminiVertexAI() {
		if s.tokenProvider == nil {
			return nil, fmt.Errorf("gemini token provider not configured")
		}
		accessToken, err = s.tokenProvider.GetAccessToken(ctx, account)
		if err != nil {
			return nil, err
		}
	}

	baseURL, err := s.vertexListBaseURL(account)
	if err != nil {
		return nil, err
	}

	pageToken := ""
	models := make([]VertexCatalogModel, 0)
	for {
		reqURL := strings.TrimRight(baseURL, "/") + vertexCatalogListPath
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
		if reqErr != nil {
			return nil, reqErr
		}
		query := req.URL.Query()
		query.Set("pageSize", fmt.Sprintf("%d", vertexCatalogListPageSize))
		if strings.TrimSpace(pageToken) != "" {
			query.Set("pageToken", pageToken)
		}
		if account.IsGeminiVertexExpress() {
			apiKey := strings.TrimSpace(account.GetCredential("api_key"))
			if apiKey == "" {
				return nil, fmt.Errorf("missing Gemini API key for Vertex catalog")
			}
			query.Set("key", apiKey)
		} else {
			req.Header.Set("Authorization", "Bearer "+accessToken)
		}
		req.URL.RawQuery = query.Encode()

		resp, doErr := s.httpUpstream.Do(req, proxyURL, account.ID, account.Concurrency)
		if doErr != nil {
			return nil, doErr
		}
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxImportBodyBytes))
		_ = resp.Body.Close()
		if readErr != nil {
			return nil, readErr
		}
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			return nil, newAccountModelImportUpstreamStatusErrorForOperation("vertex official model listing failed", resp.StatusCode, body)
		}

		var payload vertexPublisherModelsResponse
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, fmt.Errorf("failed to parse Vertex publisher models response: %w", err)
		}
		for _, item := range payload.PublisherModels {
			modelID := normalizeRegistryID(item.Name)
			if modelID == "" {
				continue
			}
			models = append(models, VertexCatalogModel{
				ID:               modelID,
				DisplayName:      FormatModelCatalogDisplayName(modelID),
				LaunchStage:      strings.TrimSpace(item.LaunchStage),
				OfficialResource: strings.TrimSpace(item.Name),
				UpstreamSource:   vertexCatalogOfficialSource,
			})
		}
		pageToken = strings.TrimSpace(payload.NextPageToken)
		if pageToken == "" {
			break
		}
	}

	return uniqueVertexCatalogModels(models), nil
}

func (s *VertexUpstreamCatalogService) checkCallable(
	ctx context.Context,
	account *Account,
	proxyURL string,
	modelID string,
) (bool, string) {
	req, err := s.buildCountTokensRequest(ctx, account, modelID)
	if err != nil {
		return false, strings.TrimSpace(err.Error())
	}

	resp, err := s.httpUpstream.Do(req, proxyURL, account.ID, account.Concurrency)
	if err != nil {
		return false, "request failed"
	}

	body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxImportBodyBytes))
	_ = resp.Body.Close()
	if readErr != nil {
		return false, "response read failed"
	}
	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		return true, vertexCatalogCallableReason
	}
	return false, summarizeVertexAvailabilityReason(resp.StatusCode, body)
}

func (s *VertexUpstreamCatalogService) buildCountTokensRequest(
	ctx context.Context,
	account *Account,
	modelID string,
) (*http.Request, error) {
	modelID = normalizeVertexUpstreamModelID(modelID)
	if modelID == "" {
		return nil, fmt.Errorf("missing model")
	}
	body := []byte(`{"contents":[{"role":"user","parts":[{"text":"ping"}]}]}`)

	if account.IsGeminiVertexExpress() {
		apiKey := strings.TrimSpace(account.GetCredential("api_key"))
		if apiKey == "" {
			return nil, fmt.Errorf("missing Gemini API key for Vertex Express")
		}
		baseURL, err := s.vertexCountTokensBaseURL(account)
		if err != nil {
			return nil, err
		}
		actionPath, err := account.GeminiVertexExpressModelActionPath(modelID, "countTokens")
		if err != nil {
			return nil, err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(baseURL, "/")+actionPath, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		query := req.URL.Query()
		query.Set("key", apiKey)
		req.URL.RawQuery = query.Encode()
		req.Header.Set("Content-Type", "application/json")
		return req, nil
	}

	if s.tokenProvider == nil {
		return nil, fmt.Errorf("gemini token provider not configured")
	}
	accessToken, err := s.tokenProvider.GetAccessToken(ctx, account)
	if err != nil {
		return nil, err
	}
	baseURL, err := s.vertexCountTokensBaseURL(account)
	if err != nil {
		return nil, err
	}
	actionPath, err := account.GeminiVertexModelActionPath(modelID, "countTokens")
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(baseURL, "/")+actionPath, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func (s *VertexUpstreamCatalogService) extraCandidateModels(
	account *Account,
	officialSet map[string]struct{},
) []string {
	candidates := make(map[string]struct{})
	add := func(modelID string) {
		modelID = normalizeVertexUpstreamModelID(modelID)
		if modelID == "" {
			return
		}
		if _, exists := officialSet[modelID]; exists {
			return
		}
		candidates[modelID] = struct{}{}
	}

	for _, model := range geminicli.DefaultModels {
		add(model.ID)
	}
	for alias, target := range vertexLegacyUpstreamModelAliases {
		add(alias)
		add(target)
	}
	for _, source := range account.GetModelMapping() {
		add(source)
	}

	items := make([]string, 0, len(candidates))
	for modelID := range candidates {
		items = append(items, modelID)
	}
	sort.Strings(items)
	return items
}

func (s *VertexUpstreamCatalogService) cacheKey(account *Account) string {
	authMode := "vertex_service_account"
	baseURL := strings.TrimSpace(account.GetGeminiVertexBaseURL(geminicli.VertexAIBaseURL))
	projectID := strings.TrimSpace(account.GetGeminiVertexProjectID())
	location := strings.TrimSpace(account.GetGeminiVertexLocation())
	rawMapping, _ := account.Credentials["model_mapping"].(map[string]any)
	mappingSig := modelMappingSignature(rawMapping)
	if account.IsGeminiVertexExpress() {
		authMode = "vertex_express"
		baseURL = strings.TrimSpace(account.GetGeminiVertexExpressBaseURL(geminicli.VertexAIBaseURL))
		projectID = ""
		location = strings.TrimSpace(account.GetGeminiVertexLocation())
	}

	return strings.Join([]string{
		fmt.Sprintf("account=%d", account.ID),
		fmt.Sprintf("auth=%s", authMode),
		fmt.Sprintf("project=%s", projectID),
		fmt.Sprintf("location=%s", location),
		fmt.Sprintf("base=%s", strings.TrimRight(baseURL, "/")),
		fmt.Sprintf("mapping=%d", mappingSig),
	}, "|")
}

func (s *VertexUpstreamCatalogService) resolveProxyURL(ctx context.Context, account *Account) (string, error) {
	if account == nil || account.ProxyID == nil {
		return "", nil
	}
	if account.Proxy != nil {
		return account.Proxy.URL(), nil
	}
	if s.proxyRepo == nil {
		return "", nil
	}
	proxy, err := s.proxyRepo.GetByID(ctx, *account.ProxyID)
	if err != nil || proxy == nil {
		return "", err
	}
	return proxy.URL(), nil
}

func (s *VertexUpstreamCatalogService) vertexListBaseURL(account *Account) (string, error) {
	if account.IsGeminiVertexExpress() {
		return s.validateUpstreamBaseURL(account.GetGeminiVertexExpressBaseURL(geminicli.VertexAIBaseURL))
	}
	return s.validateUpstreamBaseURL(account.GetGeminiVertexBaseURL(geminicli.VertexAIBaseURL))
}

func (s *VertexUpstreamCatalogService) vertexCountTokensBaseURL(account *Account) (string, error) {
	if account.IsGeminiVertexExpress() {
		return s.validateUpstreamBaseURL(account.GetGeminiVertexExpressBaseURL(geminicli.VertexAIBaseURL))
	}
	return s.validateUpstreamBaseURL(account.GetGeminiVertexBaseURL(geminicli.VertexAIBaseURL))
}

func (s *VertexUpstreamCatalogService) validateUpstreamBaseURL(raw string) (string, error) {
	if s.cfg == nil {
		normalized, err := urlvalidator.ValidateURLFormat(raw, false)
		if err != nil {
			return "", fmt.Errorf("invalid base_url: %w", err)
		}
		return normalized, nil
	}
	if s.cfg != nil && !s.cfg.Security.URLAllowlist.Enabled {
		normalized, err := urlvalidator.ValidateURLFormat(raw, s.cfg.Security.URLAllowlist.AllowInsecureHTTP)
		if err != nil {
			return "", fmt.Errorf("invalid base_url: %w", err)
		}
		return normalized, nil
	}
	normalized, err := urlvalidator.ValidateHTTPSURL(raw, urlvalidator.ValidationOptions{
		AllowedHosts:     s.cfg.Security.URLAllowlist.UpstreamHosts,
		RequireAllowlist: true,
		AllowPrivate:     s.cfg.Security.URLAllowlist.AllowPrivateHosts,
	})
	if err != nil {
		return "", fmt.Errorf("invalid base_url: %w", err)
	}
	return normalized, nil
}

func (s *VertexUpstreamCatalogService) vertexLogFields(account *Account, extra ...zap.Field) []zap.Field {
	authMode := "vertex_service_account"
	baseURL := ""
	projectID := ""
	location := ""
	if account != nil {
		if account.IsGeminiVertexExpress() {
			authMode = "vertex_express"
			baseURL = account.GetGeminiVertexExpressBaseURL(geminicli.VertexAIBaseURL)
		} else {
			baseURL = account.GetGeminiVertexBaseURL(geminicli.VertexAIBaseURL)
		}
		projectID = strings.TrimSpace(account.GetGeminiVertexProjectID())
		location = strings.TrimSpace(account.GetGeminiVertexLocation())
	}
	fields := []zap.Field{
		zap.Int64("account_id", accountID(account)),
		zap.String("auth_mode", authMode),
		zap.String("base_host", extractImportBaseHost(baseURL)),
		zap.String("vertex_project", projectID),
		zap.String("vertex_location", location),
	}
	return append(fields, extra...)
}

func summarizeVertexAvailabilityReason(statusCode int, body []byte) string {
	if parsed, err := googleapi.ParseError(string(body)); err == nil && parsed != nil {
		status := strings.TrimSpace(parsed.Error.Status)
		message := strings.TrimSpace(parsed.Error.Message)
		switch {
		case status != "" && message != "":
			return fmt.Sprintf("status %d %s: %s", statusCode, status, message)
		case status != "":
			return fmt.Sprintf("status %d %s", statusCode, status)
		case message != "":
			return fmt.Sprintf("status %d: %s", statusCode, message)
		}
	}
	if message := truncateImportBody(body); message != "" {
		return fmt.Sprintf("status %d: %s", statusCode, message)
	}
	return fmt.Sprintf("status %d", statusCode)
}

func uniqueVertexCatalogModels(models []VertexCatalogModel) []VertexCatalogModel {
	seen := make(map[string]struct{}, len(models))
	result := make([]VertexCatalogModel, 0, len(models))
	for _, model := range models {
		model.ID = normalizeRegistryID(model.ID)
		if model.ID == "" {
			continue
		}
		if _, exists := seen[model.ID]; exists {
			continue
		}
		seen[model.ID] = struct{}{}
		if strings.TrimSpace(model.DisplayName) == "" {
			model.DisplayName = FormatModelCatalogDisplayName(model.ID)
		}
		result = append(result, model)
	}
	return result
}

func sortVertexCatalogModels(models []VertexCatalogModel) {
	sort.Slice(models, func(i, j int) bool {
		return models[i].ID < models[j].ID
	})
}

func cloneVertexCatalogResult(input *VertexCatalogResult) *VertexCatalogResult {
	if input == nil {
		return nil
	}
	return &VertexCatalogResult{
		OfficialModels: append([]VertexCatalogModel(nil), input.OfficialModels...),
		VerifiedExtras: append([]VertexCatalogModel(nil), input.VerifiedExtras...),
		CallableUnion:  append([]VertexCatalogModel(nil), input.CallableUnion...),
	}
}

func accountID(account *Account) int64 {
	if account == nil {
		return 0
	}
	return account.ID
}
