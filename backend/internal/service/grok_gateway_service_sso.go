package service

import (
	"bufio"
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
	"github.com/google/uuid"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

const (
	grokQuickImageWaitTimeout  = 12 * time.Second
	grokQuickImagePollInterval = 800 * time.Millisecond
	grokReverseVideoRequestTag = "grokrev"
)

type grokReverseExecution struct {
	ResponseID     string
	ConversationID string
	AssetID        string
	Model          string
	Message        string
	Tokens         []string
	ImageURLs      []string
	VideoURLs      []string
	RawLines       [][]byte
}

type grokSSOValidation struct {
	MappedModel string
}

func (s *GrokGatewayService) forwardSSOChatCompletions(ctx context.Context, c *gin.Context, account *Account, body []byte) (*GrokGatewayForwardResult, error) {
	var req apicompat.ChatCompletionsRequest
	if err := json.Unmarshal(body, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"type": "invalid_request_error", "message": "Failed to parse request body"}})
		return nil, fmt.Errorf("parse grok chat request: %w", err)
	}
	prompt := grokChatPrompt(req.Messages)
	modeID := strings.TrimSpace(gjson.GetBytes(body, "mode_id").String())
	validation, err := s.validateSSORequest(account, req.Model, "chat", body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"type": "invalid_request_error", "message": err.Error()}})
		return nil, err
	}
	exec, err := s.executeSSOReverseRequest(ctx, account, validation.MappedModel, modeID, prompt, nil)
	if err != nil {
		return nil, err
	}
	exec.Message = grokBuildChatLikeOutput(exec)
	if grokShouldWaitQuickImage(modeID, validation.MappedModel) {
		exec.ImageURLs = s.waitForQuickImageFinal(account, exec)
		exec.Message = grokBuildChatLikeOutput(exec)
	}

	startTime := time.Now()
	if req.Stream {
		firstTokenMs, writeErr := s.writeSSOChatStream(c, req.Model, exec, startTime)
		if writeErr != nil {
			return nil, writeErr
		}
		return &GrokGatewayForwardResult{
			Result: &ForwardResult{
				RequestID:     exec.ResponseID,
				Model:         req.Model,
				UpstreamModel: validation.MappedModel,
				Stream:        true,
				Duration:      time.Since(startTime),
				FirstTokenMs:  firstTokenMs,
				MediaType:     grokMediaTypeForExec(exec),
				ImageCount:    len(exec.ImageURLs),
				MediaURL:      firstMediaURL(exec.ImageURLs),
			},
			RouteMode:         GrokRouteModeSSO,
			Endpoint:          grokEndpointChatCompletions,
			MediaType:         grokMediaTypeForExec(exec),
			UpstreamRequestID: exec.ResponseID,
		}, nil
	}

	resp := grokBuildResponsesResponse(exec, req.Model)
	chatResp := apicompat.ResponsesToChatCompletions(resp, req.Model)
	c.JSON(http.StatusOK, chatResp)
	return &GrokGatewayForwardResult{
		Result: &ForwardResult{
			RequestID:     exec.ResponseID,
			Model:         req.Model,
			UpstreamModel: validation.MappedModel,
			Stream:        false,
			Duration:      time.Since(startTime),
			MediaType:     grokMediaTypeForExec(exec),
			ImageCount:    len(exec.ImageURLs),
			MediaURL:      firstMediaURL(exec.ImageURLs),
		},
		RouteMode:         GrokRouteModeSSO,
		Endpoint:          grokEndpointChatCompletions,
		MediaType:         grokMediaTypeForExec(exec),
		UpstreamRequestID: exec.ResponseID,
	}, nil
}

