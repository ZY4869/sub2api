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
	defaultAccountRateLimitRecoveryProbeInterval = time.Minute
	defaultAccountRateLimitRecoveryProbeRetry    = 30 * time.Minute
	defaultAccountRateLimitRecoveryProbeTimeout  = 45 * time.Second
	accountRateLimitRecoveryProbeReasonCode      = "auto_recovery_probe_failed"
)

type accountRateLimitRecoveryProbeExecutor interface {
	RunTestBackgroundDetailed(ctx context.Context, input ScheduledTestExecutionInput) (*BackgroundAccountTestResult, error)
}

type accountRateLimitRecoveryProbeRecoverer interface {
	RecoverAccountAfterSuccessfulTest(ctx context.Context, accountID int64) (*SuccessfulTestRecoveryResult, error)
}

type AccountRateLimitRecoveryProbeService struct {
	accountRepo       AccountRepository
	accountTestRunner accountRateLimitRecoveryProbeExecutor
	recoverer         accountRateLimitRecoveryProbeRecoverer
	interval          time.Duration
	retryDelay        time.Duration
	testTimeout       time.Duration
	now               func() time.Time
	stopCh            chan struct{}
	stopOnce          sync.Once
	wg                sync.WaitGroup
}

func NewAccountRateLimitRecoveryProbeService(
	accountRepo AccountRepository,
	accountTestRunner accountRateLimitRecoveryProbeExecutor,
	recoverer accountRateLimitRecoveryProbeRecoverer,
	interval time.Duration,
) *AccountRateLimitRecoveryProbeService {
	if interval <= 0 {
		interval = defaultAccountRateLimitRecoveryProbeInterval
	}
	return &AccountRateLimitRecoveryProbeService{
		accountRepo:       accountRepo,
		accountTestRunner: accountTestRunner,
		recoverer:         recoverer,
		interval:          interval,
		retryDelay:        defaultAccountRateLimitRecoveryProbeRetry,
		testTimeout:       defaultAccountRateLimitRecoveryProbeTimeout,
		now:               time.Now,
		stopCh:            make(chan struct{}),
	}
}

