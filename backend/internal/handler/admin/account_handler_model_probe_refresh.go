package admin

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

const accountModelProbeRefreshTimeout = 45 * time.Second

func (h *AccountHandler) scheduleAccountModelProbeRefresh(before *service.Account, after *service.Account, trigger string) {
	if h == nil || h.accountModelImportService == nil || after == nil || after.ID <= 0 {
		return
	}
	if !shouldScheduleAccountModelProbeRefresh(before, after) {
		return
	}

	accountID := after.ID
	trigger = strings.TrimSpace(trigger)
	if trigger == "" {
		trigger = "save"
	}
	go h.runAccountModelProbeRefresh(accountID, trigger)
}

func shouldScheduleAccountModelProbeRefresh(before *service.Account, after *service.Account) bool {
	if after == nil || after.ID <= 0 {
		return false
	}
	if before == nil {
		return true
	}
	return accountModelProbeRefreshSignature(before) != accountModelProbeRefreshSignature(after)
}

func accountModelProbeRefreshSignature(account *service.Account) string {
	if account == nil {
		return ""
	}

	var scopeMap map[string]any
	if scope, ok := service.ExtractAccountModelScopeV2(account.Extra); ok && scope != nil {
		scopeMap = scope.ToMap()
	}

	allowSourceProtocol := service.IsProtocolGatewayAccount(account)
	payload := map[string]any{
		"platform":                   strings.TrimSpace(strings.ToLower(account.Platform)),
		"type":                       strings.TrimSpace(strings.ToLower(account.Type)),
		"proxy_id":                   accountModelProbeRefreshProxyID(account),
		"model_mapping":              account.GetModelMapping(),
		"model_scope_v2":             scopeMap,
		"gateway_protocol":           service.GetAccountGatewayProtocol(account),
		"gateway_accepted_protocols": service.GetAccountGatewayAcceptedProtocols(account),
		"manual_models": service.AccountManualModelsToExtraValue(
			service.AccountManualModelsFromExtra(account.Extra, allowSourceProtocol),
			allowSourceProtocol,
		),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	return string(data)
}

func accountModelProbeRefreshProxyID(account *service.Account) int64 {
	if account == nil || account.ProxyID == nil {
		return 0
	}
	return *account.ProxyID
}

func (h *AccountHandler) runAccountModelProbeRefresh(accountID int64, trigger string) {
	ctx, cancel := context.WithTimeout(context.Background(), accountModelProbeRefreshTimeout)
	defer cancel()

	slog.Info(
		"account_model_probe_refresh_started",
		"account_id", accountID,
		"trigger", trigger,
	)

	account, err := h.adminService.GetAccount(ctx, accountID)
	if err != nil || account == nil {
		slog.Warn(
			"account_model_probe_refresh_failed",
			"account_id", accountID,
			"trigger", trigger,
			"stage", "load_account",
			"error", err,
		)
		return
	}

	probe, err := h.accountModelImportService.ProbeAccountModels(ctx, account)
	if err != nil {
		slog.Warn(
			"account_model_probe_refresh_failed",
			"account_id", accountID,
			"trigger", trigger,
			"stage", "probe_models",
			"error", err,
		)
		return
	}

	updatedAt := time.Now().UTC()
	probeSource := strings.TrimSpace(firstNonEmptyString(probe.ProbeSource, service.AccountModelProbeSnapshotSourcePolicyUpdate))
	updates := service.BuildAccountModelAvailabilitySnapshotExtra(
		service.BuildAccountModelProjection(ctx, account, h.modelRegistryService),
		probe.DetectedModels,
		updatedAt,
		service.AccountModelProbeSnapshotSourcePolicyUpdate,
		probeSource,
	)
	if account.IsOpenAIOAuth() {
		updates = service.MergeStringAnyMap(
			service.BuildOpenAIKnownModelsExtra(
				probe.DetectedModels,
				updatedAt,
				service.OpenAIKnownModelsSourceModelMapping,
			),
			updates,
		)
	}

	appliedUpdates := len(updates) > 0
	if appliedUpdates {
		mergedExtra := service.MergeStringAnyMap(account.Extra, updates)
		if _, err := h.adminService.UpdateAccount(ctx, account.ID, &service.UpdateAccountInput{Extra: mergedExtra}); err != nil {
			slog.Warn(
				"account_model_probe_refresh_failed",
				"account_id", accountID,
				"trigger", trigger,
				"stage", "persist_snapshot",
				"error", err,
			)
			return
		}
	}

	slog.Info(
		"account_model_probe_refresh_succeeded",
		"account_id", accountID,
		"trigger", trigger,
		"applied_updates", appliedUpdates,
		"detected_model_count", len(probe.DetectedModels),
		"probe_source", probeSource,
	)
}
