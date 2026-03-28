package service

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
)

func (s *AccountTestService) testGrokAccountConnection(c *gin.Context, account *Account, modelID string) error {
	if account == nil {
		return s.sendErrorAndEnd(c, "Account not found")
	}

	requestedModel := strings.TrimSpace(modelID)
	if requestedModel == "" {
		defaultModels := GrokVisibleModelIDsForAccount(account)
		if len(defaultModels) > 0 {
			requestedModel = defaultModels[0]
		}
	}

	s.sendEvent(c, TestEvent{Type: "test_start", Model: requestedModel})

	if account.IsGrokAPIKey() {
		return s.testGrokAPIKeyConnection(c, account, requestedModel)
	}
	if account.IsGrokSSO() {
		return s.testGrokSSOConnection(c, account, requestedModel)
	}

	return s.sendErrorAndEnd(c, fmt.Sprintf("Unsupported Grok account type: %s", account.Type))
}

func (s *AccountTestService) testGrokAPIKeyConnection(c *gin.Context, account *Account, requestedModel string) error {
	if s.accountModelImportService == nil {
		return s.sendErrorAndEnd(c, "Grok model probe service is not configured")
	}

	probe, err := s.accountModelImportService.ProbeAccountModels(c.Request.Context(), account)
	if err != nil {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Grok API connectivity failed: %s", err.Error()))
	}

	s.sendEvent(c, TestEvent{Type: "content", Text: fmt.Sprintf("xAI official API connectivity OK (%s)", account.GetBaseURL())})
	if probe != nil && strings.TrimSpace(probe.ProbeNotice) != "" {
		s.sendEvent(c, TestEvent{Type: "content", Text: probe.ProbeNotice})
	}
	if len(probe.DetectedModels) > 0 {
		s.sendEvent(c, TestEvent{Type: "content", Text: fmt.Sprintf("Detected %d Grok models", len(probe.DetectedModels))})
		s.sendEvent(c, TestEvent{Type: "content", Text: fmt.Sprintf("Sample models: %s", strings.Join(limitStringSlice(probe.DetectedModels, 5), ", "))})
	}
	if requestedModel != "" && len(probe.DetectedModels) > 0 && !containsNormalizedString(probe.DetectedModels, requestedModel) {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Requested model %s is not available for this Grok API key", requestedModel))
	}

	s.sendEvent(c, TestEvent{Type: "test_complete", Success: true})
	return nil
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

	s.sendEvent(c, TestEvent{Type: "content", Text: "Reverse runtime connectivity probe started"})
	s.sendEvent(c, TestEvent{Type: "content", Text: fmt.Sprintf("Tier: %s", probe.Tier)})
	s.sendEvent(c, TestEvent{Type: "content", Text: fmt.Sprintf("Capabilities: heavy=%t, video=%s/%ds", probe.Capabilities.AllowHeavyModel, probe.Capabilities.VideoMaxResolution, probe.Capabilities.VideoMaxDurationSeconds)})
	if len(probe.CapabilityModels) > 0 {
		s.sendEvent(c, TestEvent{Type: "content", Text: fmt.Sprintf("Capability-derived models: %s", strings.Join(limitStringSlice(probe.CapabilityModels, 8), ", "))})
	}
	if len(probe.VisibleModels) > 0 {
		s.sendEvent(c, TestEvent{Type: "content", Text: fmt.Sprintf("Visible models after model_mapping: %s", strings.Join(limitStringSlice(probe.VisibleModels, 8), ", "))})
	}
	if err != nil {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Grok reverse runtime probe failed: %s", err.Error()))
	}

	s.sendEvent(c, TestEvent{Type: "content", Text: fmt.Sprintf("Reverse runtime connectivity OK (requested=%s mapped=%s)", probe.RequestedModel, probe.MappedModel)})
	if probe.ResponseID != "" {
		s.sendEvent(c, TestEvent{Type: "content", Text: fmt.Sprintf("Upstream response id: %s", probe.ResponseID)})
	}
	if probe.ConversationID != "" {
		s.sendEvent(c, TestEvent{Type: "content", Text: fmt.Sprintf("Conversation id: %s", probe.ConversationID)})
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