func (s *GrokGatewayService) forwardSSOResponses(ctx context.Context, c *gin.Context, account *Account, body []byte, method string, subpath string) (*GrokGatewayForwardResult, error) {
	if strings.ToUpper(strings.TrimSpace(method)) != http.MethodPost {
		c.JSON(http.StatusNotImplemented, gin.H{"error": gin.H{"type": "invalid_request_error", "message": "Grok SSO responses subresources are not supported"}})
		return nil, fmt.Errorf("grok sso responses subresources are not supported")
	}
	var req apicompat.ResponsesRequest
	if err := json.Unmarshal(body, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"type": "invalid_request_error", "message": "Failed to parse request body"}})
		return nil, fmt.Errorf("parse grok responses request: %w", err)
	}
	prompt := grokResponsesPrompt(req.Input)
	modeID := strings.TrimSpace(gjson.GetBytes(body, "mode_id").String())
	validation, err := s.validateSSORequest(account, req.Model, "chat", body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"type": "invalid_request_error", "message": err.Error()}})
		return nil, err
	}
	exec, err := s.executeSSOReverseRequest(ctx, account, validation.MappedModel, modeID, prompt, nil)
	if err != nil {
		return nil, err
	}
	exec.Message = grokBuildChatLikeOutput(exec)
	if grokShouldWaitQuickImage(modeID, validation.MappedModel) {
		exec.ImageURLs = s.waitForQuickImageFinal(account, exec)
		exec.Message = grokBuildChatLikeOutput(exec)
	}

	startTime := time.Now()
	if req.Stream {
		firstTokenMs, writeErr := s.writeSSOResponsesStream(c, req.Model, exec, startTime)
		if writeErr != nil {
			return nil, writeErr
		}
		return &GrokGatewayForwardResult{
			Result: &ForwardResult{
				RequestID:     exec.ResponseID,
				Model:         req.Model,
				UpstreamModel: validation.MappedModel,
				Stream:        true,
				Duration:      time.Since(startTime),
				FirstTokenMs:  firstTokenMs,
				MediaType:     grokMediaTypeForExec(exec),
				ImageCount:    len(exec.ImageURLs),
				MediaURL:      firstMediaURL(exec.ImageURLs),
			},
			RouteMode:         GrokRouteModeSSO,
			Endpoint:          grokEndpointResponses,
			MediaType:         grokMediaTypeForExec(exec),
			UpstreamRequestID: exec.ResponseID,
		}, nil
	}

	resp := grokBuildResponsesResponse(exec, req.Model)
	c.JSON(http.StatusOK, resp)
	return &GrokGatewayForwardResult{
		Result: &ForwardResult{
			RequestID:     exec.ResponseID,
			Model:         req.Model,
			UpstreamModel: validation.MappedModel,
			Stream:        false,
			Duration:      time.Since(startTime),
			MediaType:     grokMediaTypeForExec(exec),
			ImageCount:    len(exec.ImageURLs),
			MediaURL:      firstMediaURL(exec.ImageURLs),
		},
		RouteMode:         GrokRouteModeSSO,
		Endpoint:          grokEndpointResponses,
		MediaType:         grokMediaTypeForExec(exec),
		UpstreamRequestID: exec.ResponseID,
	}, nil
}

func (s *GrokGatewayService) forwardSSOImagesGeneration(ctx context.Context, c *gin.Context, account *Account, body []byte) (*GrokGatewayForwardResult, error) {
	return s.forwardSSOImageWorkflow(ctx, c, account, body, grokEndpointImagesGen)
}

func (s *GrokGatewayService) forwardSSOImagesEdits(ctx context.Context, c *gin.Context, account *Account, body []byte) (*GrokGatewayForwardResult, error) {
	return s.forwardSSOImageWorkflow(ctx, c, account, body, grokEndpointImagesEdits)
}

func (s *GrokGatewayService) forwardSSOImageWorkflow(ctx context.Context, c *gin.Context, account *Account, body []byte, endpoint string) (*GrokGatewayForwardResult, error) {
	reqModel := strings.TrimSpace(gjson.GetBytes(body, "model").String())
	if reqModel == "" {
		reqModel = "grok-imagine-image"
	}
	prompt := strings.TrimSpace(gjson.GetBytes(body, "prompt").String())
	modeID := strings.TrimSpace(gjson.GetBytes(body, "mode_id").String())
	validation, err := s.validateSSORequest(account, reqModel, "image", body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"type": "invalid_request_error", "message": err.Error()}})
		return nil, err
	}
	extraPayload := map[string]any{}
	if endpoint == grokEndpointImagesEdits {
		if imageURL := strings.TrimSpace(firstNonEmptyString(
			gjson.GetBytes(body, "image").String(),
			gjson.GetBytes(body, "image_url").String(),
			gjson.GetBytes(body, "input_image").String(),
		)); imageURL != "" {
			extraPayload["reference_image"] = imageURL
		}
	}
	exec, err := s.executeSSOReverseRequest(ctx, account, validation.MappedModel, modeID, prompt, extraPayload)
	if err != nil {
		return nil, err
	}
	exec.ImageURLs = s.waitForQuickImageFinal(account, exec)
	c.JSON(http.StatusOK, gin.H{
		"created": time.Now().Unix(),
		"data":    grokImageData(exec.ImageURLs),
	})
	return &GrokGatewayForwardResult{
		Result: &ForwardResult{
			RequestID:     exec.ResponseID,
			Model:         reqModel,
			UpstreamModel: validation.MappedModel,
			Stream:        false,
			MediaType:     "image",
			ImageCount:    len(exec.ImageURLs),
			ImageSize:     strings.TrimSpace(gjson.GetBytes(body, "size").String()),
			MediaURL:      firstMediaURL(exec.ImageURLs),
		},
		RouteMode:         GrokRouteModeSSO,
		Endpoint:          endpoint,
		MediaType:         "image",
		UpstreamRequestID: exec.ResponseID,
	}, nil
}

