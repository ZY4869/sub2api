package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

func (s *OpenAIGatewayService) openAIFirstOutputTimeout(reasoningEffort string) time.Duration {
	if s == nil || s.cfg == nil || s.cfg.Gateway.OpenAIFirstOutputTimeoutSeconds <= 0 {
		return 0
	}
	seconds := s.cfg.Gateway.OpenAIFirstOutputTimeoutSeconds
	switch strings.ToLower(strings.TrimSpace(reasoningEffort)) {
	case "high", "xhigh", "max":
		if override := s.cfg.Gateway.OpenAIHighEffortFirstOutputTimeoutSeconds; override > 0 {
			seconds = override
		}
	}
	return time.Duration(seconds) * time.Second
}

func (s *OpenAIGatewayService) newOpenAIFirstOutputTimeoutError(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	startTime time.Time,
	originalModel string,
	reasoningEffort string,
	timeout time.Duration,
	responseHeaders http.Header,
) *UpstreamFailoverError {
	elapsed := time.Since(startTime)
	accountID := int64(0)
	accountName := ""
	platform := PlatformOpenAI
	if account != nil {
		accountID = account.ID
		accountName = account.Name
		platform = RoutingPlatformForAccount(account)
		if strings.TrimSpace(platform) == "" {
			platform = account.Platform
		}
	}
	logger.LegacyPrintf(
		"service.openai_gateway",
		"OpenAI first output timeout: account=%d model=%s effort=%s elapsed=%s limit=%s",
		accountID, originalModel, reasoningEffort, elapsed, timeout,
	)
	appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
		Platform:           platform,
		AccountID:          accountID,
		AccountName:        accountName,
		UpstreamStatusCode: http.StatusGatewayTimeout,
		UpstreamRequestID:  strings.TrimSpace(responseHeaders.Get("x-request-id")),
		Kind:               "first_output_timeout",
		Message:            "OpenAI upstream produced no semantic output before the deadline",
		Detail:             fmt.Sprintf("elapsed_ms=%d timeout_ms=%d", elapsed.Milliseconds(), timeout.Milliseconds()),
	})
	if s != nil && s.rateLimitService != nil && account != nil {
		s.rateLimitService.HandleStreamTimeout(ctx, account, originalModel)
	}
	return &UpstreamFailoverError{
		StatusCode:            http.StatusGatewayTimeout,
		ResponseBody:          []byte(`{"error":{"type":"first_output_timeout","message":"Upstream produced no output before the deadline"}}`),
		ResponseHeaders:       responseHeaders.Clone(),
		TempUnscheduleAccount: true,
	}
}

func openAISSEDataHasSemanticOutput(data []byte) bool {
	if len(data) == 0 || bytesEqualString(data, "[DONE]") || !gjson.ValidBytes(data) {
		return false
	}
	eventType := strings.TrimSpace(gjson.GetBytes(data, "type").String())
	switch eventType {
	case "",
		"response.created",
		"response.in_progress",
		"response.queued",
		"response.started":
		return false
	case "response.output_text.delta",
		"response.refusal.delta",
		"response.function_call_arguments.delta",
		"response.reasoning_text.delta",
		"response.reasoning_summary_text.delta",
		"response.image_generation_call.partial_image",
		"response.completed",
		"response.done",
		"response.failed",
		"response.incomplete",
		"response.cancelled",
		"response.canceled":
		return true
	}
	if strings.Contains(eventType, ".delta") || strings.Contains(eventType, ".done") {
		return true
	}
	if output := gjson.GetBytes(data, "response.output"); output.Exists() && output.IsArray() && len(output.Array()) > 0 {
		return true
	}
	if choices := gjson.GetBytes(data, "choices"); choices.Exists() && choices.IsArray() {
		for _, choice := range choices.Array() {
			if strings.TrimSpace(choice.Get("delta.content").String()) != "" ||
				strings.TrimSpace(choice.Get("delta.refusal").String()) != "" ||
				choice.Get("delta.tool_calls").Exists() ||
				strings.TrimSpace(choice.Get("message.content").String()) != "" ||
				choice.Get("message.tool_calls").Exists() {
				return true
			}
		}
	}
	return false
}

func bytesEqualString(value []byte, want string) bool {
	return len(value) == len(want) && string(value) == want
}
