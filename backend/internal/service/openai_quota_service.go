package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	chatGPTQuotaUsageURL          = "https://chatgpt.com/backend-api/wham/usage"
	chatGPTQuotaResetURL          = "https://chatgpt.com/backend-api/wham/rate-limit-reset-credits/consume"
	openAIQuotaUpstreamTimeout    = 20 * time.Second
	openAIQuotaOriginator         = "Codex Desktop"
	openAIQuotaLanguage           = "zh-CN"
	openAIQuotaResetCreditsNoData = "OpenAI quota usage did not include reset credits"
)

type OpenAIRateLimitWindow struct {
	UsedPercent        float64 `json:"used_percent"`
	LimitWindowSeconds int64   `json:"limit_window_seconds"`
	ResetAfterSeconds  int64   `json:"reset_after_seconds"`
	ResetAt            int64   `json:"reset_at"`
}

type OpenAIRateLimit struct {
	Allowed         bool                   `json:"allowed"`
	LimitReached    bool                   `json:"limit_reached"`
	PrimaryWindow   *OpenAIRateLimitWindow `json:"primary_window,omitempty"`
	SecondaryWindow *OpenAIRateLimitWindow `json:"secondary_window,omitempty"`
}

type OpenAIAdditionalRateLimit struct {
	LimitName      string           `json:"limit_name"`
	MeteredFeature string           `json:"metered_feature"`
	RateLimit      *OpenAIRateLimit `json:"rate_limit,omitempty"`
}

type OpenAIRateLimitResetCredits struct {
	AvailableCount int `json:"available_count"`
}

type OpenAIQuotaUsage struct {
	UserID                string                       `json:"user_id,omitempty"`
	AccountID             string                       `json:"account_id,omitempty"`
	Email                 string                       `json:"email,omitempty"`
	PlanType              string                       `json:"plan_type,omitempty"`
	RateLimit             *OpenAIRateLimit             `json:"rate_limit,omitempty"`
	AdditionalRateLimits  []OpenAIAdditionalRateLimit  `json:"additional_rate_limits,omitempty"`
	RateLimitResetCredits *OpenAIRateLimitResetCredits `json:"rate_limit_reset_credits,omitempty"`
	FetchedAt             int64                        `json:"fetched_at"`
}

type OpenAIQuotaResetCredit struct {
	ID              string `json:"id,omitempty"`
	ResetType       string `json:"reset_type,omitempty"`
	Status          string `json:"status,omitempty"`
	GrantedAt       string `json:"granted_at,omitempty"`
	ExpiresAt       string `json:"expires_at,omitempty"`
	RedeemStartedAt string `json:"redeem_started_at,omitempty"`
	RedeemedAt      string `json:"redeemed_at,omitempty"`
}

type OpenAIQuotaResetResult struct {
	Code         string                  `json:"code"`
	Credit       *OpenAIQuotaResetCredit `json:"credit,omitempty"`
	WindowsReset int                     `json:"windows_reset"`
}

type OpenAIQuotaService struct {
	accountRepo          AccountRepository
	proxyRepo            ProxyRepository
	tokenProvider        *OpenAITokenProvider
	privacyClientFactory PrivacyClientFactory
	usageURL             string
	resetURL             string
}

func NewOpenAIQuotaService(
	accountRepo AccountRepository,
	proxyRepo ProxyRepository,
	tokenProvider *OpenAITokenProvider,
	privacyClientFactory PrivacyClientFactory,
) *OpenAIQuotaService {
	return &OpenAIQuotaService{
		accountRepo:          accountRepo,
		proxyRepo:            proxyRepo,
		tokenProvider:        tokenProvider,
		privacyClientFactory: privacyClientFactory,
		usageURL:             chatGPTQuotaUsageURL,
		resetURL:             chatGPTQuotaResetURL,
	}
}

