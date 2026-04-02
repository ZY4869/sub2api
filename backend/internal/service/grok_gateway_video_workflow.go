package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

func (s *GrokGatewayService) forwardGrokVideoChatCompletions(ctx context.Context, c *gin.Context, account *Account, body []byte) (*GrokGatewayForwardResult, error) {
	req, err := grokBuildVideoWorkflowRequestFromChatBody(body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"type": "invalid_request_error", "message": "Failed to parse request body"}})
		return nil, err
	}
	return s.forwardGrokVideoChatLike(ctx, c, account, req)
}

func (s *GrokGatewayService) forwardGrokVideoResponses(ctx context.Context, c *gin.Context, account *Account, body []byte) (*GrokGatewayForwardResult, error) {
	req, err := grokBuildVideoWorkflowRequestFromResponsesBody(body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"type": "invalid_request_error", "message": "Failed to parse request body"}})
		return nil, err
	}
	return s.forwardGrokVideoResponsesLike(ctx, c, account, req)
}

func (s *GrokGatewayService) forwardGrokVideoCreate(ctx context.Context, c *gin.Context, account *Account, body []byte) (*GrokGatewayForwardResult, error) {
	req, err := grokBuildVideoWorkflowRequestFromVideosBody(body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"type": "invalid_request_error", "message": "Failed to parse request body"}})
		return nil, err
	}
	if err := grokValidateVideoWorkflowRequest(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"type": "invalid_request_error", "message": err.Error()}})
		return nil, err
	}

	startTime := time.Now()
	result, err := s.runGrokVideoWorkflow(ctx, c, account, req)
	if err != nil {
		return nil, err
	}
	c.JSON(http.StatusOK, grokBuildVideoCreateResponse(result, req))
	return &GrokGatewayForwardResult{
		Result:            grokVideoForwardResult(req, result, false, nil, time.Since(startTime)),
		RouteMode:         s.RouteMode(account),
		Endpoint:          grokEndpointVideosGen,
		MediaType:         "video",
		UpstreamRequestID: strings.TrimSpace(result.RequestID),
	}, nil
}

func (s *GrokGatewayService) forwardGrokVideoChatLike(ctx context.Context, c *gin.Context, account *Account, req *grokVideoWorkflowRequest) (*GrokGatewayForwardResult, error) {
	if err := grokValidateVideoWorkflowRequest(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"type": "invalid_request_error", "message": err.Error()}})
		return nil, err
	}

	startTime := time.Now()
	result, err := s.runGrokVideoWorkflow(ctx, c, account, req)
	if err != nil {
		return nil, err
	}

	var firstTokenMs *int
	if req.Stream {
		firstTokenMs, err = s.writeGrokVideoChatStream(c, req.RequestedModel, result, startTime)
		if err != nil {
			return nil, err
		}
	} else {
		c.JSON(http.StatusOK, apicompat.ResponsesToChatCompletions(grokBuildVideoResponsesResponse(result, req.RequestedModel), req.RequestedModel))
	}

	return &GrokGatewayForwardResult{
		Result:            grokVideoForwardResult(req, result, req.Stream, firstTokenMs, time.Since(startTime)),
		RouteMode:         s.RouteMode(account),
		Endpoint:          grokEndpointChatCompletions,
		MediaType:         "video",
		UpstreamRequestID: strings.TrimSpace(result.RequestID),
	}, nil
}

func (s *GrokGatewayService) forwardGrokVideoResponsesLike(ctx context.Context, c *gin.Context, account *Account, req *grokVideoWorkflowRequest) (*GrokGatewayForwardResult, error) {
	if err := grokValidateVideoWorkflowRequest(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"type": "invalid_request_error", "message": err.Error()}})
		return nil, err
	}

	startTime := time.Now()
	result, err := s.runGrokVideoWorkflow(ctx, c, account, req)
	if err != nil {
		return nil, err
	}

	var firstTokenMs *int
	if req.Stream {
		firstTokenMs, err = s.writeGrokVideoResponsesStream(c, req.RequestedModel, result, startTime)
		if err != nil {
			return nil, err
		}
	} else {
		c.JSON(http.StatusOK, grokBuildVideoResponsesResponse(result, req.RequestedModel))
	}

	return &GrokGatewayForwardResult{
		Result:            grokVideoForwardResult(req, result, req.Stream, firstTokenMs, time.Since(startTime)),
		RouteMode:         s.RouteMode(account),
		Endpoint:          grokEndpointResponses,
		MediaType:         "video",
		UpstreamRequestID: strings.TrimSpace(result.RequestID),
	}, nil
}

