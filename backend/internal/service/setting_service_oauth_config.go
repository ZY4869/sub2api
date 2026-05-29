package service

import (
	"context"
	"fmt"
	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"strings"
)

func (s *SettingService) GetLinuxDoConnectOAuthConfig(ctx context.Context) (config.LinuxDoConnectConfig, error) {
	if s == nil || s.cfg == nil {
		return config.LinuxDoConnectConfig{}, infraerrors.ServiceUnavailable("CONFIG_NOT_READY", "config not loaded")
	}
	effective := s.cfg.LinuxDo
	keys := []string{SettingKeyLinuxDoConnectEnabled, SettingKeyLinuxDoConnectClientID, SettingKeyLinuxDoConnectClientSecret, SettingKeyLinuxDoConnectRedirectURL}
	settings, err := s.settingRepo.GetMultiple(ctx, keys)
	if err != nil {
		return config.LinuxDoConnectConfig{}, fmt.Errorf("get linuxdo connect settings: %w", err)
	}
	if raw, ok := settings[SettingKeyLinuxDoConnectEnabled]; ok {
		effective.Enabled = raw == "true"
	}
	if v, ok := settings[SettingKeyLinuxDoConnectClientID]; ok && strings.TrimSpace(v) != "" {
		effective.ClientID = strings.TrimSpace(v)
	}
	if v, ok := settings[SettingKeyLinuxDoConnectClientSecret]; ok && strings.TrimSpace(v) != "" {
		effective.ClientSecret = strings.TrimSpace(v)
	}
	if v, ok := settings[SettingKeyLinuxDoConnectRedirectURL]; ok && strings.TrimSpace(v) != "" {
		effective.RedirectURL = strings.TrimSpace(v)
	}
	if !effective.Enabled {
		return config.LinuxDoConnectConfig{}, infraerrors.NotFound("OAUTH_DISABLED", "oauth login is disabled")
	}
	if strings.TrimSpace(effective.ClientID) == "" {
		return config.LinuxDoConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth client id not configured")
	}
	if strings.TrimSpace(effective.AuthorizeURL) == "" {
		return config.LinuxDoConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth authorize url not configured")
	}
	if strings.TrimSpace(effective.TokenURL) == "" {
		return config.LinuxDoConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth token url not configured")
	}
	if strings.TrimSpace(effective.UserInfoURL) == "" {
		return config.LinuxDoConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth userinfo url not configured")
	}
	if strings.TrimSpace(effective.RedirectURL) == "" {
		return config.LinuxDoConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth redirect url not configured")
	}
	if strings.TrimSpace(effective.FrontendRedirectURL) == "" {
		return config.LinuxDoConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth frontend redirect url not configured")
	}
	if err := config.ValidateAbsoluteHTTPURL(effective.AuthorizeURL); err != nil {
		return config.LinuxDoConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth authorize url invalid")
	}
	if err := config.ValidateAbsoluteHTTPURL(effective.TokenURL); err != nil {
		return config.LinuxDoConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth token url invalid")
	}
	if err := config.ValidateAbsoluteHTTPURL(effective.UserInfoURL); err != nil {
		return config.LinuxDoConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth userinfo url invalid")
	}
	if err := config.ValidateAbsoluteHTTPURL(effective.RedirectURL); err != nil {
		return config.LinuxDoConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth redirect url invalid")
	}
	if err := config.ValidateFrontendRedirectURL(effective.FrontendRedirectURL); err != nil {
		return config.LinuxDoConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth frontend redirect url invalid")
	}
	method := strings.ToLower(strings.TrimSpace(effective.TokenAuthMethod))
	switch method {
	case "", "client_secret_post", "client_secret_basic":
		if strings.TrimSpace(effective.ClientSecret) == "" {
			return config.LinuxDoConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth client secret not configured")
		}
	case "none":
		if !effective.UsePKCE {
			return config.LinuxDoConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth pkce must be enabled when token_auth_method=none")
		}
	default:
		return config.LinuxDoConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth token_auth_method invalid")
	}
	return effective, nil
}

