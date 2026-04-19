package service

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Wei-Shaw/sub2api/internal/pkg/geminicli"
	"github.com/Wei-Shaw/sub2api/internal/pkg/googleapi"
	"github.com/Wei-Shaw/sub2api/internal/util/responseheaders"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"io"
	"net/http"
	"strings"
	"time"
)

func (s *GeminiMessagesCompatService) handleNonStreamingResponse(c *gin.Context, resp *http.Response, originalModel string) (*ClaudeUsage, string, *string, error) {
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, "", nil, s.writeClaudeError(c, http.StatusBadGateway, "upstream_error", "Failed to read upstream response")
	}
	SetOpsTraceUpstreamResponse(c, "gemini_upstream_response", body, resp.Header.Get("Content-Type"), false)
	unwrappedBody, err := unwrapGeminiResponse(body)
	if err != nil {
		return nil, "", nil, s.writeClaudeError(c, http.StatusBadGateway, "upstream_error", "Failed to parse upstream response")
	}
	var geminiResp map[string]any
	if err := json.Unmarshal(unwrappedBody, &geminiResp); err != nil {
		return nil, "", nil, s.writeClaudeError(c, http.StatusBadGateway, "upstream_error", "Failed to parse upstream response")
	}
	resolvedServiceTier := extractGeminiResolvedServiceTierFromResponse(unwrappedBody, resp.Header)
	claudeResp, usage, convErr := convertGeminiToClaudeMessage(geminiResp, originalModel, unwrappedBody)
	if convErr != nil {
		var compatErr *geminiCompatResponseError
		if errors.As(convErr, &compatErr) {
			return nil, compatErr.responseID, resolvedServiceTier, s.writeClaudeError(c, compatErr.statusCode, compatErr.errorType, compatErr.message)
		}
		return nil, "", resolvedServiceTier, s.writeClaudeError(c, http.StatusBadGateway, "upstream_error", convErr.Error())
	}
	responseID := strings.TrimSpace(stringValueFromAny(claudeResp["google_response_id"]))
	if c != nil && c.Writer != nil && c.Writer.Header().Get("x-request-id") == "" && responseID != "" {
		c.Header("x-request-id", responseID)
	}
	c.JSON(http.StatusOK, claudeResp)
	return usage, responseID, resolvedServiceTier, nil
}

