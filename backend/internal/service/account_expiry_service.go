package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/protocolruntime"
)

const defaultAccountExpiryProbeTimeout = 45 * time.Second

type AccountExpiryService struct {
	accountRepo       AccountRepository
	accountTestRunner interface {
		RunTestBackgroundDetailed(ctx context.Context, input ScheduledTestExecutionInput) (*BackgroundAccountTestResult, error)
	}
	interval    time.Duration
	testTimeout time.Duration
	now         func() time.Time
	stopCh      chan struct{}
	stopOnce    sync.Once
	wg          sync.WaitGroup
}

func NewAccountExpiryService(
	accountRepo AccountRepository,
	accountTestRunner interface {
		RunTestBackgroundDetailed(ctx context.Context, input ScheduledTestExecutionInput) (*BackgroundAccountTestResult, error)
	},
	interval time.Duration,
) *AccountExpiryService {
	if interval <= 0 {
		interval = time.Minute
	}
	return &AccountExpiryService{
		accountRepo:       accountRepo,
		accountTestRunner: accountTestRunner,
		interval:          interval,
		testTimeout:       defaultAccountExpiryProbeTimeout,
		now:               time.Now,
		stopCh:            make(chan struct{}),
	}
}

func (s *AccountExpiryService) SetNow(now func() time.Time) {
	if s == nil || now == nil {
		return
	}
	s.now = now
}

func (s *AccountExpiryService) Start() {
	if s == nil || s.accountRepo == nil || s.accountTestRunner == nil || s.interval <= 0 {
		return
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		s.runOnce(context.Background())
		for {
			select {
			case <-ticker.C:
				s.runOnce(context.Background())
			case <-s.stopCh:
				return
			}
		}
	}()
}

func (s *AccountExpiryService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
	s.wg.Wait()
}

func (s *AccountExpiryService) runOnce(ctx context.Context) {
	accounts, err := s.listManagedAccounts(ctx)
	if err != nil {
		slog.Warn("account_expiry_probe_list_failed", "error", err)
		return
	}
	now := s.now().UTC()
	for index := range accounts {
		account := accounts[index]
		if !shouldRunAccountExpiryProbe(&account, now) {
			continue
		}
		s.runProbeForAccount(ctx, &account, now)
	}
}

func (s *AccountExpiryService) listManagedAccounts(ctx context.Context) ([]Account, error) {
	if s == nil || s.accountRepo == nil {
		return []Account{}, nil
	}
	params := pagination.PaginationParams{Page: 1, PageSize: 10000}
	accounts, _, err := s.accountRepo.ListWithFilters(ctx, params, "", "", "", "", 0, AccountLifecycleNormal, "")
	if err != nil {
		return nil, err
	}
	return accounts, nil
}

func shouldRunAccountExpiryProbe(account *Account, now time.Time) bool {
	if account == nil || !account.AutoPauseOnExpired || account.ExpiresAt == nil {
		return false
	}
	if !IsManagedRuntimeAccount(account) {
		return false
	}
	expiresAt := account.ExpiresAt.UTC()
	if expiresAt.After(now) {
		return false
	}
	if nextCheckAt := parseAccountExpiryProbeTime(account.Extra, accountExpiryProbeNextCheckAtKey); nextCheckAt != nil && nextCheckAt.After(now) {
		return false
	}
	if checkedAt := parseAccountExpiryProbeTime(account.Extra, accountExpiryProbeCheckedAtKey); checkedAt != nil && !checkedAt.Before(expiresAt) {
		if account.ExpiresAt != nil && AccountExpiryProbePriorityUntil(account) != nil {
			return false
		}
	}
	return true
}

