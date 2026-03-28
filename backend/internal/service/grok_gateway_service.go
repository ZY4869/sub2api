package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/util/responseheaders"
	"github.com/gin-gonic/gin"
)

type GrokGatewayService struct {
	gatewayService       *GatewayService
	httpUpstream         HTTPUpstream
	rateLimitService     *RateLimitService
	cfg                  *config.Config
	reverseClient        *GrokReverseClient
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

func (s *GrokGatewayService) RouteMode(account *Account) string {
	if account != nil && account.IsGrokSSO() {
		return GrokRouteModeSSO
	}
	return GrokRouteModeAPIKey
}

func (s *GrokGatewayService) ForwardChatCompletions(ctx context.Context, c *gin.Context, account *Account, body []byte) (*GrokGatewayForwardResult, error) {
	if s.RouteMode(account) == GrokRouteModeSSO {
		return s.forwardSSOChatCompletions(ctx, c, account, body)
	}
	return s.forwardAPIKeyChatCompletions(ctx, c, account, body)
}

func (s *GrokGatewayService) ForwardResponses(ctx context.Context, c *gin.Context, account *Account, body []byte, method string, subpath string) (*GrokGatewayForwardResult, error) {
	if s.RouteMode(account) == GrokRouteModeSSO {
		return s.forwardSSOResponses(ctx, c, account, body, method, subpath)
	}
	return s.forwardAPIKeyResponses(ctx, c, account, body, method, subpath)
}

func (s *GrokGatewayService) ForwardImagesGeneration(ctx context.Context, c *gin.Context, account *Account, body []byte) (*GrokGatewayForwardResult, error) {
	if s.RouteMode(account) == GrokRouteModeSSO {
		return s.forwardSSOImagesGeneration(ctx, c, account, body)
	}
	return s.forwardAPIKeyImagesGeneration(ctx, c, account, body)
}

func (s *GrokGatewayService) ForwardImagesEdits(ctx context.Context, c *gin.Context, account *Account, body []byte) (*GrokGatewayForwardResult, error) {
	if s.RouteMode(account) == GrokRouteModeSSO {
		return s.forwardSSOImagesEdits(ctx, c, account, body)
	}
	return s.forwardAPIKeyImagesEdits(ctx, c, account, body)
}

func (s *GrokGatewayService) ForwardVideosGeneration(ctx context.Context, c *gin.Context, account *Account, body []byte) (*GrokGatewayForwardResult, error) {
	if s.RouteMode(account) == GrokRouteModeSSO {
		return s.forwardSSOVideosGeneration(ctx, c, account, body)
	}
	return s.forwardAPIKeyVideosGeneration(ctx, c, account, body)
}

func (s *GrokGatewayService) ForwardVideoStatus(ctx context.Context, c *gin.Context, account *Account, requestID string) (*GrokGatewayForwardResult, error) {
	if s.RouteMode(account) == GrokRouteModeSSO {
		return s.forwardSSOVideoStatus(ctx, c, account, requestID)
	}
	return s.forwardAPIKeyVideoStatus(ctx, c, account, requestID)
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

func (s *GrokGatewayService) newJSONRequest(ctx context.Context, method string, url string, body []byte) (*http.Request, error) {
	var reader io.Reader
	if len(body) > 0 {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return nil, err
	}
	return req, nil
}

func (s *GrokGatewayService) handleHTTPError(ctx context.Context, resp *http.Response, c *gin.Context, account *Account, routeMode string) error {
	if resp == nil {
		return fmt.Errorf("upstream response is nil")
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

func (s *GrokGatewayService) shouldFailoverStatus(statusCode int) bool {
	switch statusCode {
	case 401, 402, 403, 429, 529:
		return true
	default:
		return statusCode >= 500
	}
}

func readJSONMap(body []byte) (map[string]any, error) {
	if len(body) == 0 {
		return map[string]any{}, nil
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}