func (s *SettingService) GetSocialOAuthConfig(ctx context.Context, provider string) (SocialOAuthConfig, error) {
	provider = NormalizeOAuthProvider(provider)
	if provider == "" {
		return SocialOAuthConfig{}, ErrOAuthProviderUnsupported
	}
	effective := defaultSocialOAuthConfig(provider)
	var keys []string
	switch provider {
	case AuthProviderGitHub:
		keys = []string{SettingKeyGitHubOAuthEnabled, SettingKeyGitHubOAuthClientID, SettingKeyGitHubOAuthClientSecret, SettingKeyGitHubOAuthRedirectURL}
	case AuthProviderGoogle:
		keys = []string{SettingKeyGoogleOAuthEnabled, SettingKeyGoogleOAuthClientID, SettingKeyGoogleOAuthClientSecret, SettingKeyGoogleOAuthRedirectURL}
	case AuthProviderDingTalk:
		keys = []string{SettingKeyDingTalkOAuthEnabled, SettingKeyDingTalkOAuthClientID, SettingKeyDingTalkOAuthClientSecret, SettingKeyDingTalkOAuthRedirectURL}
	}
	settings, err := s.settingRepo.GetMultiple(ctx, keys)
	if err != nil {
		return SocialOAuthConfig{}, err
	}
	switch provider {
	case AuthProviderGitHub:
		effective.Enabled = settings[SettingKeyGitHubOAuthEnabled] == "true"
		effective.ClientID = strings.TrimSpace(settings[SettingKeyGitHubOAuthClientID])
		effective.ClientSecret = strings.TrimSpace(settings[SettingKeyGitHubOAuthClientSecret])
		effective.RedirectURL = strings.TrimSpace(settings[SettingKeyGitHubOAuthRedirectURL])
	case AuthProviderGoogle:
		effective.Enabled = settings[SettingKeyGoogleOAuthEnabled] == "true"
		effective.ClientID = strings.TrimSpace(settings[SettingKeyGoogleOAuthClientID])
		effective.ClientSecret = strings.TrimSpace(settings[SettingKeyGoogleOAuthClientSecret])
		effective.RedirectURL = strings.TrimSpace(settings[SettingKeyGoogleOAuthRedirectURL])
	case AuthProviderDingTalk:
		effective.Enabled = settings[SettingKeyDingTalkOAuthEnabled] == "true"
		effective.ClientID = strings.TrimSpace(settings[SettingKeyDingTalkOAuthClientID])
		effective.ClientSecret = strings.TrimSpace(settings[SettingKeyDingTalkOAuthClientSecret])
		effective.RedirectURL = strings.TrimSpace(settings[SettingKeyDingTalkOAuthRedirectURL])
	}
	if !effective.Enabled {
		return SocialOAuthConfig{}, infraerrors.NotFound("OAUTH_DISABLED", "oauth login is disabled")
	}
	if strings.TrimSpace(effective.ClientID) == "" {
		return SocialOAuthConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth client id not configured")
	}
	if strings.TrimSpace(effective.ClientSecret) == "" {
		return SocialOAuthConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth client secret not configured")
	}
	if strings.TrimSpace(effective.RedirectURL) == "" {
		return SocialOAuthConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth redirect url not configured")
	}
	if err := config.ValidateAbsoluteHTTPURL(effective.AuthorizeURL); err != nil {
		return SocialOAuthConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth authorize url invalid")
	}
	if err := config.ValidateAbsoluteHTTPURL(effective.TokenURL); err != nil {
		return SocialOAuthConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth token url invalid")
	}
	if err := config.ValidateAbsoluteHTTPURL(effective.UserInfoURL); err != nil {
		return SocialOAuthConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth user info url invalid")
	}
	if err := config.ValidateAbsoluteHTTPURL(effective.RedirectURL); err != nil {
		return SocialOAuthConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth redirect url invalid")
	}
	if err := config.ValidateFrontendRedirectURL(effective.FrontendRedirectURL); err != nil {
		return SocialOAuthConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth frontend redirect url invalid")
	}
	return effective, nil
}