func (s *AccountExpiryService) runProbeForAccount(ctx context.Context, account *Account, checkedAt time.Time) {
	startedAt := time.Now()
	requestID := firstNonEmptyString(requestIDFromContext(ctx), "generated:"+generateRequestID())
	if nextWindowEnd, waitingSummary := nextAccountExpiryProbeWindow(account, checkedAt); nextWindowEnd != nil {
		slog.Info(
			"account_expiry_probe_waiting_window",
			"request_id", requestID,
			"account_id", account.ID,
			"lifecycle_state", NormalizeAccountLifecycleInput(account.LifecycleState),
			"expires_at", account.ExpiresAt,
			"checked_at", checkedAt,
			"next_check_at", nextWindowEnd,
			"summary", waitingSummary,
		)
		if err := s.accountRepo.UpdateExtra(ctx, account.ID, BuildAccountExpiryProbeExtra(
			checkedAt,
			AccountExpiryProbeStatusWaiting,
			waitingSummary,
			nextWindowEnd,
			AccountExpiryProbePriorityUntil(account),
		)); err != nil {
			slog.Warn(
				"account_expiry_probe_wait_update_failed",
				"request_id", requestID,
				"account_id", account.ID,
				"lifecycle_state", NormalizeAccountLifecycleInput(account.LifecycleState),
				"expires_at", account.ExpiresAt,
				"checked_at", checkedAt,
				"next_check_at", nextWindowEnd,
				"error", err,
			)
		}
		protocolruntime.RecordRecoveryProbeResult("expiry_probe", AccountExpiryProbeStatusWaiting, time.Since(startedAt).Milliseconds())
		return
	}

	probeCtx, cancel := context.WithTimeout(ctx, s.testTimeout)
	defer cancel()
	probeCtx = EnsureRequestMetadata(probeCtx)
	SetProbeActionMetadata(probeCtx, "expiry_probe")
	protocolruntime.RecordRecoveryProbeStarted("expiry_probe")
	requestID = firstNonEmptyString(requestIDFromContext(probeCtx), requestID)

	slog.Info(
		"account_expiry_probe_started",
		"request_id", requestID,
		"account_id", account.ID,
		"lifecycle_state", NormalizeAccountLifecycleInput(account.LifecycleState),
		"expires_at", account.ExpiresAt,
		"checked_at", checkedAt,
	)
	result, err := s.accountTestRunner.RunTestBackgroundDetailed(probeCtx, ScheduledTestExecutionInput{
		AccountID:     account.ID,
		Prompt:        accountDaily5HPrompt,
		TestMode:      "real_forward",
		OperationType: UsageOperationTypeScheduledTest,
	})
	if err != nil && (result == nil || strings.TrimSpace(result.ErrorMessage) == "") {
		result = &BackgroundAccountTestResult{
			Status:       "failed",
			ErrorMessage: err.Error(),
		}
	}

	if result != nil && strings.EqualFold(strings.TrimSpace(result.Status), "success") {
		s.handleSuccessfulProbe(ctx, account, checkedAt, startedAt, requestID)
		return
	}
	s.handleFailedProbe(ctx, account, checkedAt, result, startedAt, requestID)
}

func nextAccountExpiryProbeWindow(account *Account, now time.Time) (*time.Time, string) {
	if account == nil {
		return nil, ""
	}
	for _, candidate := range []struct {
		ts      *time.Time
		summary string
	}{
		{account.RateLimitResetAt, "Account is still rate-limited; expiry probe will retry after the rate-limit window ends."},
		{account.TempUnschedulableUntil, "Account is temporarily unschedulable; expiry probe will retry after the temporary window ends."},
		{account.OverloadUntil, "Account is overloaded; expiry probe will retry after the overload window ends."},
		{account.SessionWindowEnd, "Account is inside an active session window; expiry probe will retry after the session window ends."},
	} {
		if candidate.ts != nil && now.Before(candidate.ts.UTC()) {
			ts := candidate.ts.UTC()
			return &ts, candidate.summary
		}
	}
	return nil, ""
}

func (s *AccountExpiryService) handleSuccessfulProbe(ctx context.Context, account *Account, checkedAt time.Time, startedAt time.Time, requestID string) {
	if account == nil {
		return
	}
	base := checkedAt
	if account.ExpiresAt != nil && account.ExpiresAt.After(base) {
		base = account.ExpiresAt.UTC()
	}
	extensionDays := AccountExpiryProbeExtensionDaysFromExtra(account.Extra)
	newExpiry := base.Add(time.Duration(extensionDays) * 24 * time.Hour)
	account.ExpiresAt = &newExpiry
	account.Schedulable = true
	if NormalizeAdminAccountStatusInput(account.Status) == StatusDisabled {
		account.Status = StatusActive
	}
	if account.Extra == nil {
		account.Extra = map[string]any{}
	}
	for key, value := range BuildAccountExpiryProbeExtra(
		checkedAt,
		AccountExpiryProbeStatusSuccess,
		fmt.Sprintf("Expiry probe passed and extended the account by %d day(s).", extensionDays),
		nil,
		&newExpiry,
	) {
		account.Extra[key] = value
	}
	if err := s.accountRepo.Update(ctx, account); err != nil {
		slog.Warn(
			"account_expiry_probe_failed_update",
			"request_id", requestID,
			"account_id", account.ID,
			"lifecycle_state", NormalizeAccountLifecycleInput(account.LifecycleState),
			"expires_at", account.ExpiresAt,
			"checked_at", checkedAt,
			"error", err,
		)
		return
	}
	protocolruntime.RecordRecoveryProbeSuccess("expiry_probe")
	protocolruntime.RecordRecoveryProbeResult("expiry_probe", AccountExpiryProbeStatusSuccess, time.Since(startedAt).Milliseconds())
	slog.Info(
		"account_expiry_probe_success",
		"request_id", requestID,
		"account_id", account.ID,
		"lifecycle_state", NormalizeAccountLifecycleInput(account.LifecycleState),
		"checked_at", checkedAt,
		"new_expires_at", newExpiry,
		"summary", account.Extra[accountExpiryProbeSummaryKey],
	)
}

