package routes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func newGatewayRoutesTestRouter() *gin.Engine {
	return newGatewayRoutesTestRouterForPlatform(service.PlatformOpenAI)
}

func newGatewayRoutesTestRouterWithAuth(auth gin.HandlerFunc) *gin.Engine {
	return newGatewayRoutesTestRouterWithAuthAndGooglePlatform(auth, service.PlatformOpenAI)
}

func newGatewayRoutesTestRouterForPlatform(platform string) *gin.Engine {
	return newGatewayRoutesTestRouterWithAuthAndGooglePlatform(func(c *gin.Context) {
		apiKey := newGatewayRoutesTestAPIKey(platform)
		c.Set(string(servermiddleware.ContextKeyAPIKey), apiKey)
		c.Set(string(servermiddleware.ContextKeyUser), servermiddleware.AuthSubject{
			UserID:      apiKey.User.ID,
			Concurrency: apiKey.User.Concurrency,
		})
		c.Next()
	}, platform)
}

func newGatewayRoutesTestRouterWithAuthAndGooglePlatform(auth gin.HandlerFunc, googlePlatform string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	cfg := &config.Config{RunMode: config.RunModeSimple}

	apiKeyService := service.NewAPIKeyService(
		&gatewayRouteAPIKeyRepoStub{apiKey: newGatewayRoutesTestAPIKey(googlePlatform)},
		nil,
		nil,
		nil,
		nil,
		nil,
		cfg,
	)

	authMiddleware := func(c *gin.Context) {
		if auth != nil {
			auth(c)
			return
		}
		c.Next()
	}

	RegisterGatewayRoutes(
		router,
		&handler.Handlers{
			Gateway:       &handler.GatewayHandler{},
			OpenAIGateway: &handler.OpenAIGatewayHandler{},
			GrokGateway:   &handler.GrokGatewayHandler{},
		},
		servermiddleware.APIKeyAuthMiddleware(authMiddleware),
		apiKeyService,
		nil,
		nil,
		nil,
		cfg,
	)

	return router
}

func newGatewayRoutesTestAPIKey(platform string) *service.APIKey {
	groupID := int64(1)
	return &service.APIKey{
		ID:      1,
		UserID:  1,
		Key:     "test-gateway-key",
		Name:    "gateway-test",
		Status:  service.StatusActive,
		GroupID: &groupID,
		Group: &service.Group{
			ID:       groupID,
			Platform: platform,
		},
		User: &service.User{
			ID:          1,
			Status:      service.StatusActive,
			Role:        "user",
			Concurrency: 1,
		},
	}
}

type gatewayRouteAPIKeyRepoStub struct {
	apiKey *service.APIKey
}

