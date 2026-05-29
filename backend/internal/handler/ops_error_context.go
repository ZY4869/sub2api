package handler

import (
	"context"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

const (
	opsModelKey            = "ops_model"
	opsStreamKey           = "ops_stream"
	opsRequestBodyKey      = "ops_request_body"
	opsAccountIDKey        = "ops_account_id"
	opsUpstreamModelKey    = "ops_upstream_model"
	opsUpstreamEndpointKey = "ops_upstream_endpoint"
	opsRequestTypeKey      = "ops_request_type"

	// 错误过滤匹配常量 — shouldSkipOpsErrorLog 和错误分类共用
	opsErrContextCanceled            = "context canceled"
	opsErrNoAvailableAccounts        = "no available accounts"
	opsErrInvalidAPIKey              = "invalid_api_key"
	opsErrAPIKeyRequired             = "api_key_required"
	opsErrInsufficientBalance        = "insufficient balance"
	opsErrInsufficientAccountBalance = "insufficient account balance"
	opsErrInsufficientQuota          = "insufficient_quota"

	// 上游错误码常量 — 错误分类 (normalizeOpsErrorType / classifyOpsPhase / classifyOpsIsBusinessLimited)
	opsCodeInsufficientBalance  = "INSUFFICIENT_BALANCE"
	opsCodeUsageLimitExceeded   = "USAGE_LIMIT_EXCEEDED"
	opsCodeSubscriptionNotFound = "SUBSCRIPTION_NOT_FOUND"
	opsCodeSubscriptionInvalid  = "SUBSCRIPTION_INVALID"
	opsCodeUserInactive         = "USER_INACTIVE"
)

func setOpsRequestContext(c *gin.Context, model string, stream bool, requestBody []byte) {
	if c == nil {
		return
	}
	model = strings.TrimSpace(model)
	rawModel := model
	if len(requestBody) > 0 {
		if parsedModel := strings.TrimSpace(gjson.GetBytes(requestBody, "model").String()); parsedModel != "" {
			rawModel = parsedModel
		}
	}
	c.Set(opsModelKey, model)
	c.Set(opsStreamKey, stream)
	if len(requestBody) > 0 {
		c.Set(opsRequestBodyKey, requestBody)
	}
	if c.Request != nil && len(requestBody) > 0 {
		payloadHash := service.HashUsageRequestPayload(requestBody)
		if payloadHash != "" {
			ctx := context.WithValue(c.Request.Context(), ctxkey.RequestPayloadHash, payloadHash)
			c.Request = c.Request.WithContext(ctx)
		}
	}
	if c.Request != nil && rawModel != "" {
		ctx := service.EnsureRequestMetadata(c.Request.Context())
		ctx = context.WithValue(ctx, ctxkey.Model, rawModel)
		c.Request = c.Request.WithContext(ctx)
		return
	}
	if c.Request != nil {
		c.Request = c.Request.WithContext(service.EnsureRequestMetadata(c.Request.Context()))
	}
}

func setOpsEndpointContext(c *gin.Context, upstreamModel string, requestType service.RequestType) {
	if c == nil {
		return
	}
	if upstreamModel = strings.TrimSpace(upstreamModel); upstreamModel != "" {
		c.Set(opsUpstreamModelKey, upstreamModel)
	}
	if normalized := requestType.Normalize(); normalized != service.RequestTypeUnknown {
		c.Set(opsRequestTypeKey, int16(normalized))
	}
}

func attachOpsRequestBodyToEntry(c *gin.Context, entry *service.OpsInsertErrorLogInput) {
	if c == nil || entry == nil {
		return
	}
	v, ok := c.Get(opsRequestBodyKey)
	if !ok {
		return
	}
	raw, ok := v.([]byte)
	if !ok || len(raw) == 0 {
		return
	}
	entry.RequestBodyJSON, entry.RequestBodyTruncated, entry.RequestBodyBytes = service.PrepareOpsRequestBodyForQueue(raw)
	opsErrorLogSanitized.Add(1)
}

func setOpsSelectedAccount(c *gin.Context, accountID int64, platform ...string) {
	if c == nil || accountID <= 0 {
		return
	}
	c.Set(opsAccountIDKey, accountID)
	if c.Request != nil {
		ctx := context.WithValue(c.Request.Context(), ctxkey.AccountID, accountID)
		if len(platform) > 0 {
			p := strings.TrimSpace(platform[0])
			if p != "" {
				ctx = context.WithValue(ctx, ctxkey.Platform, p)
			}
		}
		c.Request = c.Request.WithContext(ctx)
	}
}

func setOpsUpstreamEndpoint(c *gin.Context, endpoint string) {
	if c == nil {
		return
	}
	if trimmed := strings.TrimSpace(endpoint); trimmed != "" {
		c.Set(opsUpstreamEndpointKey, trimmed)
	}
}

func setOpsSelectedAccountDetails(c *gin.Context, account *service.Account) {
	if account == nil {
		return
	}
	setOpsSelectedAccount(c, account.ID, account.Platform)
	setOpsUpstreamEndpoint(c, GetUpstreamEndpointForAccount(c, account))
}

func resolveOpsUpstreamEndpoint(c *gin.Context, platform string) string {
	if c != nil {
		if value, ok := c.Get(opsUpstreamEndpointKey); ok {
			if endpoint, ok := value.(string); ok && strings.TrimSpace(endpoint) != "" {
				return strings.TrimSpace(endpoint)
			}
		}
	}
	return GetUpstreamEndpoint(c, platform)
}

func resolveOpsUpstreamModel(c *gin.Context) string {
	if c == nil {
		return ""
	}
	if v, ok := c.Get(opsUpstreamModelKey); ok {
		if s, ok := v.(string); ok {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

func resolveOpsRequestType(c *gin.Context, stream bool) *int16 {
	if c != nil {
		if v, ok := c.Get(opsRequestTypeKey); ok {
			switch t := v.(type) {
			case int16:
				return &t
			case int:
				value := int16(t)
				return &value
			}
		}
	}

	requestType := service.RequestTypeFromLegacy(stream, false)
	if c != nil && c.Request != nil && isOpenAIWSUpgradeRequest(c.Request) {
		requestType = service.RequestTypeWSV2
	}
	if requestType == service.RequestTypeUnknown {
		return nil
	}
	value := int16(requestType)
	return &value
}

func latestOpsUpstreamURL(events []*service.OpsUpstreamErrorEvent) string {
	for i := len(events) - 1; i >= 0; i-- {
		if events[i] == nil {
			continue
		}
		if url := strings.TrimSpace(events[i].UpstreamURL); url != "" {
			return url
		}
	}
	return ""
}