func (s *GrokGatewayService) runGrokVideoWorkflow(ctx context.Context, c *gin.Context, account *Account, req *grokVideoWorkflowRequest) (*grokVideoResult, error) {
	if account != nil && account.IsGrokSSO() {
		return s.runSSOGrokVideoWorkflow(ctx, c, account, req)
	}
	return s.runAPIKeyGrokVideoWorkflow(ctx, c, account, req)
}

func (s *GrokGatewayService) runAPIKeyGrokVideoWorkflow(ctx context.Context, c *gin.Context, account *Account, req *grokVideoWorkflowRequest) (*grokVideoResult, error) {
	upstreamModel := grokResolveVideoUpstreamModel(account, req.RequestedModel)
	payloadBody, endpoint, err := grokBuildAPIKeyVideoPayload(req, upstreamModel)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"type": "invalid_request_error", "message": err.Error()}})
		return nil, err
	}

	startTime := time.Now()
	resp, err := s.doAPIKeyRequest(ctx, c, account, http.MethodPost, endpoint, payloadBody)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		return nil, s.handleHTTPError(ctx, resp, c, account, GrokRouteModeAPIKey)
	}

	bodyBytes, err := readUpstreamResponseBodyLimited(resp.Body, resolveUpstreamResponseReadLimit(s.cfg))
	if err != nil {
		return nil, err
	}
	result := grokParseVideoResultBody(bodyBytes, req.RequestedModel, upstreamModel)
	if result == nil {
		result = &grokVideoResult{
			Model:         req.RequestedModel,
			UpstreamModel: upstreamModel,
			Provider:      "grok",
		}
	}
	if strings.TrimSpace(result.RequestID) == "" && strings.TrimSpace(result.URL) == "" {
		msg := "Grok video upstream returned no request_id or final url"
		s.writeGrokVideoError(c, http.StatusBadGateway, "upstream_error", msg)
		s.logGrokVideoWorkflow(req, result, GrokRouteModeAPIKey, time.Since(startTime), 0, "invalid_create_response")
		return nil, fmt.Errorf("%s", msg)
	}
	if grokIsVideoSuccessStatus(result.Status) && strings.TrimSpace(result.URL) != "" {
		s.logGrokVideoWorkflow(req, result, GrokRouteModeAPIKey, time.Since(startTime), 0, result.Status)
		return result, nil
	}

	pollCtx, cancel := context.WithTimeout(ctx, s.grokVideoWaitTimeout())
	defer cancel()
	rounds := 0
	for {
		if err := pollCtx.Err(); err != nil {
			s.writeGrokVideoError(c, http.StatusGatewayTimeout, "upstream_timeout", "Grok video generation timed out")
			s.logGrokVideoWorkflow(req, result, GrokRouteModeAPIKey, time.Since(startTime), rounds, "timeout")
			return nil, fmt.Errorf("grok video generation timed out")
		}
		rounds++
		polled, pollErr := s.fetchAPIKeyVideoStatusOnce(pollCtx, c, account, result.RequestID, req.RequestedModel, upstreamModel)
		if pollErr != nil {
			s.logGrokVideoWorkflow(req, result, GrokRouteModeAPIKey, time.Since(startTime), rounds, "poll_failed")
			return nil, pollErr
		}
		if polled != nil {
			result = polled
		}
		if result != nil && grokIsVideoSuccessStatus(result.Status) && strings.TrimSpace(result.URL) != "" {
			s.logGrokVideoWorkflow(req, result, GrokRouteModeAPIKey, time.Since(startTime), rounds, result.Status)
			return result, nil
		}
		if result != nil && grokIsVideoFailureStatus(result.Status) {
			message := firstNonEmptyString(result.Status, "failed")
			s.writeGrokVideoError(c, http.StatusBadGateway, "upstream_error", "Grok video generation failed: "+message)
			s.logGrokVideoWorkflow(req, result, GrokRouteModeAPIKey, time.Since(startTime), rounds, result.Status)
			return nil, fmt.Errorf("grok video generation failed: %s", message)
		}
		if err := sleepWithContext(pollCtx, s.grokVideoPollInterval()); err != nil {
			s.writeGrokVideoError(c, http.StatusGatewayTimeout, "upstream_timeout", "Grok video generation timed out")
			s.logGrokVideoWorkflow(req, result, GrokRouteModeAPIKey, time.Since(startTime), rounds, "timeout")
			return nil, fmt.Errorf("grok video generation timed out")
		}
	}
}

