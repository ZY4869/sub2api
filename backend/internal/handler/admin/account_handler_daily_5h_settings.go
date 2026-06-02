package admin

import (
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *AccountHandler) GetDaily5HTriggerSettings(c *gin.Context) {
	if h.settingService == nil {
		response.ErrorFrom(c, infraerrors.New(http.StatusInternalServerError, "SETTING_SERVICE_UNAVAILABLE", "setting service unavailable"))
		return
	}
	settings, err := h.settingService.GetAccountDaily5HTriggerSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.AccountDaily5HTriggerSettingsView{
		Settings:   accountDaily5HSettingsDTO(settings),
		Candidates: accountDaily5HCandidateDTOs(h.settingService.ListDaily5HTriggerCandidates(c.Request.Context())),
	})
}

func (h *AccountHandler) UpdateDaily5HTriggerSettings(c *gin.Context) {
	if h.settingService == nil {
		response.ErrorFrom(c, infraerrors.New(http.StatusInternalServerError, "SETTING_SERVICE_UNAVAILABLE", "setting service unavailable"))
		return
	}
	var req dto.AccountDaily5HTriggerSettings
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequestKey(c, "admin.account.invalid_request", "Invalid request: %s", err.Error())
		return
	}
	updated, err := h.settingService.UpdateAccountDaily5HTriggerSettings(c.Request.Context(), accountDaily5HSettingsFromDTO(req))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.AccountDaily5HTriggerSettingsView{
		Settings:   accountDaily5HSettingsDTO(updated),
		Candidates: accountDaily5HCandidateDTOs(h.settingService.ListDaily5HTriggerCandidates(c.Request.Context())),
	})
}

func accountDaily5HSettingsDTO(settings *service.AccountDaily5HTriggerSettings) dto.AccountDaily5HTriggerSettings {
	if settings == nil {
		settings = service.DefaultAccountDaily5HTriggerSettings()
	}
	return dto.AccountDaily5HTriggerSettings{
		Enabled:               settings.Enabled,
		SelectedAccountTypes:  append([]string(nil), settings.SelectedAccountTypes...),
		IncludePausedAccounts: settings.IncludePausedAccounts,
		IgnoreFreeAccounts:    settings.IgnoreFreeAccounts,
		OpenAIModelMode:       accountDaily5HModelSettingsDTO(settings.OpenAIModel),
		AnthropicModelMode:    accountDaily5HModelSettingsDTO(settings.AnthropicModel),
		GeminiModelMode:       accountDaily5HModelSettingsDTO(settings.GeminiModel),
	}
}

func accountDaily5HModelSettingsDTO(settings service.AccountDaily5HTriggerModelSettings) dto.AccountDaily5HTriggerModelSettings {
	return dto.AccountDaily5HTriggerModelSettings{
		Mode:         settings.Mode,
		FixedModelID: settings.FixedModelID,
	}
}

func accountDaily5HSettingsFromDTO(req dto.AccountDaily5HTriggerSettings) *service.AccountDaily5HTriggerSettings {
	return &service.AccountDaily5HTriggerSettings{
		Enabled:               req.Enabled,
		SelectedAccountTypes:  append([]string(nil), req.SelectedAccountTypes...),
		IncludePausedAccounts: req.IncludePausedAccounts,
		IgnoreFreeAccounts:    req.IgnoreFreeAccounts,
		OpenAIModel: service.AccountDaily5HTriggerModelSettings{
			Mode:         req.OpenAIModelMode.Mode,
			FixedModelID: req.OpenAIModelMode.FixedModelID,
		},
		AnthropicModel: service.AccountDaily5HTriggerModelSettings{
			Mode:         req.AnthropicModelMode.Mode,
			FixedModelID: req.AnthropicModelMode.FixedModelID,
		},
		GeminiModel: service.AccountDaily5HTriggerModelSettings{
			Mode:         req.GeminiModelMode.Mode,
			FixedModelID: req.GeminiModelMode.FixedModelID,
		},
	}
}

func accountDaily5HCandidateDTOs(items []service.AccountDaily5HTriggerAccountTypeSummary) []dto.AccountDaily5HTriggerAccountTypeSummary {
	if len(items) == 0 {
		return []dto.AccountDaily5HTriggerAccountTypeSummary{}
	}
	out := make([]dto.AccountDaily5HTriggerAccountTypeSummary, 0, len(items))
	for _, item := range items {
		models := make([]dto.AccountDaily5HTriggerModelOption, 0, len(item.Models))
		for _, model := range item.Models {
			models = append(models, dto.AccountDaily5HTriggerModelOption{
				ModelID:       model.ModelID,
				DisplayName:   model.DisplayName,
				Provider:      model.Provider,
				ProviderLabel: model.ProviderLabel,
				AccountCount:  model.AccountCount,
			})
		}
		out = append(out, dto.AccountDaily5HTriggerAccountTypeSummary{
			AccountType: item.AccountType,
			Count:       item.Count,
			Models:      models,
		})
	}
	return out
}