func (s *GeminiMessagesCompatService) handleStreamingResponse(c *gin.Context, resp *http.Response, startTime time.Time, originalModel string) (*geminiStreamResult, error) {
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		return nil, errors.New("streaming not supported")
	}
	reader := bufio.NewReader(resp.Body)
	prefetched := make([]*geminiStreamChunk, 0, 4)
	var lastAnalysis geminiResponseAnalysis
	hasAnalysis := false
	for {
		chunk, done, err := readNextGeminiStreamChunk(reader, true)
		if err != nil {
			return nil, fmt.Errorf("stream read error: %w", err)
		}
		if chunk != nil {
			prefetched = append(prefetched, chunk)
			lastAnalysis = analyzeGeminiResponse(chunk.response, chunk.raw)
			hasAnalysis = true
			if lastAnalysis.promptBlocked() {
				return nil, s.writeClaudeError(c, http.StatusBadRequest, "invalid_request_error", buildGeminiBlockedMessage(lastAnalysis))
			}
			if lastAnalysis.hasRenderableParts() {
				break
			}
		}
		if done {
			if hasAnalysis && strings.EqualFold(lastAnalysis.FinishReason, "SAFETY") {
				return nil, s.writeClaudeError(c, http.StatusBadRequest, "invalid_request_error", buildGeminiNoCandidateMessage(lastAnalysis))
			}
			return nil, s.writeClaudeError(c, http.StatusBadGateway, "upstream_error", buildGeminiNoCandidateMessage(lastAnalysis))
		}
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	if c.Writer.Header().Get("x-request-id") == "" && strings.TrimSpace(lastAnalysis.ResponseID) != "" {
		c.Header("x-request-id", lastAnalysis.ResponseID)
	}
	c.Status(http.StatusOK)

	emitter := newGeminiClaudeStreamEmitter(c.Writer, flusher, startTime, originalModel)
	for _, chunk := range prefetched {
		emitter.consumeResponse(chunk.response, chunk.raw)
	}
	for {
		chunk, done, err := readNextGeminiStreamChunk(reader, true)
		if err != nil {
			return nil, fmt.Errorf("stream read error: %w", err)
		}
		if done {
			break
		}
		if chunk != nil {
			emitter.consumeResponse(chunk.response, chunk.raw)
		}
	}
	return emitter.finalize(), nil
}
func writeSSE(w io.Writer, event string, data any) {
	if event != "" {
		_, _ = fmt.Fprintf(w, "event: %s\n", event)
	}
	b, _ := json.Marshal(data)
	_, _ = fmt.Fprintf(w, "data: %s\n\n", string(b))
}
func randomHex(nBytes int) string {
	b := make([]byte, nBytes)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
func (s *GeminiMessagesCompatService) writeClaudeError(c *gin.Context, status int, errType, message string) error {
	c.JSON(status, gin.H{"type": "error", "error": gin.H{"type": errType, "message": message}})
	return fmt.Errorf("%s", message)
}
func (s *GeminiMessagesCompatService) writeGoogleError(c *gin.Context, status int, message string) error {
	c.JSON(status, gin.H{"error": gin.H{"code": status, "message": message, "status": googleapi.HTTPStatusToGoogleStatus(status)}})
	return fmt.Errorf("%s", message)
}
func unwrapIfNeeded(isOAuth bool, raw []byte) []byte {
	if !isOAuth {
		return raw
	}
	inner, err := unwrapGeminiResponse(raw)
	if err != nil {
		return raw
	}
	return inner
}
func collectGeminiSSE(body io.Reader, isOAuth bool) (map[string]any, *ClaudeUsage, error) {
	reader := bufio.NewReader(body)
	var last map[string]any
	var lastWithParts map[string]any
	collectedParts := make([]map[string]any, 0)
	usage := &ClaudeUsage{}
	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			trimmed := strings.TrimRight(line, "\r\n")
			if strings.HasPrefix(trimmed, "data:") {
				payload := strings.TrimSpace(strings.TrimPrefix(trimmed, "data:"))
				switch payload {
				case "", "[DONE]":
					if payload == "[DONE]" {
						return mergeCollectedPartsToResponse(pickGeminiCollectResult(last, lastWithParts), collectedParts), usage, nil
					}
				default:
					var parsed map[string]any
					var rawBytes []byte
					if isOAuth {
						innerBytes, err := unwrapGeminiResponse([]byte(payload))
						if err == nil {
							rawBytes = innerBytes
							_ = json.Unmarshal(innerBytes, &parsed)
						}
					} else {
						rawBytes = []byte(payload)
						_ = json.Unmarshal(rawBytes, &parsed)
					}
					if parsed != nil {
						last = parsed
						if u := extractGeminiUsage(rawBytes); u != nil {
							usage = u
						}
						if parts := extractGeminiParts(parsed); len(parts) > 0 {
							lastWithParts = parsed
							collectedParts = append(collectedParts, parts...)
						}
					}
				}
			}
		}
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, nil, err
		}
	}
	return mergeCollectedPartsToResponse(pickGeminiCollectResult(last, lastWithParts), collectedParts), usage, nil
}
func pickGeminiCollectResult(last map[string]any, lastWithParts map[string]any) map[string]any {
	if lastWithParts != nil {
		return lastWithParts
	}
	if last != nil {
		return last
	}
	return map[string]any{}
}

type geminiNativeStreamResult struct {
	usage               *ClaudeUsage
	firstTokenMs        *int
	responseID          string
	resolvedServiceTier *string
}

