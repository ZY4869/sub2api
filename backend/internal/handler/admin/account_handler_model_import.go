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
		response.InternalErrorKey(c, "admin.account.model_import_service_missing", "Account model import service unavailable")
		return
	}
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequestKey(c, "admin.account.invalid_id", "Invalid account ID")
		return
	}
	var req ImportAccountModelsRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		response.BadRequestKey(c, "admin.account.invalid_request", "Invalid request: %s", err.Error())
		return
	}
	ctx := c.Request.Context()
	account, err := h.adminService.GetAccount(ctx, accountID)
	if err != nil || account == nil {
		response.NotFound(c, response.LocalizedMessage(c, "admin.account.not_found", "Account not found"))
		return
	}
	result, err := h.accountModelImportService.ImportAccountModels(ctx, account, req.Trigger, req.Models)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if len(result.DetectedModels) > 0 {
		updatedAt := time.Now().UTC()
		updates := service.BuildAccountModelAvailabilitySnapshotExtra(
			service.BuildAccountModelProjection(ctx, account, h.modelRegistryService),
			result.DetectedModels,
			updatedAt,
			service.AccountModelProbeSnapshotSourceImportModels,
			result.ProbeSource,
		)
		if account.IsOpenAIOAuth() {
			updates = service.MergeStringAnyMap(
				service.BuildOpenAIKnownModelsExtra(
					result.DetectedModels,
					updatedAt,
					service.OpenAIKnownModelsSourceImportModels,
				),
				updates,
			)
		}
		mergedExtra := service.MergeStringAnyMap(account.Extra, updates)
		if _, updateErr := h.adminService.UpdateAccount(ctx, account.ID, &service.UpdateAccountInput{Extra: mergedExtra}); updateErr != nil {
			slog.Warn(
				"account_model_probe_snapshot_update_failed",
				"account_id", account.ID,
				"source", service.AccountModelProbeSnapshotSourceImportModels,
				"error", updateErr,
			)
		}
	}
	response.Success(c, result)
}

type probeProtocolGatewayModelsRequest struct {
	GatewayProtocol   string                       `json:"gateway_protocol" binding:"required,oneof=openai anthropic gemini mixed"`
	AcceptedProtocols []string                     `json:"accepted_protocols"`
	BaseURL           string                       `json:"base_url"`
	APIKey            string                       `json:"api_key" binding:"required"`
	ProxyID           *int64                       `json:"proxy_id"`
	TargetProvider    string                       `json:"target_provider"`
	TargetModelID     string                       `json:"target_model_id"`
	ManualModels      []service.AccountManualModel `json:"manual_models"`
}

type probeProtocolGatewayModelItem struct {
	ID                 string `json:"id"`
	DisplayName        string `json:"display_name"`
	Provider           string `json:"provider,omitempty"`
	ProviderLabel      string `json:"provider_label,omitempty"`
	RegistryState      string `json:"registry_state"`
	RegistryModel      string `json:"registry_model_id,omitempty"`
	SourceProtocol     string `json:"source_protocol,omitempty"`
	UpstreamSource     string `json:"upstream_source,omitempty"`
	Availability       string `json:"availability,omitempty"`
	AvailabilityReason string `json:"availability_reason,omitempty"`
}

type probeProtocolGatewayModelsResponse struct {
	ProbeSource             string                          `json:"probe_source"`
	ProbeNotice             string                          `json:"probe_notice,omitempty"`
	ResolvedUpstreamURL     string                          `json:"resolved_upstream_url,omitempty"`
	ResolvedUpstreamHost    string                          `json:"resolved_upstream_host,omitempty"`
	ResolvedUpstreamService string                          `json:"resolved_upstream_service,omitempty"`
	Models                  []probeProtocolGatewayModelItem `json:"models"`
}

type probeAccountModelsRequest struct {
	Platform     string                       `json:"platform" binding:"required"`
	Type         string                       `json:"type" binding:"required"`
	Credentials  map[string]any               `json:"credentials"`
	Extra        map[string]any               `json:"extra"`
	ManualModels []service.AccountManualModel `json:"manual_models"`
	ProxyID      *int64                       `json:"proxy_id"`
}

