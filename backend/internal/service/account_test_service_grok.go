package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

func (s *AccountTestService) testGrokAccountConnection(c *gin.Context, account *Account, modelID string) error {
	if account == nil {
		return s.sendErrorAndEnd(c, "Account not found")
	}

	requestedModel := strings.TrimSpace(modelID)
	if requestedModel == "" {
		defaultModels := GrokDefaultTestModelIDsForAccount(account)
		if len(defaultModels) > 0 {
			requestedModel = defaultModels[0]
		}
	}

	s.sendEvent(c, TestEvent{Type: "test_start", Model: requestedModel})

	if account.IsGrokAPIKey() || account.IsGrokOAuth() {
		return s.testGrokOfficialConnection(c, account, requestedModel)
	}
	if account.IsGrokSSO() {
		return s.testGrokSSOConnection(c, account, requestedModel)
	}

	return s.sendErrorAndEnd(c, fmt.Sprintf("Unsupported Grok account type: %s", account.Type))
}

func (s *AccountTestService) testGrokOfficialConnection(c *gin.Context, account *Account, requestedModel string) error {
	if s.accountModelImportService == nil {
		return s.sendErrorAndEnd(c, "Grok model probe service is not configured")
	}

	probe, err := s.accountModelImportService.ProbeAccountModels(c.Request.Context(), account)
	if err != nil {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Grok official runtime connectivity failed: %s", err.Error()))
	}

	s.sendEvent(c, TestEvent{Type: "log", Text: fmt.Sprintf("Grok official runtime connectivity OK (%s)", account.GetBaseURL())})
	if probe != nil && strings.TrimSpace(probe.ProbeNotice) != "" {
		s.sendEvent(c, TestEvent{Type: "log", Text: probe.ProbeNotice})
	}
	if len(probe.DetectedModels) > 0 {
		s.sendEvent(c, TestEvent{Type: "log", Text: fmt.Sprintf("Detected %d Grok models", len(probe.DetectedModels))})
		s.sendEvent(c, TestEvent{Type: "log", Text: fmt.Sprintf("Sample models: %s", strings.Join(limitStringSlice(probe.DetectedModels, 5), ", "))})
	}
	if requestedModel != "" && len(probe.DetectedModels) > 0 && !containsNormalizedString(probe.DetectedModels, requestedModel) {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Requested model %s is not available for this Grok official runtime", requestedModel))
	}
	if err := s.testGrokRealResponsesCall(c, account, requestedModel); err != nil {
		return s.sendErrorAndEnd(c, err.Error())
	}

	s.sendEvent(c, TestEvent{Type: "test_complete", Success: true})
	return nil
}

func (s *AccountTestService) testGrokRealResponsesCall(c *gin.Context, account *Account, requestedModel string) error {
	if s.grokGatewayService == nil {
		return grokRealCallUserError("Grok real call service is not configured")
	}
	requestedModel = strings.TrimSpace(requestedModel)
	if requestedModel == "" {
		requestedModel = DefaultGrokBuildTextModelID()
	}
	body, err := json.Marshal(map[string]any{
		"model":  requestedModel,
		"input":  "Output exactly: OK",
		"stream": false,
	})
	if err != nil {
		return grokRealCallUserError("Grok real model call failed: failed to build request")
	}
	resp, meta, err := s.grokGatewayService.doGrokOfficialRequest(c.Request.Context(), c, account, http.MethodPost, grokEndpointResponses, body)
	if err != nil {
		return grokRealCallUserError(fmt.Sprintf("Grok real model call failed: %s", err.Error()))
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		bodyBytes, _ := readUpstreamResponseBodyLimited(resp.Body, resolveUpstreamResponseReadLimit(s.cfg))
		msg := sanitizeUpstreamErrorMessage(strings.TrimSpace(extractUpstreamErrorMessage(bodyBytes)))
		if msg == "" {
			msg = http.StatusText(resp.StatusCode)
		}
		if s.rateLimitService != nil {
			s.rateLimitService.HandleUpstreamError(c.Request.Context(), account, resp.StatusCode, resp.Header, bodyBytes)
		}
		setOpsUpstreamError(c, resp.StatusCode, msg, "")
		appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
			Platform:           PlatformGrok,
			AccountID:          account.ID,
			AccountName:        account.Name,
			UpstreamStatusCode: resp.StatusCode,
			UpstreamRequestID:  grokUpstreamRequestID(resp.Header),
			Kind:               "account_test_real_call",
			Message:            msg,
		})
		logger.FromContext(c.Request.Context()).Warn("grok.account_test_real_call_failed", grokUpstreamLogFields(account, meta, resp.StatusCode, grokUpstreamRequestID(resp.Header))...)
		return grokRealCallUserError(fmt.Sprintf("Grok real model call failed: upstream status %d: %s", resp.StatusCode, msg))
	}
	bodyBytes, err := readUpstreamResponseBodyLimited(resp.Body, resolveUpstreamResponseReadLimit(s.cfg))
	if err != nil {
		return grokRealCallUserError("Grok real model call failed: failed to read upstream response")
	}
	text := strings.TrimSpace(extractGrokResponsesText(bodyBytes))
	if text != "" {
		s.sendEvent(c, TestEvent{Type: "content", Text: text})
	}
	s.sendEvent(c, TestEvent{Type: "log", Text: fmt.Sprintf("Grok real model call OK (%s)", requestedModel)})
	return nil
}

