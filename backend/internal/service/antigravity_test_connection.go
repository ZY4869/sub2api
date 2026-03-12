package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Wei-Shaw/sub2api/internal/pkg/antigravity"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/google/uuid"
	"github.com/tidwall/gjson"
	"io"
	"strings"
)

type TestConnectionResult struct {
	Text        string
	MappedModel string
}

func (s *AntigravityGatewayService) TestConnection(ctx context.Context, account *Account, modelID string) (*TestConnectionResult, error) {
	if s.tokenProvider == nil {
		return nil, errors.New("antigravity token provider not configured")
	}
	accessToken, err := s.tokenProvider.GetAccessToken(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("获取 access_token 失败: %w", err)
	}
	projectID := strings.TrimSpace(account.GetCredential("project_id"))
	mappedModel := s.getMappedModel(account, modelID)
	if mappedModel == "" {
		return nil, fmt.Errorf("model %s not in whitelist", modelID)
	}
	var requestBody []byte
	if strings.HasPrefix(modelID, "gemini-") {
		requestBody, err = s.buildGeminiTestRequest(projectID, mappedModel)
	} else {
		requestBody, err = s.buildClaudeTestRequest(projectID, mappedModel)
	}
	if err != nil {
		return nil, fmt.Errorf("构建请求失败: %w", err)
	}
	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	baseURL := resolveAntigravityForwardBaseURL()
	if baseURL == "" {
		return nil, errors.New("no antigravity forward base url configured")
	}
	availableURLs := []string{baseURL}
	var lastErr error
	for urlIdx, baseURL := range availableURLs {
		req, err := antigravity.NewAPIRequestWithURL(ctx, baseURL, "streamGenerateContent", accessToken, requestBody)
		if err != nil {
			lastErr = err
			continue
		}
		logger.LegacyPrintf("service.antigravity_gateway", "[antigravity-Test] account=%s request_size=%d url=%s", account.Name, len(requestBody), req.URL.String())
		resp, err := s.httpUpstream.Do(req, proxyURL, account.ID, account.Concurrency)
		if err != nil {
			lastErr = fmt.Errorf("请求失败: %w", err)
			if shouldAntigravityFallbackToNextURL(err, 0) && urlIdx < len(availableURLs)-1 {
				logger.LegacyPrintf("service.antigravity_gateway", "[antigravity-Test] URL fallback: %s -> %s", baseURL, availableURLs[urlIdx+1])
				continue
			}
			return nil, lastErr
		}
		respBody, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		_ = resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("读取响应失败: %w", err)
		}
		if shouldAntigravityFallbackToNextURL(nil, resp.StatusCode) && urlIdx < len(availableURLs)-1 {
			logger.LegacyPrintf("service.antigravity_gateway", "[antigravity-Test] URL fallback (HTTP %d): %s -> %s", resp.StatusCode, baseURL, availableURLs[urlIdx+1])
			continue
		}
		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("API 返回 %d: %s", resp.StatusCode, string(respBody))
		}
		text := extractTextFromSSEResponse(respBody)
		antigravity.DefaultURLAvailability.MarkSuccess(baseURL)
		return &TestConnectionResult{Text: text, MappedModel: mappedModel}, nil
	}
	return nil, lastErr
}
func (s *AntigravityGatewayService) buildGeminiTestRequest(projectID, model string) ([]byte, error) {
	payload := map[string]any{"contents": []map[string]any{{"role": "user", "parts": []map[string]any{{"text": "."}}}}, "systemInstruction": map[string]any{"parts": []map[string]any{{"text": antigravity.GetDefaultIdentityPatch()}}}, "generationConfig": map[string]any{"maxOutputTokens": 1}}
	payloadBytes, _ := json.Marshal(payload)
	return s.wrapV1InternalRequest(projectID, model, payloadBytes)
}
func (s *AntigravityGatewayService) buildClaudeTestRequest(projectID, mappedModel string) ([]byte, error) {
	claudeReq := &antigravity.ClaudeRequest{Model: mappedModel, Messages: []antigravity.ClaudeMessage{{Role: "user", Content: json.RawMessage(`"."`)}}, MaxTokens: 1, Stream: false}
	return antigravity.TransformClaudeToGemini(claudeReq, projectID, mappedModel)
}
func (s *AntigravityGatewayService) getClaudeTransformOptions(ctx context.Context) antigravity.TransformOptions {
	opts := antigravity.DefaultTransformOptions()
	if s.settingService == nil {
		return opts
	}
	opts.EnableIdentityPatch = s.settingService.IsIdentityPatchEnabled(ctx)
	opts.IdentityPatch = s.settingService.GetIdentityPatchPrompt(ctx)
	return opts
}
func extractTextFromSSEResponse(respBody []byte) string {
	var texts []string
	lines := bytes.Split(respBody, []byte("\n"))
	for _, line := range lines {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		if bytes.HasPrefix(line, []byte("data:")) {
			line = bytes.TrimPrefix(line, []byte("data:"))
			line = bytes.TrimSpace(line)
		}
		if len(line) == 0 || line[0] != '{' {
			continue
		}
		var data map[string]any
		if err := json.Unmarshal(line, &data); err != nil {
			continue
		}
		response, ok := data["response"].(map[string]any)
		if !ok {
			response = data
		}
		candidates, ok := response["candidates"].([]any)
		if !ok || len(candidates) == 0 {
			continue
		}
		candidate, ok := candidates[0].(map[string]any)
		if !ok {
			continue
		}
		content, ok := candidate["content"].(map[string]any)
		if !ok {
			continue
		}
		parts, ok := content["parts"].([]any)
		if !ok {
			continue
		}
		for _, part := range parts {
			if partMap, ok := part.(map[string]any); ok {
				if text, ok := partMap["text"].(string); ok && text != "" {
					texts = append(texts, text)
				}
			}
		}
	}
	return strings.Join(texts, "")
}
func injectIdentityPatchToGeminiRequest(body []byte) ([]byte, error) {
	var request map[string]any
	if err := json.Unmarshal(body, &request); err != nil {
		return nil, fmt.Errorf("解析 Gemini 请求失败: %w", err)
	}
	if sysInst, ok := request["systemInstruction"].(map[string]any); ok {
		if parts, ok := sysInst["parts"].([]any); ok {
			for _, part := range parts {
				if partMap, ok := part.(map[string]any); ok {
					if text, ok := partMap["text"].(string); ok {
						if strings.Contains(text, "You are Antigravity") {
							return body, nil
						}
					}
				}
			}
		}
	}
	identityPatch := antigravity.GetDefaultIdentityPatch()
	newPart := map[string]any{"text": identityPatch}
	if existing, ok := request["systemInstruction"].(map[string]any); ok {
		if parts, ok := existing["parts"].([]any); ok {
			existing["parts"] = append([]any{newPart}, parts...)
		} else {
			existing["parts"] = []any{newPart}
		}
	} else {
		request["systemInstruction"] = map[string]any{"parts": []any{newPart}}
	}
	return json.Marshal(request)
}
func (s *AntigravityGatewayService) wrapV1InternalRequest(projectID, model string, originalBody []byte) ([]byte, error) {
	var request any
	if err := json.Unmarshal(originalBody, &request); err != nil {
		return nil, fmt.Errorf("解析请求体失败: %w", err)
	}
	wrapped := map[string]any{"project": projectID, "requestId": "agent-" + uuid.New().String(), "userAgent": "antigravity", "requestType": "agent", "model": model, "request": request}
	return json.Marshal(wrapped)
}
func (s *AntigravityGatewayService) unwrapV1InternalResponse(body []byte) ([]byte, error) {
	result := gjson.GetBytes(body, "response")
	if result.Exists() {
		return []byte(result.Raw), nil
	}
	return body, nil
}
func isModelNotFoundError(statusCode int, body []byte) bool {
	if statusCode != 404 {
		return false
	}
	bodyStr := strings.ToLower(string(body))
	keywords := []string{"model not found", "unknown model", "not found"}
	for _, keyword := range keywords {
		if strings.Contains(bodyStr, keyword) {
			return true
		}
	}
	return true
}
