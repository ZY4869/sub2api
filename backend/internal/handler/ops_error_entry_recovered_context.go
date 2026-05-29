package handler

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type recoveredOpsUpstreamContext struct {
	events               []*service.OpsUpstreamErrorEvent
	accountID            *int64
	upstreamStatusCode   *int
	upstreamErrorMessage *string
	upstreamErrorDetail  *string
	effectiveStatus      int
	messageForLog        string
}

func collectRecoveredOpsUpstreamContext(c *gin.Context) (recoveredOpsUpstreamContext, bool) {
	var recovered recoveredOpsUpstreamContext
	if v, ok := c.Get(service.OpsUpstreamErrorsKey); ok {
		if arr, ok := v.([]*service.OpsUpstreamErrorEvent); ok && len(arr) > 0 {
			recovered.events = arr
		}
	}
	if !hasRecoveredOpsUpstreamContext(c, recovered.events) {
		return recoveredOpsUpstreamContext{}, false
	}

	if len(recovered.events) > 0 {
		last := recovered.events[len(recovered.events)-1]
		if last != nil {
			if last.AccountID > 0 {
				v := last.AccountID
				recovered.accountID = &v
			}
			if last.UpstreamStatusCode > 0 {
				code := last.UpstreamStatusCode
				recovered.upstreamStatusCode = &code
			}
			if msg := strings.TrimSpace(last.Message); msg != "" {
				recovered.upstreamErrorMessage = &msg
			}
			if detail := strings.TrimSpace(last.Detail); detail != "" {
				recovered.upstreamErrorDetail = &detail
			}
		}
	}
	fillRecoveredOpsUpstreamFieldsFromContext(c, &recovered)

	// If we still have nothing meaningful, skip.
	if recovered.upstreamStatusCode == nil && recovered.upstreamErrorMessage == nil && recovered.upstreamErrorDetail == nil && len(recovered.events) == 0 {
		return recoveredOpsUpstreamContext{}, false
	}

	if recovered.upstreamStatusCode != nil {
		recovered.effectiveStatus = *recovered.upstreamStatusCode
	}
	recovered.messageForLog = buildRecoveredOpsMessage(recovered.effectiveStatus, recovered.upstreamErrorMessage)
	return recovered, true
}

func hasRecoveredOpsUpstreamContext(c *gin.Context, events []*service.OpsUpstreamErrorEvent) bool {
	if len(events) > 0 {
		return true
	}
	if v, ok := c.Get(service.OpsUpstreamStatusCodeKey); ok {
		switch t := v.(type) {
		case int:
			if t > 0 {
				return true
			}
		case int64:
			if t > 0 {
				return true
			}
		}
	}
	if v, ok := c.Get(service.OpsUpstreamErrorMessageKey); ok {
		if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
			return true
		}
	}
	if v, ok := c.Get(service.OpsUpstreamErrorDetailKey); ok {
		if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
			return true
		}
	}
	return false
}

func fillRecoveredOpsUpstreamFieldsFromContext(c *gin.Context, recovered *recoveredOpsUpstreamContext) {
	if c == nil || recovered == nil {
		return
	}
	if recovered.upstreamStatusCode == nil {
		if v, ok := c.Get(service.OpsUpstreamStatusCodeKey); ok {
			switch t := v.(type) {
			case int:
				if t > 0 {
					code := t
					recovered.upstreamStatusCode = &code
				}
			case int64:
				if t > 0 {
					code := int(t)
					recovered.upstreamStatusCode = &code
				}
			}
		}
	}
	if recovered.upstreamErrorMessage == nil {
		if v, ok := c.Get(service.OpsUpstreamErrorMessageKey); ok {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				msg := strings.TrimSpace(s)
				recovered.upstreamErrorMessage = &msg
			}
		}
	}
	if recovered.upstreamErrorDetail == nil {
		if v, ok := c.Get(service.OpsUpstreamErrorDetailKey); ok {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				detail := strings.TrimSpace(s)
				recovered.upstreamErrorDetail = &detail
			}
		}
	}
}