func isGeminiInsufficientScope(headers http.Header, body []byte) bool {
	if strings.Contains(strings.ToLower(headers.Get("Www-Authenticate")), "insufficient_scope") {
		return true
	}
	lower := strings.ToLower(string(body))
	return strings.Contains(lower, "insufficient authentication scopes") || strings.Contains(lower, "access_token_scope_insufficient")
}
func (s *GeminiMessagesCompatService) ForwardAIStudioGET(ctx context.Context, account *Account, path string) (*UpstreamHTTPResult, error) {
	if account == nil {
		return nil, errors.New("account is nil")
	}
	path = strings.TrimSpace(path)
	if path == "" || !strings.HasPrefix(path, "/") {
		return nil, errors.New("invalid path")
	}
	pathOnly, _, _ := strings.Cut(path, "?")
	if account.IsGeminiVertexExpress() || account.IsGeminiVertexAI() {
		if s.vertexCatalogService == nil {
			return nil, fmt.Errorf("vertex catalog service is not configured")
		}
		catalog, err := s.vertexCatalogService.GetCatalog(ctx, account, false)
		if err != nil {
			return nil, err
		}
		switch {
		case pathOnly == "/v1beta/models":
			return buildGeminiVertexCatalogModelsResponseFromCatalog(catalog.CallableUnion)
		case strings.HasPrefix(pathOnly, "/v1beta/models/"):
			return buildGeminiVertexCatalogModelResponseFromCatalog(strings.TrimPrefix(pathOnly, "/v1beta/models/"), catalog.CallableUnion)
		default:
			return nil, fmt.Errorf("unsupported vertex AI GET path: %s", path)
		}
	}
	baseURL := account.GetGeminiBaseURL(geminicli.AIStudioBaseURL)
	normalizedBaseURL, err := s.validateUpstreamBaseURL(baseURL)
	if err != nil {
		return nil, err
	}
	fullURL := strings.TrimRight(normalizedBaseURL, "/") + path
	var proxyURL string
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, err
	}
	switch account.Type {
	case AccountTypeAPIKey:
		apiKey := strings.TrimSpace(account.GetCredential("api_key"))
		if apiKey == "" {
			return nil, errors.New("gemini api_key not configured")
		}
		req.Header.Set("x-goog-api-key", apiKey)
	case AccountTypeOAuth:
		if s.tokenProvider == nil {
			return nil, errors.New("gemini token provider not configured")
		}
		accessToken, err := s.tokenProvider.GetAccessToken(ctx, account)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+accessToken)
	default:
		return nil, fmt.Errorf("unsupported account type: %s", account.Type)
	}
	resp, err := s.httpUpstream.Do(req, proxyURL, account.ID, account.Concurrency)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	wwwAuthenticate := resp.Header.Get("Www-Authenticate")
	filteredHeaders := responseheaders.FilterHeaders(resp.Header, s.responseHeaderFilter)
	if wwwAuthenticate != "" {
		filteredHeaders.Set("Www-Authenticate", wwwAuthenticate)
	}
	return &UpstreamHTTPResult{StatusCode: resp.StatusCode, Headers: filteredHeaders, Body: body}, nil
}
func unwrapGeminiResponse(raw []byte) ([]byte, error) {
	result := gjson.GetBytes(raw, "response")
	if result.Exists() && result.Type == gjson.JSON {
		return []byte(result.Raw), nil
	}
	return raw, nil
}

type geminiCompatResponseError struct {
	statusCode int
	errorType  string
	message    string
	responseID string
}

func (e *geminiCompatResponseError) Error() string {
	return e.message
}