func (h *AccountHandler) ProbeModels(c *gin.Context) {
	if h.accountModelImportService == nil {
		response.InternalErrorKey(c, "admin.account.model_import_service_missing", "Account model import service unavailable")
		return
	}
	var req probeAccountModelsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequestKey(c, "admin.account.invalid_request", "Invalid request: %s", err.Error())
		return
	}

	draftAccount := &service.Account{
		Name:        "account-probe",
		Platform:    strings.TrimSpace(req.Platform),
		Type:        strings.TrimSpace(req.Type),
		Status:      service.StatusActive,
		Credentials: service.MergeStringAnyMap(nil, req.Credentials),
		Extra:       service.MergeStringAnyMap(nil, req.Extra),
	}
	if strings.EqualFold(draftAccount.Platform, service.PlatformGemini) {
		draftAccount.Credentials = service.NormalizeGeminiCredentialsForStorage(draftAccount.Type, draftAccount.Credentials)
	}
	if manualModels := service.AccountManualModelsToExtraValue(req.ManualModels, service.IsProtocolGatewayAccount(draftAccount)); len(manualModels) > 0 {
		if draftAccount.Extra == nil {
			draftAccount.Extra = map[string]any{}
		}
		draftAccount.Extra["manual_models"] = manualModels
	}
	if req.ProxyID != nil {
		draftAccount.ProxyID = req.ProxyID
		if proxy, err := h.adminService.GetProxy(c.Request.Context(), *req.ProxyID); err == nil && proxy != nil {
			draftAccount.Proxy = proxy
		}
	}

	h.writeProbeModelsResponse(c, draftAccount)
}

func (h *AccountHandler) ProbeProtocolGatewayModels(c *gin.Context) {
	if h.accountModelImportService == nil {
		response.InternalErrorKey(c, "admin.account.model_import_service_missing", "Account model import service unavailable")
		return
	}
	var req probeProtocolGatewayModelsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequestKey(c, "admin.account.invalid_request", "Invalid request: %s", err.Error())
		return
	}
	descriptor, ok := service.ProtocolGatewayDescriptorByID(req.GatewayProtocol)
	if !ok {
		response.BadRequestKey(c, "admin.account.gateway_protocol_invalid", "Invalid gateway protocol")
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
	if provider := service.NormalizeModelProvider(req.TargetProvider); provider != "" {
		draftAccount.Extra["gateway_test_provider"] = provider
	}
	if modelID := strings.TrimSpace(req.TargetModelID); modelID != "" {
		draftAccount.Extra["gateway_test_model_id"] = modelID
	}
	if manualModels := service.AccountManualModelsToExtraValue(req.ManualModels, true); len(manualModels) > 0 {
		draftAccount.Extra["manual_models"] = manualModels
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

	h.writeProbeModelsResponse(c, draftAccount)
}

func (h *AccountHandler) writeProbeModelsResponse(c *gin.Context, draftAccount *service.Account) {
	result, err := h.accountModelImportService.ProbeAccountModels(c.Request.Context(), draftAccount)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	items := make([]probeProtocolGatewayModelItem, 0, len(result.Models))
	for _, detectedModel := range result.Models {
		modelID := detectedModel.ID
		item := probeProtocolGatewayModelItem{
			ID:                 modelID,
			DisplayName:        detectedModel.DisplayName,
			Provider:           detectedModel.Provider,
			ProviderLabel:      detectedModel.ProviderLabel,
			RegistryState:      "missing",
			SourceProtocol:     detectedModel.SourceProtocol,
			UpstreamSource:     detectedModel.UpstreamSource,
			Availability:       detectedModel.Availability,
			AvailabilityReason: detectedModel.AvailabilityReason,
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
		ProbeSource:             result.ProbeSource,
		ProbeNotice:             result.ProbeNotice,
		ResolvedUpstreamURL:     result.ResolvedUpstreamURL,
		ResolvedUpstreamHost:    result.ResolvedUpstreamHost,
		ResolvedUpstreamService: result.ResolvedUpstreamService,
		Models:                  items,
	})
}

func (h *AccountHandler) GetAvailableModels(c *gin.Context) {
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequestKey(c, "admin.account.invalid_id", "Invalid account ID")
		return
	}
	account, err := h.adminService.GetAccount(c.Request.Context(), accountID)
	if err != nil {
		response.NotFound(c, response.LocalizedMessage(c, "admin.account.not_found", "Account not found"))
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
