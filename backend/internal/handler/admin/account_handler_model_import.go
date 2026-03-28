package admin

import (
	"context"
	"errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"time"
)

func (h *AccountHandler) ImportModels(c *gin.Context) {
	if h.accountModelImportService == nil {
		response.InternalError(c, "Account model import service unavailable")
		return
	}
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid account ID")
		return
	}
	var req ImportAccountModelsRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	ctx := c.Request.Context()
	account, err := h.adminService.GetAccount(ctx, accountID)
	if err != nil || account == nil {
		response.NotFound(c, "Account not found")
		return
	}
	result, err := h.accountModelImportService.ImportAccountModels(ctx, account, req.Trigger, req.Models)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if account.IsOpenAIOAuth() {
		updates := service.BuildOpenAIKnownModelsExtra(
			result.DetectedModels,
			time.Now().UTC(),
			service.OpenAIKnownModelsSourceImportModels,
		)
		mergedExtra := service.MergeStringAnyMap(account.Extra, updates)
		if _, updateErr := h.adminService.UpdateAccount(ctx, account.ID, &service.UpdateAccountInput{Extra: mergedExtra}); updateErr != nil {
			slog.Warn(
				"openai_known_models_snapshot_update_failed",
				"account_id", account.ID,
				"source", service.OpenAIKnownModelsSourceImportModels,
				"error", updateErr,
			)
		}
	}
	response.Success(c, result)
}

type probeProtocolGatewayModelsRequest struct {
	GatewayProtocol   string   `json:"gateway_protocol" binding:"required,oneof=openai anthropic gemini mixed"`
	AcceptedProtocols []string `json:"accepted_protocols"`
	BaseURL           string   `json:"base_url"`
	APIKey            string   `json:"api_key" binding:"required"`
	ProxyID           *int64   `json:"proxy_id"`
}

type probeProtocolGatewayModelItem struct {
	ID             string `json:"id"`
	DisplayName    string `json:"display_name"`
	RegistryState  string `json:"registry_state"`
	RegistryModel  string `json:"registry_model_id,omitempty"`
	SourceProtocol string `json:"source_protocol,omitempty"`
}

type probeProtocolGatewayModelsResponse struct {
	ProbeSource string                          `json:"probe_source"`
	ProbeNotice string                          `json:"probe_notice,omitempty"`
	Models      []probeProtocolGatewayModelItem `json:"models"`
}

func (h *AccountHandler) ProbeProtocolGatewayModels(c *gin.Context) {
	if h.accountModelImportService == nil {
		response.InternalError(c, "Account model import service unavailable")
		return
	}
	var req probeProtocolGatewayModelsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	descriptor, ok := service.ProtocolGatewayDescriptorByID(req.GatewayProtocol)
	if !ok {
		response.BadRequest(c, "Invalid gateway protocol")
		return
	}

	draftAccount := &service.Account{
		Name:     "protocol-gateway-probe",
		Platform: service.PlatformProtocolGateway,
		Type:     service.AccountTypeAPIKey,
		Status:   service.StatusActive,
		Credentials: map[string]any{
			"api_key":  strings.TrimSpace(req.APIKey),
			"base_url": strings.TrimSpace(req.BaseURL),
		},
		Extra: map[string]any{
			"gateway_protocol": descriptor.ID,
			"gateway_accepted_protocols": service.NormalizeGatewayAcceptedProtocols(descriptor.ID, map[string]any{
				"gateway_accepted_protocols": req.AcceptedProtocols,
			}),
		},
	}
	if strings.TrimSpace(req.BaseURL) == "" {
		draftAccount.Credentials["base_url"] = descriptor.DefaultBaseURL
	}
	if req.ProxyID != nil {
		draftAccount.ProxyID = req.ProxyID
		if proxy, err := h.adminService.GetProxy(c.Request.Context(), *req.ProxyID); err == nil && proxy != nil {
			draftAccount.Proxy = proxy
		}
	}

	result, err := h.accountModelImportService.ProbeAccountModels(c.Request.Context(), draftAccount)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	items := make([]probeProtocolGatewayModelItem, 0, len(result.Models))
	for _, detectedModel := range result.Models {
		modelID := detectedModel.ID
		item := probeProtocolGatewayModelItem{
			ID:             modelID,
			DisplayName:    detectedModel.DisplayName,
			RegistryState:  "missing",
			SourceProtocol: detectedModel.SourceProtocol,
		}
		if h.modelRegistryService != nil {
			if detail, detailErr := h.modelRegistryService.GetDetail(c.Request.Context(), modelID); detailErr == nil && detail != nil {
				item.RegistryState = "existing"
				item.RegistryModel = detail.ID
			} else if resolution, resolutionErr := h.modelRegistryService.ExplainResolution(c.Request.Context(), modelID); resolutionErr == nil && resolution != nil {
				item.RegistryState = "existing"
				item.RegistryModel = firstNonEmptyString(resolution.EffectiveID, resolution.CanonicalID, resolution.Entry.ID)
			}
		}
		items = append(items, item)
	}

	response.Success(c, probeProtocolGatewayModelsResponse{
		ProbeSource: result.ProbeSource,
		ProbeNotice: result.ProbeNotice,
		Models:      items,
	})
}