func (s *GrokGatewayService) fetchAPIKeyVideoStatusOnce(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	requestID string,
	requestedModel string,
	upstreamModel string,
) (*grokVideoResult, error) {
	resp, err := s.doAPIKeyRequest(ctx, c, account, http.MethodGet, "/v1/videos/"+strings.TrimSpace(requestID), nil)
	if err != nil {
		s.writeGrokVideoError(c, http.StatusBadGateway, "upstream_error", sanitizeUpstreamErrorMessage(err.Error()))
		return nil, fmt.Errorf("grok video status request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	bodyBytes, readErr := readUpstreamResponseBodyLimited(resp.Body, resolveUpstreamResponseReadLimit(s.cfg))
	if readErr != nil {
		return nil, readErr
	}
	if resp.StatusCode >= 400 {
		s.writeJSONResponse(c, resp, bodyBytes)
		return nil, fmt.Errorf("grok video status request failed: %d", resp.StatusCode)
	}
	return grokParseVideoResultBody(bodyBytes, requestedModel, upstreamModel), nil
}

func (s *GrokGatewayService) runSSOGrokVideoWorkflow(ctx context.Context, c *gin.Context, account *Account, req *grokVideoWorkflowRequest) (*grokVideoResult, error) {
	validationBody, _ := json.Marshal(map[string]any{
		"duration_seconds": req.Seconds,
		"resolution":       req.Resolution,
	})
	validation, err := s.validateSSORequest(account, req.RequestedModel, "video", validationBody)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"type": "invalid_request_error", "message": err.Error()}})
		return nil, err
	}

	extraPayload := map[string]any{
		"media_type":       "video",
		"aspect_ratio":     req.AspectRatio,
		"resolution":       req.Resolution,
		"duration_seconds": req.Seconds,
	}
	if strings.TrimSpace(req.ImageURL) != "" {
		extraPayload["reference_image"] = strings.TrimSpace(req.ImageURL)
	}
	if strings.TrimSpace(req.VideoURL) != "" {
		extraPayload["reference_video"] = strings.TrimSpace(req.VideoURL)
	}

	startTime := time.Now()
	exec, err := s.executeSSOReverseRequest(ctx, account, validation.MappedModel, req.ModeID, req.Prompt, extraPayload)
	if err != nil {
		return nil, err
	}
	requestID := grokEncodeReverseVideoRequestID(exec.ConversationID, exec.ResponseID)
	result := &grokVideoResult{
		RequestID:     requestID,
		Status:        "processing",
		Model:         req.RequestedModel,
		UpstreamModel: validation.MappedModel,
		Resolution:    req.Resolution,
		AspectRatio:   req.AspectRatio,
		Seconds:       req.Seconds,
		MimeType:      "video/mp4",
		Provider:      "grok",
	}
	if len(exec.VideoURLs) > 0 {
		result.Status = "completed"
		result.URL = firstMediaURL(exec.VideoURLs)
		result.CompletedAt = time.Now()
		s.logGrokVideoWorkflow(req, result, GrokRouteModeSSO, time.Since(startTime), 0, result.Status)
		return result, nil
	}
	if strings.TrimSpace(exec.ConversationID) == "" && strings.TrimSpace(exec.AssetID) == "" {
		msg := "Grok reverse video request did not return pollable identifiers"
		s.writeGrokVideoError(c, http.StatusBadGateway, "upstream_error", msg)
		s.logGrokVideoWorkflow(req, result, GrokRouteModeSSO, time.Since(startTime), 0, "invalid_create_response")
		return nil, fmt.Errorf("%s", msg)
	}

	pollCtx, cancel := context.WithTimeout(ctx, s.grokVideoWaitTimeout())
	defer cancel()
	rounds := 0
	for {
		if err := pollCtx.Err(); err != nil {
			s.writeGrokVideoError(c, http.StatusGatewayTimeout, "upstream_timeout", "Grok video generation timed out")
			s.logGrokVideoWorkflow(req, result, GrokRouteModeSSO, time.Since(startTime), rounds, "timeout")
			return nil, fmt.Errorf("grok video generation timed out")
		}
		rounds++
		polled, pollErr := s.fetchSSOVideoStatusOnce(c, account, exec, result.RequestID, req.RequestedModel, validation.MappedModel)
		if pollErr != nil {
			s.logGrokVideoWorkflow(req, result, GrokRouteModeSSO, time.Since(startTime), rounds, "poll_failed")
			return nil, pollErr
		}
		if polled != nil {
			result = polled
		}
		if result != nil && grokIsVideoSuccessStatus(result.Status) && strings.TrimSpace(result.URL) != "" {
			s.logGrokVideoWorkflow(req, result, GrokRouteModeSSO, time.Since(startTime), rounds, result.Status)
			return result, nil
		}
		if result != nil && grokIsVideoFailureStatus(result.Status) {
			message := firstNonEmptyString(result.Status, "failed")
			s.writeGrokVideoError(c, http.StatusBadGateway, "upstream_error", "Grok video generation failed: "+message)
			s.logGrokVideoWorkflow(req, result, GrokRouteModeSSO, time.Since(startTime), rounds, result.Status)
			return nil, fmt.Errorf("grok video generation failed: %s", message)
		}
		if err := sleepWithContext(pollCtx, s.grokVideoPollInterval()); err != nil {
			s.writeGrokVideoError(c, http.StatusGatewayTimeout, "upstream_timeout", "Grok video generation timed out")
			s.logGrokVideoWorkflow(req, result, GrokRouteModeSSO, time.Since(startTime), rounds, "timeout")
			return nil, fmt.Errorf("grok video generation timed out")
		}
	}
}

