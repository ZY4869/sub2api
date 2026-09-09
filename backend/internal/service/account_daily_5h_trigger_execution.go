package service

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/protocolruntime"
)

const (
	accountDaily5HTaskDateKey  = "daily_5h_trigger_task_date"
	accountDaily5HAttemptsKey  = "daily_5h_trigger_attempts"
	accountDaily5HNextRetryKey = "daily_5h_trigger_next_retry_at"
	accountDaily5HStoppedKey   = "daily_5h_trigger_retry_stopped"
	accountDaily5HMaxAttempts  = 3
)

func accountDaily5HLocation() *time.Location {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.FixedZone("Asia/Shanghai", 8*60*60)
	}
	return location
}

func (s *AccountDaily5HTriggerService) stopped(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	case <-s.stopCh:
		return true
	default:
		return false
	}
}

func (s *AccountDaily5HTriggerService) dueSettings(ctx context.Context) (*AccountDaily5HTriggerSettings, time.Time) {
	now := s.now().In(s.location)
	settings, err := s.settingService.GetAccountDaily5HTriggerSettings(ctx)
	if err != nil {
		slog.Warn("account_daily_5h_trigger_settings_failed", "error", err)
		return nil, now
	}
	if settings == nil || !settings.Enabled {
		return nil, now
	}
	clock, err := time.Parse("15:04", settings.TriggerTime)
	if err != nil || settings.TriggerTime != clock.Format("15:04") {
		return nil, now
	}
	due := time.Date(now.Year(), now.Month(), now.Day(), clock.Hour(), clock.Minute(), 0, 0, s.location)
	if now.Before(due) || (settings.SkipCNHolidaysAndWeekends && accountDaily5HShouldSkipCNNonWorkday(now)) {
		return nil, now
	}
	return settings, now
}

func (s *AccountDaily5HTriggerService) runOnce(ctx context.Context) {
	if !s.runMu.TryLock() {
		return
	}
	defer s.runMu.Unlock()
	if s.stopped(ctx) {
		return
	}
	settings, now := s.dueSettings(ctx)
	if settings == nil {
		return
	}
	localDate := now.Format("2006-01-02")
	accounts, err := s.listManagedAccounts(ctx)
	if err != nil {
		slog.Warn("account_daily_5h_trigger_list_failed", "error", err)
		return
	}
	for start := 0; start < len(accounts); start += 4 {
		if s.stopped(ctx) {
			return
		}
		settings, now = s.dueSettings(ctx)
		if settings == nil || now.Format("2006-01-02") != localDate {
			return
		}
		batch := accounts[start:min(start+4, len(accounts))]
		run := func(batchCtx context.Context) {
			batchCtx, cancel := context.WithTimeout(batchCtx, 50*time.Second)
			defer cancel()
			var wg sync.WaitGroup
			for _, account := range batch {
				if s.stopped(batchCtx) {
					break
				}
				wg.Add(1)
				go func(id int64) {
					defer wg.Done()
					// Reload after earlier batches: lifecycle, windows and policy may have changed.
					current, err := s.accountRepo.GetByID(batchCtx, id)
					if err != nil || current == nil {
						slog.Warn("account_daily_5h_trigger_reload_failed", "account_id", id, "error", err)
						return
					}
					s.runAccount(batchCtx, settings, current, localDate)
				}(account.ID)
			}
			wg.Wait()
		}
		// A batch is bounded by 45-second requests; renew the existing lease before each batch.
		if s.leaderGate != nil {
			if !s.leaderGate.RunIfLeader(ctx, accountDaily5HTriggerJobName, periodicJobLeaderTTL(s.interval), run) {
				return
			}
		} else {
			run(ctx)
		}
	}
}

func daily5HAttempts(extra map[string]any, localDate string) int {
	if stringValueFromAny(extra[accountDaily5HTaskDateKey]) != localDate {
		return 0
	}
	return parseExtraInt(extra[accountDaily5HAttemptsKey])
}

func daily5HSucceeded(extra map[string]any, localDate string) bool {
	if AccountDaily5HLastLocalDate(extra) != localDate {
		return false
	}
	status := strings.TrimSpace(stringValueFromAny(extra[accountDaily5HLastStatusKey]))
	// Old records containing only a completion date remain idempotent.
	return status == "" || status == AccountDaily5HTriggerStatusSuccess
}

func (s *AccountDaily5HTriggerService) saveDaily5HState(ctx context.Context, id int64, state map[string]any) bool {
	if err := s.accountRepo.UpdateExtra(ctx, id, state); err != nil {
		slog.Warn("account_daily_5h_trigger_update_failed", "account_id", id, "error", err)
		return false
	}
	return true
}

func daily5HWaitingUntil(account *Account, reason string) *time.Time {
	switch reason {
	case AccountDaily5HSkipReasonRateLimited:
		return account.RateLimitResetAt
	case AccountDaily5HSkipReasonTempUnsched:
		return account.TempUnschedulableUntil
	case AccountDaily5HSkipReasonOverloaded:
		return account.OverloadUntil
	case AccountDaily5HSkipReasonSessionWindow:
		return account.SessionWindowEnd
	}
	return nil
}