func (s *GrokGatewayService) forwardSSOVideosGeneration(ctx context.Context, c *gin.Context, account *Account, body []byte) (*GrokGatewayForwardResult, error) {
	reqModel := strings.TrimSpace(gjson.GetBytes(body, "model").String())
	if reqModel == "" {
		reqModel = "grok-imagine-video"
	}
	prompt := strings.TrimSpace(gjson.GetBytes(body, "prompt").String())
	modeID := strings.TrimSpace(gjson.GetBytes(body, "mode_id").String())
	validation, err := s.validateSSORequest(account, reqModel, "video", body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"type": "invalid_request_error", "message": err.Error()}})
		return nil, err
	}
	exec, err := s.executeSSOReverseRequest(ctx, account, validation.MappedModel, modeID, prompt, map[string]any{"media_type": "video"})
	if err != nil {
		return nil, err
	}
	requestID := grokEncodeReverseVideoRequestID(exec.ConversationID, exec.ResponseID)
	c.JSON(http.StatusOK, gin.H{
		"request_id": requestID,
		"status":     "processing",
	})
	return &GrokGatewayForwardResult{
		Result: &ForwardResult{
			RequestID:     requestID,
			Model:         reqModel,
			UpstreamModel: validation.MappedModel,
			Stream:        false,
			MediaType:     "video",
		},
		RouteMode:         GrokRouteModeSSO,
		Endpoint:          grokEndpointVideosGen,
		MediaType:         "video",
		UpstreamRequestID: exec.ResponseID,
		SkipUsageRecord:   true,
	}, nil
}

func (s *GrokGatewayService) forwardSSOVideoStatus(ctx context.Context, c *gin.Context, account *Account, requestID string) (*GrokGatewayForwardResult, error) {
	conversationID, responseID := grokDecodeReverseVideoRequestID(requestID)
	if conversationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"type": "invalid_request_error", "message": "invalid grok reverse video request_id"}})
		return nil, fmt.Errorf("invalid grok reverse video request id")
	}
	resp, err := s.reverseClient.ProbeConversation(account, conversationID, 20*time.Second)
	if err != nil {
		safeErr := sanitizeUpstreamErrorMessage(err.Error())
		setOpsUpstreamError(c, 0, safeErr, "")
		appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
			Platform:           PlatformGrok,
			AccountID:          account.ID,
			AccountName:        account.Name,
			UpstreamStatusCode: 0,
			Kind:               "request_error",
			Message:            safeErr,
		})
		return nil, fmt.Errorf("grok reverse conversation probe failed: %s", safeErr)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		return nil, s.handleHTTPError(ctx, resp, c, account, GrokRouteModeSSO)
	}
	bodyBytes, err := readUpstreamResponseBodyLimited(resp.Body, resolveUpstreamResponseReadLimit(s.cfg))
	if err != nil {
		return nil, err
	}

	videoURLs := grokCollectURLsFromJSON(bodyBytes, true)
	status := strings.ToLower(strings.TrimSpace(firstNonEmptyString(
		gjson.GetBytes(bodyBytes, "status").String(),
		gjson.GetBytes(bodyBytes, "task.status").String(),
		gjson.GetBytes(bodyBytes, "taskResult.status").String(),
	)))
	if status == "" {
		if len(videoURLs) > 0 {
			status = "completed"
		} else {
			status = "processing"
		}
	}
	model := strings.TrimSpace(firstNonEmptyString(
		gjson.GetBytes(bodyBytes, "model").String(),
		gjson.GetBytes(bodyBytes, "task.model").String(),
	))
	c.JSON(http.StatusOK, gin.H{
		"request_id": requestID,
		"status":     status,
		"data":       grokVideoData(videoURLs),
	})

	result := &GrokGatewayForwardResult{
		Result: &ForwardResult{
			RequestID:     "grok-video:" + requestID,
			Model:         model,
			UpstreamModel: model,
			Stream:        false,
			MediaType:     "video",
			MediaURL:      firstMediaURL(videoURLs),
		},
		RouteMode:         GrokRouteModeSSO,
		Endpoint:          grokEndpointVideosStatus,
		MediaType:         "video",
		UpstreamRequestID: responseID,
		SkipUsageRecord:   !grokIsTerminalVideoStatus(status),
	}
	if status == "failed" || status == "error" || status == "cancelled" {
		result.SkipUsageRecord = true
		result.FailedUsage = &GrokFailedUsageInfo{
			RequestID:     "grok-video:" + requestID,
			Model:         model,
			UpstreamModel: model,
			ErrorCode:     status,
			ErrorMessage:  strings.TrimSpace(firstNonEmptyString(gjson.GetBytes(bodyBytes, "error.message").String(), gjson.GetBytes(bodyBytes, "message").String())),
			MediaType:     "video",
		}
	}
	return result, nil
}

