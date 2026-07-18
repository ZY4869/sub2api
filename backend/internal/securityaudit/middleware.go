package securityaudit

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	mw "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func GatewayMiddleware(service *PromptService, protocol string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if service == nil || c.Request == nil || c.Request.Body == nil {
			c.Next()
			return
		}
		body, err := io.ReadAll(c.Request.Body)
		_ = c.Request.Body.Close()
		c.Request.Body = io.NopCloser(bytes.NewReader(body))
		if err != nil || len(bytes.TrimSpace(body)) == 0 {
			c.Next()
			return
		}
		req := requestFromGin(c, protocol, body)
		result := service.Evaluate(c.Request.Context(), req)
		c.Request.Body = io.NopCloser(bytes.NewReader(body))
		if result.Error == nil || result.Allowed {
			c.Next()
			return
		}
		writeGatewayAuditError(c, protocol, result.Error)
	}
}

func GatewayMiddlewareWhen(service *PromptService, protocol string, shouldAudit func(*gin.Context) bool) gin.HandlerFunc {
	base := GatewayMiddleware(service, protocol)
	return func(c *gin.Context) {
		if shouldAudit != nil && !shouldAudit(c) {
			c.Next()
			return
		}
		base(c)
	}
}

func requestFromGin(c *gin.Context, protocol string, body []byte) Request {
	apiKey, _ := mw.GetAPIKeyFromContext(c)
	group := currentGroup(c, apiKey)
	requestID, _ := c.Request.Context().Value(ctxkey.RequestID).(string)
	clientRequestID, _ := c.Request.Context().Value(ctxkey.ClientRequestID).(string)
	req := Request{
		RequestID: requestID, ClientRequestID: clientRequestID, Protocol: protocol,
		Provider: providerFromGroup(group), Endpoint: c.FullPath(), Body: body, Stage: "http",
	}
	if apiKey != nil {
		req.APIKeyID = &apiKey.ID
		req.APIKeyName = apiKey.Name
		req.UserID = &apiKey.UserID
		if apiKey.User != nil {
			req.Username, req.UserEmail = apiKey.User.Username, apiKey.User.Email
		}
	}
	if group != nil {
		req.GroupID = &group.ID
		req.GroupName = group.Name
	}
	req.Model = extractModel(body)
	return req
}

func currentGroup(c *gin.Context, apiKey *service.APIKey) *service.Group {
	if apiKey != nil && apiKey.Group != nil {
		return apiKey.Group
	}
	if c != nil && c.Request != nil {
		if group, ok := c.Request.Context().Value(ctxkey.Group).(*service.Group); ok {
			return group
		}
	}
	return nil
}

func providerFromGroup(group *service.Group) string {
	if group == nil {
		return ""
	}
	return group.Platform
}

func writeGatewayAuditError(c *gin.Context, protocol string, err error) {
	status := http.StatusServiceUnavailable
	code := ErrorCodeUnavailable
	if reason := infraerrors.Reason(err); reason != "" {
		code = reason
	}
	if code == ErrorCodeBlocked {
		status, code = http.StatusForbidden, ErrorCodeBlocked
	}
	switch protocol {
	case ProtocolGemini:
		c.AbortWithStatusJSON(status, gin.H{"error": gin.H{"code": status, "message": "prompt audit blocked the request", "status": code}})
	case ProtocolAnthropic:
		c.AbortWithStatusJSON(status, gin.H{"type": "error", "error": gin.H{"type": "permission_error", "message": "prompt audit blocked the request", "code": code}})
	default:
		errorType := "service_unavailable_error"
		if status == http.StatusForbidden {
			errorType = "invalid_request_error"
		}
		c.AbortWithStatusJSON(status, gin.H{"error": gin.H{"type": errorType, "message": "prompt audit blocked the request", "code": code}})
	}
}

func extractModel(body []byte) string {
	var payload struct {
		Model string `json:"model"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	return strings.TrimSpace(payload.Model)
}
