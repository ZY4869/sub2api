package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type OpenAIAlphaSearchForwardResult struct {
	RequestID        string
	StatusCode       int
	ResponseHeaders  http.Header
	UpstreamEndpoint string
	Duration         time.Duration
}

func (s *OpenAIGatewayService) ForwardAlphaSearch(ctx context.Context, c *gin.Context, account *Account, body []byte) (*OpenAIAlphaSearchForwardResult, error) {
	if s == nil || c == nil || account == nil {
		return nil, fmt.Errorf("service, context, and account are required")
	}
	account = ResolveProtocolGatewayInboundAccount(account, PlatformOpenAI)
	if account == nil {
		return nil, fmt.Errorf("openai account is required")
	}
	if !account.IsOpenAIOAuth() && !account.IsOpenAIApiKey() {
		return nil, fmt.Errorf("alpha/search requires an OpenAI OAuth or API key account")
	}
	startTime := time.Now()
	requestedModel, upstreamModel := alphaSearchModels(account, body)
	if upstreamModel != "" && upstreamModel != requestedModel {
		if mapped, mapErr := sjson.SetBytes(body, "model", upstreamModel); mapErr == nil {
			body = mapped
		}
	}
	body = sanitizeAlphaSearchBody(body)
	token, _, err := s.GetAccessToken(ctx, account)
	if err != nil {
		return nil, err
	}
	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	// Keep ops traces on the public alpha/search request. PAT fallback builds a
	// synthetic Responses prompt and must not replace the trace with that full
	// prompt or duplicate sensitive search content.
	setOpsUpstreamRequestBody(c, body)
	if err := s.ensureOpenAIAlphaSearchAuthMetadata(ctx, account, token, proxyURL); err != nil {
		return nil, err
	}
	if account.IsOpenAIPersonalAccessToken() {
		return s.forwardAlphaSearchViaResponsesWebSearch(ctx, c, account, body, token, proxyURL, requestedModel, upstreamModel)
	}
	req, err := s.buildOpenAIAlphaSearchRequest(ctx, c, account, body, token)
	if err != nil {
		return nil, err
	}
	upstreamStart := time.Now()
	resp, err := s.httpUpstream.Do(MarkOpenAIHTTPUpstreamRequest(req), proxyURL, account.ID, account.Concurrency)
	SetOpsLatencyMs(c, OpsUpstreamLatencyMsKey, time.Since(upstreamStart).Milliseconds())
	if err != nil {
		return nil, newOpenAITransportFailoverError(c, account, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	statusCode := resp.StatusCode
	if statusCode >= http.StatusBadRequest {
		respBody, _ := readUpstreamResponseBodyLimitedFromResponse(resp, 2<<20)
		upstreamMsg := sanitizeUpstreamErrorMessage(strings.TrimSpace(extractUpstreamErrorMessage(respBody)))
		if s.shouldFailoverAlphaSearchResponse(account, statusCode, upstreamMsg, respBody) {
			appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
				Platform:           RoutingPlatformForAccount(account),
				AccountID:          account.ID,
				AccountName:        account.Name,
				UpstreamStatusCode: statusCode,
				UpstreamRequestID:  resp.Header.Get("x-request-id"),
				Kind:               "failover",
				Message:            upstreamMsg,
			})
			if s.shouldFailoverOpenAIUpstreamResponse(statusCode, upstreamMsg, respBody) {
				resp.Body = io.NopCloser(bytes.NewReader(respBody))
				s.handleFailoverSideEffects(ctx, resp, account)
			}
			return nil, &UpstreamFailoverError{
				StatusCode:             statusCode,
				ResponseBody:           respBody,
				ResponseHeaders:        resp.Header.Clone(),
				RetryableOnSameAccount: account.IsPoolMode() && account.IsPoolModeRetryableStatus(statusCode),
			}
		}
		writeOpenAIPassthroughResponseHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)
		c.Status(statusCode)
		if _, copyErr := c.Writer.Write(respBody); copyErr != nil {
			return nil, copyErr
		}
		if flusher, ok := c.Writer.(http.Flusher); ok {
			flusher.Flush()
		}
		return &OpenAIAlphaSearchForwardResult{
			RequestID:        resp.Header.Get("x-request-id"),
			StatusCode:       statusCode,
			ResponseHeaders:  resp.Header.Clone(),
			UpstreamEndpoint: EndpointAlphaSearch,
			Duration:         time.Since(startTime),
		}, nil
	}
	writeOpenAIPassthroughResponseHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)
	c.Status(statusCode)
	if _, copyErr := io.Copy(c.Writer, resp.Body); copyErr != nil {
		return nil, copyErr
	}
	if flusher, ok := c.Writer.(http.Flusher); ok {
		flusher.Flush()
	}
	result := &OpenAIAlphaSearchForwardResult{
		RequestID:        resp.Header.Get("x-request-id"),
		StatusCode:       statusCode,
		ResponseHeaders:  resp.Header.Clone(),
		UpstreamEndpoint: EndpointAlphaSearch,
		Duration:         time.Since(startTime),
	}
	return result, nil
}

