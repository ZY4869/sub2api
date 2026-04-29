package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	ModelDebugKeyModeSaved  = "saved"
	ModelDebugKeyModeManual = "manual"

	ModelDebugProtocolOpenAI    = "openai"
	ModelDebugProtocolAnthropic = "anthropic"
	ModelDebugProtocolGemini    = "gemini"

	ModelDebugEndpointResponses       = "responses"
	ModelDebugEndpointChatCompletions = "chat_completions"
	ModelDebugEndpointMessages        = "messages"
	ModelDebugEndpointGenerateContent = "generate_content"

	modelDebugAnthropicVersion = "2023-06-01"
	modelDebugEventStart       = "start"
	modelDebugEventRequest     = "request_preview"
	modelDebugEventHeaders     = "response_headers"
	modelDebugEventContent     = "content"
	modelDebugEventFinal       = "final"
	modelDebugEventError       = "error"
	modelDebugTraceAction      = "model_debug"
	modelDebugUpstreamAction   = "model_debug_upstream"
	modelDebugTraceBodyLimit   = 64 * 1024
)

type modelDebugAPIKeyReader interface {
	GetByID(ctx context.Context, id int64) (*APIKey, error)
}

type ModelDebugHTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type ModelDebugRunInput struct {
	AdminUserID     int64          `json:"-"`
	BaseURL         string         `json:"-"`
	ClientRequestID string         `json:"-"`
	KeyMode         string         `json:"key_mode"`
	APIKeyID        *int64         `json:"api_key_id,omitempty"`
	ManualAPIKey    string         `json:"manual_api_key,omitempty"`
	Protocol        string         `json:"protocol"`
	EndpointKind    string         `json:"endpoint_kind"`
	Model           string         `json:"model"`
	Stream          bool           `json:"stream"`
	RequestBody     map[string]any `json:"request_body"`
}

type ModelDebugService struct {
	apiKeyReader modelDebugAPIKeyReader
	opsService   *OpsService
	httpClient   ModelDebugHTTPClient
	cfg          *config.Config
}

func NewModelDebugService(apiKeyReader modelDebugAPIKeyReader, opsService *OpsService, cfg *config.Config) *ModelDebugService {
	client := &http.Client{Timeout: 5 * time.Minute}
	return &ModelDebugService{
		apiKeyReader: apiKeyReader,
		opsService:   opsService,
		httpClient:   client,
		cfg:          cfg,
	}
}

func (s *ModelDebugService) SetHTTPClient(client ModelDebugHTTPClient) {
	if s == nil || client == nil {
		return
	}
	s.httpClient = client
}