func ProvideOpenAIQuotaService(
	accountRepo AccountRepository,
	proxyRepo ProxyRepository,
	tokenProvider *OpenAITokenProvider,
	privacyClientFactory PrivacyClientFactory,
) *OpenAIQuotaService {
	return NewOpenAIQuotaService(accountRepo, proxyRepo, tokenProvider, privacyClientFactory)
}

func (s *OpenAIQuotaService) QueryUsage(ctx context.Context, accountID int64) (*OpenAIQuotaUsage, error) {
	startedAt := time.Now()
	accessToken, chatGPTAccountID, proxyURL, err := s.prepareUpstreamCall(ctx, accountID)
	if err != nil {
		return nil, err
	}

	client, err := s.privacyClientFactory(proxyURL)
	if err != nil {
		return nil, infraerrors.Newf(http.StatusBadGateway, "OPENAI_QUOTA_CLIENT_ERROR", "OpenAI quota upstream client failed: %v", err)
	}

	callCtx, cancel := context.WithTimeout(ctx, openAIQuotaUpstreamTimeout)
	defer cancel()

	var payload OpenAIQuotaUsage
	resp, err := client.R().
		SetContext(callCtx).
		SetHeaders(buildOpenAIQuotaHeaders(accessToken, chatGPTAccountID)).
		SetSuccessResult(&payload).
		Get(s.usageEndpoint())
	if err != nil {
		slog.Warn("openai_quota_query_failed", "account_id", accountID, "duration_ms", time.Since(startedAt).Milliseconds(), "error", err.Error())
		return nil, infraerrors.Newf(http.StatusBadGateway, "OPENAI_QUOTA_REQUEST_FAILED", "OpenAI quota query failed: %v", err)
	}
	if !resp.IsSuccessState() {
		status := resp.StatusCode
		slog.Warn("openai_quota_query_upstream_error", "account_id", accountID, "status", status, "duration_ms", time.Since(startedAt).Milliseconds())
		return nil, infraerrors.Newf(mapOpenAIQuotaUpstreamStatus(status), "OPENAI_QUOTA_UPSTREAM_ERROR", "OpenAI quota upstream returned %d", status)
	}

	payload.FetchedAt = time.Now().UTC().Unix()
	slog.Info("openai_quota_query_succeeded", "account_id", accountID, "duration_ms", time.Since(startedAt).Milliseconds(), "has_reset_credits", payload.RateLimitResetCredits != nil)
	return &payload, nil
}

func (s *OpenAIQuotaService) ResetCredit(ctx context.Context, accountID int64) (*OpenAIQuotaResetResult, error) {
	startedAt := time.Now()
	accessToken, chatGPTAccountID, proxyURL, err := s.prepareUpstreamCall(ctx, accountID)
	if err != nil {
		return nil, err
	}
	redeemRequestID, err := generateOpenAIQuotaRedeemRequestID()
	if err != nil {
		return nil, infraerrors.Newf(http.StatusInternalServerError, "OPENAI_QUOTA_REDEEM_ID_FAILED", "OpenAI quota redeem id failed: %v", err)
	}

	client, err := s.privacyClientFactory(proxyURL)
	if err != nil {
		return nil, infraerrors.Newf(http.StatusBadGateway, "OPENAI_QUOTA_CLIENT_ERROR", "OpenAI quota upstream client failed: %v", err)
	}

	callCtx, cancel := context.WithTimeout(ctx, openAIQuotaUpstreamTimeout)
	defer cancel()

	headers := buildOpenAIQuotaHeaders(accessToken, chatGPTAccountID)
	headers["content-type"] = "application/json"

	var payload OpenAIQuotaResetResult
	resp, err := client.R().
		SetContext(callCtx).
		SetHeaders(headers).
		SetBody(map[string]string{"redeem_request_id": redeemRequestID}).
		SetSuccessResult(&payload).
		Post(s.resetEndpoint())
	if err != nil {
		slog.Warn("openai_quota_reset_failed", "account_id", accountID, "duration_ms", time.Since(startedAt).Milliseconds(), "error", err.Error())
		return nil, infraerrors.Newf(http.StatusBadGateway, "OPENAI_QUOTA_RESET_REQUEST_FAILED", "OpenAI quota reset failed: %v", err)
	}
	if !resp.IsSuccessState() {
		status := resp.StatusCode
		slog.Warn("openai_quota_reset_upstream_error", "account_id", accountID, "status", status, "duration_ms", time.Since(startedAt).Milliseconds())
		return nil, infraerrors.Newf(mapOpenAIQuotaUpstreamStatus(status), "OPENAI_QUOTA_RESET_UPSTREAM_ERROR", "OpenAI quota reset upstream returned %d", status)
	}

	slog.Info("openai_quota_reset_succeeded", "account_id", accountID, "duration_ms", time.Since(startedAt).Milliseconds(), "code", payload.Code, "windows_reset", payload.WindowsReset)
	return &payload, nil
}