func (s *OpenAIGatewayService) ensureOpenAIAlphaSearchAuthMetadata(ctx context.Context, account *Account, token, proxyURL string) error {
	if s == nil || account == nil || !account.IsOpenAIPersonalAccessToken() || strings.TrimSpace(account.GetChatGPTAccountID()) != "" {
		return nil
	}
	if s.openAITokenProvider == nil || s.openAITokenProvider.openAIOAuthService == nil {
		return nil
	}
	tokenInfo, err := s.openAITokenProvider.openAIOAuthService.ValidateCodexPersonalAccessToken(ctx, token, proxyURL)
	if err != nil {
		return fmt.Errorf("validate Codex PAT metadata for alpha/search: %w", err)
	}
	credentials := shallowCopyMap(account.Credentials)
	for key, value := range s.openAITokenProvider.openAIOAuthService.BuildAccountCredentials(tokenInfo) {
		credentials[key] = value
	}
	credentials = NormalizeOpenAIPersonalAccessTokenCredentials(account, tokenInfo, credentials)
	account.Credentials = shallowCopyMap(credentials)
	if s.accountRepo != nil {
		if err := persistAccountCredentials(ctx, s.accountRepo, account, credentials); err != nil {
			return fmt.Errorf("persist Codex PAT metadata for alpha/search: %w", err)
		}
	}
	return nil
}

func alphaSearchModels(account *Account, body []byte) (string, string) {
	requested := strings.TrimSpace(gjson.GetBytes(body, "model").String())
	if requested == "" || account == nil {
		return requested, requested
	}
	upstream := normalizeOpenAIModelForUpstream(account, account.GetMappedModel(requested))
	if upstream == "" {
		upstream = requested
	}
	return requested, upstream
}

func sanitizeAlphaSearchBody(body []byte) []byte {
	var fields map[string]json.RawMessage
	if len(body) == 0 || json.Unmarshal(body, &fields) != nil || fields == nil {
		return body
	}
	changed := false
	for _, key := range []string{"prompt_cache_key", "prompt_cache_retention"} {
		if _, ok := fields[key]; ok {
			delete(fields, key)
			changed = true
		}
	}
	if !changed {
		return body
	}
	encoded, err := json.Marshal(fields)
	if err != nil {
		return body
	}
	return encoded
}