func convertGeminiToClaudeMessage(geminiResp map[string]any, originalModel string, rawData []byte) (map[string]any, *ClaudeUsage, error) {
	analysis := analyzeGeminiResponse(geminiResp, rawData)
	usage := analysis.Usage
	if usage == nil {
		usage = &ClaudeUsage{}
	}
	if analysis.promptBlocked() {
		return nil, usage, &geminiCompatResponseError{
			statusCode: http.StatusBadRequest,
			errorType:  "invalid_request_error",
			message:    buildGeminiBlockedMessage(analysis),
			responseID: analysis.ResponseID,
		}
	}
	if !analysis.hasRenderableParts() {
		statusCode := http.StatusBadGateway
		errorType := "upstream_error"
		if strings.EqualFold(analysis.FinishReason, "SAFETY") {
			statusCode = http.StatusBadRequest
			errorType = "invalid_request_error"
		}
		return nil, usage, &geminiCompatResponseError{
			statusCode: statusCode,
			errorType:  errorType,
			message:    buildGeminiNoCandidateMessage(analysis),
			responseID: analysis.ResponseID,
		}
	}

	contentBlocks := make([]any, 0, len(analysis.Parts))
	sawToolUse := false
	toolSequence := 0
	for _, part := range analysis.Parts {
		if part == nil {
			continue
		}
		if functionCall, ok := part["functionCall"].(map[string]any); ok && functionCall != nil {
			toolSequence++
			name := strings.TrimSpace(stringValueFromAny(functionCall["name"]))
			if name == "" {
				name = "tool"
			}
			sawToolUse = true
			contentBlocks = append(contentBlocks, map[string]any{
				"type":  "tool_use",
				"id":    buildGeminiToolUseID(functionCall, toolSequence),
				"name":  name,
				"input": functionCall["args"],
			})
			continue
		}
		text := stringValueFromAny(part["text"])
		if strings.TrimSpace(text) != "" {
			if thought, _ := part["thought"].(bool); thought {
				block := map[string]any{
					"type":     "thinking",
					"thinking": text,
				}
				if signature := strings.TrimSpace(stringValueFromAny(part["thoughtSignature"])); signature != "" {
					block["signature"] = signature
				}
				contentBlocks = append(contentBlocks, block)
				continue
			}
			contentBlocks = append(contentBlocks, map[string]any{"type": "text", "text": text})
			continue
		}
		if inlineData, ok := part["inlineData"].(map[string]any); ok && inlineData != nil {
			mimeType := firstNonEmptyString(
				stringValueFromAny(inlineData["mimeType"]),
				stringValueFromAny(inlineData["mime_type"]),
			)
			if mimeType == "" {
				mimeType = "image/*"
			}
			contentBlocks = append(contentBlocks, map[string]any{
				"type": "text",
				"text": fmt.Sprintf("[Gemini returned inline %s data]", mimeType),
			})
		}
	}

	stopReason := mapGeminiFinishReasonToClaudeStopReason(analysis.FinishReason)
	if sawToolUse {
		stopReason = "tool_use"
	}
	resp := map[string]any{
		"id":            "msg_" + randomHex(12),
		"type":          "message",
		"role":          "assistant",
		"model":         originalModel,
		"content":       contentBlocks,
		"stop_reason":   stopReason,
		"stop_sequence": nil,
		"usage": map[string]any{
			"input_tokens":  usage.InputTokens,
			"output_tokens": usage.OutputTokens,
		},
	}
	if analysis.ResponseID != "" {
		resp["google_response_id"] = analysis.ResponseID
	}
	if analysis.ModelVersion != "" {
		resp["google_model_version"] = analysis.ModelVersion
	}
	if analysis.FinishReason != "" {
		resp["google_finish_reason"] = analysis.FinishReason
	}
	if len(analysis.GroundingMetadata) > 0 {
		resp["google_grounding_metadata"] = analysis.GroundingMetadata
	}
	return resp, usage, nil
}
func extractGeminiUsage(data []byte) *ClaudeUsage {
	usage := gjson.GetBytes(data, "usageMetadata")
	if usage.Exists() {
		prompt := int(usage.Get("promptTokenCount").Int())
		cand := int(usage.Get("candidatesTokenCount").Int())
		cached := int(usage.Get("cachedContentTokenCount").Int())
		thoughts := int(usage.Get("thoughtsTokenCount").Int())
		if prompt < cached {
			prompt = cached
		}
		return &ClaudeUsage{InputTokens: prompt - cached, OutputTokens: cand + thoughts, CacheReadInputTokens: cached}
	}
	interactionUsage := gjson.GetBytes(data, "usage")
	if !interactionUsage.Exists() {
		return nil
	}
	totalInput := int(interactionUsage.Get("total_input_tokens").Int())
	if totalInput == 0 {
		totalInput = int(interactionUsage.Get("totalInputTokens").Int())
	}
	totalOutput := int(interactionUsage.Get("total_output_tokens").Int())
	if totalOutput == 0 {
		totalOutput = int(interactionUsage.Get("totalOutputTokens").Int())
	}
	totalCached := int(interactionUsage.Get("total_cached_tokens").Int())
	if totalCached == 0 {
		totalCached = int(interactionUsage.Get("totalCachedTokens").Int())
	}
	totalThought := int(interactionUsage.Get("total_thought_tokens").Int())
	if totalThought == 0 {
		totalThought = int(interactionUsage.Get("totalThoughtTokens").Int())
	}
	if totalThought == 0 {
		totalThought = int(interactionUsage.Get("total_reasoning_tokens").Int())
	}
	if totalThought == 0 {
		totalThought = int(interactionUsage.Get("totalReasoningTokens").Int())
	}
	if totalInput == 0 && totalOutput == 0 && totalCached == 0 && totalThought == 0 {
		return nil
	}
	billableInput := totalInput - totalCached
	if billableInput < 0 {
		billableInput = 0
	}
	return &ClaudeUsage{
		InputTokens:          billableInput,
		OutputTokens:         totalOutput + totalThought,
		CacheReadInputTokens: totalCached,
	}
}
func asInt(v any) (int, bool) {
	switch t := v.(type) {
	case float64:
		return int(t), true
	case int:
		return t, true
	case int64:
		return int(t), true
	case json.Number:
		i, err := t.Int64()
		if err != nil {
			return 0, false
		}
		return int(i), true
	default:
		return 0, false
	}
}
func ensureGeminiFunctionCallThoughtSignatures(body []byte) []byte {
	if !bytes.Contains(body, []byte(`"functionCall"`)) {
		return body
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return body
	}
	contentsAny, ok := payload["contents"].([]any)
	if !ok || len(contentsAny) == 0 {
		return body
	}
	modified := false
	for _, c := range contentsAny {
		cm, ok := c.(map[string]any)
		if !ok {
			continue
		}
		partsAny, ok := cm["parts"].([]any)
		if !ok || len(partsAny) == 0 {
			continue
		}
		for _, p := range partsAny {
			pm, ok := p.(map[string]any)
			if !ok || pm == nil {
				continue
			}
			if fc, ok := pm["functionCall"].(map[string]any); !ok || fc == nil {
				continue
			}
			ts, _ := pm["thoughtSignature"].(string)
			if strings.TrimSpace(ts) == "" {
				pm["thoughtSignature"] = geminiDummyThoughtSignature
				modified = true
			}
		}
	}
	if !modified {
		return body
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return body
	}
	return b
}

