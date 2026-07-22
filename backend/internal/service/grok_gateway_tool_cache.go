package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	gocache "github.com/patrickmn/go-cache"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const (
	grokToolPromptCacheTTL       = 5 * time.Minute
	grokToolPromptCacheKeyPrefix = "grok_tool_"
)

type grokRuntimeIdentityContextKey struct{}

type GrokRuntimeIdentity struct {
	UserID   int64
	APIKeyID int64
}

type grokToolPromptCacheEntry struct {
	PromptCacheKey string
}

func (s *GrokGatewayService) SetRuntimeIdentity(ctx context.Context, identity GrokRuntimeIdentity) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if identity.UserID <= 0 && identity.APIKeyID <= 0 {
		return ctx
	}
	return context.WithValue(ctx, grokRuntimeIdentityContextKey{}, identity)
}

func grokRuntimeIdentityFromContext(ctx context.Context) GrokRuntimeIdentity {
	if ctx == nil {
		return GrokRuntimeIdentity{}
	}
	if identity, ok := ctx.Value(grokRuntimeIdentityContextKey{}).(GrokRuntimeIdentity); ok {
		return identity
	}
	return GrokRuntimeIdentity{}
}

func (s *GrokGatewayService) ensureToolPromptCache() *gocache.Cache {
	if s == nil {
		return nil
	}
	if s.toolPromptCache == nil {
		s.toolPromptCache = gocache.New(grokToolPromptCacheTTL, time.Minute)
	}
	return s.toolPromptCache
}

func (s *GrokGatewayService) applyGrokToolPromptCache(ctx context.Context, account *Account, body []byte) []byte {
	if len(body) == 0 || !gjson.ValidBytes(body) {
		return body
	}
	if strings.TrimSpace(gjson.GetBytes(body, "prompt_cache_key").String()) != "" {
		return body
	}
	tools := gjson.GetBytes(body, "tools")
	if !tools.Exists() || !tools.IsArray() || len(tools.Array()) == 0 {
		return body
	}
	cache := s.ensureToolPromptCache()
	if cache == nil {
		return body
	}
	cacheKey := buildGrokToolPromptCacheLookupKey(ctx, account, body)
	if cacheKey == "" {
		return body
	}
	promptCacheKey := ""
	if cached, ok := cache.Get(cacheKey); ok {
		if entry, castOK := cached.(grokToolPromptCacheEntry); castOK {
			promptCacheKey = strings.TrimSpace(entry.PromptCacheKey)
		}
	}
	if promptCacheKey == "" {
		promptCacheKey = grokToolPromptCacheKeyPrefix + shortSHA256(cacheKey, 32)
		cache.Set(cacheKey, grokToolPromptCacheEntry{PromptCacheKey: promptCacheKey}, grokToolPromptCacheTTL)
	}
	next, err := sjson.SetBytes(body, "prompt_cache_key", promptCacheKey)
	if err != nil {
		return body
	}
	return next
}

func buildGrokToolPromptCacheLookupKey(ctx context.Context, account *Account, body []byte) string {
	identity := grokRuntimeIdentityFromContext(ctx)
	accountID := int64(0)
	routeMode := ""
	if account != nil {
		accountID = account.ID
		routeMode = grokRouteModeForAccount(account)
	}
	model := strings.TrimSpace(gjson.GetBytes(body, "model").String())
	session := grokToolPromptCacheSessionSeed(body)
	toolFingerprint := grokToolFingerprint(body)
	if toolFingerprint == "" {
		return ""
	}
	parts := []string{
		"v1",
		"account", fmt.Sprintf("%d", accountID),
		"route", routeMode,
		"user", fmt.Sprintf("%d", identity.UserID),
		"api_key", fmt.Sprintf("%d", identity.APIKeyID),
		"session", session,
		"model", model,
		"tools", toolFingerprint,
	}
	return strings.Join(parts, "|")
}

func grokToolPromptCacheSessionSeed(body []byte) string {
	for _, path := range []string{"session_id", "conversation_id"} {
		if value := strings.TrimSpace(gjson.GetBytes(body, path).String()); value != "" {
			return shortSHA256(value, 24)
		}
	}
	inputFingerprint := grokRequestInputFingerprint(body)
	if inputFingerprint != "" {
		return inputFingerprint
	}
	return "no-session"
}

func grokToolFingerprint(body []byte) string {
	raw := gjson.GetBytes(body, "tools").Raw
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return shortSHA256(raw, 32)
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return shortSHA256(raw, 32)
	}
	return shortSHA256(string(canonical), 32)
}

func grokRequestInputFingerprint(body []byte) string {
	var parts []string
	for _, path := range []string{"input", "messages", "instructions"} {
		if raw := strings.TrimSpace(gjson.GetBytes(body, path).Raw); raw != "" {
			parts = append(parts, raw)
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return shortSHA256(strings.Join(parts, "\n"), 24)
}

func shortSHA256(value string, chars int) string {
	sum := sha256.Sum256([]byte(value))
	out := hex.EncodeToString(sum[:])
	if chars > 0 && chars < len(out) {
		return out[:chars]
	}
	return out
}
