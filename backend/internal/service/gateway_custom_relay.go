package service

import (
	"fmt"
	"net/url"
	"strings"
)

const (
	anthropicMessagesPath    = "/v1/messages"
	anthropicCountTokensPath = "/v1/messages/count_tokens"
)

func resolveGatewayProxyURL(account *Account) string {
	if account == nil || account.ProxyID == nil || account.Proxy == nil {
		return ""
	}
	if account.IsCustomBaseURLEnabled() && account.GetCustomBaseURL() != "" {
		return ""
	}
	return account.Proxy.URL()
}

func (s *GatewayService) resolveAnthropicTargetURL(account *Account, path, defaultURL string) (string, error) {
	if account == nil {
		return defaultURL, nil
	}
	if account.Type == AccountTypeAPIKey {
		baseURL := account.GetBaseURL()
		if baseURL == "" {
			return defaultURL, nil
		}
		validatedURL, err := s.validateUpstreamBaseURL(baseURL)
		if err != nil {
			return "", err
		}
		return strings.TrimRight(validatedURL, "/") + path + "?beta=true", nil
	}
	if !account.IsCustomBaseURLEnabled() {
		return defaultURL, nil
	}
	customURL := account.GetCustomBaseURL()
	if customURL == "" {
		return "", fmt.Errorf("custom_base_url is enabled but not configured for account %d", account.ID)
	}
	validatedURL, err := s.validateUpstreamBaseURL(customURL)
	if err != nil {
		return "", err
	}
	return s.buildCustomRelayURL(validatedURL, path, account), nil
}

func (s *GatewayService) buildCustomRelayURL(baseURL, path string, account *Account) string {
	relayURL, err := url.Parse(strings.TrimRight(baseURL, "/") + path)
	if err != nil {
		return strings.TrimRight(baseURL, "/") + path + "?beta=true"
	}
	query := relayURL.Query()
	query.Set("beta", "true")
	if account != nil && account.ProxyID != nil && account.Proxy != nil {
		if proxyURL := strings.TrimSpace(account.Proxy.URL()); proxyURL != "" {
			query.Set("proxy", proxyURL)
		}
	}
	relayURL.RawQuery = query.Encode()
	return relayURL.String()
}