func extractGeminiParts(geminiResp map[string]any) []map[string]any {
	if candidates, ok := geminiResp["candidates"].([]any); ok && len(candidates) > 0 {
		if cand, ok := candidates[0].(map[string]any); ok {
			if content, ok := cand["content"].(map[string]any); ok {
				if partsAny, ok := content["parts"].([]any); ok && len(partsAny) > 0 {
					out := make([]map[string]any, 0, len(partsAny))
					for _, p := range partsAny {
						pm, ok := p.(map[string]any)
						if !ok {
							continue
						}
						out = append(out, pm)
					}
					return out
				}
			}
		}
	}
	return nil
}
func computeGeminiTextDelta(seen, incoming string) (delta, newSeen string) {
	incoming = strings.TrimSuffix(incoming, "\u0000")
	if incoming == "" {
		return "", seen
	}
	if strings.HasPrefix(incoming, seen) {
		return strings.TrimPrefix(incoming, seen), incoming
	}
	if strings.HasPrefix(seen, incoming) {
		return "", seen
	}
	return incoming, seen + incoming
}
func mapGeminiFinishReasonToClaudeStopReason(finishReason string) string {
	switch strings.ToUpper(strings.TrimSpace(finishReason)) {
	case "MAX_TOKENS":
		return "max_tokens"
	case "STOP":
		return "end_turn"
	default:
		return "end_turn"
	}
}