func (s *GrokGatewayService) validateSSORequest(account *Account, requestedModel string, mediaType string, body []byte) (*grokSSOValidation, error) {
	if account == nil || !account.IsGrokSSO() {
		return nil, fmt.Errorf("grok sso account is required")
	}
	requestedModel = strings.TrimSpace(requestedModel)
	if requestedModel == "" {
		return nil, fmt.Errorf("model is required")
	}
	mappedModel, matched := account.ResolveMappedModel(requestedModel)
	if len(account.GetModelMapping()) > 0 && !matched {
		return nil, fmt.Errorf("model %s is not enabled for this Grok account", requestedModel)
	}
	if mappedModel == "" {
		mappedModel = requestedModel
	}
	caps := ResolveGrokCapabilities(account.Extra)
	if IsGrokHeavyModel(mappedModel) && !caps.AllowHeavyModel {
		return nil, fmt.Errorf("model %s requires Grok heavy tier", requestedModel)
	}
	if mediaType == "video" {
		durationSeconds := int(firstPositiveInt64(
			gjson.GetBytes(body, "duration_seconds").Int(),
			gjson.GetBytes(body, "duration").Int(),
		))
		if caps.VideoMaxDurationSeconds > 0 && durationSeconds > caps.VideoMaxDurationSeconds {
			return nil, fmt.Errorf("video duration %ds exceeds tier limit %ds", durationSeconds, caps.VideoMaxDurationSeconds)
		}
		resolution := grokNormalizeVideoResolution(firstNonEmptyString(
			gjson.GetBytes(body, "resolution").String(),
			gjson.GetBytes(body, "size").String(),
		))
		if resolution != "" && grokResolutionRank(resolution) > grokResolutionRank(caps.VideoMaxResolution) {
			return nil, fmt.Errorf("video resolution %s exceeds tier limit %s", resolution, caps.VideoMaxResolution)
		}
	}
	return &grokSSOValidation{MappedModel: mappedModel}, nil
}

