package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

var (
	ErrRegistrationDisabled          = infraerrors.Forbidden("REGISTRATION_DISABLED", "registration is currently disabled")
	ErrSettingNotFound               = infraerrors.NotFound("SETTING_NOT_FOUND", "setting not found")
	ErrGoogleBatchGCSProfileNotFound = infraerrors.NotFound("GOOGLE_BATCH_GCS_PROFILE_NOT_FOUND", "google batch gcs profile not found")
	ErrGoogleBatchGCSProfileExists   = infraerrors.Conflict("GOOGLE_BATCH_GCS_PROFILE_EXISTS", "google batch gcs profile already exists")
	ErrDefaultSubGroupInvalid        = infraerrors.BadRequest(
		"DEFAULT_SUBSCRIPTION_GROUP_INVALID",
		"default subscription group must exist and be subscription type",
	)
	ErrDefaultSubGroupDuplicate = infraerrors.BadRequest(
		"DEFAULT_SUBSCRIPTION_GROUP_DUPLICATE",
		"default subscription group cannot be duplicated",
	)
)

type SettingRepository interface {
	Get(ctx context.Context, key string) (*Setting, error)
	GetValue(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string) error
	GetMultiple(ctx context.Context, keys []string) (map[string]string, error)
	SetMultiple(ctx context.Context, settings map[string]string) error
	GetAll(ctx context.Context) (map[string]string, error)
	Delete(ctx context.Context, key string) error
}

type DefaultSubscriptionGroupReader interface {
	GetByID(ctx context.Context, id int64) (*Group, error)
}

type SettingService struct {
	settingRepo                     SettingRepository
	defaultSubGroupReader           DefaultSubscriptionGroupReader
	accountDaily5HCandidateProvider AccountDaily5HTriggerCandidateProvider
	cfg                             *config.Config
	updateCallbacksMu               sync.RWMutex
	onUpdateCallbacks               []func()
	onS3Update                      func()
	version                         string
}

type AccountDaily5HTriggerCandidateProvider interface {
	ListDaily5HTriggerCandidates(ctx context.Context) []AccountDaily5HTriggerAccountTypeSummary
}

func NewSettingService(settingRepo SettingRepository, cfg *config.Config) *SettingService {
	return &SettingService{settingRepo: settingRepo, cfg: cfg}
}

func (s *SettingService) addOnUpdateCallback(callback func()) {
	if s == nil || callback == nil {
		return
	}
	s.updateCallbacksMu.Lock()
	defer s.updateCallbacksMu.Unlock()
	s.onUpdateCallbacks = append(s.onUpdateCallbacks, callback)
}

func (s *SettingService) notifyUpdateCallbacks() {
	if s == nil {
		return
	}

	s.updateCallbacksMu.RLock()
	callbacks := append([]func(){}, s.onUpdateCallbacks...)
	s.updateCallbacksMu.RUnlock()

	for _, callback := range callbacks {
		if callback == nil {
			continue
		}
		func() {
			defer func() {
				if recovered := recover(); recovered != nil {
					slog.Error("setting update callback panicked", "panic", recovered)
				}
			}()
			callback()
		}()
	}
}

func defaultSocialOAuthConfig(provider string) SocialOAuthConfig {
	switch NormalizeOAuthProvider(provider) {
	case AuthProviderGitHub:
		return SocialOAuthConfig{
			Provider:                  AuthProviderGitHub,
			AuthorizeURL:              "https://github.com/login/oauth/authorize",
			TokenURL:                  "https://github.com/login/oauth/access_token",
			UserInfoURL:               "https://api.github.com/user",
			Scopes:                    "read:user user:email",
			FrontendRedirectURL:       "/auth/social/callback",
			TokenAuthMethod:           "client_secret_post",
			UserInfoEmailPath:         "email",
			UserInfoIDPath:            "id",
			UserInfoUsernamePath:      "login",
			UserInfoAvatarPath:        "avatar_url",
			UserInfoEmailVerifiedPath: "verified",
		}
	case AuthProviderGoogle:
		return SocialOAuthConfig{
			Provider:                  AuthProviderGoogle,
			AuthorizeURL:              "https://accounts.google.com/o/oauth2/v2/auth",
			TokenURL:                  "https://oauth2.googleapis.com/token",
			UserInfoURL:               "https://openidconnect.googleapis.com/v1/userinfo",
			Scopes:                    "openid email profile",
			FrontendRedirectURL:       "/auth/social/callback",
			TokenAuthMethod:           "client_secret_post",
			UsePKCE:                   true,
			UserInfoEmailPath:         "email",
			UserInfoIDPath:            "sub",
			UserInfoUsernamePath:      "name",
			UserInfoAvatarPath:        "picture",
			UserInfoEmailVerifiedPath: "email_verified",
		}
	default:
		return SocialOAuthConfig{}
	}
}

