package handler

import (
	"context"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
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
) (bool, error) {
	if moderationService == nil || input == nil {
		return false, nil
	}
	content := service.ExtractModerationTextFromJSONBody([]byte(input.Content))
	if strings.TrimSpace(content) == "" {
		return false, nil
	}
	recordInput := *input
	recordInput.Content = content
	decision, err := moderationService.CheckBlock(ctx, &recordInput)
	if err != nil || decision == nil {
		return false, err
	}
	return decision.Blocked, nil
}

func contentModerationOpenAIBlockResponse(c *gin.Context) {
	c.JSON(http.StatusForbidden, gin.H{
		"error": gin.H{
			"type":    "policy_violation",
			"code":    "content_policy_keyword_blocked",
			"message": "Request blocked by local content policy keywords",
		},
	})
}

func contentModerationAnthropicBlockResponse(c *gin.Context) {
	c.JSON(http.StatusForbidden, gin.H{
		"type": "error",
		"error": gin.H{
			"type":    "policy_error",
			"message": "Request blocked by local content policy keywords",
		},
	})
}

func buildContentModerationRecordInput(c *gin.Context, sourceEndpoint, provider, model string, body []byte) *service.ContentModerationRecordInput {
	if c == nil || len(body) == 0 {
		return nil
	}
	record := &service.ContentModerationRecordInput{
		SourceEndpoint: strings.TrimSpace(sourceEndpoint),
		Provider:       strings.TrimSpace(provider),
		Model:          strings.TrimSpace(model),
		Content:        string(body),
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