func (s *GrokGatewayService) executeSSOReverseRequest(ctx context.Context, account *Account, mappedModel string, modeID string, prompt string, extra map[string]any) (*grokReverseExecution, error) {
	payload := map[string]any{
		"temporary": true,
		"message":   prompt,
		"modelName": mappedModel,
	}
	if modeID = strings.TrimSpace(modeID); modeID != "" {
		payload["mode_id"] = modeID
	}
	for key, value := range extra {
		payload[key] = value
	}
	s.logGrokPromptDiagnostics(prompt, mappedModel, modeID)

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal grok reverse payload: %w", err)
	}
	resp, err := s.reverseClient.DoAppChat(ctx, account, body)
	if err != nil {
		return nil, fmt.Errorf("grok reverse upstream request failed: %s", sanitizeUpstreamErrorMessage(err.Error()))
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		bodyBytes, _ := readUpstreamResponseBodyLimited(resp.Body, resolveUpstreamResponseReadLimit(s.cfg))
		msg := sanitizeUpstreamErrorMessage(extractUpstreamErrorMessage(bodyBytes))
		if msg == "" {
			msg = fmt.Sprintf("grok reverse upstream error: %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("%s", msg)
	}
	return grokParseReverseStream(resp.Body, mappedModel)
}

func grokParseReverseStream(reader io.Reader, mappedModel string) (*grokReverseExecution, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), defaultMaxLineSize)
	exec := &grokReverseExecution{Model: mappedModel}
	var messageBuilder strings.Builder
	seenImageURLs := make(map[string]struct{})
	seenVideoURLs := make(map[string]struct{})

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if data, ok := extractOpenAISSEDataLine(line); ok {
			line = strings.TrimSpace(data)
		}
		if line == "" || line == "[DONE]" {
			continue
		}
		raw := []byte(line)
		exec.RawLines = append(exec.RawLines, append([]byte(nil), raw...))
		if errMsg := strings.TrimSpace(firstNonEmptyString(
			gjson.GetBytes(raw, "error.message").String(),
			gjson.GetBytes(raw, "result.error.message").String(),
		)); errMsg != "" {
			return nil, fmt.Errorf("%s", sanitizeUpstreamErrorMessage(errMsg))
		}
		if token := strings.TrimSpace(gjson.GetBytes(raw, "result.response.token").String()); token != "" {
			exec.Tokens = append(exec.Tokens, token)
			messageBuilder.WriteString(token)
		}
		exec.ResponseID = firstNonEmptyString(exec.ResponseID, gjson.GetBytes(raw, "result.response.responseId").String(), gjson.GetBytes(raw, "result.response.id").String())
		exec.ConversationID = firstNonEmptyString(exec.ConversationID, gjson.GetBytes(raw, "result.response.conversationId").String())
		exec.AssetID = firstNonEmptyString(exec.AssetID,
			gjson.GetBytes(raw, "result.response.assetId").String(),
			gjson.GetBytes(raw, "result.response.generatedAssetId").String(),
			gjson.GetBytes(raw, "result.response.modelResponse.assetId").String(),
		)
		if message := strings.TrimSpace(firstNonEmptyString(
			gjson.GetBytes(raw, "result.response.modelResponse.message").String(),
			gjson.GetBytes(raw, "result.response.modelResponse.response").String(),
			gjson.GetBytes(raw, "result.response.message").String(),
		)); message != "" {
			exec.Message = message
		}
		for _, url := range grokCollectURLsFromJSON(raw, false) {
			if grokLooksLikeVideoURL(url) {
				if _, ok := seenVideoURLs[url]; ok {
					continue
				}
				seenVideoURLs[url] = struct{}{}
				exec.VideoURLs = append(exec.VideoURLs, url)
				continue
			}
			if _, ok := seenImageURLs[url]; ok {
				continue
			}
			seenImageURLs[url] = struct{}{}
			exec.ImageURLs = append(exec.ImageURLs, url)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if exec.Message == "" {
		exec.Message = strings.TrimSpace(messageBuilder.String())
	}
	if exec.ResponseID == "" {
		exec.ResponseID = "resp_" + uuid.NewString()
	}
	return exec, nil
}

func (s *GrokGatewayService) waitForQuickImageFinal(account *Account, exec *grokReverseExecution) []string {
	if exec == nil {
		return nil
	}
	if urls := grokFilterPreviewURLs(exec.ImageURLs); len(urls) > 0 {
		return urls
	}
	deadline := time.Now().Add(grokQuickImageWaitTimeout)
	if strings.TrimSpace(exec.ConversationID) == "" {
		if urls := s.probeGrokAssetURLs(account, firstNonEmptyString(exec.AssetID, exec.ResponseID), deadline); len(urls) > 0 {
			return urls
		}
		return nil
	}
	if urls := s.probeGrokConversationURLs(account, exec.ConversationID, deadline); len(urls) > 0 {
		return urls
	}
	if urls := s.probeGrokAssetURLs(account, firstNonEmptyString(exec.AssetID, exec.ResponseID), deadline); len(urls) > 0 {
		return urls
	}
	return nil
}

func (s *GrokGatewayService) probeGrokConversationURLs(account *Account, conversationID string, deadline time.Time) []string {
	for time.Now().Before(deadline) {
		resp, err := s.reverseClient.ProbeConversation(account, conversationID, 10*time.Second)
		if err == nil && resp != nil {
			bodyBytes, _ := readUpstreamResponseBodyLimited(resp.Body, resolveUpstreamResponseReadLimit(s.cfg))
			_ = resp.Body.Close()
			if resp.StatusCode < 400 {
				if urls := grokFilterPreviewURLs(grokCollectURLsFromJSON(bodyBytes, false)); len(urls) > 0 {
					return urls
				}
			}
		}
		time.Sleep(grokQuickImagePollInterval)
	}
	return nil
}