func (s *AccountExpiryService) handleFailedProbe(ctx context.Context, account *Account, checkedAt time.Time, result *BackgroundAccountTestResult, startedAt time.Time, requestID string) {
	if account == nil {
		return
	}
	status := AccountExpiryProbeStatusDisabled
	summary := "Expiry probe failed; account has been disabled."
	errorCode := "expiry_probe_failed"
	if result != nil && strings.TrimSpace(result.ErrorMessage) != "" {
		summary = strings.TrimSpace(result.ErrorMessage)
	}
	if result != nil && (NormalizeAccountLifecycleInput(result.CurrentLifecycleState) == AccountLifecycleBlacklisted ||
		result.BlacklistAdviceDecision == string(BlacklistAdviceAutoBlacklisted) ||
		result.BlacklistAdviceDecision == string(BlacklistAdviceRecommendBlacklist)) {
		status = AccountExpiryProbeStatusBlacklisted
		errorCode = firstNonEmptyHardBanString(result.BlacklistAdviceDecision, errorCode)
		purgeAt := checkedAt.Add(AccountBlacklistRetention)
		if err := s.accountRepo.MarkBlacklisted(ctx, account.ID, errorCode, summary, checkedAt, purgeAt); err != nil {
			slog.Warn(
				"account_expiry_probe_failed_update",
				"request_id", requestID,
				"account_id", account.ID,
				"lifecycle_state", NormalizeAccountLifecycleInput(account.LifecycleState),
				"checked_at", checkedAt,
				"probe_status", AccountExpiryProbeStatusBlacklisted,
				"error", err,
			)
			status = AccountExpiryProbeStatusFailed
		}
	}
	if status != AccountExpiryProbeStatusBlacklisted {
		account.Schedulable = false
		account.Status = StatusDisabled
		if account.Extra == nil {
			account.Extra = map[string]any{}
		}
		for key, value := range BuildAccountExpiryProbeExtra(checkedAt, status, summary, nil, nil) {
			account.Extra[key] = value
		}
		if err := s.accountRepo.Update(ctx, account); err != nil {
			slog.Warn(
				"account_expiry_probe_failed_update",
				"request_id", requestID,
				"account_id", account.ID,
				"lifecycle_state", NormalizeAccountLifecycleInput(account.LifecycleState),
				"checked_at", checkedAt,
				"probe_status", status,
				"error", err,
			)
			return
		}
	}
	if status == AccountExpiryProbeStatusBlacklisted {
		if err := s.accountRepo.UpdateExtra(ctx, account.ID, BuildAccountExpiryProbeExtra(checkedAt, status, summary, nil, nil)); err != nil {
			slog.Warn(
				"account_expiry_probe_failed_update",
				"request_id", requestID,
				"account_id", account.ID,
				"lifecycle_state", NormalizeAccountLifecycleInput(account.LifecycleState),
				"checked_at", checkedAt,
				"probe_status", status,
				"error", err,
			)
		}
		protocolruntime.RecordRecoveryProbeBlacklisted("expiry_probe")
		slog.Info(
			"account_expiry_probe_blacklisted",
			"request_id", requestID,
			"account_id", account.ID,
			"lifecycle_state", NormalizeAccountLifecycleInput(account.LifecycleState),
			"checked_at", checkedAt,
			"probe_status", status,
			"summary", summary,
		)
	} else {
		protocolruntime.RecordRecoveryProbeRetry("expiry_probe")
		slog.Info(
			"account_expiry_probe_disabled",
			"request_id", requestID,
			"account_id", account.ID,
			"lifecycle_state", NormalizeAccountLifecycleInput(account.LifecycleState),
			"checked_at", checkedAt,
			"probe_status", status,
			"summary", summary,
		)
	}
	protocolruntime.RecordRecoveryProbeResult("expiry_probe", status, time.Since(startedAt).Milliseconds())
}

func requestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if requestID, _ := ctx.Value(ctxkey.RequestID).(string); strings.TrimSpace(requestID) != "" {
		return strings.TrimSpace(requestID)
	}
	if clientRequestID, _ := ctx.Value(ctxkey.ClientRequestID).(string); strings.TrimSpace(clientRequestID) != "" {
		return strings.TrimSpace(clientRequestID)
	}
	return ""
}
