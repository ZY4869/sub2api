package admin

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *ModelCatalogHandler) RunDebug(c *gin.Context) {
	if h == nil || h.modelDebugService == nil {
		response.Error(c, http.StatusServiceUnavailable, "model debug service is unavailable")
		return
	}

	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "admin authentication required")
		return
	}

	var req service.ModelDebugRunInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	req.AdminUserID = subject.UserID
	req.BaseURL = modelDebugBaseURL(c.Request)
	if clientRequestID, _ := c.Request.Context().Value(ctxkey.ClientRequestID).(string); strings.TrimSpace(clientRequestID) != "" {
		req.ClientRequestID = strings.TrimSpace(clientRequestID)
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)
	c.Writer.Flush()

	send := func(event string, payload any) error {
		body, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		if _, err := c.Writer.WriteString("event: " + event + "\n"); err != nil {
			return err
		}
		if _, err := c.Writer.WriteString("data: " + string(body) + "\n\n"); err != nil {
			return err
		}
		c.Writer.Flush()
		return nil
	}

	_ = h.modelDebugService.Run(c.Request.Context(), req, send)
}

func modelDebugBaseURL(req *http.Request) string {
	if req == nil {
		return ""
	}
	scheme := strings.TrimSpace(req.Header.Get("X-Forwarded-Proto"))
	if scheme == "" {
		if req.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	host := strings.TrimSpace(req.Host)
	if host == "" && req.URL != nil {
		host = strings.TrimSpace(req.URL.Host)
	}
	if host == "" {
		return ""
	}
	return scheme + "://" + host
}