func (s *GrokGatewayService) probeGrokAssetURLs(account *Account, assetID string, deadline time.Time) []string {
	assetID = strings.TrimSpace(assetID)
	if assetID == "" {
		return nil
	}
	for time.Now().Before(deadline) {
		resp, err := s.reverseClient.ProbeAsset(account, assetID, 10*time.Second)
		if err == nil && resp != nil {
			bodyBytes, _ := readUpstreamResponseBodyLimited(resp.Body, resolveUpstreamResponseReadLimit(s.cfg))
			_ = resp.Body.Close()
			if resp.StatusCode < 400 {
				if urls := grokFilterPreviewURLs(grokCollectURLsFromJSON(bodyBytes, false)); len(urls) > 0 {
					return urls
				}
			}
		}
		time.Sleep(grokQuickImagePollInterval)
	}
	return nil
}

func (s *GrokGatewayService) writeSSOChatStream(c *gin.Context, model string, exec *grokReverseExecution, startTime time.Time) (*int, error) {
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
	created := apicompat.ResponsesStreamEvent{
		Type: "response.created",
		Response: &apicompat.ResponsesResponse{
			ID:     exec.ResponseID,
			Object: "response",
			Model:  model,
			Status: "in_progress",
		},
	}
	firstTokenMs := (*int)(nil)
	for _, chunk := range apicompat.ResponsesEventToChatChunks(&created, state) {
		sse, _ := apicompat.ChatChunkToSSE(chunk)
		if _, err := io.WriteString(c.Writer, sse); err != nil {
			return nil, nil
		}
	}
	for _, token := range grokStreamTokens(exec) {
		if firstTokenMs == nil {
			ms := int(time.Since(startTime).Milliseconds())
			firstTokenMs = &ms
		}
		event := apicompat.ResponsesStreamEvent{Type: "response.output_text.delta", OutputIndex: 0, Delta: token}
		for _, chunk := range apicompat.ResponsesEventToChatChunks(&event, state) {
			sse, _ := apicompat.ChatChunkToSSE(chunk)
			if _, err := io.WriteString(c.Writer, sse); err != nil {
				return firstTokenMs, nil
			}
		}
		flusher.Flush()
	}
	completed := apicompat.ResponsesStreamEvent{Type: "response.completed", Response: grokBuildResponsesResponse(exec, model)}
	for _, chunk := range apicompat.ResponsesEventToChatChunks(&completed, state) {
		sse, _ := apicompat.ChatChunkToSSE(chunk)
		if _, err := io.WriteString(c.Writer, sse); err != nil {
			return firstTokenMs, nil
		}
	}
	if finalChunks := apicompat.FinalizeResponsesChatStream(state); len(finalChunks) > 0 {
		for _, chunk := range finalChunks {
			sse, _ := apicompat.ChatChunkToSSE(chunk)
			if _, err := io.WriteString(c.Writer, sse); err != nil {
				return firstTokenMs, nil
			}
		}
	}
	_, _ = io.WriteString(c.Writer, "data: [DONE]\n\n")
	flusher.Flush()
	return firstTokenMs, nil
}

func (s *GrokGatewayService) writeSSOResponsesStream(c *gin.Context, model string, exec *grokReverseExecution, startTime time.Time) (*int, error) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("streaming not supported")
	}

	writeEvent := func(payload any) error {
		raw, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		_, err = io.WriteString(c.Writer, "data: "+string(raw)+"\n\n")
		if err == nil {
			flusher.Flush()
		}
		return err
	}

	if err := writeEvent(apicompat.ResponsesStreamEvent{
		Type: "response.created",
		Response: &apicompat.ResponsesResponse{ID: exec.ResponseID, Object: "response", Model: model, Status: "in_progress"},
	}); err != nil {
		return nil, nil
	}

	var firstTokenMs *int
	for _, token := range grokStreamTokens(exec) {
		if firstTokenMs == nil {
			ms := int(time.Since(startTime).Milliseconds())
			firstTokenMs = &ms
		}
		if err := writeEvent(apicompat.ResponsesStreamEvent{Type: "response.output_text.delta", OutputIndex: 0, Delta: token}); err != nil {
			return firstTokenMs, nil
		}
	}
	if err := writeEvent(apicompat.ResponsesStreamEvent{Type: "response.completed", Response: grokBuildResponsesResponse(exec, model)}); err != nil {
		return firstTokenMs, nil
	}
	_, _ = io.WriteString(c.Writer, "data: [DONE]\n\n")
	flusher.Flush()
	return firstTokenMs, nil
}

