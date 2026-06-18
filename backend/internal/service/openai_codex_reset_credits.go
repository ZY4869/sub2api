package service

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/google/uuid"
)

type OpenAICodexResetCreditService struct {
	accountRepo AccountRepository
	client      OpenAICodexAppServerRateLimitClient
	enabled     bool
}

func NewOpenAICodexResetCreditService(
	accountRepo AccountRepository,
	client OpenAICodexAppServerRateLimitClient,
	cfg *config.Config,
) *OpenAICodexResetCreditService {
	enabled := true
	if cfg != nil {
		enabled = cfg.OpenAICodex.ResetCreditsEnabled
	}
	return &OpenAICodexResetCreditService{
		accountRepo: accountRepo,
		client:      client,
		enabled:     enabled,
	}
}

func (s *OpenAICodexResetCreditService) ReadResetCredits(ctx context.Context, account *Account) (*OpenAICodexResetCreditsSnapshot, error) {
	startedAt := time.Now()
	if err := s.validateReady(account); err != nil {
		return nil, err
	}
	auth, err := openAICodexAuthTokensFromAccount(account)
	if err != nil {
		return nil, err
	}

	slog.Info("openai_reset_credits_read_started", "account_id", account.ID, "action", "read")
	raw, err := s.client.ReadRateLimits(ctx, auth)
	if err != nil {
		slog.Warn("openai_reset_credits_read_failed", "account_id", account.ID, "action", "read", "duration_ms", time.Since(startedAt).Milliseconds(), "error", err.Error())
		if isOpenAIResetCreditsUnsupportedError(err) {
			snapshot := openAICodexUnsupportedResetCreditsSnapshot(time.Now().UTC())
			_ = s.persistSnapshot(ctx, account, openAICodexUnsupportedResetCreditsExtra(snapshot))
			return snapshot, nil
		}
		return nil, err
	}
	if raw == nil {
		return nil, infraerrors.ServiceUnavailable("OPENAI_CODEX_APP_SERVER_INVALID_RESPONSE", "Codex app-server 返回空响应")
	}
	if strings.TrimSpace(raw.Status) == "" {
		raw.Status = openAIResetCreditsStatusUnknownOrUnsupported
	}

	snapshot := openAICodexResetCreditsSnapshotFromAppServer(raw)
	if err := s.persistSnapshot(ctx, account, raw.ExtraUpdates); err != nil {
		slog.Warn("openai_reset_credits_persist_failed", "account_id", account.ID, "action", "read", "duration_ms", time.Since(startedAt).Milliseconds(), "error", err.Error())
	}
	slog.Info("openai_reset_credits_read_succeeded", "account_id", account.ID, "action", "read", "duration_ms", time.Since(startedAt).Milliseconds(), "has_count", snapshot != nil && snapshot.AvailableCount != nil)
	return snapshot, nil
}

