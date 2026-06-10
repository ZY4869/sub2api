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
const accountExpiryJobName = "account_expiry_probe"

type AccountExpiryService struct {
	accountRepo       AccountRepository
	accountTestRunner interface {
		RunTestBackgroundDetailed(ctx context.Context, input ScheduledTestExecutionInput) (*BackgroundAccountTestResult, error)
	}
	leaderGate  PeriodicJobLeaderGate
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

func (s *AccountExpiryService) SetLeaderGate(gate PeriodicJobLeaderGate) {
	if s == nil {
		return
	}
	s.leaderGate = gate
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

		s.runLeaderOnce(context.Background())
		for {
			select {
			case <-ticker.C:
				s.runLeaderOnce(context.Background())
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

func (s *AccountExpiryService) runLeaderOnce(ctx context.Context) bool {
	if s == nil {
		return false
	}
	if s.leaderGate == nil {
		s.runOnce(ctx)
		return true
	}
	return s.leaderGate.RunIfLeader(ctx, accountExpiryJobName, periodicJobLeaderTTL(s.interval), s.runOnce)
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
		if s.tryAutoRenewAccount(ctx, &account, now) {
			continue
		}
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

func (s *AccountExpiryService) tryAutoRenewAccount(ctx context.Context, account *Account, now time.Time) bool {
	if account == nil || !account.AutoRenewEnabled || account.ExpiresAt == nil {
		return false
	}
	if !IsManagedRuntimeAccount(account) {
		return false
	}
	previousExpiry := account.ExpiresAt.UTC()
	if previousExpiry.After(now.UTC()) {
		return false
	}
	period, err := NormalizeAccountAutoRenewPeriod(account.AutoRenewPeriod)
	if err != nil {
		s.logAutoRenewFailed(ctx, account, now, previousExpiry, strings.TrimSpace(account.AutoRenewPeriod), err)
		return false
	}
	nextExpiry := nextAccountAutoRenewExpiry(previousExpiry, period, now)
	if !nextExpiry.After(previousExpiry) {
		s.logAutoRenewFailed(ctx, account, now, previousExpiry, period, fmt.Errorf("computed renewal is not after previous expiration"))
		return false
	}
	if !nextExpiry.After(now.UTC()) {
		s.logAutoRenewFailed(ctx, account, now, previousExpiry, period, fmt.Errorf("computed renewal is not after current time"))
		return false
	}
	requestID := firstNonEmptyString(requestIDFromContext(ctx), "generated:"+generateRequestID())
	slog.Info(
		"account_auto_renew_started",
		"request_id", requestID,
		"account_id", account.ID,
		"lifecycle_state", NormalizeAccountLifecycleInput(account.LifecycleState),
		"period", period,
		"previous_expires_at", previousExpiry,
		"new_expires_at", nextExpiry,
	)

	updatedAccount := *account
	updatedAccount.Credentials = cloneStringAnyMap(account.Credentials)
	updatedAccount.Extra = cloneStringAnyMap(account.Extra)
	updatedAccount.AutoRenewPeriod = period
	updatedAccount.ExpiresAt = &nextExpiry
	if updatedAccount.Extra == nil {
		updatedAccount.Extra = map[string]any{}
	}
	summary := fmt.Sprintf("Auto renewed account from %s to %s by %s.", previousExpiry.Format(time.RFC3339), nextExpiry.Format(time.RFC3339), period)
	for key, value := range BuildAccountAutoRenewExtra(now, AccountAutoRenewStatusSuccess, period, previousExpiry, &nextExpiry, summary) {
		updatedAccount.Extra[key] = value
	}
	if err := s.accountRepo.Update(ctx, &updatedAccount); err != nil {
		s.logAutoRenewFailed(ctx, account, now, previousExpiry, period, err)
		return false
	}
	syncAccountMonthlyUsagePeriod(ctx, s.accountRepo, &updatedAccount, &previousExpiry, AccountUsagePeriodSourceExpiry)
	slog.Info(
		"account_auto_renew_success",
		"request_id", requestID,
		"account_id", account.ID,
		"lifecycle_state", NormalizeAccountLifecycleInput(account.LifecycleState),
		"period", period,
		"previous_expires_at", previousExpiry,
		"new_expires_at", nextExpiry,
		"summary", summary,
	)
	return true
}

func (s *AccountExpiryService) logAutoRenewFailed(ctx context.Context, account *Account, checkedAt time.Time, previousExpiry time.Time, period string, err error) {
	if account == nil {
		return
	}
	requestID := firstNonEmptyString(requestIDFromContext(ctx), "generated:"+generateRequestID())
	slog.Warn(
		"account_auto_renew_failed",
		"request_id", requestID,
		"account_id", account.ID,
		"lifecycle_state", NormalizeAccountLifecycleInput(account.LifecycleState),
		"period", strings.TrimSpace(period),
		"previous_expires_at", previousExpiry,
		"checked_at", checkedAt,
		"error", err,
	)
	if s == nil || s.accountRepo == nil {
		return
	}
	updates := BuildAccountAutoRenewExtra(
		checkedAt,
		AccountAutoRenewStatusFailed,
		strings.TrimSpace(period),
		previousExpiry,
		nil,
		firstNonEmptyString(strings.TrimSpace(accountAutoRenewErrString(err)), "Auto renew failed."),
	)
	if updateErr := s.accountRepo.UpdateExtra(ctx, account.ID, updates); updateErr != nil {
		slog.Warn(
			"account_auto_renew_failed_update_failed",
			"request_id", requestID,
			"account_id", account.ID,
			"error", updateErr,
		)
	}
}

func addAccountAutoRenewPeriod(base time.Time, period string) time.Time {
	switch period {
	case AccountAutoRenewPeriodQuarter:
		return base.AddDate(0, 3, 0)
	case AccountAutoRenewPeriodYear:
		return base.AddDate(1, 0, 0)
	default:
		return base.AddDate(0, 1, 0)
	}
}

func nextAccountAutoRenewExpiry(previousExpiry time.Time, period string, now time.Time) time.Time {
	now = now.UTC()
	previousExpiry = previousExpiry.UTC()
	monthsPerPeriod := 1
	switch period {
	case AccountAutoRenewPeriodQuarter:
		monthsPerPeriod = 3
	case AccountAutoRenewPeriodYear:
		monthsPerPeriod = 12
	}
	elapsedMonths := (now.Year()-previousExpiry.Year())*12 + int(now.Month()) - int(previousExpiry.Month())
	periods := elapsedMonths / monthsPerPeriod
	if periods < 1 {
		periods = 1
	}
	nextExpiry := previousExpiry.AddDate(0, periods*monthsPerPeriod, 0)
	for !nextExpiry.After(now) {
		nextExpiry = addAccountAutoRenewPeriod(nextExpiry, period)
	}
	return nextExpiry
}

func accountAutoRenewErrString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
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
	oldExpiresAt := cloneTimePtr(account.ExpiresAt)
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
	syncAccountMonthlyUsagePeriod(ctx, s.accountRepo, account, oldExpiresAt, AccountUsagePeriodSourceExpiry)
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
