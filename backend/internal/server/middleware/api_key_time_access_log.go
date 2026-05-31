package middleware

import (
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func logAPIKeyTimeAccessDenied(c *gin.Context, apiKey *service.APIKey, eval service.TimeAccessEvaluation, now time.Time) {
	if c == nil || c.Request == nil || apiKey == nil {
		return
	}
	requestID, _ := c.Request.Context().Value(ctxkey.RequestID).(string)
	if strings.TrimSpace(requestID) == "" {
		requestID, _ = c.Request.Context().Value(ctxkey.ClientRequestID).(string)
	}
	userID := apiKey.UserID
	if apiKey.User != nil && apiKey.User.ID > 0 {
		userID = apiKey.User.ID
	}
	logger.FromContext(c.Request.Context()).Warn(
		"api key time access denied",
		zap.String("component", "audit.api_key_time_access"),
		zap.String("scope", apiKeyTimeAccessDeniedScope(apiKey, now)),
		zap.String("reason", strings.TrimSpace(eval.Reason)),
		zap.Int64("api_key_id", apiKey.ID),
		zap.Int64("user_id", userID),
		zap.String("request_id", strings.TrimSpace(requestID)),
	)
}

func apiKeyTimeAccessDeniedScope(apiKey *service.APIKey, now time.Time) string {
	if apiKey == nil {
		return "api_key"
	}
	if apiKey.StartsAt != nil && now.Before(*apiKey.StartsAt) {
		return "api_key"
	}
	if apiKey.User != nil {
		eval := service.EvaluateTimeAccessPolicy(apiKey.User.APIKeyAccessTimePolicy, now)
		if !eval.Allowed {
			return "user"
		}
	}
	return "api_key"
}
