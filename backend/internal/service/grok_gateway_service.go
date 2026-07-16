package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/util/responseheaders"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type GrokGatewayService struct {
	gatewayService       *GatewayService
	httpUpstream         HTTPUpstream
	rateLimitService     *RateLimitService
	cfg                  *config.Config
	reverseClient        *GrokReverseClient
	tokenProvider        *GrokTokenProvider
	accountRepo          AccountRepository
	responseHeaderFilter *responseheaders.CompiledHeaderFilter
}

func NewGrokGatewayService(
	gatewayService *GatewayService,
	httpUpstream HTTPUpstream,
	rateLimitService *RateLimitService,
	reverseClient *GrokReverseClient,
	cfg *config.Config,
) *GrokGatewayService {
	return &GrokGatewayService{
		gatewayService:       gatewayService,
		httpUpstream:         httpUpstream,
		rateLimitService:     rateLimitService,
		cfg:                  cfg,
		reverseClient:        reverseClient,
		responseHeaderFilter: compileResponseHeaderFilter(cfg),
	}
}

func (s *GrokGatewayService) SetGrokRuntimeDependencies(tokenProvider *GrokTokenProvider, accountRepo AccountRepository) {
	if s == nil {
		return
	}
	s.tokenProvider = tokenProvider
	s.accountRepo = accountRepo
}

func (s *GrokGatewayService) RouteMode(account *Account) string {
	if account == nil {
		return GrokRouteModeAPIKey
	}
	switch {
	case account.IsGrokSSO():
		return GrokRouteModeSSOReverse
	case account.IsGrokOAuth():
		return GrokRouteModeOAuthBuild
	default:
		return GrokRouteModeAPIKey
	}
}

func (s *GrokGatewayService) ForwardChatCompletions(ctx context.Context, c *gin.Context, account *Account, body []byte) (*GrokGatewayForwardResult, error) {
	switch s.RouteMode(account) {
	case GrokRouteModeSSOReverse:
		return s.forwardSSOChatCompletions(ctx, c, account, body)
	case GrokRouteModeOAuthBuild:
		return s.forwardOAuthBuildChatCompletions(ctx, c, account, body)
	default:
		return s.forwardAPIKeyChatCompletions(ctx, c, account, body)
	}
}

func (s *GrokGatewayService) ForwardResponses(ctx context.Context, c *gin.Context, account *Account, body []byte, method string, subpath string) (*GrokGatewayForwardResult, error) {
	switch s.RouteMode(account) {
	case GrokRouteModeSSOReverse:
		return s.forwardSSOResponses(ctx, c, account, body, method, subpath)
	case GrokRouteModeOAuthBuild:
		return s.forwardOAuthBuildResponses(ctx, c, account, body, method, subpath)
	default:
		return s.forwardAPIKeyResponses(ctx, c, account, body, method, subpath)
	}
}

func (s *GrokGatewayService) ForwardMessagesCompat(ctx context.Context, c *gin.Context, account *Account, body []byte) (*GrokGatewayForwardResult, error) {
	switch s.RouteMode(account) {
	case GrokRouteModeSSOReverse:
		return s.forwardSSOMessagesCompat(ctx, c, account, body)
	case GrokRouteModeOAuthBuild:
		return s.forwardOAuthBuildMessagesCompat(ctx, c, account, body)
	default:
		return s.forwardAPIKeyMessagesCompat(ctx, c, account, body)
	}
}

func (s *GrokGatewayService) ForwardAnthropicCountTokensCompat(ctx context.Context, c *gin.Context, account *Account, body []byte) (*AnthropicCountTokensBridgeResult, error) {
	if s.RouteMode(account) == GrokRouteModeSSOReverse {
		writeAnthropicError(c, http.StatusNotFound, "not_found_error", "count_tokens endpoint is not supported for Grok SSO accounts", "")
		return nil, fmt.Errorf("grok sso count_tokens is not supported")
	}
	return s.forwardOfficialAnthropicCountTokensCompat(ctx, c, account, body)
}