func grokBuildResponsesResponse(exec *grokReverseExecution, model string) *apicompat.ResponsesResponse {
	text := grokBuildChatLikeOutput(exec)
	return &apicompat.ResponsesResponse{
		ID:     exec.ResponseID,
		Object: "response",
		Model:  model,
		Status: "completed",
		Output: []apicompat.ResponsesOutput{{
			Type:   "message",
			ID:     "msg_" + exec.ResponseID,
			Role:   "assistant",
			Status: "completed",
			Content: []apicompat.ResponsesContentPart{{
				Type: "output_text",
				Text: text,
			}},
		}},
		Usage: &apicompat.ResponsesUsage{InputTokens: 0, OutputTokens: 0, TotalTokens: 0},
	}
}

func grokBuildChatLikeOutput(exec *grokReverseExecution) string {
	if exec == nil {
		return ""
	}
	if len(exec.ImageURLs) > 0 {
		return strings.Join(exec.ImageURLs, "\n")
	}
	if len(exec.VideoURLs) > 0 {
		return strings.Join(exec.VideoURLs, "\n")
	}
	if strings.TrimSpace(exec.Message) != "" {
		return strings.TrimSpace(exec.Message)
	}
	if len(exec.Tokens) > 0 {
		return strings.Join(exec.Tokens, "")
	}
	return ""
}

func grokStreamTokens(exec *grokReverseExecution) []string {
	if exec == nil {
		return nil
	}
	if len(exec.ImageURLs) > 0 {
		return []string{strings.Join(exec.ImageURLs, "\n")}
	}
	if len(exec.VideoURLs) > 0 {
		return []string{strings.Join(exec.VideoURLs, "\n")}
	}
	if len(exec.Tokens) > 0 {
		return exec.Tokens
	}
	if strings.TrimSpace(exec.Message) == "" {
		return nil
	}
	return []string{strings.TrimSpace(exec.Message)}
}