func (s *AccountRateLimitRecoveryProbeService) Start() {
	if s == nil || s.accountRepo == nil || s.accountTestRunner == nil || s.recoverer == nil {
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

func (s *AccountRateLimitRecoveryProbeService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
	s.wg.Wait()
}

func (s *AccountRateLimitRecoveryProbeService) runOnce(ctx context.Context) {
	if s == nil || s.accountRepo == nil || s.accountTestRunner == nil || s.recoverer == nil {
		return
	}
	accounts, err := s.accountRepo.ListActive(ctx)
	if err != nil {
		slog.Warn("account_auto_recovery_probe_list_failed", "error", err)
		return
	}
	now := s.now().UTC()
	for index := range accounts {
		account := accounts[index]
		if !shouldRunAccountAutoRecoveryProbe(&account, now) {
			continue
		}
		s.runProbeForAccount(ctx, &account, now)
	}
}

func shouldRunAccountAutoRecoveryProbe(account *Account, now time.Time) bool {
	if account == nil || account.RateLimitResetAt == nil {
		return false
	}
	if NormalizeAccountLifecycleInput(account.LifecycleState) == AccountLifecycleBlacklisted {
		return false
	}
	if NormalizeAccountRateLimitReasonInput(parseExtraString(account.Extra["rate_limit_reason"])) != AccountRateLimitReasonUsage7d {
		return false
	}
	if account.RateLimitResetAt.After(now) {
		return false
	}
	if nextRetryAt := parseAccountAutoRecoveryProbeTime(account.Extra, accountAutoRecoveryProbeNextRetryKey); nextRetryAt != nil && nextRetryAt.After(now) {
		return false
	}
	if checkedAt := parseAccountAutoRecoveryProbeTime(account.Extra, accountAutoRecoveryProbeCheckedAtKey); checkedAt != nil && !checkedAt.Before(account.RateLimitResetAt.UTC()) {
		return false
	}
	return true
}

func (s *AccountRateLimitRecoveryProbeService) runProbeForAccount(ctx context.Context, account *Account, checkedAt time.Time) {
	probeCtx, cancel := context.WithTimeout(ctx, s.testTimeout)
	defer cancel()
	probeCtx = EnsureRequestMetadata(probeCtx)
	SetProbeActionMetadata(probeCtx, "test")
	protocolruntime.RecordRecoveryProbeStarted(normalizeRecoveryProbeMetricReason(parseExtraString(account.Extra["rate_limit_reason"])))

	slog.Info("account_auto_recovery_probe_started", "account_id", account.ID, "probe_action", "test", "rate_limit_reset_at", account.RateLimitResetAt)
	result, err := s.accountTestRunner.RunTestBackgroundDetailed(probeCtx, ScheduledTestExecutionInput{
		AccountID: account.ID,
	})
	if err != nil && (result == nil || strings.TrimSpace(result.ErrorMessage) == "") {
		result = &BackgroundAccountTestResult{
			Status:       "failed",
			ErrorMessage: err.Error(),
		}
	}

	if result != nil && strings.EqualFold(strings.TrimSpace(result.Status), "success") {
		s.handleSuccessfulProbe(ctx, account.ID, checkedAt)
		return
	}

	status, summary, blacklisted, errorCode, nextRetryAt := classifyAccountAutoRecoveryProbeFailure(result, checkedAt, s.retryDelay)
	currentLifecycleState := account.LifecycleState
	if result != nil && strings.TrimSpace(result.CurrentLifecycleState) != "" {
		currentLifecycleState = result.CurrentLifecycleState
	}
	if blacklisted && NormalizeAccountLifecycleInput(currentLifecycleState) != AccountLifecycleBlacklisted {
		purgeAt := checkedAt.Add(AccountBlacklistRetention)
		if markErr := s.accountRepo.MarkBlacklisted(ctx, account.ID, firstNonEmptyHardBanString(errorCode, accountRateLimitRecoveryProbeReasonCode), summary, checkedAt, purgeAt); markErr != nil {
			slog.Warn("account_auto_recovery_probe_blacklist_failed", "account_id", account.ID, "error", markErr)
			status = AccountAutoRecoveryProbeStatusRetryScheduled
			summary = "Auto-blacklist failed; retrying automatically in 30 minutes."
			blacklisted = false
			errorCode = "blacklist_failed"
			nextRetryAt = recoveryProbeTimePtr(checkedAt.Add(s.retryDelay))
		}
	}

	if updateErr := s.accountRepo.UpdateExtra(ctx, account.ID, BuildAccountAutoRecoveryProbeExtra(
		checkedAt,
		status,
		summary,
		blacklisted,
		nextRetryAt,
		errorCode,
	)); updateErr != nil {
		slog.Warn("account_auto_recovery_probe_update_failed", "account_id", account.ID, "error", updateErr)
	}
	recordRecoveryProbeOutcome(status, errorCode)
	slog.Info("account_auto_recovery_probe_finished", "account_id", account.ID, "probe_action", status, "blacklisted", blacklisted, "error_code", errorCode)
}

func (s *AccountRateLimitRecoveryProbeService) handleSuccessfulProbe(ctx context.Context, accountID int64, checkedAt time.Time) {
	ctx = EnsureRequestMetadata(ctx)
	SetProbeActionMetadata(ctx, "recover")
	if _, recoverErr := s.recoverer.RecoverAccountAfterSuccessfulTest(ctx, accountID); recoverErr != nil {
		slog.Warn("account_auto_recovery_probe_recover_failed", "account_id", accountID, "error", recoverErr)
		if updateErr := s.accountRepo.UpdateExtra(ctx, accountID, BuildAccountAutoRecoveryProbeExtra(
			checkedAt,
			AccountAutoRecoveryProbeStatusRetryScheduled,
			firstNonEmptyHardBanString(recoverErr.Error(), "Account recovery failed; retrying automatically in 30 minutes."),
			false,
			recoveryProbeTimePtr(checkedAt.Add(s.retryDelay)),
			"recover_failed",
		)); updateErr != nil {
			slog.Warn("account_auto_recovery_probe_update_failed", "account_id", accountID, "error", updateErr)
		}
		recordRecoveryProbeOutcome(AccountAutoRecoveryProbeStatusRetryScheduled, "recover_failed")
		return
	}

	if updateErr := s.accountRepo.UpdateExtra(ctx, accountID, BuildAccountAutoRecoveryProbeExtra(
		checkedAt,
		AccountAutoRecoveryProbeStatusSuccess,
		"7-day limit window recovered and background test passed.",
		false,
		nil,
		"",
	)); updateErr != nil {
		slog.Warn("account_auto_recovery_probe_update_failed", "account_id", accountID, "error", updateErr)
	}
	recordRecoveryProbeOutcome(AccountAutoRecoveryProbeStatusSuccess, "recover")
	slog.Info("account_auto_recovery_probe_succeeded", "account_id", accountID, "probe_action", "recover")
}

func classifyAccountAutoRecoveryProbeFailure(result *BackgroundAccountTestResult, checkedAt time.Time, retryDelay time.Duration) (string, string, bool, string, *time.Time) {
	if result == nil {
		return AccountAutoRecoveryProbeStatusRetryScheduled, "Background test returned no result; retrying automatically in 30 minutes.", false, "empty_result", recoveryProbeTimePtr(checkedAt.Add(retryDelay))
	}

	errorMessage := strings.TrimSpace(result.ErrorMessage)
	if errorMessage == "" {
		errorMessage = "Background test failed."
	}
	if NormalizeAccountLifecycleInput(result.CurrentLifecycleState) == AccountLifecycleBlacklisted ||
		result.BlacklistAdviceDecision == string(BlacklistAdviceAutoBlacklisted) ||
		result.BlacklistAdviceDecision == string(BlacklistAdviceRecommendBlacklist) {
		return AccountAutoRecoveryProbeStatusBlacklisted, errorMessage, true, firstNonEmptyHardBanString(result.BlacklistAdviceDecision, accountRateLimitRecoveryProbeReasonCode), nil
	}
	if isTransientAccountAutoRecoveryProbeError(errorMessage) {
		return AccountAutoRecoveryProbeStatusRetryScheduled, errorMessage, false, "transient_error", recoveryProbeTimePtr(checkedAt.Add(retryDelay))
	}
	return AccountAutoRecoveryProbeStatusBlacklisted, errorMessage, true, accountRateLimitRecoveryProbeReasonCode, nil
}

func isTransientAccountAutoRecoveryProbeError(message string) bool {
	normalized := strings.ToLower(strings.TrimSpace(message))
	for _, keyword := range []string{
		"timeout", "timed out", "deadline exceeded", "temporarily unavailable",
		"connection reset", "connection refused", "eof", "tls", "dial tcp",
		"gateway", "bad gateway", "service unavailable", "no such host", "proxyconnect",
	} {
		if strings.Contains(normalized, keyword) {
			return true
		}
	}
	return false
}

func recoveryProbeTimePtr(value time.Time) *time.Time {
	t := value.UTC()
	return &t
}

func normalizeRecoveryProbeMetricReason(reason string) string {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return "unknown"
	}
	return reason
}

func recordRecoveryProbeOutcome(status string, reason string) {
	reason = normalizeRecoveryProbeMetricReason(reason)
	switch strings.TrimSpace(status) {
	case AccountAutoRecoveryProbeStatusSuccess:
		protocolruntime.RecordRecoveryProbeSuccess(reason)
	case AccountAutoRecoveryProbeStatusBlacklisted:
		protocolruntime.RecordRecoveryProbeBlacklisted(reason)
	case AccountAutoRecoveryProbeStatusRetryScheduled:
		protocolruntime.RecordRecoveryProbeRetry(reason)
	}
}