func (s *gatewayRouteAPIKeyRepoStub) Create(context.Context, *service.APIKey) error { return nil }
func (s *gatewayRouteAPIKeyRepoStub) GetByID(context.Context, int64) (*service.APIKey, error) {
	return s.apiKey, nil
}
func (s *gatewayRouteAPIKeyRepoStub) GetKeyAndOwnerID(context.Context, int64) (string, int64, error) {
	if s.apiKey == nil {
		return "", 0, service.ErrAPIKeyNotFound
	}
	return s.apiKey.Key, s.apiKey.UserID, nil
}
func (s *gatewayRouteAPIKeyRepoStub) GetByKey(_ context.Context, key string) (*service.APIKey, error) {
	if s.apiKey == nil || s.apiKey.Key != key {
		return nil, service.ErrAPIKeyNotFound
	}
	return s.apiKey, nil
}
func (s *gatewayRouteAPIKeyRepoStub) GetByKeyForAuth(ctx context.Context, key string) (*service.APIKey, error) {
	return s.GetByKey(ctx, key)
}
func (s *gatewayRouteAPIKeyRepoStub) Update(context.Context, *service.APIKey) error { return nil }
func (s *gatewayRouteAPIKeyRepoStub) Delete(context.Context, int64) error           { return nil }
func (s *gatewayRouteAPIKeyRepoStub) ListByUserID(context.Context, int64, pagination.PaginationParams, service.APIKeyListFilters) ([]service.APIKey, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (s *gatewayRouteAPIKeyRepoStub) VerifyOwnership(context.Context, int64, []int64) ([]int64, error) {
	return nil, nil
}
func (s *gatewayRouteAPIKeyRepoStub) CountByUserID(context.Context, int64) (int64, error) {
	return 0, nil
}
func (s *gatewayRouteAPIKeyRepoStub) ExistsByKey(context.Context, string) (bool, error) {
	return false, nil
}
func (s *gatewayRouteAPIKeyRepoStub) ListByGroupID(context.Context, int64, pagination.PaginationParams) ([]service.APIKey, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (s *gatewayRouteAPIKeyRepoStub) SearchAPIKeys(context.Context, int64, string, int) ([]service.APIKey, error) {
	return nil, nil
}
func (s *gatewayRouteAPIKeyRepoStub) ClearGroupIDByGroupID(context.Context, int64) (int64, error) {
	return 0, nil
}
func (s *gatewayRouteAPIKeyRepoStub) UpdateGroupIDByUserAndGroup(context.Context, int64, int64, int64) (int64, error) {
	return 0, nil
}
func (s *gatewayRouteAPIKeyRepoStub) CountByGroupID(context.Context, int64) (int64, error) {
	return 0, nil
}
func (s *gatewayRouteAPIKeyRepoStub) ListKeysByUserID(context.Context, int64) ([]string, error) {
	return nil, nil
}
func (s *gatewayRouteAPIKeyRepoStub) ListKeysByGroupID(context.Context, int64) ([]string, error) {
	return nil, nil
}
func (s *gatewayRouteAPIKeyRepoStub) GetAPIKeyGroups(context.Context, int64) ([]service.APIKeyGroupBinding, error) {
	return nil, nil
}
func (s *gatewayRouteAPIKeyRepoStub) SetAPIKeyGroups(context.Context, int64, []service.APIKeyGroupBinding) error {
	return nil
}
func (s *gatewayRouteAPIKeyRepoStub) IncrementAPIKeyGroupQuotaUsed(context.Context, int64, int64, float64) error {
	return nil
}
func (s *gatewayRouteAPIKeyRepoStub) IncrementQuotaUsed(context.Context, int64, float64) (float64, error) {
	return 0, nil
}
func (s *gatewayRouteAPIKeyRepoStub) UpdateLastUsed(context.Context, int64, time.Time) error {
	return nil
}
func (s *gatewayRouteAPIKeyRepoStub) IncrementRateLimitUsage(context.Context, int64, float64) error {
	return nil
}
func (s *gatewayRouteAPIKeyRepoStub) ResetRateLimitWindows(context.Context, int64) error { return nil }
func (s *gatewayRouteAPIKeyRepoStub) GetRateLimitData(context.Context, int64) (*service.APIKeyRateLimitData, error) {
	return nil, nil
}

func sampleRequestPath(pattern string) string {
	sample := pattern
	if strings.Contains(sample, "*modelAction") {
		sample = strings.ReplaceAll(sample, "*modelAction", "gemini-2.5-pro:predict")
	}
	sample = strings.ReplaceAll(sample, "{batch}", "batch-123")
	sample = strings.ReplaceAll(sample, "{store}", "default-store")
	sample = strings.ReplaceAll(sample, "{document}", "doc-123")
	sample = strings.ReplaceAll(sample, "{operation}", "op-123")
	sample = strings.ReplaceAll(sample, "{model}", "gemini-2.5-pro")
	sample = strings.ReplaceAll(sample, ":project", "demo-project")
	sample = strings.ReplaceAll(sample, ":location", "us-central1")
	sample = strings.ReplaceAll(sample, ":request_id", "req_123")
	sample = strings.ReplaceAll(sample, ":model", "gemini-2.5-pro")
	sample = strings.ReplaceAll(sample, ":action", ":register")
	sample = strings.ReplaceAll(sample, "*subpath", "sample")
	return path.Clean(sample)
}

func sampleRequestBody(pattern string) string {
	switch {
	case strings.Contains(pattern, "/messages"):
		return `{"model":"claude-sonnet-4.5","messages":[{"role":"user","content":"hello"}]}`
	case strings.Contains(pattern, "/chat/completions"):
		return `{"model":"gpt-5.4","messages":[{"role":"user","content":"hello"}]}`
	case strings.Contains(pattern, "/responses"):
		return `{"model":"gpt-5.4","input":"hello"}`
	case strings.Contains(pattern, "/embeddings"):
		return `{"model":"gemini-2.5-flash","input":"hello"}`
	case strings.Contains(pattern, "/interactions"):
		return `{"model":"gemini-2.5-flash","input":"hello"}`
	case strings.Contains(pattern, "/images/"):
		return `{"model":"grok-image-1","prompt":"draw a cat"}`
	case strings.Contains(pattern, "/videos"):
		return `{"model":"grok-video-1","prompt":"animate a cat"}`
	default:
		return `{}`
	}
}

func supportedPlatformForEntry(entry service.PublicEndpointRegistryEntry) string {
	for _, capability := range entry.Capabilities {
		if capability.Mode != service.ProtocolCapabilityReject {
			return capability.RuntimePlatform
		}
	}
	return service.PlatformOpenAI
}

func routerForRoute(entry service.PublicEndpointRegistryEntry, route service.PublicEndpointRoute) *gin.Engine {
	if strings.HasPrefix(route.Pattern, "/grok/v1/") {
		return newGatewayRoutesTestRouterForPlatform(service.PlatformGrok)
	}
	if strings.HasPrefix(entry.HandlerFamily, "grok_") {
		return newGatewayRoutesTestRouterForPlatform(service.PlatformOpenAI)
	}
	return newGatewayRoutesTestRouterForPlatform(supportedPlatformForEntry(entry))
}

func handlerFamilyForRegisteredRoute(handlerName string) string {
	switch {
	case strings.Contains(handlerName, ".AnthropicMessages-fm"), strings.Contains(handlerName, ".AnthropicCountTokens-fm"):
		return "anthropic_messages"
	case strings.Contains(handlerName, ".OpenAIChatCompletions-fm"):
		return "openai_chat_completions"
	case strings.Contains(handlerName, ".OpenAIResponses-fm"), strings.Contains(handlerName, ".OpenAIResponsesWebSocket-fm"):
		return "openai_responses"
	case strings.Contains(handlerName, ".GrokImagesGeneration-fm"):
		return "grok_images_generation"
	case strings.Contains(handlerName, ".GrokImagesEdits-fm"):
		return "grok_images_edits"
	case strings.Contains(handlerName, ".GrokVideosGeneration-fm"):
		return "grok_videos_generation"
	case strings.Contains(handlerName, ".GrokVideosStatus-fm"):
		return "grok_videos_status"
	case strings.Contains(handlerName, ".GeminiModels-fm"),
		strings.Contains(handlerName, ".GeminiV1BetaListModels-fm"),
		strings.Contains(handlerName, ".GeminiV1BetaGetModel-fm"):
		return "gemini_models"
	case strings.Contains(handlerName, ".GatewayV1ModelsList-fm"):
		return "gateway_v1_models_list"
	case strings.Contains(handlerName, ".GatewayV1ModelsGet-fm"):
		return "gateway_v1_models_get"
	case strings.Contains(handlerName, ".GatewayV1ModelsAction-fm"):
		return "gateway_v1_models_action"
	case strings.Contains(handlerName, ".GeminiFiles-fm"):
		return "gemini_files"
	case strings.Contains(handlerName, ".GeminiFilesUpload-fm"):
		return "gemini_files_upload"
	case strings.Contains(handlerName, ".GeminiFilesDownload-fm"):
		return "gemini_files_download"
	case strings.Contains(handlerName, ".GeminiBatches-fm"):
		return "gemini_batches"
	case strings.Contains(handlerName, ".GeminiEmbeddings-fm"):
		return "gemini_embeddings"
	case strings.Contains(handlerName, ".GeminiCachedContents-fm"):
		return "gemini_cached_contents"
	case strings.Contains(handlerName, ".GeminiFileSearchStores-fm"):
		return "gemini_file_search_stores"
	case strings.Contains(handlerName, ".GeminiDocuments-fm"):
		return "gemini_documents"
	case strings.Contains(handlerName, ".GeminiOperations-fm"):
		return "gemini_operations"
	case strings.Contains(handlerName, ".GeminiUploadOperations-fm"):
		return "gemini_upload_operations"
	case strings.Contains(handlerName, ".GeminiInteractions-fm"):
		return "gemini_interactions"
	case strings.Contains(handlerName, ".GeminiOpenAICompat-fm"):
		return "gemini_openai_compat"
	case strings.Contains(handlerName, ".GeminiLive-fm"):
		return "gemini_live"
	case strings.Contains(handlerName, ".GeminiLiveAuthTokens-fm"):
		return "gemini_live_auth_tokens"
	case strings.Contains(handlerName, ".GoogleBatchArchiveBatch-fm"):
		return "google_batch_archive_batches"
	case strings.Contains(handlerName, ".GoogleBatchArchiveFileDownload-fm"):
		return "google_batch_archive_files"
	case strings.Contains(handlerName, ".VertexModels-fm"):
		return "vertex_models"
	case strings.Contains(handlerName, ".VertexBatchPredictionJobs-fm"):
		return "vertex_batch_prediction_jobs"
	default:
		return ""
	}
}

func preferredDynamicRoute(entry service.PublicEndpointRegistryEntry) service.PublicEndpointRoute {
	if strings.HasPrefix(entry.HandlerFamily, "grok_") {
		for _, route := range entry.Routes {
			if !strings.HasPrefix(route.Pattern, "/grok/v1/") {
				return route
			}
		}
	}
	return entry.Routes[0]
}

func splitRoutePattern(value string) []string {
	trimmed := strings.Trim(strings.TrimSpace(value), "/")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "/")
}

func registeredRouteSpecificity(pattern string) int {
	score := 0
	for _, segment := range splitRoutePattern(pattern) {
		switch {
		case strings.HasPrefix(segment, "*"):
			score += 1
		case strings.HasPrefix(segment, ":"):
			score += 2
		default:
			score += 4
		}
	}
	return score
}

func matchRegisteredRoutePath(samplePath string, registeredPattern string) bool {
	pathSegments := splitRoutePattern(samplePath)
	patternSegments := splitRoutePattern(registeredPattern)
	if len(patternSegments) == 0 {
		return len(pathSegments) == 0
	}
	pathIndex := 0
	for patternIndex, segment := range patternSegments {
		switch {
		case strings.HasPrefix(segment, "*"):
			return pathIndex < len(pathSegments) && patternIndex == len(patternSegments)-1
		case pathIndex >= len(pathSegments):
			return false
		case strings.HasPrefix(segment, ":"):
			pathIndex++
		case pathSegments[pathIndex] != segment:
			return false
		default:
			pathIndex++
		}
	}
	return pathIndex == len(pathSegments)
}

func findRegisteredRouteForPattern(registered map[string]gin.RouteInfo, method string, publicPattern string) (gin.RouteInfo, bool) {
	if route, ok := registered[method+" "+publicPattern]; ok {
		return route, true
	}

	samplePath := sampleRequestPath(publicPattern)
	bestScore := -1
	bestRoute := gin.RouteInfo{}
	for _, route := range registered {
		if route.Method != method {
			continue
		}
		if !matchRegisteredRoutePath(samplePath, route.Path) {
			continue
		}
		score := registeredRouteSpecificity(route.Path)
		if score > bestScore {
			bestScore = score
			bestRoute = route
		}
	}
	if bestScore < 0 {
		return gin.RouteInfo{}, false
	}
	return bestRoute, true
}

func assertExplicitGatewayReject(t *testing.T, body string) {
	t.Helper()

	if reason := strings.TrimSpace(gjson.Get(body, "error.reason").String()); reason != "" {
		require.Equal(t, reason, gjson.Get(body, "error.code").String())
		return
	}

	reason := strings.TrimSpace(gjson.Get(body, "error.details.0.reason").String())
	require.NotEmpty(t, reason)
	require.Contains(t, []string{
		service.GatewayReasonRouteMismatch,
		service.GatewayReasonUnsupportedAction,
		service.GatewayReasonPublicEndpointUnsupported,
	}, reason)
}

func TestGatewayRoutesOpenAIResponsesCompactPathIsRegistered(t *testing.T) {
	router := newGatewayRoutesTestRouter()

	for _, path := range []string{"/v1/responses/compact", "/responses/compact"} {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"model":"gpt-5"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		require.NotEqual(t, http.StatusNotFound, w.Code, "path=%s should hit OpenAI responses handler", path)
	}
}

func TestGatewayRoutesGrokMessagesReturnUnsupported(t *testing.T) {
	router := newGatewayRoutesTestRouterWithAuth(func(c *gin.Context) {
		groupID := int64(1)
		c.Set(string(servermiddleware.ContextKeyAPIKey), &service.APIKey{
			GroupID: &groupID,
			Group:   &service.Group{Platform: service.PlatformGrok},
		})
		c.Next()
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"model":"grok-4"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Equal(t, service.GatewayReasonRouteMismatch, gjson.Get(w.Body.String(), "error.reason").String())
	require.Equal(t, service.GatewayReasonRouteMismatch, gjson.Get(w.Body.String(), "error.code").String())
}

func TestGatewayRoutesResponsesWebSocketRejectsGrokGroup(t *testing.T) {
	router := newGatewayRoutesTestRouterWithAuth(func(c *gin.Context) {
		groupID := int64(1)
		c.Set(string(servermiddleware.ContextKeyAPIKey), &service.APIKey{
			GroupID: &groupID,
			Group:   &service.Group{Platform: service.PlatformGrok},
		})
		c.Next()
	})

	req := httptest.NewRequest(http.MethodGet, "/responses", nil)
	req.Header.Set("Accept-Language", "en")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
	require.Equal(t, service.GatewayReasonPublicEndpointUnsupported, gjson.Get(w.Body.String(), "error.reason").String())
	require.Equal(t, service.GatewayReasonPublicEndpointUnsupported, gjson.Get(w.Body.String(), "error.code").String())
}

func TestGatewayRoutesChatCompletionsRejectAnthropicGroup(t *testing.T) {
	router := newGatewayRoutesTestRouterWithAuth(func(c *gin.Context) {
		groupID := int64(1)
		c.Set(string(servermiddleware.ContextKeyAPIKey), &service.APIKey{
			GroupID: &groupID,
			Group:   &service.Group{Platform: service.PlatformAnthropic},
		})
		c.Next()
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"claude-3-7-sonnet"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", "en")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
	require.Equal(t, service.GatewayReasonPublicEndpointUnsupported, gjson.Get(w.Body.String(), "error.reason").String())
	require.Equal(t, service.GatewayReasonPublicEndpointUnsupported, gjson.Get(w.Body.String(), "error.code").String())
}

func TestGatewayRoutesRegisterPublicProtocolEndpoints(t *testing.T) {
	router := newGatewayRoutesTestRouter()
	registered := make(map[string]gin.RouteInfo, len(router.Routes()))
	for _, route := range router.Routes() {
		registered[route.Method+" "+route.Path] = route
	}

	entries := service.PublicEndpointRegistryEntries()
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].CanonicalEndpoint < entries[j].CanonicalEndpoint
	})

	for _, entry := range entries {
		entry := entry
		for _, route := range entry.Routes {
			route := route
			t.Run(route.Method+" "+route.Pattern, func(t *testing.T) {
				registeredRoute, ok := findRegisteredRouteForPattern(registered, route.Method, route.Pattern)
				require.True(t, ok, "route should be registered")
				require.Equal(t, entry.CanonicalEndpoint, service.NormalizeInboundEndpoint(sampleRequestPath(route.Pattern)))
				expectedHandlerFamily := entry.HandlerFamily
				if strings.TrimSpace(route.RegisteredHandlerFamily) != "" {
					expectedHandlerFamily = route.RegisteredHandlerFamily
				}
				require.Equal(t, expectedHandlerFamily, handlerFamilyForRegisteredRoute(registeredRoute.Handler), "handler=%s", registeredRoute.Handler)
			})
		}
	}
}

func TestGatewayRoutesEveryRegistryPublicEndpointHasDynamicSample(t *testing.T) {
	entries := service.PublicEndpointRegistryEntries()
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].CanonicalEndpoint < entries[j].CanonicalEndpoint
	})

	for _, entry := range entries {
		entry := entry
		route := preferredDynamicRoute(entry)
		t.Run(route.Method+" "+route.Pattern, func(t *testing.T) {
			router := routerForRoute(entry, route)
			samplePath := sampleRequestPath(route.Pattern)
			req := httptest.NewRequest(route.Method, samplePath, strings.NewReader(sampleRequestBody(route.Pattern)))
			if route.Method != http.MethodGet && route.Method != http.MethodDelete && route.Method != http.MethodHead {
				req.Header.Set("Content-Type", "application/json")
			}
			req.Header.Set("Authorization", "Bearer test-gateway-key")
			req.Header.Set("x-goog-api-key", "test-gateway-key")
			req.Header.Set("Accept-Language", "en")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			require.Equal(t, entry.CanonicalEndpoint, service.NormalizeInboundEndpoint(samplePath))
			require.Falsef(
				t,
				w.Code == http.StatusNotFound && strings.Contains(strings.ToLower(w.Body.String()), "404 page not found"),
				"route %s %s should hit a handler or return an explicit gateway rejection, got default 404 for sample path %s",
				route.Method,
				route.Pattern,
				samplePath,
			)
			if w.Code == http.StatusBadRequest || w.Code == http.StatusNotFound {
				assertExplicitGatewayReject(t, w.Body.String())
			}
		})
	}
}