func (s *SettingService) SetDefaultSubscriptionGroupReader(reader DefaultSubscriptionGroupReader) {
	s.defaultSubGroupReader = reader
}

func (s *SettingService) SetAccountDaily5HTriggerCandidateProvider(provider AccountDaily5HTriggerCandidateProvider) {
	s.accountDaily5HCandidateProvider = provider
}

func (s *SettingService) ListDaily5HTriggerCandidates(ctx context.Context) []AccountDaily5HTriggerAccountTypeSummary {
	if s == nil || s.accountDaily5HCandidateProvider == nil {
		return []AccountDaily5HTriggerAccountTypeSummary{}
	}
	items := s.accountDaily5HCandidateProvider.ListDaily5HTriggerCandidates(ctx)
	if items == nil {
		return []AccountDaily5HTriggerAccountTypeSummary{}
	}
	return items
}

func parseCustomMenuItemURLs(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return nil
	}

	var items []struct {
		URL      string `json:"url"`
		PageMode string `json:"page_mode"`
	}
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil
	}

	urls := make([]string, 0, len(items))
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item.PageMode), "markdown") {
			continue
		}
		if item.URL != "" {
			urls = append(urls, item.URL)
		}
	}
	return urls
}

func (s *SettingService) IsBudgetRectifierEnabled(ctx context.Context) bool {
	settings, err := s.GetRectifierSettings(ctx)
	if err != nil {
		return true
	}
	return settings.Enabled && settings.ThinkingBudgetEnabled
}

func (s *SettingService) SetBetaPolicySettings(ctx context.Context, settings *BetaPolicySettings) error {
	if settings == nil {
		return fmt.Errorf("settings cannot be nil")
	}

	validActions := map[string]bool{
		BetaPolicyActionPass:   true,
		BetaPolicyActionFilter: true,
		BetaPolicyActionBlock:  true,
	}
	validScopes := map[string]bool{
		BetaPolicyScopeAll:     true,
		BetaPolicyScopeOAuth:   true,
		BetaPolicyScopeAPIKey:  true,
		BetaPolicyScopeBedrock: true,
	}
	for i, rule := range settings.Rules {
		if rule.BetaToken == "" {
			return fmt.Errorf("rule[%d]: beta_token cannot be empty", i)
		}
		if !validActions[rule.Action] {
			return fmt.Errorf("rule[%d]: invalid action %q", i, rule.Action)
		}
		if !validScopes[rule.Scope] {
			return fmt.Errorf("rule[%d]: invalid scope %q", i, rule.Scope)
		}
	}

	data, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("marshal beta policy settings: %w", err)
	}
	return s.settingRepo.Set(ctx, SettingKeyBetaPolicySettings, string(data))
}

func (s *SettingService) SetStreamTimeoutSettings(ctx context.Context, settings *StreamTimeoutSettings) error {
	if settings == nil {
		return fmt.Errorf("settings cannot be nil")
	}
	if settings.TempUnschedMinutes < 1 || settings.TempUnschedMinutes > 60 {
		return fmt.Errorf("temp_unsched_minutes must be between 1-60")
	}
	if settings.ThresholdCount < 1 || settings.ThresholdCount > 10 {
		return fmt.Errorf("threshold_count must be between 1-10")
	}
	if settings.ThresholdWindowMinutes < 1 || settings.ThresholdWindowMinutes > 60 {
		return fmt.Errorf("threshold_window_minutes must be between 1-60")
	}

	switch settings.Action {
	case StreamTimeoutActionTempUnsched, StreamTimeoutActionError, StreamTimeoutActionNone:
	default:
		return fmt.Errorf("invalid action: %s", settings.Action)
	}

	data, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("marshal stream timeout settings: %w", err)
	}
	return s.settingRepo.Set(ctx, SettingKeyStreamTimeoutSettings, string(data))
}

func logClaudeCodeVersionBoundsFallback(err error) {
	if err == nil {
		return
	}
	slog.Warn("failed to load claude code version bounds, skipping version bound check", "error", err)
}
