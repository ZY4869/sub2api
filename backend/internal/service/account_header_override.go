package service

import (
	"fmt"
	"net/http"
	"strings"
)

const AccountRequestHeadersCredentialKey = "request_headers"

var blockedAccountRequestHeaderOverrides = map[string]struct{}{
	"authorization":       {},
	"x-api-key":           {},
	"api-key":             {},
	"x-goog-api-key":      {},
	"cookie":              {},
	"set-cookie":          {},
	"host":                {},
	"content-length":      {},
	"connection":          {},
	"transfer-encoding":   {},
	"proxy-authorization": {},
	"proxy-authenticate":  {},
	"te":                  {},
	"trailer":             {},
	"upgrade":             {},
	"x-grok-conv-id":      {},
}

func ApplyAccountRequestHeaderOverrides(req *http.Request, account *Account) {
	if req == nil {
		return
	}
	for key, values := range AccountRequestHeaderOverrides(account) {
		req.Header.Del(key)
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
}

func AccountRequestHeaderOverrides(account *Account) http.Header {
	if !accountSupportsRequestHeaderOverrides(account) {
		return http.Header{}
	}
	raw := firstAccountRequestHeaderOverrideValue(account)
	rawMap, ok := raw.(map[string]any)
	if !ok || len(rawMap) == 0 {
		return http.Header{}
	}
	out := http.Header{}
	for rawName, rawValue := range rawMap {
		name := normalizeAccountRequestHeaderName(rawName)
		if name == "" || isBlockedAccountRequestHeaderOverride(name) {
			continue
		}
		values := normalizeAccountRequestHeaderValues(rawValue)
		if len(values) == 0 {
			continue
		}
		out[http.CanonicalHeaderKey(name)] = values
	}
	return out
}

func accountSupportsRequestHeaderOverrides(account *Account) bool {
	if account == nil || account.Type != AccountTypeAPIKey {
		return false
	}
	switch EffectiveProtocol(account) {
	case PlatformOpenAI, PlatformAnthropic, PlatformGrok, PlatformDeepSeek, PlatformOpenRouter, PlatformProtocolGateway:
		return true
	default:
		return false
	}
}

func firstAccountRequestHeaderOverrideValue(account *Account) any {
	if account == nil {
		return nil
	}
	for _, key := range []string{AccountRequestHeadersCredentialKey, "request_header_overrides", "header_overrides"} {
		if account.Credentials != nil {
			if value, ok := account.Credentials[key]; ok {
				return value
			}
		}
		if account.Extra != nil {
			if value, ok := account.Extra[key]; ok {
				return value
			}
		}
	}
	return nil
}

func normalizeAccountRequestHeaderName(value string) string {
	name := strings.TrimSpace(value)
	if name == "" || len(name) > 128 || !isHTTPToken(name) {
		return ""
	}
	return strings.ToLower(name)
}

func isBlockedAccountRequestHeaderOverride(name string) bool {
	_, blocked := blockedAccountRequestHeaderOverrides[strings.ToLower(strings.TrimSpace(name))]
	return blocked
}

func normalizeAccountRequestHeaderValues(raw any) []string {
	switch value := raw.(type) {
	case string:
		return normalizeAccountRequestHeaderStringValues([]string{value})
	case []string:
		return normalizeAccountRequestHeaderStringValues(value)
	case []any:
		values := make([]string, 0, len(value))
		for _, item := range value {
			values = append(values, fmt.Sprint(item))
		}
		return normalizeAccountRequestHeaderStringValues(values)
	case nil:
		return nil
	default:
		return normalizeAccountRequestHeaderStringValues([]string{fmt.Sprint(value)})
	}
}

func normalizeAccountRequestHeaderStringValues(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" || strings.ContainsAny(trimmed, "\r\n") {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}

func isHTTPToken(value string) bool {
	for _, ch := range value {
		if ch <= 32 || ch >= 127 {
			return false
		}
		switch ch {
		case '(', ')', '<', '>', '@', ',', ';', ':', '\\', '"', '/', '[', ']', '?', '=', '{', '}':
			return false
		}
	}
	return true
}