func (s *ModelDebugService) Run(
	ctx context.Context,
	input ModelDebugRunInput,
	send func(event string, payload any) error,
) error {
	startedAt := time.Now().UTC()
	debugRunID := uuid.NewString()
	log := logger.FromContext(ctx).With(
		zap.String("component", "service.model_debug"),
		zap.String("debug_run_id", debugRunID),
		zap.String("protocol", strings.TrimSpace(input.Protocol)),
		zap.String("endpoint_kind", strings.TrimSpace(input.EndpointKind)),
		zap.String("model", strings.TrimSpace(input.Model)),
		zap.Bool("stream", input.Stream),
		zap.Int64("admin_user_id", input.AdminUserID),
	)

	if err := send(modelDebugEventStart, map[string]any{
		"debug_run_id":  debugRunID,
		"protocol":      input.Protocol,
		"endpoint_kind": input.EndpointKind,
		"model":         input.Model,
		"stream":        input.Stream,
	}); err != nil {
		return err
	}

	requestPlan, apiKeyValue, err := s.prepareRequest(ctx, input)
	if err != nil {
		log.Warn("model debug validation failed", zap.Error(err))
		_ = send(modelDebugEventError, map[string]any{
			"debug_run_id": debugRunID,
			"message":      err.Error(),
		})
		s.recordTrace(ctx, modelDebugTraceAction, debugRunID, input, requestPlan, modelDebugTraceResult{
			Status:     "error",
			StatusCode: http.StatusBadRequest,
			Message:    err.Error(),
			Duration:   time.Since(startedAt),
		})
		return err
	}

	requestPreviewPayload := map[string]any{
		"debug_run_id": debugRunID,
		"method":       http.MethodPost,
		"url":          requestPlan.URL,
		"path":         requestPlan.Path,
		"headers":      requestPlan.PreviewHeaders,
		"body":         requestPlan.BodyPreview,
	}
	if err := send(modelDebugEventRequest, requestPreviewPayload); err != nil {
		return err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, requestPlan.URL, bytes.NewReader(requestPlan.BodyBytes))
	if err != nil {
		_ = send(modelDebugEventError, map[string]any{"debug_run_id": debugRunID, "message": err.Error()})
		s.recordTrace(ctx, modelDebugTraceAction, debugRunID, input, requestPlan, modelDebugTraceResult{
			Status:     "error",
			StatusCode: http.StatusInternalServerError,
			Message:    err.Error(),
			Duration:   time.Since(startedAt),
		})
		return err
	}
	for key, value := range requestPlan.Headers {
		httpReq.Header.Set(key, value)
	}
	if apiKeyValue == "" {
		return infraerrors.BadRequest("MODEL_DEBUG_API_KEY_REQUIRED", "api key is required")
	}

	log.Info("model debug request start")
	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		log.Warn("model debug upstream request failed", zap.Error(err))
		_ = send(modelDebugEventError, map[string]any{
			"debug_run_id": debugRunID,
			"message":      err.Error(),
		})
		s.recordTrace(ctx, modelDebugTraceAction, debugRunID, input, requestPlan, modelDebugTraceResult{
			Status:     "error",
			StatusCode: http.StatusBadGateway,
			Message:    err.Error(),
			Duration:   time.Since(startedAt),
		})
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	responseHeaders := modelDebugHeaderPreview(resp.Header)
	upstreamRequestID := extractModelDebugUpstreamRequestID(resp.Header)
	if err := send(modelDebugEventHeaders, map[string]any{
		"debug_run_id":        debugRunID,
		"status_code":         resp.StatusCode,
		"headers":             responseHeaders,
		"upstream_request_id": upstreamRequestID,
	}); err != nil {
		return err
	}

	bodyPreview, bytesReceived, streamErr := s.streamResponse(resp, send, debugRunID)
	duration := time.Since(startedAt)
	if streamErr != nil {
		log.Warn("model debug response stream failed", zap.Error(streamErr))
		_ = send(modelDebugEventError, map[string]any{
			"debug_run_id": debugRunID,
			"message":      streamErr.Error(),
		})
		s.recordTrace(ctx, modelDebugTraceAction, debugRunID, input, requestPlan, modelDebugTraceResult{
			Status:             "error",
			StatusCode:         resp.StatusCode,
			UpstreamStatusCode: &resp.StatusCode,
			UpstreamRequestID:  upstreamRequestID,
			Headers:            responseHeaders,
			ResponsePreview:    bodyPreview,
			Message:            streamErr.Error(),
			Duration:           duration,
			BytesReceived:      bytesReceived,
		})
		return streamErr
	}

	finalPayload := map[string]any{
		"debug_run_id":        debugRunID,
		"status_code":         resp.StatusCode,
		"bytes_received":      bytesReceived,
		"upstream_request_id": upstreamRequestID,
	}
	if err := send(modelDebugEventFinal, finalPayload); err != nil {
		return err
	}

	resultStatus := "success"
	if resp.StatusCode >= http.StatusBadRequest {
		resultStatus = "error"
	}
	log.Info("model debug request completed", zap.Int("status_code", resp.StatusCode), zap.Int("bytes_received", bytesReceived))
	s.recordTrace(ctx, modelDebugTraceAction, debugRunID, input, requestPlan, modelDebugTraceResult{
		Status:             resultStatus,
		StatusCode:         resp.StatusCode,
		UpstreamStatusCode: &resp.StatusCode,
		UpstreamRequestID:  upstreamRequestID,
		Headers:            responseHeaders,
		ResponsePreview:    bodyPreview,
		Duration:           duration,
		BytesReceived:      bytesReceived,
	})
	s.recordTrace(ctx, modelDebugUpstreamAction, debugRunID, input, requestPlan, modelDebugTraceResult{
		Status:             resultStatus,
		StatusCode:         resp.StatusCode,
		UpstreamStatusCode: &resp.StatusCode,
		UpstreamRequestID:  upstreamRequestID,
		Headers:            responseHeaders,
		ResponsePreview:    bodyPreview,
		Duration:           duration,
		BytesReceived:      bytesReceived,
	})
	return nil
}

