package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *SettingHandler) GetRectifierSettings(c *gin.Context) {
	settings, err := h.settingService.GetRectifierSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.RectifierSettings{Enabled: settings.Enabled, ThinkingSignatureEnabled: settings.ThinkingSignatureEnabled, ThinkingBudgetEnabled: settings.ThinkingBudgetEnabled})
}

type UpdateRectifierSettingsRequest struct {
	Enabled                  bool `json:"enabled"`
	ThinkingSignatureEnabled bool `json:"thinking_signature_enabled"`
	ThinkingBudgetEnabled    bool `json:"thinking_budget_enabled"`
}

func (h *SettingHandler) UpdateRectifierSettings(c *gin.Context) {
	var req UpdateRectifierSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	settings := &service.RectifierSettings{Enabled: req.Enabled, ThinkingSignatureEnabled: req.ThinkingSignatureEnabled, ThinkingBudgetEnabled: req.ThinkingBudgetEnabled}
	if err := h.settingService.SetRectifierSettings(c.Request.Context(), settings); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	updatedSettings, err := h.settingService.GetRectifierSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.RectifierSettings{Enabled: updatedSettings.Enabled, ThinkingSignatureEnabled: updatedSettings.ThinkingSignatureEnabled, ThinkingBudgetEnabled: updatedSettings.ThinkingBudgetEnabled})
}
func (h *SettingHandler) GetBetaPolicySettings(c *gin.Context) {
	settings, err := h.settingService.GetBetaPolicySettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	rules := make([]dto.BetaPolicyRule, len(settings.Rules))
	for i, r := range settings.Rules {
		rules[i] = dto.BetaPolicyRule(r)
	}
	response.Success(c, dto.BetaPolicySettings{Rules: rules})
}

type UpdateBetaPolicySettingsRequest struct {
	Rules []dto.BetaPolicyRule `json:"rules"`
}