func (s *GrokGatewayService) fetchSSOVideoStatusOnce(
	c *gin.Context,
	account *Account,
	exec *grokReverseExecution,
	requestID string,
	requestedModel string,
	upstreamModel string,
) (*grokVideoResult, error) {
	if exec == nil {
		return nil, fmt.Errorf("missing grok reverse execution")
	}
	if strings.TrimSpace(exec.ConversationID) != "" {
		resp, err := s.reverseClient.ProbeConversation(account, exec.ConversationID, 20*time.Second)
		if err != nil {
			s.writeGrokVideoError(c, http.StatusBadGateway, "upstream_error", sanitizeUpstreamErrorMessage(err.Error()))
			return nil, fmt.Errorf("grok reverse conversation probe failed: %w", err)
		}
		bodyBytes, statusErr := s.readGrokReverseStatusBody(c, resp)
		if statusErr != nil {
			return nil, statusErr
		}
		result := grokParseVideoResultBody(bodyBytes, requestedModel, upstreamModel)
		if result == nil {
			result = &grokVideoResult{}
		}
		result.RequestID = requestID
		if result.URL != "" || grokIsVideoFailureStatus(result.Status) {
			return result, nil
		}
	}
	if strings.TrimSpace(exec.AssetID) != "" {
		resp, err := s.reverseClient.ProbeAsset(account, exec.AssetID, 20*time.Second)
		if err != nil {
			s.writeGrokVideoError(c, http.StatusBadGateway, "upstream_error", sanitizeUpstreamErrorMessage(err.Error()))
			return nil, fmt.Errorf("grok reverse asset probe failed: %w", err)
		}
		bodyBytes, statusErr := s.readGrokReverseStatusBody(c, resp)
		if statusErr != nil {
			return nil, statusErr
		}
		result := grokParseVideoResultBody(bodyBytes, requestedModel, upstreamModel)
		if result == nil {
			result = &grokVideoResult{}
		}
		result.RequestID = requestID
		return result, nil
	}
	return &grokVideoResult{
		RequestID:     requestID,
		Status:        "processing",
		Model:         requestedModel,
		UpstreamModel: upstreamModel,
		MimeType:      "video/mp4",
		Provider:      "grok",
	}, nil
}

func (s *GrokGatewayService) readGrokReverseStatusBody(c *gin.Context, resp *http.Response) ([]byte, error) {
	if resp == nil {
		return nil, fmt.Errorf("nil grok reverse response")
	}
	defer func() { _ = resp.Body.Close() }()
	bodyBytes, err := readUpstreamResponseBodyLimited(resp.Body, resolveUpstreamResponseReadLimit(s.cfg))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		s.writeJSONResponse(c, resp, bodyBytes)
		return nil, fmt.Errorf("grok reverse status request failed: %d", resp.StatusCode)
	}
	return bodyBytes, nil
}

