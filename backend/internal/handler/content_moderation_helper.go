package handler

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/protocolruntime"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

func submitContentModerationAudit(
	ctx context.Context,
	moderationService *service.ContentModerationService,
	input *service.ContentModerationRecordInput,
) {
	if moderationService == nil || input == nil {
		return
	}
	content := service.ExtractModerationTextFromJSONBody([]byte(input.Content))
	if strings.TrimSpace(content) == "" {
		return
	}

	recordInput := *input
	recordInput.Content = content
	go func(localCtx context.Context, localInput service.ContentModerationRecordInput) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.FromContext(localCtx).With(
					zap.String("component", "handler.content_moderation"),
					zap.Any("panic", recovered),
					zap.String("source_endpoint", strings.TrimSpace(localInput.SourceEndpoint)),
				).Error("content moderation audit task panicked")
			}
		}()
		moderationService.RecordAudit(localCtx, &localInput)
	}(ctx, recordInput)
}

func checkContentModerationKeywordBlock(
	ctx context.Context,
	moderationService *service.ContentModerationService,
	input *service.ContentModerationRecordInput,
) (*service.ContentModerationKeywordDecision, error) {
	if moderationService == nil || input == nil {
		return nil, nil
	}
	content := service.ExtractModerationTextFromJSONBody([]byte(input.Content))
	if strings.TrimSpace(content) == "" {
		return nil, nil
	}
	recordInput := *input
	recordInput.Content = content
	service.RecordContentModerationRepeatedPromptSignal(&recordInput, content, time.Now().UTC())
	decision, err := moderationService.CheckBlock(ctx, &recordInput)
	if err != nil || decision == nil || !decision.Blocked {
		return nil, err
	}
	return decision, nil
}

func contentModerationOpenAIBlockResponse(c *gin.Context, decision *service.ContentModerationKeywordDecision) {
	code, message := contentModerationBlockError(decision)
	recordContentModerationBlock(c, decision)
	c.JSON(http.StatusForbidden, gin.H{
		"error": gin.H{
			"type":    "policy_violation",
			"code":    code,
			"message": message,
		},
	})
}

func contentModerationAnthropicBlockResponse(c *gin.Context, decision *service.ContentModerationKeywordDecision) {
	code, message := contentModerationBlockError(decision)
	recordContentModerationBlock(c, decision)
	c.JSON(http.StatusForbidden, gin.H{
		"type": "error",
		"error": gin.H{
			"type":    "policy_error",
			"code":    code,
			"message": message,
		},
	})
}

func contentModerationGeminiBlockResponse(c *gin.Context, decision *service.ContentModerationKeywordDecision) {
	code, message := contentModerationBlockError(decision)
	messageKey := "gateway.gemini.content_policy_keyword_blocked"
	if code == service.ContentModerationErrorCodeUnavailableBlocked {
		messageKey = "gateway.gemini.content_moderation_unavailable_blocked"
	}
	recordContentModerationBlock(c, decision)
	googleErrorWithReason(c, http.StatusForbidden, code, messageKey, message)
}

func contentModerationBlockError(decision *service.ContentModerationKeywordDecision) (string, string) {
	if decision != nil && strings.TrimSpace(decision.ErrorReason) == service.ContentModerationReasonModerationUnavailable {
		return service.ContentModerationErrorCodeUnavailableBlocked, "Request blocked because content moderation is temporarily unavailable"
	}
	return "content_policy_blocked", "Request blocked by content policy"
}

func recordContentModerationBlock(c *gin.Context, decision *service.ContentModerationKeywordDecision) {
	reason := "content_policy_blocked"
	if decision != nil && strings.TrimSpace(decision.ErrorReason) != "" {
		reason = strings.TrimSpace(decision.ErrorReason)
	}
	code, _ := contentModerationBlockError(decision)
	protocolruntime.RecordContentModerationBlock(reason)
	logger.FromContext(c.Request.Context()).Warn(
		"content moderation blocked request",
		zap.String("reason", reason),
		zap.String("code", code),
	)
}

func buildContentModerationRecordInput(c *gin.Context, sourceEndpoint, provider, model string, body []byte) *service.ContentModerationRecordInput {
	if c == nil || len(body) == 0 {
		return nil
	}
	ctx := context.Background()
	if c.Request != nil {
		ctx = c.Request.Context()
	}
	record := &service.ContentModerationRecordInput{
		SourceEndpoint:  strings.TrimSpace(sourceEndpoint),
		Provider:        strings.TrimSpace(provider),
		Model:           firstNonEmptyHandlerString(model, gjson.GetBytes(body, "model").String()),
		Content:         string(body),
		RequestID:       service.ContentModerationRequestIDFromContext(ctx),
		ClientRequestID: service.ContentModerationClientRequestIDFromContext(ctx),
	}
	if subject, ok := middleware2.GetAuthSubjectFromContext(c); ok && subject.UserID > 0 {
		userID := subject.UserID
		record.UserID = &userID
	}
	if apiKey, ok := middleware2.GetAPIKeyFromContext(c); ok && apiKey != nil && apiKey.ID > 0 {
		apiKeyID := apiKey.ID
		record.APIKeyID = &apiKeyID
	}
	return record
}