func grokChatPrompt(messages []apicompat.ChatMessage) string {
	parts := make([]string, 0, len(messages))
	for _, message := range messages {
		if text := grokRawContentText(message.Content); text != "" {
			parts = append(parts, text)
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func grokResponsesPrompt(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var plain string
	if err := json.Unmarshal(raw, &plain); err == nil {
		return strings.TrimSpace(plain)
	}
	var items []apicompat.ResponsesInputItem
	if err := json.Unmarshal(raw, &items); err != nil {
		return ""
	}
	parts := make([]string, 0, len(items))
	for _, item := range items {
		if text := grokRawContentText(item.Content); text != "" {
			parts = append(parts, text)
		}
		if strings.TrimSpace(item.Output) != "" {
			parts = append(parts, strings.TrimSpace(item.Output))
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func grokRawContentText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var plain string
	if err := json.Unmarshal(raw, &plain); err == nil {
		return strings.TrimSpace(plain)
	}
	var parts []map[string]any
	if err := json.Unmarshal(raw, &parts); err != nil {
		return ""
	}
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		for _, key := range []string{"text", "input_text", "content"} {
			if text, ok := part[key].(string); ok && strings.TrimSpace(text) != "" {
				values = append(values, strings.TrimSpace(text))
			}
		}
	}
	return strings.TrimSpace(strings.Join(values, "\n"))
}

func grokShouldWaitQuickImage(modeID string, mappedModel string) bool {
	modeID = strings.TrimSpace(strings.ToLower(modeID))
	mappedModel = strings.TrimSpace(strings.ToLower(mappedModel))
	target := modeID
	if target == "" {
		target = mappedModel
	}
	return target == "quick-image" || strings.HasPrefix(target, "quick-image-")
}

func grokCollectURLsFromJSON(raw []byte, videoOnly bool) []string {
	if len(raw) == 0 {
		return nil
	}
	var payload any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil
	}
	seen := map[string]struct{}{}
	result := make([]string, 0)
	var walk func(value any)
	walk = func(value any) {
		switch typed := value.(type) {
		case map[string]any:
			for key, child := range typed {
				if text, ok := child.(string); ok {
					text = strings.TrimSpace(text)
					if strings.HasPrefix(text, "http") && (strings.Contains(strings.ToLower(key), "url") || grokLooksLikeMediaURL(text)) {
						if !videoOnly || grokLooksLikeVideoURL(text) {
							if _, exists := seen[text]; !exists {
								seen[text] = struct{}{}
								result = append(result, text)
							}
						}
					}
				}
				walk(child)
			}
		case []any:
			for _, child := range typed {
				walk(child)
			}
		}
	}
	walk(payload)
	return result
}

func grokLooksLikeMediaURL(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return strings.Contains(value, ".png") || strings.Contains(value, ".jpg") || strings.Contains(value, ".jpeg") || strings.Contains(value, ".webp") || strings.Contains(value, ".gif") || grokLooksLikeVideoURL(value)
}

func grokLooksLikeVideoURL(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return strings.Contains(value, ".mp4") || strings.Contains(value, ".mov") || strings.Contains(value, ".webm")
}

func grokFilterPreviewURLs(urls []string) []string {
	result := make([]string, 0, len(urls))
	seen := make(map[string]struct{}, len(urls))
	for _, url := range urls {
		trimmed := strings.TrimSpace(url)
		if trimmed == "" {
			continue
		}
		lower := strings.ToLower(trimmed)
		if strings.Contains(lower, "preview") || strings.Contains(lower, "thumbnail") {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func grokMediaTypeForExec(exec *grokReverseExecution) string {
	if exec == nil {
		return ""
	}
	if len(exec.ImageURLs) > 0 {
		return "image"
	}
	if len(exec.VideoURLs) > 0 {
		return "video"
	}
	return ""
}

func grokImageData(urls []string) []gin.H {
	items := make([]gin.H, 0, len(urls))
	for _, url := range urls {
		items = append(items, gin.H{"url": url})
	}
	return items
}

func grokVideoData(urls []string) []gin.H {
	items := make([]gin.H, 0, len(urls))
	for _, url := range urls {
		items = append(items, gin.H{"url": url})
	}
	return items
}

func grokEncodeReverseVideoRequestID(conversationID string, responseID string) string {
	return grokReverseVideoRequestTag + ":" + strings.TrimSpace(conversationID) + ":" + strings.TrimSpace(responseID)
}

func grokDecodeReverseVideoRequestID(requestID string) (string, string) {
	parts := strings.SplitN(strings.TrimSpace(requestID), ":", 3)
	if len(parts) != 3 || parts[0] != grokReverseVideoRequestTag {
		return "", ""
	}
	return strings.TrimSpace(parts[1]), strings.TrimSpace(parts[2])
}

func grokNormalizeVideoResolution(value string) string {
	normalized := strings.TrimSpace(strings.ToLower(value))
	switch normalized {
	case "480p", "720p", "1080p":
		return normalized
	}
	if strings.Contains(normalized, "1280x720") {
		return "720p"
	}
	if strings.Contains(normalized, "854x480") || strings.Contains(normalized, "640x480") {
		return "480p"
	}
	if strings.Contains(normalized, "1920x1080") {
		return "1080p"
	}
	return normalized
}

func grokResolutionRank(value string) int {
	switch grokNormalizeVideoResolution(value) {
	case "1080p":
		return 3
	case "720p":
		return 2
	case "480p":
		return 1
	default:
		return 0
	}
}

func (s *GrokGatewayService) logGrokPromptDiagnostics(prompt string, mappedModel string, modeID string) {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return
	}
	nonASCII := 0
	hasCJK := false
	lowerPrompt := strings.ToLower(prompt)
	categories := make([]string, 0, 4)
	for _, r := range prompt {
		if r > 127 {
			nonASCII++
		}
		if (r >= 0x4E00 && r <= 0x9FFF) || (r >= 0x3040 && r <= 0x30FF) || (r >= 0xAC00 && r <= 0xD7AF) {
			hasCJK = true
		}
	}
	for category, keywords := range map[string][]string{
		"photo":        {"photo", "photoreal", "realistic"},
		"illustration": {"illustration", "anime", "cartoon"},
		"portrait":     {"portrait", "headshot", "selfie"},
		"landscape":    {"landscape", "mountain", "cityscape"},
	} {
		for _, keyword := range keywords {
			if strings.Contains(lowerPrompt, keyword) {
				categories = append(categories, category)
				break
			}
		}
	}
	logger.L().Debug("grok.prompt_diagnostics",
		zap.String("platform", PlatformGrok),
		zap.String("route_mode", GrokRouteModeSSO),
		zap.String("model", mappedModel),
		zap.String("mode_id", strings.TrimSpace(modeID)),
		zap.String("message_hash", hashSensitiveValueForLog(prompt)),
		zap.Int("message_len", len(prompt)),
		zap.Int("non_ascii_count", nonASCII),
		zap.Bool("has_cjk", hasCJK),
		zap.Bool("has_image_keywords", len(categories) > 0),
		zap.Strings("image_keyword_categories", categories),
	)
}