func grokResolveVideoUpstreamModel(account *Account, requestedModel string) string {
	requestedModel = strings.TrimSpace(requestedModel)
	if requestedModel == "" {
		return GrokModelImagineVideo
	}
	mappedModel := requestedModel
	if account != nil {
		mappedModel = strings.TrimSpace(account.GetMappedModel(requestedModel))
	}
	if mappedModel == "" {
		mappedModel = requestedModel
	}
	if account != nil && account.IsGrokAPIKey() && mappedModel == NormalizeGrokPublicModelID(mappedModel) {
		if resolved := GrokAPIKeyResolvedUpstreamModel(mappedModel); resolved != "" {
			return resolved
		}
	}
	return mappedModel
}

func grokBuildAPIKeyVideoPayload(req *grokVideoWorkflowRequest, upstreamModel string) ([]byte, string, error) {
	if req == nil {
		return nil, "", fmt.Errorf("missing grok video request")
	}
	payload := map[string]any{
		"model":            strings.TrimSpace(upstreamModel),
		"prompt":           strings.TrimSpace(req.Prompt),
		"aspect_ratio":     strings.TrimSpace(req.AspectRatio),
		"resolution":       strings.TrimSpace(req.Resolution),
		"duration_seconds": req.Seconds,
	}
	endpoint := grokEndpointVideosGen
	if strings.TrimSpace(req.ImageURL) != "" {
		payload["image_url"] = strings.TrimSpace(req.ImageURL)
	}
	if strings.TrimSpace(req.VideoURL) != "" {
		endpoint = "/v1/videos/edits"
		payload["video_url"] = strings.TrimSpace(req.VideoURL)
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, "", fmt.Errorf("marshal grok video payload: %w", err)
	}
	return body, endpoint, nil
}

func grokParseVideoResultBody(body []byte, requestedModel string, upstreamModel string) *grokVideoResult {
	if len(body) == 0 {
		return nil
	}
	url := strings.TrimSpace(firstNonEmptyString(
		gjson.GetBytes(body, "video.url").String(),
		gjson.GetBytes(body, "result.url").String(),
		gjson.GetBytes(body, "url").String(),
		firstMediaURL(grokCollectURLsFromJSON(body, true)),
	))
	status := grokCanonicalVideoStatus(firstNonEmptyString(
		gjson.GetBytes(body, "status").String(),
		gjson.GetBytes(body, "video.status").String(),
		gjson.GetBytes(body, "result.status").String(),
	), url != "")
	model := strings.TrimSpace(firstNonEmptyString(
		gjson.GetBytes(body, "model").String(),
		gjson.GetBytes(body, "video.model").String(),
		gjson.GetBytes(body, "result.model").String(),
		requestedModel,
	))
	publicModel := NormalizeGrokPublicModelID(model)
	if publicModel == "" {
		publicModel = model
	}
	return &grokVideoResult{
		RequestID: strings.TrimSpace(firstNonEmptyString(
			gjson.GetBytes(body, "request_id").String(),
			gjson.GetBytes(body, "id").String(),
			gjson.GetBytes(body, "video.request_id").String(),
		)),
		Status:        status,
		Model:         publicModel,
		UpstreamModel: strings.TrimSpace(firstNonEmptyString(upstreamModel, model)),
		URL:           url,
		ThumbnailURL: strings.TrimSpace(firstNonEmptyString(
			gjson.GetBytes(body, "video.thumbnail_url").String(),
			gjson.GetBytes(body, "thumbnail_url").String(),
		)),
		Resolution: strings.TrimSpace(firstNonEmptyString(
			gjson.GetBytes(body, "video.resolution").String(),
			gjson.GetBytes(body, "video.resolution_name").String(),
			gjson.GetBytes(body, "resolution").String(),
		)),
		AspectRatio: strings.TrimSpace(firstNonEmptyString(
			gjson.GetBytes(body, "video.aspect_ratio").String(),
			gjson.GetBytes(body, "aspect_ratio").String(),
		)),
		Seconds: grokExtractPositiveJSONInt(body,
			"video.duration_seconds",
			"duration_seconds",
			"video.duration",
			"duration",
			"seconds",
		),
		MimeType: strings.TrimSpace(firstNonEmptyString(
			gjson.GetBytes(body, "video.mime_type").String(),
			gjson.GetBytes(body, "mime_type").String(),
		)),
		Provider:    "grok",
		CompletedAt: time.Now(),
	}
}