func (s *GrokGatewayService) ForwardImagesGeneration(ctx context.Context, c *gin.Context, account *Account, body []byte) (*GrokGatewayForwardResult, error) {
	switch s.RouteMode(account) {
	case GrokRouteModeSSOReverse:
		return s.forwardSSOImagesGeneration(ctx, c, account, body)
	case GrokRouteModeOAuthBuild:
		return s.forwardOAuthBuildImagesGeneration(ctx, c, account, body)
	default:
		return s.forwardAPIKeyImagesGeneration(ctx, c, account, body)
	}
}

func (s *GrokGatewayService) ForwardImagesEdits(ctx context.Context, c *gin.Context, account *Account, body []byte) (*GrokGatewayForwardResult, error) {
	switch s.RouteMode(account) {
	case GrokRouteModeSSOReverse:
		return s.forwardSSOImagesEdits(ctx, c, account, body)
	case GrokRouteModeOAuthBuild:
		return s.forwardOAuthBuildImagesEdits(ctx, c, account, body)
	default:
		return s.forwardAPIKeyImagesEdits(ctx, c, account, body)
	}
}

func (s *GrokGatewayService) ForwardVideosGeneration(ctx context.Context, c *gin.Context, account *Account, body []byte) (*GrokGatewayForwardResult, error) {
	switch s.RouteMode(account) {
	case GrokRouteModeSSOReverse:
		return s.forwardSSOVideosGeneration(ctx, c, account, body)
	case GrokRouteModeOAuthBuild:
		return s.forwardOAuthBuildVideosGeneration(ctx, c, account, body)
	default:
		return s.forwardAPIKeyVideosGeneration(ctx, c, account, body)
	}
}

func (s *GrokGatewayService) ForwardVideosEdit(ctx context.Context, c *gin.Context, account *Account, body []byte) (*GrokGatewayForwardResult, error) {
	switch s.RouteMode(account) {
	case GrokRouteModeSSOReverse:
		return s.forwardSSOVideosEdit(ctx, c, account, body)
	case GrokRouteModeOAuthBuild:
		return s.forwardOAuthBuildVideosEdit(ctx, c, account, body)
	default:
		return s.forwardAPIKeyVideosEdit(ctx, c, account, body)
	}
}

func (s *GrokGatewayService) ForwardVideosExtension(ctx context.Context, c *gin.Context, account *Account, body []byte) (*GrokGatewayForwardResult, error) {
	switch s.RouteMode(account) {
	case GrokRouteModeSSOReverse:
		return s.forwardSSOVideosExtension(ctx, c, account, body)
	case GrokRouteModeOAuthBuild:
		return s.forwardOAuthBuildVideosExtension(ctx, c, account, body)
	default:
		return s.forwardAPIKeyVideosExtension(ctx, c, account, body)
	}
}

func (s *GrokGatewayService) ForwardVideoStatus(ctx context.Context, c *gin.Context, account *Account, requestID string) (*GrokGatewayForwardResult, error) {
	switch s.RouteMode(account) {
	case GrokRouteModeSSOReverse:
		return s.forwardSSOVideoStatus(ctx, c, account, requestID)
	case GrokRouteModeOAuthBuild:
		return s.forwardOAuthBuildVideoStatus(ctx, c, account, requestID)
	default:
		return s.forwardAPIKeyVideoStatus(ctx, c, account, requestID)
	}
}

func (s *GrokGatewayService) validatedBaseURL(raw string, fallback string) (string, error) {
	candidate := strings.TrimSpace(raw)
	if candidate == "" {
		candidate = fallback
	}
	if s.gatewayService != nil {
		return s.gatewayService.validateUpstreamBaseURL(candidate)
	}
	return strings.TrimRight(candidate, "/"), nil
}

func (s *GrokGatewayService) writeJSONResponse(c *gin.Context, resp *http.Response, body []byte) {
	if c == nil || resp == nil {
		return
	}
	if s.responseHeaderFilter != nil {
		responseheaders.WriteFilteredHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)
	}
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/json"
	}
	c.Data(resp.StatusCode, contentType, body)
}