func (s *OpenAIQuotaService) usageEndpoint() string {
	if s != nil && strings.TrimSpace(s.usageURL) != "" {
		return strings.TrimSpace(s.usageURL)
	}
	return chatGPTQuotaUsageURL
}

func (s *OpenAIQuotaService) resetEndpoint() string {
	if s != nil && strings.TrimSpace(s.resetURL) != "" {
		return strings.TrimSpace(s.resetURL)
	}
	return chatGPTQuotaResetURL
}

func (s *OpenAIQuotaService) ReadResetCredits(ctx context.Context, account *Account) (*OpenAIResetCreditsSnapshot, error) {
	if account == nil {
		return nil, ErrAccountNilInput
	}
	usage, err := s.QueryUsage(ctx, account.ID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	snapshot := openAIResetCreditsSnapshotFromQuotaUsage(usage, now)
	if snapshot == nil {
		snapshot = &OpenAIResetCreditsSnapshot{
			UpdatedAt:         now,
			Source:            openAIResetCreditsSourceWham,
			Status:            openAIResetCreditsStatusUnknownOrUnsupported,
			UnsupportedReason: openAIQuotaResetCreditsNoData,
		}
	}
	updates := openAIResetCreditsExtraFromSnapshot(snapshot)
	mergeAccountExtra(account, updates)
	if s != nil && s.accountRepo != nil && account.ID > 0 {
		if err := s.accountRepo.UpdateExtra(ctx, account.ID, updates); err != nil {
			slog.Warn("openai_quota_reset_credits_persist_failed", "account_id", account.ID, "error", err.Error())
		}
	}
	return snapshot, nil
}

func (s *OpenAIQuotaService) prepareUpstreamCall(ctx context.Context, accountID int64) (accessToken, chatGPTAccountID, proxyURL string, err error) {
	if s == nil || s.accountRepo == nil || s.tokenProvider == nil || s.privacyClientFactory == nil {
		return "", "", "", infraerrors.New(http.StatusInternalServerError, "OPENAI_QUOTA_NOT_CONFIGURED", "OpenAI quota service is not configured")
	}
	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return "", "", "", err
	}
	if account == nil {
		return "", "", "", ErrAccountNotFound
	}
	if account.Platform != PlatformOpenAI {
		return "", "", "", infraerrors.BadRequest("OPENAI_QUOTA_INVALID_PLATFORM", "account is not an OpenAI account")
	}
	if account.Type != AccountTypeOAuth {
		return "", "", "", infraerrors.BadRequest("OPENAI_QUOTA_INVALID_TYPE", "account is not an OpenAI OAuth account")
	}

	chatGPTAccountID = strings.TrimSpace(account.GetCredential("chatgpt_account_id"))
	if chatGPTAccountID == "" {
		chatGPTAccountID = strings.TrimSpace(account.GetCredential("organization_id"))
	}
	if chatGPTAccountID == "" {
		return "", "", "", infraerrors.BadRequest("OPENAI_QUOTA_MISSING_ACCOUNT_ID", "OpenAI chatgpt_account_id is missing")
	}

	accessToken, err = s.tokenProvider.GetAccessToken(ctx, account)
	if err != nil {
		return "", "", "", infraerrors.Newf(http.StatusBadGateway, "OPENAI_QUOTA_TOKEN_UNAVAILABLE", "OpenAI access token unavailable: %v", err)
	}
	if strings.TrimSpace(accessToken) == "" {
		return "", "", "", infraerrors.New(http.StatusBadGateway, "OPENAI_QUOTA_TOKEN_UNAVAILABLE", "OpenAI access token is empty")
	}

	if account.ProxyID != nil {
		switch {
		case account.Proxy != nil:
			proxyURL = account.Proxy.URL()
		case s.proxyRepo != nil:
			proxy, proxyErr := s.proxyRepo.GetByID(ctx, *account.ProxyID)
			if proxyErr == nil && proxy != nil {
				proxyURL = proxy.URL()
			}
		}
	}
	return accessToken, chatGPTAccountID, proxyURL, nil
}