func grokCanonicalVideoStatus(status string, hasURL bool) string {
	normalized := strings.TrimSpace(strings.ToLower(status))
	switch normalized {
	case "done", "completed", "succeeded", "success":
		return "completed"
	case "queued", "pending", "processing", "running", "in_progress", "submitted":
		return "processing"
	case "failed", "error", "cancelled", "canceled", "expired":
		if normalized == "canceled" {
			return "cancelled"
		}
		return normalized
	default:
		if hasURL {
			return "completed"
		}
		if normalized == "" {
			return "processing"
		}
		return normalized
	}
}

func grokIsVideoSuccessStatus(status string) bool {
	return strings.TrimSpace(strings.ToLower(status)) == "completed"
}

func grokIsVideoFailureStatus(status string) bool {
	switch strings.TrimSpace(strings.ToLower(status)) {
	case "failed", "error", "cancelled", "expired":
		return true
	default:
		return false
	}
}

func grokValidateVideoWorkflowRequest(req *grokVideoWorkflowRequest) error {
	if req == nil {
		return fmt.Errorf("video request is required")
	}
	if strings.TrimSpace(req.RequestedModel) == "" {
		req.RequestedModel = GrokModelImagineVideo
	}
	if !grokIsVideoRequestModel(req.RequestedModel) {
		return fmt.Errorf("model %s is not a Grok video model", req.RequestedModel)
	}
	if strings.TrimSpace(req.Prompt) == "" {
		return fmt.Errorf("prompt is required")
	}
	return nil
}

func grokVideoForwardResult(req *grokVideoWorkflowRequest, result *grokVideoResult, stream bool, firstTokenMs *int, duration time.Duration) *ForwardResult {
	model := ""
	if req != nil {
		model = strings.TrimSpace(req.RequestedModel)
	}
	if model == "" && result != nil {
		model = strings.TrimSpace(result.Model)
	}
	upstreamModel := ""
	requestID := model
	mediaURL := ""
	if result != nil {
		upstreamModel = strings.TrimSpace(result.UpstreamModel)
		requestID = firstNonEmptyString(strings.TrimSpace(result.RequestID), requestID)
		mediaURL = strings.TrimSpace(result.URL)
	}
	return &ForwardResult{
		RequestID:     requestID,
		Model:         model,
		UpstreamModel: upstreamModel,
		Stream:        stream,
		Duration:      duration,
		FirstTokenMs:  firstTokenMs,
		MediaType:     "video",
		MediaURL:      mediaURL,
	}
}

func (s *GrokGatewayService) writeGrokVideoChatStream(c *gin.Context, model string, result *grokVideoResult, startTime time.Time) (*int, error) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("streaming not supported")
	}

	state := apicompat.NewResponsesEventToChatState()
	state.Model = model

	var firstTokenMs *int
	writeChunks := func(chunks []apicompat.ChatCompletionsChunk) error {
		for _, chunk := range chunks {
			if firstTokenMs == nil {
				ms := int(time.Since(startTime).Milliseconds())
				firstTokenMs = &ms
			}
			sse, err := apicompat.ChatChunkToSSE(chunk)
			if err != nil {
				return err
			}
			if _, err := io.WriteString(c.Writer, sse); err != nil {
				return err
			}
			flusher.Flush()
		}
		return nil
	}

	events := []*apicompat.ResponsesStreamEvent{
		{
			Type: "response.created",
			Response: &apicompat.ResponsesResponse{
				ID:     strings.TrimSpace(result.RequestID),
				Object: "response",
				Model:  model,
				Status: "in_progress",
			},
		},
		{Type: "response.output_video.added", OutputIndex: 0, Video: result.MediaVideo()},
		{Type: "response.output_video.done", OutputIndex: 0, Video: result.MediaVideo()},
		{Type: "response.completed", Response: grokBuildVideoResponsesResponse(result, model)},
	}
	for _, event := range events {
		if err := writeChunks(apicompat.ResponsesEventToChatChunks(event, state)); err != nil {
			return firstTokenMs, nil
		}
	}
	if finalChunks := apicompat.FinalizeResponsesChatStream(state); len(finalChunks) > 0 {
		if err := writeChunks(finalChunks); err != nil {
			return firstTokenMs, nil
		}
	}
	_, _ = io.WriteString(c.Writer, "data: [DONE]\n\n")
	flusher.Flush()
	return firstTokenMs, nil
}

