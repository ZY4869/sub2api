package service

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"go.uber.org/zap"
)

func (s *GatewayService) maybeInjectAnthropicCacheTTL1h(ctx context.Context, account *Account, body []byte) []byte {
	if s == nil || s.settingService == nil || account == nil || len(body) == 0 {
		return body
	}
	if !account.IsAnthropicOAuthOrSetupToken() {
		return body
	}
	if !s.settingService.IsAnthropicCacheTTL1hInjectionEnabled(ctx) {
		return body
	}

	updated, changed := injectAnthropicCacheControlTTL(body, "1h")
	if changed {
		logger.FromContext(ctx).With(
			zap.String("component", "service.gateway"),
			zap.Int64("account_id", account.ID),
			zap.String("account_type", string(account.Type)),
		).Debug("anthropic cache_control.ttl injected", zap.String("ttl", "1h"))
	}
	return updated
}

func injectAnthropicCacheControlTTL(body []byte, ttl string) ([]byte, bool) {
	ttl = strings.TrimSpace(ttl)
	if len(body) == 0 || ttl == "" {
		return body, false
	}
	if !bytes.Contains(body, []byte(`"cache_control"`)) {
		return body, false
	}

	updated := body
	changed := false

	system := gjson.GetBytes(body, "system")
	if system.IsArray() {
		blocks := system.Array()
		for i := range blocks {
			ccType := strings.TrimSpace(blocks[i].Get("cache_control.type").String())
			if ccType != "ephemeral" {
				continue
			}
			existingTTL := strings.TrimSpace(blocks[i].Get("cache_control.ttl").String())
			if existingTTL == ttl {
				continue
			}
			next, err := sjson.SetBytes(updated, fmt.Sprintf("system.%d.cache_control.ttl", i), ttl)
			if err != nil {
				continue
			}
			updated = next
			changed = true
		}
	}

	messages := gjson.GetBytes(body, "messages")
	if messages.IsArray() {
		msgItems := messages.Array()
		for mi := range msgItems {
			content := msgItems[mi].Get("content")
			if !content.IsArray() {
				continue
			}
			parts := content.Array()
			for ci := range parts {
				ccType := strings.TrimSpace(parts[ci].Get("cache_control.type").String())
				if ccType != "ephemeral" {
					continue
				}
				existingTTL := strings.TrimSpace(parts[ci].Get("cache_control.ttl").String())
				if existingTTL == ttl {
					continue
				}
				next, err := sjson.SetBytes(updated, fmt.Sprintf("messages.%d.content.%d.cache_control.ttl", mi, ci), ttl)
				if err != nil {
					continue
				}
				updated = next
				changed = true
			}
		}
	}

	return updated, changed
}
