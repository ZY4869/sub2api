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
		defaultModels := DefaultGrokModelIDsForTier(ResolveGrokTier(account.Extra))
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
	token := strings.TrimSpace(account.GetGrokSSOToken())
	if token == "" {
		return s.sendErrorAndEnd(c, "Grok SSO account is missing sso_token")
	}

	tier := ResolveGrokTier(account.Extra)
	capabilities := ResolveGrokCapabilities(account.Extra)
	if requestedModel != "" && IsGrokHeavyModel(requestedModel) && !capabilities.AllowHeavyModel {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Requested model %s requires heavy tier", requestedModel))
	}

	s.sendEvent(c, TestEvent{Type: "content", Text: "SSO token present and normalized"})
	s.sendEvent(c, TestEvent{Type: "content", Text: fmt.Sprintf("Tier: %s", tier)})
	s.sendEvent(c, TestEvent{Type: "content", Text: fmt.Sprintf("Capabilities: heavy=%t, video=%s/%ds", capabilities.AllowHeavyModel, capabilities.VideoMaxResolution, capabilities.VideoMaxDurationSeconds)})
	models := DefaultGrokModelIDsForTier(tier)
	if len(models) > 0 {
		s.sendEvent(c, TestEvent{Type: "content", Text: fmt.Sprintf("Capability-derived models: %s", strings.Join(limitStringSlice(models, 6), ", "))})
	}
	s.sendEvent(c, TestEvent{Type: "content", Text: "Reverse runtime validation is pending in the current build; this check verifies the stored capability profile only"})
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