func (s *GrokGatewayService) writeGrokVideoResponsesStream(c *gin.Context, model string, result *grokVideoResult, startTime time.Time) (*int, error) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("streaming not supported")
	}

	var firstTokenMs *int
	writeEvent := func(payload any) error {
		if firstTokenMs == nil {
			ms := int(time.Since(startTime).Milliseconds())
			firstTokenMs = &ms
		}
		raw, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		if _, err := io.WriteString(c.Writer, "data: "+string(raw)+"\n\n"); err != nil {
			return err
		}
		flusher.Flush()
		return nil
	}

	events := []any{
		apicompat.ResponsesStreamEvent{
			Type: "response.created",
			Response: &apicompat.ResponsesResponse{
				ID:     strings.TrimSpace(result.RequestID),
				Object: "response",
				Model:  model,
				Status: "in_progress",
			},
		},
		apicompat.ResponsesStreamEvent{Type: "response.output_video.added", OutputIndex: 0, Video: result.MediaVideo()},
		apicompat.ResponsesStreamEvent{Type: "response.output_video.done", OutputIndex: 0, Video: result.MediaVideo()},
		apicompat.ResponsesStreamEvent{Type: "response.completed", Response: grokBuildVideoResponsesResponse(result, model)},
	}
	for _, event := range events {
		if err := writeEvent(event); err != nil {
			return firstTokenMs, nil
		}
	}
	_, _ = io.WriteString(c.Writer, "data: [DONE]\n\n")
	flusher.Flush()
	return firstTokenMs, nil
}

func (s *GrokGatewayService) writeGrokVideoError(c *gin.Context, status int, errType string, message string) {
	if c == nil || c.Writer == nil || c.Writer.Written() {
		return
	}
	c.JSON(status, gin.H{"error": gin.H{"type": errType, "message": message}})
}

func (s *GrokGatewayService) grokVideoPollInterval() time.Duration {
	if s != nil && s.cfg != nil && s.cfg.Gateway.GrokVideoPollIntervalSeconds > 0 {
		return time.Duration(s.cfg.Gateway.GrokVideoPollIntervalSeconds) * time.Second
	}
	return 2 * time.Second
}

func (s *GrokGatewayService) grokVideoWaitTimeout() time.Duration {
	if s != nil && s.cfg != nil && s.cfg.Gateway.GrokVideoWaitTimeoutSeconds > 0 {
		return time.Duration(s.cfg.Gateway.GrokVideoWaitTimeoutSeconds) * time.Second
	}
	return 180 * time.Second
}

func (s *GrokGatewayService) logGrokVideoWorkflow(
	req *grokVideoWorkflowRequest,
	result *grokVideoResult,
	routeMode string,
	duration time.Duration,
	pollRounds int,
	finalStatus string,
) {
	requestedModel := ""
	canonicalModel := ""
	upstreamModel := ""
	entryPoint := ""
	requestID := ""
	if req != nil {
		requestedModel = strings.TrimSpace(req.RequestedModel)
		canonicalModel = NormalizeGrokPublicModelID(req.RequestedModel)
		entryPoint = strings.TrimSpace(req.EntryPoint)
	}
	if result != nil {
		upstreamModel = strings.TrimSpace(result.UpstreamModel)
		requestID = strings.TrimSpace(result.RequestID)
		if canonicalModel == "" {
			canonicalModel = NormalizeGrokPublicModelID(result.Model)
		}
	}
	logger.L().Info("grok.video_workflow",
		zap.String("entrypoint", entryPoint),
		zap.String("route_mode", strings.TrimSpace(routeMode)),
		zap.String("requested_model", requestedModel),
		zap.String("canonical_model", canonicalModel),
		zap.String("upstream_model", upstreamModel),
		zap.Int("poll_rounds", pollRounds),
		zap.Int64("latency_ms", duration.Milliseconds()),
		zap.String("final_status", strings.TrimSpace(finalStatus)),
		zap.String("request_id", requestID),
	)
}
