package service

import (
	"net/http"
	"strings"
)

const (
	openRouterHTTPRefererCredentialKey = "http_referer"
	openRouterTitleCredentialKey       = "openrouter_title"
)

func applyOpenRouterAttributionHeaders(account *Account, headers map[string]string) {
	if account == nil || account.Platform != PlatformOpenRouter || headers == nil {
		return
	}
	if referer := strings.TrimSpace(account.GetCredential(openRouterHTTPRefererCredentialKey)); referer != "" {
		headers["HTTP-Referer"] = referer
	}
	if title := strings.TrimSpace(account.GetCredential(openRouterTitleCredentialKey)); title != "" {
		headers["X-OpenRouter-Title"] = title
	}
}

func applyOpenRouterAttributionRequestHeaders(account *Account, headers http.Header) {
	if account == nil || account.Platform != PlatformOpenRouter || headers == nil {
		return
	}
	if referer := strings.TrimSpace(account.GetCredential(openRouterHTTPRefererCredentialKey)); referer != "" {
		headers.Set("HTTP-Referer", referer)
	}
	if title := strings.TrimSpace(account.GetCredential(openRouterTitleCredentialKey)); title != "" {
		headers.Set("X-OpenRouter-Title", title)
	}
}