func (s *OpenAICodexResetCreditService) ConsumeResetCredit(ctx context.Context, account *Account, idempotencyKey string) (*OpenAICodexResetCreditConsumeResult, error) {
	startedAt := time.Now()
	if err := s.validateReady(account); err != nil {
		return nil, err
	}
	auth, err := openAICodexAuthTokensFromAccount(account)
	if err != nil {
		return nil, err
	}
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if idempotencyKey == "" {
		idempotencyKey = uuid.NewString()
	}

	slog.Info("openai_reset_credit_consume_started", "account_id", account.ID, "action", "consume")
	raw, err := s.client.ConsumeResetCredit(ctx, auth, idempotencyKey)
	if err != nil {
		slog.Warn("openai_reset_credit_consume_failed", "account_id", account.ID, "action", "consume", "duration_ms", time.Since(startedAt).Milliseconds(), "error", err.Error())
		if isOpenAIResetCreditsUnsupportedError(err) {
			snapshot := openAICodexUnsupportedResetCreditsSnapshot(time.Now().UTC())
			_ = s.persistSnapshot(ctx, account, openAICodexUnsupportedResetCreditsExtra(snapshot))
		}
		return nil, err
	}
	if raw == nil || raw.Snapshot == nil {
		return nil, infraerrors.ServiceUnavailable("OPENAI_CODEX_APP_SERVER_INVALID_RESPONSE", "Codex app-server 返回空响应")
	}
	status := strings.TrimSpace(raw.Status)
	updates := MergeStringAnyMap(nil, raw.Snapshot.ExtraUpdates)
	now := time.Now().UTC().Format(time.RFC3339)
	updates[openAIResetCreditLastConsumeStatusExtraKey] = status
	updates[openAIResetCreditLastConsumeUpdatedAtExtraKey] = now
	if err := s.persistSnapshot(ctx, account, updates); err != nil {
		slog.Warn("openai_reset_credit_consume_persist_failed", "account_id", account.ID, "action", "consume", "duration_ms", time.Since(startedAt).Milliseconds(), "result_status", status, "error", err.Error())
	}

	result := &OpenAICodexResetCreditConsumeResult{
		Status:   status,
		Snapshot: openAICodexResetCreditsSnapshotFromAppServer(raw.Snapshot),
	}
	slog.Info("openai_reset_credit_consume_succeeded", "account_id", account.ID, "action", "consume", "duration_ms", time.Since(startedAt).Milliseconds(), "result_status", status, "has_count", result.Snapshot != nil && result.Snapshot.AvailableCount != nil)
	if status == openAIResetCreditConsumeStatusNoCredit {
		return result, infraerrors.New(http.StatusConflict, "OPENAI_RESET_CREDITS_NO_CREDIT", "没有可用的 OpenAI 真实重置次数")
	}
	if status == openAIResetCreditConsumeStatusNothingToReset {
		return result, infraerrors.New(http.StatusConflict, "OPENAI_RESET_CREDITS_NOTHING_TO_RESET", "当前没有可重置的 OpenAI 限额窗口")
	}
	return result, nil
}

func (s *OpenAICodexResetCreditService) validateReady(account *Account) error {
	if s == nil || !s.enabled {
		return infraerrors.ServiceUnavailable("OPENAI_CODEX_RESET_CREDITS_DISABLED", "OpenAI 真实重置次数未启用")
	}
	if s.client == nil {
		return infraerrors.ServiceUnavailable("OPENAI_CODEX_APP_SERVER_NOT_CONFIGURED", "Codex app-server 未配置")
	}
	if account == nil {
		return ErrAccountNilInput
	}
	if !account.IsOpenAIOAuth() {
		return infraerrors.BadRequest("OPENAI_RESET_CREDITS_UNSUPPORTED_ACCOUNT", "仅 OpenAI OAuth 账号支持真实重置次数")
	}
	return nil
}

func openAICodexAuthTokensFromAccount(account *Account) (OpenAICodexAppServerAuthTokens, error) {
	if account == nil {
		return OpenAICodexAppServerAuthTokens{}, ErrAccountNilInput
	}
	accessToken := strings.TrimSpace(account.GetOpenAIAccessToken())
	if accessToken == "" {
		return OpenAICodexAppServerAuthTokens{}, infraerrors.BadRequest("OPENAI_ACCESS_TOKEN_MISSING", "OpenAI OAuth access_token 缺失")
	}
	chatGPTAccountID := strings.TrimSpace(account.GetChatGPTAccountID())
	if chatGPTAccountID == "" {
		return OpenAICodexAppServerAuthTokens{}, infraerrors.BadRequest("OPENAI_CHATGPT_ACCOUNT_ID_MISSING", "OpenAI OAuth chatgpt_account_id 缺失")
	}
	return OpenAICodexAppServerAuthTokens{
		AccessToken:      accessToken,
		ChatGPTAccountID: chatGPTAccountID,
		ChatGPTPlanType:  normalizeOpenAIPlanType(account.GetCredential("plan_type")),
	}, nil
}

