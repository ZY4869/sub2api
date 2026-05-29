package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

func (s *AccountUsageService) probeOpenAICodexSnapshot(ctx context.Context, account *Account) (map[string]any, *time.Time, error) {
	if account == nil || !isChatGPTOpenAIOAuthAccount(account) {
		return nil, nil, nil
	}
	// Probe the Codex quota headers using Codex models.
	// This avoids upstream fallbacks where non-Codex models may return shared/degenerate snapshots.
	probeModels := []string{openAICodexScopeNormal}
	if isOpenAIProPlan(account) {
		probeModels = append(probeModels, openAICodexScopeSpark)
	}

	var (
		mergedUpdates map[string]any
		mergedResetAt *time.Time
		joinedErr     error
		succeeded     bool
	)
	for _, modelID := range probeModels {
		updates, resetAt, err := s.probeOpenAICodexSnapshotForModel(ctx, account, modelID)
		if err != nil {
			joinedErr = errors.Join(joinedErr, err)
			continue
		}
		if len(updates) == 0 && resetAt == nil {
			continue
		}
		succeeded = true
		mergedUpdates = mergeStringAnyMap(mergedUpdates, updates)
		if resetAt != nil && (mergedResetAt == nil || resetAt.After(*mergedResetAt)) {
			resetAtCopy := resetAt.UTC()
			mergedResetAt = &resetAtCopy
		}
	}
	if !succeeded {
		return nil, nil, joinedErr
	}
	if isOpenAIProPlan(account) {
		maybeWarnOpenAICodexProbeDegenerate(account, mergedUpdates)
	}
	return mergedUpdates, mergedResetAt, nil
}

func (s *AccountUsageService) probeOpenAICodexSnapshotForModel(ctx context.Context, account *Account, modelID string) (map[string]any, *time.Time, error) {
	if s != nil && s.openAICodexScopeProbe != nil {
		return s.openAICodexScopeProbe(ctx, account, modelID)
	}
	if account == nil || !isChatGPTOpenAIOAuthAccount(account) {
		return nil, nil, nil
	}
	accessToken := account.GetOpenAIAccessToken()
	if accessToken == "" {
		return nil, nil, fmt.Errorf("no access token available")
	}

	probeModelID := resolveOpenAICodexProbeModelID(account, modelID)
	reqCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	reqCtx = WithOpenAICodexRequestModel(reqCtx, probeModelID)
	scope, ok := resolveOpenAICodexSnapshotScopeFromContext(reqCtx, account, probeModelID)
	if !ok {
		return nil, nil, fmt.Errorf("openai codex probe scope could not be resolved for model %q", probeModelID)
	}
	reqCtx = withOpenAICodexResolvedQuotaScope(reqCtx, scope)

	if isOpenAIProPlan(account) {
		wsUpdates, wsResetAt, wsSnapshot, wsSource, wsDiag, wsErr := s.probeOpenAICodexSnapshotForModelWS(reqCtx, account, probeModelID, scope)
		if wsErr == nil && (len(wsUpdates) > 0 || wsResetAt != nil) {
			util5h, util7d := extractCodexUsagePercentsFromUpdates(scope, wsUpdates)
			primaryUsed, secondaryUsed, primaryWindow, secondaryWindow, primaryResetAfter, secondaryResetAfter := codexProbeSnapshotLogFields(wsSnapshot)
			wsReadMessages := 0
			wsLastEventType := ""
			wsReadExitReason := ""
			if wsDiag != nil {
				wsReadMessages = wsDiag.ReadMessages
				wsLastEventType = wsDiag.LastEventType
				wsReadExitReason = wsDiag.ReadExitReason
			}
			slog.Info(
				"openai_codex_snapshot_scope_resolved",
				"account_id", account.ID,
				"requested_model", probeModelID,
				"upstream_model", probeModelID,
				"resolved_scope", scope,
				"snapshot_source", wsSource,
				"probe_transport", "ws",
				"ws_read_messages", wsReadMessages,
				"ws_last_event_type", wsLastEventType,
				"ws_read_exit_reason", wsReadExitReason,
				"x_cx_primary_used_percent", primaryUsed,
				"x_cx_secondary_used_percent", secondaryUsed,
				"x_cx_primary_window_minutes", primaryWindow,
				"x_cx_secondary_window_minutes", secondaryWindow,
				"x_cx_primary_reset_after_seconds", primaryResetAfter,
				"x_cx_secondary_reset_after_seconds", secondaryResetAfter,
				"utilization_5h_percent", util5h,
				"utilization_7d_percent", util7d,
			)
			state := s.persistOpenAICodexProbeSnapshot(reqCtx, account, wsUpdates)
			if state != nil {
				if state.AccountResetAt != nil {
					return wsUpdates, state.AccountResetAt, nil
				}
				if state.ScopeResetAt != nil {
					return wsUpdates, state.ScopeResetAt, nil
				}
			}
			return wsUpdates, wsResetAt, nil
		}
	}

	if s != nil && s.openAICodexScopeProbeHTTP != nil {
		return s.openAICodexScopeProbeHTTP(reqCtx, account, modelID)
	}
	return s.probeOpenAICodexSnapshotForModelHTTP(reqCtx, account, accessToken, probeModelID, scope)
}