type modelDebugPreparedRequest struct {
	Path           string
	URL            string
	Headers        map[string]string
	PreviewHeaders map[string]string
	BodyBytes      []byte
	BodyPreview    map[string]any
}

type modelDebugTraceResult struct {
	Status             string
	StatusCode         int
	UpstreamStatusCode *int
	UpstreamRequestID  string
	Headers            map[string]string
	ResponsePreview    string
	Message            string
	Duration           time.Duration
	BytesReceived      int
}

func (s *ModelDebugService) prepareRequest(ctx context.Context, input ModelDebugRunInput) (modelDebugPreparedRequest, string, error) {
	keyMode := strings.TrimSpace(strings.ToLower(input.KeyMode))
	protocol := strings.TrimSpace(strings.ToLower(input.Protocol))
	endpointKind := strings.TrimSpace(strings.ToLower(input.EndpointKind))
	modelID := strings.TrimSpace(input.Model)
	baseURL := strings.TrimRight(strings.TrimSpace(input.BaseURL), "/")
	if baseURL == "" {
		return modelDebugPreparedRequest{}, "", infraerrors.BadRequest("MODEL_DEBUG_BASE_URL_REQUIRED", "base url is required")
	}
	if modelID == "" {
		return modelDebugPreparedRequest{}, "", infraerrors.BadRequest("MODEL_DEBUG_MODEL_REQUIRED", "model is required")
	}
	if !isAllowedModelDebugEndpoint(protocol, endpointKind) {
		return modelDebugPreparedRequest{}, "", infraerrors.BadRequest("MODEL_DEBUG_ENDPOINT_INVALID", "protocol and endpoint_kind combination is not supported")
	}

	apiKeyValue, err := s.resolveAPIKeyValue(ctx, input, keyMode)
	if err != nil {
		return modelDebugPreparedRequest{}, "", err
	}

	bodyPreview := cloneModelDebugBody(input.RequestBody)
	switch protocol {
	case ModelDebugProtocolOpenAI, ModelDebugProtocolAnthropic:
		bodyPreview["model"] = modelID
		bodyPreview["stream"] = input.Stream
	case ModelDebugProtocolGemini:
		delete(bodyPreview, "model")
		delete(bodyPreview, "stream")
	}

	bodyBytes, err := json.Marshal(bodyPreview)
	if err != nil {
		return modelDebugPreparedRequest{}, "", infraerrors.BadRequest("MODEL_DEBUG_REQUEST_BODY_INVALID", "request_body must be valid json")
	}

	path := buildModelDebugPath(protocol, endpointKind, modelID, input.Stream)
	headers := map[string]string{
		"Content-Type": "application/json",
		"User-Agent":   "sub2api-admin-model-debug/1.0",
	}
	switch protocol {
	case ModelDebugProtocolOpenAI:
		headers["Authorization"] = "Bearer " + apiKeyValue
	case ModelDebugProtocolAnthropic:
		headers["x-api-key"] = apiKeyValue
		headers["anthropic-version"] = modelDebugAnthropicVersion
	case ModelDebugProtocolGemini:
		headers["x-goog-api-key"] = apiKeyValue
	}
	if input.Stream {
		headers["Accept"] = "text/event-stream"
	}
	return modelDebugPreparedRequest{
		Path:           path,
		URL:            baseURL + path,
		Headers:        headers,
		PreviewHeaders: redactModelDebugHeaders(headers),
		BodyBytes:      bodyBytes,
		BodyPreview:    bodyPreview,
	}, apiKeyValue, nil
}