func (s *OpenAIGatewayService) buildOpenAIAlphaSearchRequest(ctx context.Context, c *gin.Context, account *Account, body []byte, token string) (*http.Request, error) {
	targetURL, err := resolveOpenAIAlphaSearchTargetURL(account, s.validateUpstreamBaseURL)
	if err != nil {
		return nil, err
	}
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("parse alpha search URL: %w", err)
	}
	if c != nil && c.Request != nil && c.Request.URL != nil {
		query := parsedURL.Query()
		for key, values := range c.Request.URL.Query() {
			for _, value := range values {
				query.Add(key, value)
			}
		}
		parsedURL.RawQuery = query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, parsedURL.String(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req = req.WithContext(WithHTTPUpstreamProfile(req.Context(), HTTPUpstreamProfileOpenAI))
	authHeaders, err := s.buildOpenAIAuthenticationHeaders(ctx, account, token)
	if err != nil {
		return nil, fmt.Errorf("build openai authentication headers: %w", err)
	}
	for key, values := range authHeaders {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	applyOpenRouterAttributionRequestHeaders(account, req.Header)
	if account.IsOpenAIOAuth() {
		req.Host = "chatgpt.com"
		if accountID := account.GetChatGPTAccountID(); accountID != "" {
			req.Header.Set("chatgpt-account-id", accountID)
		}
		if account.IsChatGPTAccountFedRAMP() {
			req.Header.Set("x-openai-fedramp", "true")
		}
	}
	if c != nil && c.Request != nil {
		for key, values := range c.Request.Header {
			if !openaiAllowedHeaders[strings.ToLower(key)] {
				continue
			}
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if account.IsOpenAIOAuth() {
		if req.Header.Get("Originator") == "" {
			req.Header.Set("Originator", "codex_cli_rs")
		}
		if req.Header.Get("Version") == "" {
			req.Header.Set("Version", codexCLIVersion)
		}
		if req.Header.Get("User-Agent") == "" {
			req.Header.Set("User-Agent", codexCLIUserAgent)
		}
		stripOpenAIAlphaSearchResponsesHeaders(req.Header)
		s.applyCodexOAuthUserAgentPolicy(ctx, req.Header, account)
		enforceCodexIdentityHeadersWithConfig(ctx, req.Header, account, s.cfg)
	}
	if customUA := account.GetOpenAIUserAgent(); customUA != "" {
		req.Header.Set("User-Agent", customUA)
	}
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}
	ApplyAccountRequestHeaderOverrides(req, account)
	if account.IsOpenAIOAuth() {
		// Header overrides are applied last for consistency with other gateway
		// paths; keep Responses-only headers out of standalone alpha/search even
		// when an account override contains one.
		stripOpenAIAlphaSearchResponsesHeaders(req.Header)
	}
	return MarkOpenAIHTTPUpstreamRequest(req), nil
}

func (s *OpenAIGatewayService) forwardAlphaSearchViaResponsesWebSearch(ctx context.Context, c *gin.Context, account *Account, alphaBody []byte, token, proxyURL, requestedModel, upstreamModel string) (*OpenAIAlphaSearchForwardResult, error) {
	startTime := time.Now()
	model := strings.TrimSpace(upstreamModel)
	if model == "" {
		model = strings.TrimSpace(requestedModel)
	}
	if model == "" {
		model = "gpt-4o"
	}
	responsesBody, err := buildAlphaSearchResponsesWebSearchBody(alphaBody, model)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, chatgptCodexURL, bytes.NewReader(responsesBody))
	if err != nil {
		return nil, err
	}
	req = req.WithContext(WithHTTPUpstreamProfile(req.Context(), HTTPUpstreamProfileOpenAI))
	authHeaders, err := s.buildOpenAIAuthenticationHeaders(ctx, account, token)
	if err != nil {
		return nil, fmt.Errorf("build openai authentication headers: %w", err)
	}
	for key, values := range authHeaders {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	req.Host = "chatgpt.com"
	if chatgptAccountID := account.GetChatGPTAccountID(); chatgptAccountID != "" {
		req.Header.Set("chatgpt-account-id", chatgptAccountID)
	}
	if account.IsChatGPTAccountFedRAMP() {
		req.Header.Set("x-openai-fedramp", "true")
	}
	if c != nil && c.Request != nil {
		for key, values := range c.Request.Header {
			lower := strings.ToLower(key)
			if lower != "x-codex-turn-metadata" && lower != "originator" && lower != "version" && lower != "user-agent" {
				continue
			}
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("OpenAI-Beta", "responses=experimental")
	if req.Header.Get("Originator") == "" {
		req.Header.Set("Originator", "codex_cli_rs")
	}
	if req.Header.Get("Version") == "" {
		req.Header.Set("Version", codexCLIVersion)
	}
	if customUA := account.GetOpenAIUserAgent(); customUA != "" {
		req.Header.Set("User-Agent", customUA)
	} else if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", codexCLIUserAgent)
	}
	if sessionID := strings.TrimSpace(gjson.GetBytes(alphaBody, "id").String()); sessionID != "" {
		isolated := isolateOpenAISessionID(getAPIKeyIDFromContext(c), sessionID)
		req.Header.Set("Session_ID", isolated)
		req.Header.Set("Conversation_ID", isolated)
	}
	s.applyCodexOAuthUserAgentPolicy(ctx, req.Header, account)
	enforceCodexIdentityHeadersWithConfig(ctx, req.Header, account, s.cfg)
	ApplyAccountRequestHeaderOverrides(req, account)
	upstreamStart := time.Now()
	resp, err := s.httpUpstream.Do(MarkOpenAIHTTPUpstreamRequest(req), proxyURL, account.ID, account.Concurrency)
	SetOpsLatencyMs(c, OpsUpstreamLatencyMsKey, time.Since(upstreamStart).Milliseconds())
	if err != nil {
		return nil, newOpenAITransportFailoverError(c, account, err)
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, err := readUpstreamResponseBodyLimitedFromResponse(resp, 2<<20)
	if err != nil {
		return nil, fmt.Errorf("read alpha search responses fallback response: %w", err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		upstreamMsg := sanitizeUpstreamErrorMessage(strings.TrimSpace(extractUpstreamErrorMessage(respBody)))
		if s.shouldFailoverOpenAIUpstreamResponse(resp.StatusCode, upstreamMsg, respBody) {
			appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: RoutingPlatformForAccount(account), AccountID: account.ID, AccountName: account.Name, UpstreamStatusCode: resp.StatusCode, UpstreamRequestID: resp.Header.Get("x-request-id"), Kind: "failover", Message: upstreamMsg})
			if resp.StatusCode != http.StatusUnauthorized {
				resp.Body = io.NopCloser(bytes.NewReader(respBody))
				s.handleFailoverSideEffects(ctx, resp, account)
			}
			return nil, &UpstreamFailoverError{StatusCode: resp.StatusCode, ResponseBody: respBody, ResponseHeaders: resp.Header.Clone(), RetryableOnSameAccount: account.IsPoolMode() && account.IsPoolModeRetryableStatus(resp.StatusCode)}
		}
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		writeOpenAIPassthroughResponseHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)
		contentType := resp.Header.Get("Content-Type")
		if contentType == "" {
			contentType = "application/json"
		}
		c.Data(resp.StatusCode, contentType, respBody)
		return &OpenAIAlphaSearchForwardResult{RequestID: resp.Header.Get("x-request-id"), StatusCode: resp.StatusCode, ResponseHeaders: resp.Header.Clone(), UpstreamEndpoint: EndpointResponses, Duration: time.Since(startTime)}, nil
	}
	mediaType, _, mediaTypeErr := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if mediaTypeErr != nil || !strings.EqualFold(strings.TrimSpace(mediaType), "text/event-stream") || !isValidResponsesSSEForAlphaSearch(respBody) {
		// A successful HTTP status is not sufficient for the PAT fallback: a
		// non-SSE body must never be converted to an empty alpha/search success,
		// otherwise it would be billed and reported as a valid request.
		message := "Codex Responses returned an invalid SSE response"
		appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
			Platform:           RoutingPlatformForAccount(account),
			AccountID:          account.ID,
			AccountName:        account.Name,
			UpstreamStatusCode: resp.StatusCode,
			UpstreamRequestID:  resp.Header.Get("x-request-id"),
			Kind:               "failover",
			Message:            message,
		})
		return nil, &UpstreamFailoverError{
			StatusCode:             http.StatusBadGateway,
			ResponseBody:           buildOpenAINonJSONSuccessFailoverBody(),
			ResponseHeaders:        resp.Header.Clone(),
			RetryableOnSameAccount: false,
			TempUnscheduleAccount:  true,
			Message:                message,
		}
	}
	converted := convertResponsesSSEToAlphaSearchJSON(respBody)
	c.Data(http.StatusOK, "application/json", converted)
	return &OpenAIAlphaSearchForwardResult{RequestID: resp.Header.Get("x-request-id"), StatusCode: http.StatusOK, ResponseHeaders: resp.Header.Clone(), UpstreamEndpoint: EndpointResponses, Duration: time.Since(startTime)}, nil
}

// isValidResponsesSSEForAlphaSearch verifies that a body contains at least
// one parseable Responses SSE event.  It intentionally accepts any valid
// event type so future Responses event additions do not become false errors,
// while still rejecting JSON/HTML bodies that happen to have a 2xx status.
func isValidResponsesSSEForAlphaSearch(body []byte) bool {
	text := strings.ReplaceAll(string(body), "\r\n", "\n")
	for _, block := range strings.Split(text, "\n\n") {
		for _, line := range strings.Split(block, "\n") {
			if !strings.HasPrefix(line, "data:") {
				continue
			}
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if data == "" || data == "[DONE]" {
				continue
			}
			var event map[string]any
			if json.Unmarshal([]byte(data), &event) == nil && strings.TrimSpace(gjson.GetBytes([]byte(data), "type").String()) != "" {
				return true
			}
		}
	}
	return false
}

func buildAlphaSearchResponsesWebSearchBody(alphaBody []byte, model string) ([]byte, error) {
	tool := map[string]any{"type": "web_search"}
	if raw := gjson.GetBytes(alphaBody, "settings.search_context_size"); raw.Exists() {
		var value any
		if json.Unmarshal([]byte(raw.Raw), &value) == nil {
			tool["search_context_size"] = value
		}
	}
	if raw := gjson.GetBytes(alphaBody, "settings.user_location"); raw.IsObject() {
		var value map[string]any
		if json.Unmarshal([]byte(raw.Raw), &value) == nil && len(value) > 0 {
			tool["user_location"] = value
		}
	}
	payload := map[string]any{
		"model": model, "stream": true, "store": false,
		"input": []any{map[string]any{
			"role":    "user",
			"content": []any{map[string]any{"type": "input_text", "text": alphaSearchResponsesPrompt(alphaBody)}},
		}},
		"tools": []any{tool},
	}
	return json.Marshal(payload)
}

func alphaSearchResponsesPrompt(body []byte) string {
	var b strings.Builder
	_, _ = b.WriteString("Execute this Codex standalone web.run request for another model.\n")
	_, _ = b.WriteString("Use the hosted web_search tool when current information is needed.\n")
	_, _ = b.WriteString("Return concise source-backed results with titles, URLs, dates, and direct answers when available.")
	appendJSON := func(label, path string, limit int) {
		raw := strings.TrimSpace(gjson.GetBytes(body, path).Raw)
		if raw == "" || raw == "null" {
			return
		}
		_, _ = b.WriteString("\n\n")
		_, _ = b.WriteString(label)
		_, _ = b.WriteString("\n")
		_, _ = b.WriteString(truncateAlphaSearchPrompt(raw, limit))
	}
	appendJSON("Commands JSON:", "commands", 12000)
	appendJSON("Search settings JSON:", "settings", 4000)
	appendJSON("Recent conversation/input JSON:", "input", 8000)
	return b.String()
}

func truncateAlphaSearchPrompt(value string, limit int) string {
	if limit > 0 && len(value) > limit {
		return value[:limit] + "\n...<truncated>"
	}
	return value
}

func convertResponsesSSEToAlphaSearchJSON(body []byte) []byte {
	output, results := parseResponsesSSEForAlphaSearch(body)
	result := map[string]any{"output": output}
	if len(results) > 0 {
		result["results"] = results
	}
	encoded, _ := json.Marshal(result)
	return encoded
}

func parseResponsesSSEForAlphaSearch(body []byte) (string, []map[string]any) {
	text := strings.ReplaceAll(string(body), "\r\n", "\n")
	var output strings.Builder
	var completed any
	results := make([]map[string]any, 0)
	seen := make(map[string]struct{})
	for _, block := range strings.Split(text, "\n\n") {
		data := ""
		for _, line := range strings.Split(block, "\n") {
			if strings.HasPrefix(line, "data:") {
				if data != "" {
					data += "\n"
				}
				data += strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			}
		}
		data = strings.TrimSpace(data)
		if data == "" || data == "[DONE]" {
			continue
		}
		var event map[string]any
		if json.Unmarshal([]byte(data), &event) != nil {
			continue
		}
		if event["type"] == "response.output_text.delta" {
			if delta, ok := event["delta"].(string); ok {
				_, _ = output.WriteString(delta)
			}
		}
		if event["type"] == "response.completed" {
			completed = event["response"]
		}
		collectAlphaSearchCitations(event, &results, seen)
	}
	if strings.TrimSpace(output.String()) == "" && completed != nil {
		_, _ = output.WriteString(extractCompletedResponseText(completed))
		collectAlphaSearchCitations(completed, &results, seen)
	}
	return output.String(), results
}

func extractCompletedResponseText(response any) string {
	root, ok := response.(map[string]any)
	if !ok {
		return ""
	}
	var output strings.Builder
	items, _ := root["output"].([]any)
	for _, item := range items {
		itemMap, _ := item.(map[string]any)
		if itemMap["type"] != "message" {
			continue
		}
		content, _ := itemMap["content"].([]any)
		for _, part := range content {
			partMap, _ := part.(map[string]any)
			if partMap["type"] == "output_text" {
				if value, ok := partMap["text"].(string); ok {
					_, _ = output.WriteString(value)
				}
			}
		}
	}
	return output.String()
}

func collectAlphaSearchCitations(value any, results *[]map[string]any, seen map[string]struct{}) {
	switch typed := value.(type) {
	case map[string]any:
		if typed["type"] == "url_citation" {
			if url, ok := typed["url"].(string); ok && strings.TrimSpace(url) != "" {
				url = strings.TrimSpace(url)
				if _, exists := seen[url]; !exists {
					seen[url] = struct{}{}
					item := map[string]any{"type": "text_result", "ref_id": fmt.Sprintf("turn0search%d", len(*results)), "url": url}
					if title, ok := typed["title"].(string); ok && strings.TrimSpace(title) != "" {
						item["title"] = strings.TrimSpace(title)
					}
					*results = append(*results, item)
				}
			}
		}
		for _, child := range typed {
			collectAlphaSearchCitations(child, results, seen)
		}
	case []any:
		for _, child := range typed {
			collectAlphaSearchCitations(child, results, seen)
		}
	}
}

func stripOpenAIAlphaSearchResponsesHeaders(headers http.Header) {
	for _, key := range []string{"OpenAI-Beta", "Session_ID", "Conversation_ID", "X-Codex-Beta-Features", "X-Codex-Turn-State", responsesLiteHeader} {
		headers.Del(key)
	}
}

func (s *OpenAIGatewayService) shouldFailoverAlphaSearchResponse(account *Account, statusCode int, upstreamMsg string, upstreamBody []byte) bool {
	if s.shouldFailoverOpenAIUpstreamResponse(statusCode, upstreamMsg, upstreamBody) {
		return true
	}
	return account != nil && account.IsOpenAIApiKey() && (statusCode == http.StatusNotFound || statusCode == http.StatusMethodNotAllowed)
}
