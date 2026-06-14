package service

import (
	"context"
	"log/slog"
	"strings"
	"time"
)

const (
	AccountReauthReasonCode          = "credentials_need_reauth"
	AccountReauthDeadlineExpiredCode = "reauth_deadline_expired"
	AccountReauthStatusExtraKey      = "reauth_status"
	AccountReauthGracePeriod         = 7 * 24 * time.Hour
)

type AccountReauthStatus struct {
	RequiredSince time.Time `json:"required_since"`
	DeadlineAt    time.Time `json:"deadline_at"`
	ReasonCode    string    `json:"reason_code,omitempty"`
	Message       string    `json:"message,omitempty"`
}

func ShouldAccountNeedReauth(account *Account, reasonCode string) bool {
	if account == nil || !account.IsOAuth() {
		return false
	}
	return strings.TrimSpace(reasonCode) == AccountReauthReasonCode
}

func MarkAccountNeedsReauth(ctx context.Context, repo AccountRepository, account *Account, message string, now time.Time) (*AccountReauthStatus, bool) {
	if repo == nil || account == nil {
		return nil, false
	}
	if now.IsZero() {
		now = time.Now()
	}
	if MaybeBlacklistExpiredReauth(ctx, repo, account, now) {
		return AccountReauthStatusFromExtra(account.Extra), true
	}

	status := AccountReauthStatusFromExtra(account.Extra)
	if status == nil || status.RequiredSince.IsZero() || status.DeadlineAt.IsZero() {
		requiredSince := now.UTC()
		status = &AccountReauthStatus{
			RequiredSince: requiredSince,
			DeadlineAt:    requiredSince.Add(AccountReauthGracePeriod),
		}
	}
	status.ReasonCode = AccountReauthReasonCode
	status.Message = strings.TrimSpace(message)
	account.Extra = setAccountReauthStatusExtra(account.Extra, status)
	account.Status = StatusError
	account.Schedulable = false
	account.ErrorMessage = strings.TrimSpace(message)
	account.LifecycleReasonCode = AccountReauthReasonCode
	account.LifecycleReasonMessage = strings.TrimSpace(message)
	if NormalizeAccountLifecycleInput(account.LifecycleState) == AccountLifecycleBlacklisted {
		return status, true
	}
	account.LifecycleState = AccountLifecycleNormal
	if err := repo.Update(ctx, account); err != nil {
		slog.Warn("account_reauth_mark_failed", "account_id", account.ID, "platform", account.Platform, "error", err)
		return status, false
	}
	slog.Info(
		"account_reauth_required",
		"account_id", account.ID,
		"platform", account.Platform,
		"reason_code", AccountReauthReasonCode,
		"deadline_at", status.DeadlineAt.Format(time.RFC3339),
	)
	return status, false
}

func MaybeBlacklistExpiredReauth(ctx context.Context, repo AccountRepository, account *Account, now time.Time) bool {
	if repo == nil || account == nil {
		return false
	}
	status := AccountReauthStatusFromExtra(account.Extra)
	if status == nil || status.DeadlineAt.IsZero() {
		return false
	}
	if now.IsZero() {
		now = time.Now()
	}
	if now.UTC().Before(status.DeadlineAt.UTC()) {
		return false
	}
	message := firstNonEmptyHardBanString(status.Message, account.LifecycleReasonMessage, "reauth deadline expired")
	purgeAt := now.Add(AccountBlacklistRetention)
	if err := repo.MarkBlacklisted(ctx, account.ID, AccountReauthDeadlineExpiredCode, message, now, purgeAt); err != nil {
		slog.Warn("account_reauth_blacklist_expired_failed", "account_id", account.ID, "platform", account.Platform, "error", err)
		return false
	}
	account.LifecycleState = AccountLifecycleBlacklisted
	account.LifecycleReasonCode = AccountReauthDeadlineExpiredCode
	account.LifecycleReasonMessage = message
	account.Status = StatusDisabled
	account.Schedulable = false
	slog.Info(
		"account_reauth_deadline_expired_blacklisted",
		"account_id", account.ID,
		"platform", account.Platform,
		"deadline_at", status.DeadlineAt.Format(time.RFC3339),
	)
	return true
}

func ClearAccountReauthState(ctx context.Context, repo AccountRepository, account *Account) bool {
	if repo == nil || account == nil || AccountReauthStatusFromExtra(account.Extra) == nil {
		return false
	}
	clearAccountReauthStateInMemory(account)
	if err := repo.Update(ctx, account); err != nil {
		slog.Warn("account_reauth_clear_failed", "account_id", account.ID, "platform", account.Platform, "error", err)
		return false
	}
	slog.Info("account_reauth_cleared", "account_id", account.ID, "platform", account.Platform)
	return true
}

func clearAccountReauthStateInMemory(account *Account) {
	if account == nil {
		return
	}
	if len(account.Extra) > 0 {
		nextExtra := cloneStringAnyMap(account.Extra)
		delete(nextExtra, AccountReauthStatusExtraKey)
		account.Extra = emptyMapToNil(nextExtra)
	}
	if NormalizeAccountLifecycleInput(account.LifecycleState) == AccountLifecycleBlacklisted {
		return
	}
	if strings.TrimSpace(account.LifecycleReasonCode) == AccountReauthReasonCode ||
		strings.TrimSpace(account.LifecycleReasonCode) == AccountReauthDeadlineExpiredCode {
		account.LifecycleReasonCode = ""
		account.LifecycleReasonMessage = ""
	}
	if account.Status == StatusError {
		account.Status = StatusActive
		account.ErrorMessage = ""
		account.Schedulable = true
	}
}

func AccountReauthStatusFromExtra(extra map[string]any) *AccountReauthStatus {
	if len(extra) == 0 {
		return nil
	}
	raw, ok := extra[AccountReauthStatusExtraKey]
	if !ok {
		return nil
	}
	statusMap, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	requiredSince := parseReauthTime(statusMap["required_since"])
	deadlineAt := parseReauthTime(statusMap["deadline_at"])
	if requiredSince.IsZero() && deadlineAt.IsZero() {
		return nil
	}
	return &AccountReauthStatus{
		RequiredSince: requiredSince,
		DeadlineAt:    deadlineAt,
		ReasonCode:    strings.TrimSpace(stringAny(statusMap["reason_code"])),
		Message:       strings.TrimSpace(stringAny(statusMap["message"])),
	}
}

func setAccountReauthStatusExtra(extra map[string]any, status *AccountReauthStatus) map[string]any {
	nextExtra := cloneStringAnyMap(extra)
	if nextExtra == nil {
		nextExtra = map[string]any{}
	}
	if status == nil {
		delete(nextExtra, AccountReauthStatusExtraKey)
		return emptyMapToNil(nextExtra)
	}
	nextExtra[AccountReauthStatusExtraKey] = map[string]any{
		"required_since": status.RequiredSince.UTC().Format(time.RFC3339),
		"deadline_at":    status.DeadlineAt.UTC().Format(time.RFC3339),
		"reason_code":    strings.TrimSpace(status.ReasonCode),
		"message":        strings.TrimSpace(status.Message),
	}
	return nextExtra
}

func parseReauthTime(value any) time.Time {
	switch v := value.(type) {
	case time.Time:
		return v.UTC()
	case *time.Time:
		if v == nil {
			return time.Time{}
		}
		return v.UTC()
	case string:
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(v))
		if err == nil {
			return parsed.UTC()
		}
	}
	return time.Time{}
}
