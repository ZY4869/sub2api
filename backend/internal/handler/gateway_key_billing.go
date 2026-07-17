package handler

import (
	"net/http"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type keyBillingInfoResponse struct {
	Object                  string    `json:"object"`
	SchemaVersion           int       `json:"schema_version"`
	BillingScope            string    `json:"billing_scope"`
	GroupRateMultiplier     float64   `json:"group_rate_multiplier"`
	UserRateMultiplier      *float64  `json:"user_rate_multiplier,omitempty"`
	ResolvedRateMultiplier  float64   `json:"resolved_rate_multiplier"`
	PeakRateEnabled         bool      `json:"peak_rate_enabled"`
	PeakStart               *string   `json:"peak_start,omitempty"`
	PeakEnd                 *string   `json:"peak_end,omitempty"`
	PeakRateMultiplier      *float64  `json:"peak_rate_multiplier,omitempty"`
	AppliedPeakMultiplier   *float64  `json:"applied_peak_multiplier,omitempty"`
	EffectiveRateMultiplier float64   `json:"effective_rate_multiplier"`
	Timezone                *string   `json:"timezone,omitempty"`
	ObservedAt              time.Time `json:"observed_at"`
}

func (h *GatewayHandler) KeyBillingInfo(c *gin.Context) {
	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok || apiKey == nil {
		h.errorResponse(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return
	}
	if h == nil || h.gatewayService == nil {
		h.errorResponse(c, http.StatusInternalServerError, "api_error", "Billing metadata service is unavailable")
		return
	}
	group, groupID := keyBillingGroup(apiKey)
	if group == nil || groupID <= 0 {
		h.errorResponse(c, http.StatusForbidden, "permission_error", "API key is not bound to an available group")
		return
	}

	groupRate := group.RateMultiplier
	resolvedRate := h.gatewayService.ResolveUserGroupRateMultiplier(c.Request.Context(), apiKey.UserID, groupID, groupRate)
	response := buildKeyBillingInfo(apiKey, resolvedRate, time.Now())

	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, response)
}

func keyBillingGroup(apiKey *service.APIKey) (*service.Group, int64) {
	if apiKey == nil {
		return nil, 0
	}
	if apiKey.Group != nil {
		if apiKey.GroupID != nil {
			return apiKey.Group, *apiKey.GroupID
		}
		return apiKey.Group, apiKey.Group.ID
	}
	if binding := apiKey.PrimaryGroupBinding(); binding != nil && binding.Group != nil {
		return binding.Group, binding.GroupID
	}
	return nil, 0
}

func buildKeyBillingInfo(apiKey *service.APIKey, resolvedRate float64, now time.Time) keyBillingInfoResponse {
	group, _ := keyBillingGroup(apiKey)
	groupRate := 1.0
	if group != nil {
		groupRate = group.RateMultiplier
	}
	if now.IsZero() {
		now = time.Now()
	}
	effectiveRate := resolvedRate
	if group != nil {
		effectiveRate = group.EffectiveTokenRateMultiplierAt(resolvedRate, now)
	}
	resp := keyBillingInfoResponse{
		Object:                  "sub2api.key_billing",
		SchemaVersion:           1,
		BillingScope:            "token",
		GroupRateMultiplier:     groupRate,
		ResolvedRateMultiplier:  resolvedRate,
		EffectiveRateMultiplier: effectiveRate,
		ObservedAt:              now.UTC(),
	}
	if group != nil && resolvedRate != groupRate {
		resp.UserRateMultiplier = keyBillingFloat64Ptr(resolvedRate)
	}
	if group != nil && group.PeakRateEnabled {
		resp.PeakRateEnabled = true
		resp.PeakStart = keyBillingStringPtr(group.PeakStart)
		resp.PeakEnd = keyBillingStringPtr(group.PeakEnd)
		resp.PeakRateMultiplier = keyBillingFloat64Ptr(group.PeakRateMultiplier)
		tz := timezone.Location().String()
		resp.Timezone = &tz
		if group.IsPeakRateActiveAt(now) {
			resp.AppliedPeakMultiplier = keyBillingFloat64Ptr(group.PeakRateMultiplier)
		}
	}
	return resp
}

func keyBillingStringPtr(value string) *string {
	return &value
}

func keyBillingFloat64Ptr(value float64) *float64 {
	return &value
}