func extractGrokResponsesText(body []byte) string {
	if len(bytes.TrimSpace(body)) == 0 {
		return ""
	}
	for _, path := range []string{
		"output_text",
		"response.output_text",
		"choices.0.message.content",
		"message.content",
	} {
		if value := strings.TrimSpace(gjson.GetBytes(body, path).String()); value != "" {
			return value
		}
	}
	output := gjson.GetBytes(body, "output")
	if output.IsArray() {
		var parts []string
		for _, item := range output.Array() {
			content := item.Get("content")
			if content.IsArray() {
				for _, contentItem := range content.Array() {
					text := strings.TrimSpace(firstNonEmptyString(
						contentItem.Get("text").String(),
						contentItem.Get("content").String(),
					))
					if text != "" {
						parts = append(parts, text)
					}
				}
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, "")
		}
	}
	return ""
}

type grokRealCallUserError string

func (e grokRealCallUserError) Error() string {
	return string(e)
}

func (s *AccountTestService) testGrokSSOConnection(c *gin.Context, account *Account, requestedModel string) error {
	if s.grokGatewayService == nil {
		return s.sendErrorAndEnd(c, "Grok reverse runtime is not configured")
	}

	probe, err := s.grokGatewayService.ProbeSSOAccount(c.Request.Context(), account, requestedModel)
	if probe == nil {
		probe = &GrokSSOProbeResult{
			Tier:             ResolveGrokTier(account.Extra),
			Capabilities:     ResolveGrokCapabilities(account.Extra),
			CapabilityModels: GrokCapabilityModelIDsForAccount(account),
			VisibleModels:    GrokVisibleModelIDsForAccount(account),
			RequestedModel:   strings.TrimSpace(requestedModel),
		}
	}

	s.sendEvent(c, TestEvent{Type: "status", Text: "Reverse runtime connectivity probe started"})
	s.sendEvent(c, TestEvent{Type: "log", Text: fmt.Sprintf("Tier: %s", probe.Tier)})
	s.sendEvent(c, TestEvent{Type: "log", Text: fmt.Sprintf("Capabilities: heavy=%t, video=%s/%ds", probe.Capabilities.AllowHeavyModel, probe.Capabilities.VideoMaxResolution, probe.Capabilities.VideoMaxDurationSeconds)})
	if len(probe.CapabilityModels) > 0 {
		s.sendEvent(c, TestEvent{Type: "log", Text: fmt.Sprintf("Capability-derived models: %s", strings.Join(limitStringSlice(probe.CapabilityModels, 8), ", "))})
	}
	if len(probe.VisibleModels) > 0 {
		s.sendEvent(c, TestEvent{Type: "log", Text: fmt.Sprintf("Visible models after model_mapping: %s", strings.Join(limitStringSlice(probe.VisibleModels, 8), ", "))})
	}
	if err != nil {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Grok reverse runtime probe failed: %s", err.Error()))
	}

	s.sendEvent(c, TestEvent{Type: "status", Text: fmt.Sprintf("Reverse runtime connectivity OK (requested=%s mapped=%s)", probe.RequestedModel, probe.MappedModel)})
	if probe.ResponseID != "" {
		s.sendEvent(c, TestEvent{Type: "log", Text: fmt.Sprintf("Upstream response id: %s", probe.ResponseID)})
	}
	if probe.ConversationID != "" {
		s.sendEvent(c, TestEvent{Type: "log", Text: fmt.Sprintf("Conversation id: %s", probe.ConversationID)})
	}
	s.sendEvent(c, TestEvent{Type: "test_complete", Success: true})
	return nil
}

func limitStringSlice(items []string, limit int) []string {
	if limit <= 0 || len(items) <= limit {
		return items
	}
	cloned := make([]string, limit)
	copy(cloned, items[:limit])
	return cloned
}

func containsNormalizedString(items []string, target string) bool {
	target = strings.TrimSpace(strings.ToLower(target))
	for _, item := range items {
		if strings.TrimSpace(strings.ToLower(item)) == target {
			return true
		}
	}
	return false
}