func (s *ModelDebugService) resolveAPIKeyValue(ctx context.Context, input ModelDebugRunInput, keyMode string) (string, error) {
	switch keyMode {
	case ModelDebugKeyModeSaved:
		if input.APIKeyID == nil || *input.APIKeyID <= 0 {
			return "", infraerrors.BadRequest("MODEL_DEBUG_API_KEY_ID_REQUIRED", "api_key_id is required when key_mode is saved")
		}
		if s == nil || s.apiKeyReader == nil {
			return "", infraerrors.BadRequest("MODEL_DEBUG_API_KEY_UNAVAILABLE", "saved api key lookup is unavailable")
		}
		apiKey, err := s.apiKeyReader.GetByID(ctx, *input.APIKeyID)
		if err != nil {
			return "", err
		}
		if apiKey == nil || apiKey.UserID != input.AdminUserID {
			return "", infraerrors.Forbidden("MODEL_DEBUG_API_KEY_FORBIDDEN", "api key does not belong to the current admin")
		}
		if strings.TrimSpace(apiKey.Key) == "" {
			return "", infraerrors.BadRequest("MODEL_DEBUG_API_KEY_EMPTY", "saved api key is empty")
		}
		return strings.TrimSpace(apiKey.Key), nil
	case ModelDebugKeyModeManual:
		if strings.TrimSpace(input.ManualAPIKey) == "" {
			return "", infraerrors.BadRequest("MODEL_DEBUG_MANUAL_API_KEY_REQUIRED", "manual_api_key is required when key_mode is manual")
		}
		return strings.TrimSpace(input.ManualAPIKey), nil
	default:
		return "", infraerrors.BadRequest("MODEL_DEBUG_KEY_MODE_INVALID", "key_mode must be saved or manual")
	}
}

func (s *ModelDebugService) streamResponse(
	resp *http.Response,
	send func(event string, payload any) error,
	debugRunID string,
) (string, int, error) {
	reader := bufio.NewReader(resp.Body)
	var (
		total   int
		builder strings.Builder
	)
	for {
		chunk, err := reader.ReadBytes('\n')
		if len(chunk) > 0 {
			total += len(chunk)
			text := string(chunk)
			if builder.Len() < modelDebugTraceBodyLimit {
				_, _ = builder.WriteString(limitModelDebugTraceText(text, modelDebugTraceBodyLimit-builder.Len()))
			}
			if sendErr := send(modelDebugEventContent, map[string]any{
				"debug_run_id": debugRunID,
				"chunk":        text,
			}); sendErr != nil {
				return builder.String(), total, sendErr
			}
		}
		if err == nil {
			continue
		}
		if err == io.EOF {
			break
		}
		return builder.String(), total, err
	}
	return builder.String(), total, nil
}

func (s *ModelDebugService) recordTrace(
	ctx context.Context,
	probeAction string,
	debugRunID string,
	input ModelDebugRunInput,
	request modelDebugPreparedRequest,
	result modelDebugTraceResult,
) {
	if s == nil || s.opsService == nil || input.AdminUserID <= 0 {
		return
	}

	requestJSON := marshalModelDebugTraceJSON(map[string]any{
		"method":  http.MethodPost,
		"url":     request.URL,
		"path":    request.Path,
		"headers": request.PreviewHeaders,
		"body":    request.BodyPreview,
	})
	responseJSON := marshalModelDebugTraceJSON(map[string]any{
		"status_code":         result.StatusCode,
		"upstream_request_id": result.UpstreamRequestID,
		"headers":             result.Headers,
		"bytes_received":      result.BytesReceived,
		"preview":             result.ResponsePreview,
		"message":             result.Message,
	})
	headersJSON := marshalModelDebugTraceJSON(result.Headers)
	userID := input.AdminUserID
	upstreamPath := request.Path
	_ = s.opsService.RecordRequestTrace(ctx, &OpsRecordRequestTraceInput{
		RequestID:          strings.TrimSpace(debugRunID),
		ClientRequestID:    strings.TrimSpace(input.ClientRequestID),
		UpstreamRequestID:  strings.TrimSpace(result.UpstreamRequestID),
		UserID:             &userID,
		Status:             result.Status,
		StatusCode:         result.StatusCode,
		UpstreamStatusCode: result.UpstreamStatusCode,
		DurationMs:         result.Duration.Milliseconds(),
		Trace: GatewayTraceContext{
			Normalize: ProtocolNormalizeResult{
				Platform:          strings.TrimSpace(input.Protocol),
				ProtocolIn:        strings.TrimSpace(input.Protocol),
				ProtocolOut:       strings.TrimSpace(input.Protocol),
				Channel:           "admin",
				RoutePath:         "/api/v1/admin/models/debug/run",
				UpstreamPath:      upstreamPath,
				RequestType:       "model_debug",
				RequestedModel:    strings.TrimSpace(input.Model),
				ProbeAction:       probeAction,
				Stream:            input.Stream,
				UpstreamRequestID: strings.TrimSpace(result.UpstreamRequestID),
			},
			NormalizedRequestJSON: modelDebugStringPtr(requestJSON),
			GatewayResponseJSON:   modelDebugStringPtr(responseJSON),
			ResponseHeadersJSON:   modelDebugStringPtr(headersJSON),
		},
		CreatedAt: time.Now().UTC(),
	})
}