func (s *OpenAICodexResetCreditService) persistSnapshot(ctx context.Context, account *Account, updates map[string]any) error {
	if s == nil || s.accountRepo == nil || account == nil || account.ID <= 0 || len(updates) == 0 {
		return nil
	}
	updateCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := s.accountRepo.UpdateExtra(updateCtx, account.ID, updates); err != nil {
		return err
	}
	mergeAccountExtra(account, updates)
	return nil
}

func openAICodexResetCreditsSnapshotFromAppServer(snapshot *OpenAICodexAppServerRateLimitsSnapshot) *OpenAICodexResetCreditsSnapshot {
	if snapshot == nil {
		return nil
	}
	status := strings.TrimSpace(snapshot.Status)
	if status == "" {
		if snapshot.AvailableCount != nil {
			status = openAIResetCreditsStatusAvailable
		} else {
			status = openAIResetCreditsStatusUnknownOrUnsupported
		}
	}
	return &OpenAICodexResetCreditsSnapshot{
		AvailableCount:    snapshot.AvailableCount,
		UpdatedAt:         snapshot.UpdatedAt,
		Source:            openAIResetCreditsSourceCodexAppServer,
		Status:            status,
		UnsupportedReason: snapshot.UnsupportedReason,
	}
}

func isOpenAIResetCreditsUnsupportedError(err error) bool {
	if err == nil {
		return false
	}
	var appErr *infraerrors.ApplicationError
	return errors.As(err, &appErr) && appErr.Reason == "OPENAI_CODEX_RESET_CREDITS_UNSUPPORTED"
}

func openAICodexUnsupportedResetCreditsSnapshot(now time.Time) *OpenAICodexResetCreditsSnapshot {
	return &OpenAICodexResetCreditsSnapshot{
		UpdatedAt:         now.UTC(),
		Source:            openAIResetCreditsSourceCodexAppServer,
		Status:            openAIResetCreditsStatusUnsupported,
		UnsupportedReason: "当前 Codex app-server 不支持 OpenAI 官方真实重置",
	}
}

func openAICodexUnsupportedResetCreditsExtra(snapshot *OpenAICodexResetCreditsSnapshot) map[string]any {
	if snapshot == nil {
		return nil
	}
	updatedAt := snapshot.UpdatedAt.UTC().Format(time.RFC3339)
	return map[string]any{
		openAIRateLimitsAppServerUpdatedAtExtraKey:  updatedAt,
		openAIResetCreditsStatusExtraKey:            openAIResetCreditsStatusUnsupported,
		openAIResetCreditsUnsupportedReasonExtraKey: snapshot.UnsupportedReason,
		openAIResetCreditsAvailableCountExtraKey:    nil,
		openAIResetCreditsUpdatedAtExtraKey:         nil,
	}
}

func openAICodexUnknownResetCreditsSnapshot(now time.Time) *OpenAICodexResetCreditsSnapshot {
	return &OpenAICodexResetCreditsSnapshot{
		UpdatedAt: now.UTC(),
		Source:    openAIResetCreditsSourceCodexAppServer,
		Status:    openAIResetCreditsStatusUnknownOrUnsupported,
	}
}

func openAICodexUnknownResetCreditsExtra(snapshot *OpenAICodexResetCreditsSnapshot) map[string]any {
	if snapshot == nil {
		return nil
	}
	updatedAt := snapshot.UpdatedAt.UTC().Format(time.RFC3339)
	return map[string]any{
		openAIRateLimitsAppServerUpdatedAtExtraKey:  updatedAt,
		openAIResetCreditsStatusExtraKey:            openAIResetCreditsStatusUnknownOrUnsupported,
		openAIResetCreditsUnsupportedReasonExtraKey: nil,
		openAIResetCreditsAvailableCountExtraKey:    nil,
		openAIResetCreditsUpdatedAtExtraKey:         nil,
	}
}