func (s *AccountDaily5HTriggerService) runAccount(ctx context.Context, settings *AccountDaily5HTriggerSettings, account *Account, localDate string) {
	now := s.now().UTC()
	if s.stopped(ctx) || daily5HSucceeded(account.Extra, localDate) {
		return
	}
	attempts := daily5HAttempts(account.Extra, localDate)
	if attempts >= accountDaily5HMaxAttempts {
		return
	}
	if stringValueFromAny(account.Extra[accountDaily5HTaskDateKey]) == localDate {
		if stopped, _ := account.Extra[accountDaily5HStoppedKey].(bool); stopped {
			return
		}
		// Only request retries are delayed here; runtime waits are re-evaluated after state changes.
		status := stringValueFromAny(account.Extra[accountDaily5HLastStatusKey])
		if status == "failed" || status == "running" {
			if retry := parseAccountExpiryProbeTime(account.Extra, accountDaily5HNextRetryKey); retry != nil && now.Before(*retry) {
				return
			}
		}
	}
	shouldRun, reason, summary := s.shouldRunForAccount(settings, account, now)
	if AccountReauthStatusFromExtra(account.Extra) != nil {
		shouldRun, reason, summary = false, "reauth_required", "Account requires reauthorization."
	}
	modelID := ""
	if shouldRun {
		modelID, reason, summary = s.selectModelForAccount(ctx, settings, account)
	}
	if !shouldRun || modelID == "" {
		state := BuildAccountDaily5HTriggerSkipExtra(localDate, reason, summary)
		state[accountDaily5HTaskDateKey] = localDate
		state[accountDaily5HAttemptsKey] = attempts
		state[accountDaily5HLastCheckedAtKey] = now.Format(time.RFC3339)
		state[accountDaily5HNextRetryKey] = nil
		state[accountDaily5HStoppedKey] = false
		if until := daily5HWaitingUntil(account, reason); until != nil {
			state[accountDaily5HLastStatusKey] = "waiting"
			state[accountDaily5HNextRetryKey] = until.UTC().Format(time.RFC3339)
		}
		// Avoid rewriting identical skipped rows every minute.
		if stringValueFromAny(account.Extra[accountDaily5HTaskDateKey]) == localDate &&
			stringValueFromAny(account.Extra[accountDaily5HLastSkipReasonKey]) == reason &&
			stringValueFromAny(account.Extra[accountDaily5HNextRetryKey]) == stringValueFromAny(state[accountDaily5HNextRetryKey]) {
			return
		}
		s.saveDaily5HState(ctx, account.ID, state)
		protocolruntime.RecordRecoveryProbeResult("daily_5h_trigger", AccountDaily5HTriggerStatusSkipped, 0)
		return
	}
	attempts++
	state := BuildAccountDaily5HTriggerExtra("", "running", modelID, "Daily 5H trigger in progress.")
	state[accountDaily5HTaskDateKey] = localDate
	state[accountDaily5HAttemptsKey] = attempts
	state[accountDaily5HLastCheckedAtKey] = now.Format(time.RFC3339)
	state[accountDaily5HNextRetryKey] = now.Add(5 * time.Minute).Format(time.RFC3339)
	state[accountDaily5HStoppedKey] = false
	if !s.saveDaily5HState(ctx, account.ID, state) || s.stopped(ctx) {
		return
	}
	started := time.Now()
	triggerCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	triggerCtx = EnsureRequestMetadata(triggerCtx)
	SetProbeActionMetadata(triggerCtx, "daily_5h_trigger")
	protocolruntime.RecordRecoveryProbeStarted("daily_5h_trigger")
	result, runErr := s.accountTestRunner.RunTestBackgroundDetailed(triggerCtx, ScheduledTestExecutionInput{
		AccountID: account.ID, ModelID: modelID, Prompt: accountDaily5HPrompt,
		TestMode: "real_forward", OperationType: UsageOperationTypeScheduledTest,
	})
	status, summary := AccountDaily5HTriggerStatusSuccess, "Daily 5H trigger succeeded."
	state[accountDaily5HNextRetryKey] = nil
	if runErr != nil || result == nil || !strings.EqualFold(strings.TrimSpace(result.Status), "success") {
		status = AccountDaily5HTriggerStatusFailed
		summary = firstNonEmptyString(runErrString(runErr), resultErrorMessage(result), "Daily 5H trigger failed.")
		terminal := result != nil && (result.NeedsReauth || (result.CurrentLifecycleState != "" && NormalizeAccountLifecycleInput(result.CurrentLifecycleState) != AccountLifecycleNormal))
		state[accountDaily5HStoppedKey] = terminal
		if !terminal && attempts < accountDaily5HMaxAttempts {
			delay := 5 * time.Minute
			if attempts == 2 {
				delay = 15 * time.Minute
			}
			state[accountDaily5HNextRetryKey] = s.now().UTC().Add(delay).Format(time.RFC3339)
			protocolruntime.RecordRecoveryProbeRetry("daily_5h_trigger")
		}
	} else {
		state[accountDaily5HLastLocalDateKey] = localDate
		protocolruntime.RecordRecoveryProbeSuccess("daily_5h_trigger")
	}
	state[accountDaily5HLastStatusKey] = status
	state[accountDaily5HLastSummaryKey] = summary
	state[accountDaily5HLastCheckedAtKey] = s.now().UTC().Format(time.RFC3339)
	s.saveDaily5HState(ctx, account.ID, state)
	protocolruntime.RecordRecoveryProbeResult("daily_5h_trigger", status, time.Since(started).Milliseconds())
	slog.Info("account_daily_5h_trigger_result", "request_id", requestIDFromContext(triggerCtx), "account_id", account.ID,
		"local_date", localDate, "status", status, "model_id", modelID, "attempt", attempts)
}