func (s *GrokGatewayService) handleHTTPError(ctx context.Context, resp *http.Response, c *gin.Context, account *Account, meta grokUpstreamRequestMetadata) error {
	if resp == nil {
		return fmt.Errorf("upstream response is nil")
	}
	routeMode := strings.TrimSpace(meta.RouteMode)
	if routeMode == "" {
		routeMode = s.RouteMode(account)
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	upstreamMsg := sanitizeUpstreamErrorMessage(strings.TrimSpace(extractUpstreamErrorMessage(body)))
	upstreamDetail := ""
	if s.cfg != nil && s.cfg.Gateway.LogUpstreamErrorBody {
		maxBytes := s.cfg.Gateway.LogUpstreamErrorBodyMaxBytes
		if maxBytes <= 0 {
			maxBytes = 2048
		}
		upstreamDetail = truncateString(string(body), maxBytes)
	}
	setOpsUpstreamError(c, resp.StatusCode, upstreamMsg, upstreamDetail)
	appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
		Platform:           PlatformGrok,
		AccountID:          account.ID,
		AccountName:        account.Name,
		UpstreamStatusCode: resp.StatusCode,
		UpstreamRequestID:  resp.Header.Get("x-request-id"),
		Kind:               "http_error",
		Message:            upstreamMsg,
		Detail:             upstreamDetail,
	})
	logger.FromContext(ctx).Warn("grok.upstream_http_error", grokUpstreamLogFields(account, meta, resp.StatusCode, resp.Header.Get("x-request-id"))...)
	shouldDisable := false
	if s.rateLimitService != nil {
		shouldDisable = s.rateLimitService.HandleUpstreamError(ctx, account, resp.StatusCode, resp.Header, body)
	}
	if shouldDisable || s.shouldFailoverStatus(resp.StatusCode) {
		return &UpstreamFailoverError{
			StatusCode:      resp.StatusCode,
			ResponseBody:    body,
			ResponseHeaders: resp.Header.Clone(),
		}
	}
	s.writeJSONResponse(c, resp, body)
	if upstreamMsg == "" {
		upstreamMsg = fmt.Sprintf("grok upstream error: %d", resp.StatusCode)
	}
	return fmt.Errorf("%s %s", routeMode, upstreamMsg)
}

func grokUpstreamLogFields(account *Account, meta grokUpstreamRequestMetadata, upstreamStatus int, upstreamRequestID string) []zap.Field {
	accountID := int64(0)
	accountType := ""
	if account != nil {
		accountID = account.ID
		accountType = strings.TrimSpace(account.Type)
	}
	routeMode := strings.TrimSpace(meta.RouteMode)
	if routeMode == "" && account != nil {
		routeMode = grokRouteModeForAccount(account)
	}
	fields := []zap.Field{
		zap.Int64("account_id", accountID),
		zap.String("platform", PlatformGrok),
		zap.String("type", accountType),
		zap.String("route_mode", routeMode),
		zap.String("effective_host", strings.TrimSpace(meta.EffectiveHost)),
		zap.String("effective_endpoint", strings.TrimSpace(meta.EffectiveEndpoint)),
		zap.Int("upstream_status", upstreamStatus),
	}
	if upstreamRequestID = strings.TrimSpace(upstreamRequestID); upstreamRequestID != "" {
		fields = append(fields, zap.String("upstream_request_id", upstreamRequestID))
	}
	return fields
}

func grokUpstreamRequestID(headers http.Header) string {
	for _, name := range []string{"x-request-id", "X-Request-Id", "xai-request-id", "Xai-Request-Id"} {
		if value := strings.TrimSpace(headers.Get(name)); value != "" {
			return value
		}
		if values, ok := headers[name]; ok && len(values) > 0 {
			if value := strings.TrimSpace(values[0]); value != "" {
				return value
			}
		}
	}
	return ""
}

func (s *GrokGatewayService) shouldFailoverStatus(statusCode int) bool {
	switch statusCode {
	case 401, 402, 403, 429, 529:
		return true
	default:
		return statusCode >= 500
	}
}
