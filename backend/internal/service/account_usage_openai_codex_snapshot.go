package service

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

func extractCodexUsagePercentsFromUpdates(scope string, updates map[string]any) (float64, float64) {
	if len(updates) == 0 {
		return 0, 0
	}
	switch strings.TrimSpace(scope) {
	case openAICodexScopeSpark:
		return parseExtraFloat64(updates[codexSpark5hUsedPercentKey]), parseExtraFloat64(updates[codexSpark7dUsedPercentKey])
	default:
		return parseExtraFloat64(updates["codex_5h_used_percent"]), parseExtraFloat64(updates["codex_7d_used_percent"])
	}
}

func (s *AccountUsageService) persistOpenAICodexProbeSnapshot(ctx context.Context, account *Account, updates map[string]any) *openAICodexRateLimitState {
	if s == nil || s.accountRepo == nil || account == nil || account.ID <= 0 || len(updates) == 0 {
		return nil
	}
	updateCtx, updateCancel := context.WithTimeout(ctx, 5*time.Second)
	defer updateCancel()
	return syncOpenAICodexRateLimitState(updateCtx, s.accountRepo, account, updates, time.Now())
}

func resolveOpenAICodexProbeModelID(account *Account, modelID string) string {
	modelID = strings.TrimSpace(modelID)
	if account == nil || modelID == "" {
		return modelID
	}
	if modelID != openAICodexScopeSpark {
		return modelID
	}
	mapped, matched := account.ResolveMappedModel(modelID)
	mapped = strings.TrimSpace(mapped)
	if !matched || mapped == "" {
		return modelID
	}
	if normalizeOpenAICodexQuotaScope(mapped) != openAICodexScopeSpark {
		slog.Info("openai_codex_spark_probe_mapping_ignored", "account_id", account.ID, "mapped_model", mapped)
		return modelID
	}
	return mapped
}

func extractOpenAICodexProbeSnapshotForScope(resp *http.Response, scope string) (map[string]any, *time.Time, string, error) {
	if resp == nil {
		return nil, nil, "", nil
	}
	if snapshot := ParseCodexRateLimitHeaders(resp.Header); snapshot != nil {
		baseTime := time.Now()
		updates := buildCodexUsageExtraUpdatesForScope(scope, snapshot, baseTime)
		resetAt := codexRateLimitResetAtFromSnapshot(snapshot, baseTime)
		reason := codexRateLimitReasonFromSnapshot(snapshot)
		if len(updates) > 0 {
			return updates, resetAt, reason, nil
		}
		return nil, resetAt, reason, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, nil, "", fmt.Errorf("openai codex probe returned status %d", resp.StatusCode)
	}
	return nil, nil, "", nil
}

func extractOpenAICodexProbeSnapshot(resp *http.Response) (map[string]any, *time.Time, string, error) {
	return extractOpenAICodexProbeSnapshotForScope(resp, openAICodexScopeNormal)
}

func extractOpenAICodexProbeUpdatesForScope(resp *http.Response, scope string) (map[string]any, error) {
	updates, _, _, err := extractOpenAICodexProbeSnapshotForScope(resp, scope)
	return updates, err
}

func extractOpenAICodexProbeUpdates(resp *http.Response) (map[string]any, error) {
	updates, _, _, err := extractOpenAICodexProbeSnapshot(resp)
	return updates, err
}

func mergeAccountExtra(account *Account, updates map[string]any) {
	if account == nil || len(updates) == 0 {
		return
	}
	if account.Extra == nil {
		account.Extra = make(map[string]any, len(updates))
	}
	for k, v := range updates {
		account.Extra[k] = v
	}
}

func buildCodexUsageProgressFromExtra(extra map[string]any, window string, now time.Time) *UsageProgress {
	if len(extra) == 0 {
		return nil
	}

	var (
		usedPercentKey string
		resetAfterKey  string
		resetAtKey     string
	)

	switch window {
	case "5h":
		usedPercentKey = "codex_5h_used_percent"
		resetAfterKey = "codex_5h_reset_after_seconds"
		resetAtKey = "codex_5h_reset_at"
	case "7d":
		usedPercentKey = "codex_7d_used_percent"
		resetAfterKey = "codex_7d_reset_after_seconds"
		resetAtKey = "codex_7d_reset_at"
	default:
		return nil
	}

	usedRaw, ok := extra[usedPercentKey]
	if !ok {
		return nil
	}

	progress := &UsageProgress{Utilization: parseExtraFloat64(usedRaw)}
	if resetAtRaw, ok := extra[resetAtKey]; ok {
		if resetAt, err := parseTime(fmt.Sprint(resetAtRaw)); err == nil {
			progress.ResetsAt = &resetAt
			progress.RemainingSeconds = int(time.Until(resetAt).Seconds())
			if progress.RemainingSeconds < 0 {
				progress.RemainingSeconds = 0
			}
		}
	}
	if progress.ResetsAt == nil {
		if resetAfterSeconds := parseExtraInt(extra[resetAfterKey]); resetAfterSeconds > 0 {
			base := now
			if updatedAtRaw, ok := extra["codex_usage_updated_at"]; ok {
				if updatedAt, err := parseTime(fmt.Sprint(updatedAtRaw)); err == nil {
					base = updatedAt
				}
			}
			resetAt := base.Add(time.Duration(resetAfterSeconds) * time.Second)
			progress.ResetsAt = &resetAt
			progress.RemainingSeconds = int(time.Until(resetAt).Seconds())
			if progress.RemainingSeconds < 0 {
				progress.RemainingSeconds = 0
			}
		}
	}

	// 窗口已过期（resetAt 在 now 之前）→ 额度已重置，归零
	if progress.ResetsAt != nil && !now.Before(*progress.ResetsAt) {
		progress.Utilization = 0
	}

	return progress
}