func (h *AccountHandler) GetAvailableModels(c *gin.Context) {
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid account ID")
		return
	}
	account, err := h.adminService.GetAccount(c.Request.Context(), accountID)
	if err != nil {
		response.NotFound(c, "Account not found")
		return
	}
	response.Success(c, service.BuildAvailableTestModels(c.Request.Context(), account, h.modelRegistryService))
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

type availableModelItem struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	DisplayName string `json:"display_name"`
	CreatedAt   string `json:"created_at"`
}

func (h *AccountHandler) defaultAvailableModels(ctx context.Context, account *service.Account) []availableModelItem {
	if account == nil {
		return []availableModelItem{}
	}
	runtimePlatform := service.RoutingPlatformForAccount(account)
	if runtimePlatform == service.PlatformSora {
		defaults := service.DefaultSoraModels(nil)
		items := make([]availableModelItem, 0, len(defaults))
		for _, model := range defaults {
			items = append(items, availableModelItem{ID: model.ID, Type: model.Type, DisplayName: model.DisplayName, CreatedAt: ""})
		}
		return items
	}
	if runtimePlatform == service.PlatformKiro {
		if h.modelRegistryService != nil {
			for _, exposures := range [][]string{{"test"}, {"runtime", "whitelist"}} {
				entries, err := h.modelRegistryService.GetModelsByPlatform(ctx, runtimePlatform, exposures...)
				if err != nil || len(entries) == 0 {
					continue
				}
				items := make([]availableModelItem, 0, len(entries))
				for _, entry := range entries {
					displayName := entry.DisplayName
					if displayName == "" {
						displayName = entry.ID
					}
					items = append(items, availableModelItem{ID: entry.ID, Type: "model", DisplayName: displayName, CreatedAt: ""})
				}
				return items
			}
		}
		catalog := service.KiroBuiltinModelCatalog()
		items := make([]availableModelItem, 0, len(catalog))
		for _, model := range catalog {
			items = append(items, availableModelItem{
				ID:          model.ID,
				Type:        model.Type,
				DisplayName: model.DisplayName,
				CreatedAt:   model.CreatedAt,
			})
		}
		return items
	}
	if h.modelRegistryService != nil {
		for _, exposures := range [][]string{{"test"}, {"runtime", "whitelist"}} {
			entries, err := h.modelRegistryService.GetModelsByPlatform(ctx, runtimePlatform, exposures...)
			if err != nil || len(entries) == 0 {
				continue
			}
			items := make([]availableModelItem, 0, len(entries))
			for _, entry := range entries {
				displayName := entry.DisplayName
				if displayName == "" {
					displayName = entry.ID
				}
				items = append(items, availableModelItem{ID: entry.ID, Type: "model", DisplayName: displayName, CreatedAt: ""})
			}
			return items
		}
	}
	defaults := service.BuildAvailableTestModels(ctx, account, nil)
	items := make([]availableModelItem, 0, len(defaults))
	for _, model := range defaults {
		items = append(items, availableModelItem{
			ID:          model.ID,
			Type:        model.Type,
			DisplayName: model.DisplayName,
			CreatedAt:   model.CreatedAt,
		})
	}
	return items
}
func filterAvailableModelsByMapping(defaults []availableModelItem, mapping map[string]string) []availableModelItem {
	if len(mapping) == 0 {
		return defaults
	}
	index := make(map[string]availableModelItem, len(defaults))
	for _, model := range defaults {
		index[model.ID] = model
	}
	items := make([]availableModelItem, 0, len(mapping))
	for requestedModel := range mapping {
		if model, ok := index[requestedModel]; ok {
			items = append(items, model)
			continue
		}
		items = append(items, availableModelItem{ID: requestedModel, Type: "model", DisplayName: requestedModel, CreatedAt: ""})
	}
	return items
}

func (h *AccountHandler) filterAvailableModelsByScope(ctx context.Context, defaults []availableModelItem, scope *service.AccountModelScopeV2) []availableModelItem {
	if h.modelRegistryService == nil || scope == nil {
		return defaults
	}
	allowed := map[string]struct{}{}
	appendAllowed := func(value string) {
		if resolved, ok, err := h.modelRegistryService.ResolveModel(ctx, value); err == nil && ok && resolved != "" {
			allowed[resolved] = struct{}{}
			return
		}
		if normalized := service.NormalizeModelCatalogModelID(value); normalized != "" {
			allowed[normalized] = struct{}{}
		}
	}
	for _, models := range scope.SupportedModelsByProvider {
		for _, modelID := range models {
			appendAllowed(modelID)
		}
	}
	for _, row := range scope.ManualMappingRows {
		appendAllowed(row.From)
		appendAllowed(row.To)
	}
	for from, to := range scope.ManualMappings {
		appendAllowed(from)
		appendAllowed(to)
	}
	if len(allowed) == 0 {
		return defaults
	}
	items := make([]availableModelItem, 0, len(defaults))
	for _, model := range defaults {
		if _, ok := allowed[model.ID]; ok {
			items = append(items, model)
		}
	}
	return items
}