func buildOpenAIQuotaHeaders(accessToken, chatGPTAccountID string) map[string]string {
	return map[string]string{
		"authorization":      "Bearer " + accessToken,
		"chatgpt-account-id": chatGPTAccountID,
		"oai-language":       openAIQuotaLanguage,
		"originator":         openAIQuotaOriginator,
		"accept":             "application/json",
		"origin":             "https://chatgpt.com",
		"referer":            "https://chatgpt.com/",
		"sec-fetch-site":     "same-origin",
		"sec-fetch-mode":     "cors",
		"sec-fetch-dest":     "empty",
	}
}

func openAIResetCreditsSnapshotFromQuotaUsage(usage *OpenAIQuotaUsage, now time.Time) *OpenAIResetCreditsSnapshot {
	if usage == nil {
		return nil
	}
	snapshot := &OpenAIResetCreditsSnapshot{
		UpdatedAt: now.UTC(),
		Source:    openAIResetCreditsSourceWham,
		Status:    openAIResetCreditsStatusUnknownOrUnsupported,
	}
	if usage.RateLimitResetCredits != nil {
		count := usage.RateLimitResetCredits.AvailableCount
		snapshot.AvailableCount = &count
		snapshot.Status = openAIResetCreditsStatusAvailable
	}
	return snapshot
}

func openAIResetCreditsExtraFromSnapshot(snapshot *OpenAIResetCreditsSnapshot) map[string]any {
	if snapshot == nil {
		return nil
	}
	updatedAt := snapshot.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = time.Now().UTC()
	}
	updates := map[string]any{
		openAIQuotaUsageUpdatedAtExtraKey:           updatedAt.Format(time.RFC3339),
		openAIResetCreditsStatusExtraKey:            snapshot.Status,
		openAIResetCreditsUnsupportedReasonExtraKey: nil,
		openAIResetCreditsAvailableCountExtraKey:    nil,
		openAIResetCreditsUpdatedAtExtraKey:         nil,
	}
	if strings.TrimSpace(snapshot.UnsupportedReason) != "" {
		updates[openAIResetCreditsUnsupportedReasonExtraKey] = snapshot.UnsupportedReason
	}
	if snapshot.AvailableCount != nil {
		updates[openAIResetCreditsAvailableCountExtraKey] = *snapshot.AvailableCount
		updates[openAIResetCreditsUpdatedAtExtraKey] = updatedAt.Format(time.RFC3339)
	}
	return updates
}

func maxInt64(value, minimum int64) int64 {
	if value < minimum {
		return minimum
	}
	return value
}

func generateOpenAIQuotaRedeemRequestID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	hexStr := hex.EncodeToString(b)
	return fmt.Sprintf("%s-%s-%s-%s-%s", hexStr[0:8], hexStr[8:12], hexStr[12:16], hexStr[16:20], hexStr[20:]), nil
}

func mapOpenAIQuotaUpstreamStatus(status int) int {
	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return status
	case status == http.StatusTooManyRequests:
		return http.StatusTooManyRequests
	case status >= 400:
		return http.StatusBadGateway
	default:
		return http.StatusBadGateway
	}
}