func isAllowedModelDebugEndpoint(protocol string, endpointKind string) bool {
	switch protocol {
	case ModelDebugProtocolOpenAI:
		return endpointKind == ModelDebugEndpointResponses || endpointKind == ModelDebugEndpointChatCompletions
	case ModelDebugProtocolAnthropic:
		return endpointKind == ModelDebugEndpointMessages
	case ModelDebugProtocolGemini:
		return endpointKind == ModelDebugEndpointGenerateContent
	default:
		return false
	}
}

func buildModelDebugPath(protocol string, endpointKind string, modelID string, stream bool) string {
	switch protocol {
	case ModelDebugProtocolOpenAI:
		if endpointKind == ModelDebugEndpointChatCompletions {
			return "/v1/chat/completions"
		}
		return "/v1/responses"
	case ModelDebugProtocolAnthropic:
		return "/v1/messages"
	case ModelDebugProtocolGemini:
		action := "generateContent"
		if stream {
			action = "streamGenerateContent"
		}
		path := fmt.Sprintf("/v1beta/models/%s:%s", url.PathEscape(modelID), action)
		if stream {
			return path + "?alt=sse"
		}
		return path
	default:
		return ""
	}
}

func cloneModelDebugBody(source map[string]any) map[string]any {
	if len(source) == 0 {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func redactModelDebugHeaders(headers map[string]string) map[string]string {
	if len(headers) == 0 {
		return map[string]string{}
	}
	cloned := make(map[string]string, len(headers))
	for key, value := range headers {
		lowerKey := strings.ToLower(strings.TrimSpace(key))
		switch lowerKey {
		case "authorization", "x-api-key", "x-goog-api-key":
			cloned[key] = "[REDACTED]"
		default:
			cloned[key] = value
		}
	}
	return cloned
}

func modelDebugHeaderPreview(headers http.Header) map[string]string {
	if len(headers) == 0 {
		return map[string]string{}
	}
	preview := make(map[string]string, len(headers))
	for key, values := range headers {
		preview[key] = strings.Join(values, ", ")
	}
	return preview
}

func extractModelDebugUpstreamRequestID(headers http.Header) string {
	for _, key := range []string{"x-request-id", "request-id", "anthropic-request-id", "x-goog-request-id"} {
		if value := strings.TrimSpace(headers.Get(key)); value != "" {
			return value
		}
	}
	return ""
}

func marshalModelDebugTraceJSON(value any) string {
	payload, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	if len(payload) <= modelDebugTraceBodyLimit {
		return string(payload)
	}
	return string(payload[:modelDebugTraceBodyLimit]) + "...(truncated)"
}

func modelDebugStringPtr(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func limitModelDebugTraceText(value string, limit int) string {
	if limit <= 0 || len(value) <= limit {
		return value
	}
	return value[:limit]
}
