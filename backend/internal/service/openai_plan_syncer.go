package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

// OpenAIPlanSyncer 在 usage_limit_reached 后 best-effort 同步 ChatGPT 账号计划信息。
type OpenAIPlanSyncer interface {
	SyncPlanMetadata(ctx context.Context, account *Account) error
}

type OpenAIPlanSyncerFunc func(ctx context.Context, account *Account) error

func (fn OpenAIPlanSyncerFunc) SyncPlanMetadata(ctx context.Context, account *Account) error {
	return fn(ctx, account)
}

type openAIPlanSyncService struct {
	accountRepo          AccountRepository
	proxyRepo            ProxyRepository
	privacyClientFactory PrivacyClientFactory
}

func NewOpenAIPlanSyncService(accountRepo AccountRepository, proxyRepo ProxyRepository, privacyClientFactory PrivacyClientFactory) OpenAIPlanSyncer {
	if accountRepo == nil || privacyClientFactory == nil {
		return nil
	}
	return &openAIPlanSyncService{
		accountRepo:          accountRepo,
		proxyRepo:            proxyRepo,
		privacyClientFactory: privacyClientFactory,
	}
}

func (s *openAIPlanSyncService) SyncPlanMetadata(ctx context.Context, account *Account) error {
	if s == nil || account == nil || !account.IsOpenAIOAuth() {
		return nil
	}

	accessToken := account.GetOpenAIAccessToken()
	if strings.TrimSpace(accessToken) == "" {
		return nil
	}

	proxyURL := ""
	if account.ProxyID != nil && s.proxyRepo != nil {
		proxy, err := s.proxyRepo.GetByID(ctx, *account.ProxyID)
		if err != nil {
			slog.Debug("openai_usage_limit_plan_sync_proxy_lookup_failed", "account_id", account.ID, "proxy_id", *account.ProxyID, "error", err)
		} else if proxy != nil {
			proxyURL = proxy.URL()
		}
	}

	info := fetchChatGPTAccountInfo(ctx, s.privacyClientFactory, accessToken, proxyURL, account.GetOpenAIOrganizationID())
	if info == nil || strings.TrimSpace(info.PlanType) == "" {
		return nil
	}

	credentials := cloneAccountCredentialsMap(account.Credentials)
	changed := false
	changed = setOpenAIPlanCredential(credentials, "plan_type", normalizeOpenAIPlanType(info.PlanType)) || changed
	changed = setOpenAIPlanCredential(credentials, "plan_type_raw", strings.TrimSpace(info.PlanType)) || changed
	changed = setOpenAIPlanCredential(credentials, "plan_type_label", strings.TrimSpace(info.PlanTypeLabel)) || changed
	if info.ProMultiplier > 0 {
		changed = setOpenAIPlanCredential(credentials, "pro_multiplier", info.ProMultiplier) || changed
	}
	if strings.TrimSpace(info.SubscriptionExpiresAt) != "" {
		changed = setOpenAIPlanCredential(credentials, "subscription_expires_at", strings.TrimSpace(info.SubscriptionExpiresAt)) || changed
	}
	if !changed {
		return nil
	}

	if err := persistAccountCredentials(ctx, s.accountRepo, account, credentials); err != nil {
		return fmt.Errorf("persist openai plan metadata: %w", err)
	}
	return nil
}

func setOpenAIPlanCredential(credentials map[string]any, key string, value any) bool {
	if credentials == nil || key == "" || value == nil {
		return false
	}
	if current, exists := credentials[key]; exists && fmt.Sprint(current) == fmt.Sprint(value) {
		return false
	}
	credentials[key] = value
	return true
}
