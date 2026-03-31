package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/cespare/xxhash/v2"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"net/http"
	"strconv"
	"strings"
)

func (s *GatewayService) extractCacheableContent(parsed *ParsedRequest) string {
	if parsed == nil {
		return ""
	}
	var builder strings.Builder
	if system, ok := parsed.System.([]any); ok {
		for _, part := range system {
			if partMap, ok := part.(map[string]any); ok {
				if cc, ok := partMap["cache_control"].(map[string]any); ok {
					if cc["type"] == "ephemeral" {
						if text, ok := partMap["text"].(string); ok {
							_, _ = builder.WriteString(text)
						}
					}
				}
			}
		}
	}
	systemText := builder.String()
	for _, msg := range parsed.Messages {
		if msgMap, ok := msg.(map[string]any); ok {
			if msgContent, ok := msgMap["content"].([]any); ok {
				for _, part := range msgContent {
					if partMap, ok := part.(map[string]any); ok {
						if cc, ok := partMap["cache_control"].(map[string]any); ok {
							if cc["type"] == "ephemeral" {
								return s.extractTextFromContent(msgMap["content"])
							}
						}
					}
				}
			}
		}
	}
	return systemText
}
func (s *GatewayService) extractTextFromSystem(system any) string {
	switch v := system.(type) {
	case string:
		return v
	case []any:
		var texts []string
		for _, part := range v {
			if partMap, ok := part.(map[string]any); ok {
				if text, ok := partMap["text"].(string); ok {
					texts = append(texts, text)
				}
			}
		}
		return strings.Join(texts, "")
	}
	return ""
}
func (s *GatewayService) extractTextFromContent(content any) string {
	switch v := content.(type) {
	case string:
		return v
	case []any:
		var texts []string
		for _, part := range v {
			if partMap, ok := part.(map[string]any); ok {
				if partMap["type"] == "text" {
					if text, ok := partMap["text"].(string); ok {
						texts = append(texts, text)
					}
				}
			}
		}
		return strings.Join(texts, "")
	}
	return ""
}
func (s *GatewayService) hashContent(content string) string {
	h := xxhash.Sum64String(content)
	return strconv.FormatUint(h, 36)
}
func (s *GatewayService) replaceModelInBody(body []byte, newModel string) []byte {
	if len(body) == 0 {
		return body
	}
	if current := gjson.GetBytes(body, "model"); current.Exists() && current.String() == newModel {
		return body
	}
	newBody, err := sjson.SetBytes(body, "model", newModel)
	if err != nil {
		return body
	}
	return newBody
}

type claudeOAuthNormalizeOptions struct {
	injectMetadata          bool
	metadataUserID          string
	stripSystemCacheControl bool
}

func sanitizeSystemText(text string) string {
	if text == "" {
		return text
	}
	text = strings.ReplaceAll(text, "You are OpenCode, the best coding agent on the planet.", strings.TrimSpace(claudeCodeSystemPrompt))
	return text
}
func stripCacheControlFromSystemBlocks(system any) bool {
	blocks, ok := system.([]any)
	if !ok {
		return false
	}
	changed := false
	for _, item := range blocks {
		block, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if _, exists := block["cache_control"]; !exists {
			continue
		}
		delete(block, "cache_control")
		changed = true
	}
	return changed
}
func normalizeClaudeOAuthRequestBody(body []byte, modelID string, opts claudeOAuthNormalizeOptions) ([]byte, string) {
	if len(body) == 0 {
		return body, modelID
	}
	normalizedModelID := claude.NormalizeModelID(modelID)
	if normalizedModelID == "" {
		normalizedModelID = modelID
	}

	normalized := body
	system := gjson.GetBytes(normalized, "system")
	if system.Exists() {
		switch {
		case system.Type == gjson.String:
			sanitized := sanitizeSystemText(system.String())
			if sanitized != system.String() {
				if next, err := sjson.SetBytes(normalized, "system", sanitized); err == nil {
					normalized = next
				}
			}
		case system.IsArray():
			var blocks []any
			if err := json.Unmarshal([]byte(system.Raw), &blocks); err == nil {
				changed := opts.stripSystemCacheControl && stripCacheControlFromSystemBlocks(blocks)
				for _, item := range blocks {
					block, ok := item.(map[string]any)
					if !ok {
						continue
					}
					if blockType, _ := block["type"].(string); blockType != "text" {
						continue
					}
					text, ok := block["text"].(string)
					if !ok || text == "" {
						continue
					}
					sanitized := sanitizeSystemText(text)
					if sanitized != text {
						block["text"] = sanitized
						changed = true
					}
				}
				if changed {
					if next, err := sjson.SetBytes(normalized, "system", blocks); err == nil {
						normalized = next
					}
				}
			}
		}
	}

	if !gjson.GetBytes(normalized, "tools").Exists() {
		if next, err := sjson.SetBytes(normalized, "tools", []any{}); err == nil {
			normalized = next
		}
	}
	if opts.injectMetadata && opts.metadataUserID != "" && strings.TrimSpace(gjson.GetBytes(normalized, "metadata.user_id").String()) == "" {
		if next, err := sjson.SetBytes(normalized, "metadata.user_id", opts.metadataUserID); err == nil {
			normalized = next
		}
	}
	if gjson.GetBytes(normalized, "temperature").Exists() {
		if next, err := sjson.DeleteBytes(normalized, "temperature"); err == nil {
			normalized = next
		}
	}
	if gjson.GetBytes(normalized, "tool_choice").Exists() {
		if next, err := sjson.DeleteBytes(normalized, "tool_choice"); err == nil {
			normalized = next
		}
	}
	return normalized, normalizedModelID
}
func (s *GatewayService) buildOAuthMetadataUserID(parsed *ParsedRequest, account *Account, fp *Fingerprint) string {
	if parsed == nil || account == nil {
		return ""
	}
	if parsed.MetadataUserID != "" {
		return ""
	}
	userID := strings.TrimSpace(account.GetClaudeUserID())
	if userID == "" && fp != nil {
		userID = fp.ClientID
	}
	if userID == "" {
		userID = generateClientID()
	}
	sessionHash := s.GenerateSessionHash(parsed)
	sessionID := uuid.NewString()
	if sessionHash != "" {
		seed := fmt.Sprintf("%d::%s", account.ID, sessionHash)
		sessionID = generateSessionUUID(seed)
	}
	accountUUID := strings.TrimSpace(account.GetExtraString("account_uuid"))
	if accountUUID != "" {
		return fmt.Sprintf("user_%s_account_%s_session_%s", userID, accountUUID, sessionID)
	}
	return fmt.Sprintf("user_%s_account__session_%s", userID, sessionID)
}
func isClaudeCodeClient(userAgent string, metadataUserID string) bool {
	if metadataUserID == "" {
		return false
	}
	return claudeCliUserAgentRe.MatchString(userAgent)
}
func isClaudeCodeRequest(ctx context.Context, c *gin.Context, parsed *ParsedRequest) bool {
	if IsClaudeCodeClient(ctx) {
		return true
	}
	if parsed == nil || c == nil {
		return false
	}
	return isClaudeCodeClient(c.GetHeader("User-Agent"), parsed.MetadataUserID)
}
func normalizeSystemParam(system any) any {
	raw, ok := system.(json.RawMessage)
	if !ok {
		return system
	}
	if len(raw) == 0 {
		return nil
	}
	var parsed any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil
	}
	return parsed
}
func systemIncludesClaudeCodePrompt(system any) bool {
	system = normalizeSystemParam(system)
	switch v := system.(type) {
	case string:
		return hasClaudeCodePrefix(v)
	case []any:
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				if text, ok := m["text"].(string); ok && hasClaudeCodePrefix(text) {
					return true
				}
			}
		}
	}
	return false
}
func hasClaudeCodePrefix(text string) bool {
	for _, prefix := range claudeCodePromptPrefixes {
		if strings.HasPrefix(text, prefix) {
			return true
		}
	}
	return false
}
func matchesFilterPrefix(text string) bool {
	for _, prefix := range systemBlockFilterPrefixes {
		if strings.HasPrefix(text, prefix) {
			return true
		}
	}
	return false
}
func filterSystemBlocksByPrefix(body []byte) []byte {
	sys := gjson.GetBytes(body, "system")
	if !sys.Exists() {
		return body
	}
	switch {
	case sys.Type == gjson.String:
		if matchesFilterPrefix(sys.Str) {
			result, err := sjson.DeleteBytes(body, "system")
			if err != nil {
				return body
			}
			return result
		}
	case sys.IsArray():
		var parsed []any
		if err := json.Unmarshal([]byte(sys.Raw), &parsed); err != nil {
			return body
		}
		filtered := make([]any, 0, len(parsed))
		changed := false
		for _, item := range parsed {
			if m, ok := item.(map[string]any); ok {
				if text, ok := m["text"].(string); ok && matchesFilterPrefix(text) {
					changed = true
					continue
				}
			}
			filtered = append(filtered, item)
		}
		if changed {
			result, err := sjson.SetBytes(body, "system", filtered)
			if err != nil {
				return body
			}
			return result
		}
	}
	return body
}
func injectClaudeCodePrompt(body []byte, system any) []byte {
	system = normalizeSystemParam(system)
	claudeCodePrefix := strings.TrimSpace(claudeCodeSystemPrompt)

	claudeCodeBlockRaw := `{"type":"text","text":` + strconv.Quote(claudeCodeSystemPrompt) + `,"cache_control":{"type":"ephemeral"}}`
	setSystemRaw := func(systemRaw string) []byte {
		result, err := sjson.SetRawBytes(body, "system", []byte(systemRaw))
		if err != nil {
			logger.LegacyPrintf("service.gateway", "Warning: failed to inject Claude Code prompt: %v", err)
			return body
		}
		return result
	}
	resolveRawSystemItem := func(index int, item any) string {
		if raw := gjson.GetBytes(body, fmt.Sprintf("system.%d", index)).Raw; raw != "" {
			return raw
		}
		marshaled, err := json.Marshal(item)
		if err != nil {
			return ""
		}
		return string(marshaled)
	}

	switch v := system.(type) {
	case nil:
		return setSystemRaw("[" + claudeCodeBlockRaw + "]")
	case string:
		if strings.TrimSpace(v) == "" || strings.TrimSpace(v) == strings.TrimSpace(claudeCodeSystemPrompt) {
			return setSystemRaw("[" + claudeCodeBlockRaw + "]")
		}
		merged := v
		if !strings.HasPrefix(v, claudeCodePrefix) {
			merged = claudeCodePrefix + "\n\n" + v
		}
		textBlockRaw := `{"type":"text","text":` + strconv.Quote(merged) + `}`
		return setSystemRaw("[" + claudeCodeBlockRaw + "," + textBlockRaw + "]")
	case []any:
		rawItems := make([]string, 0, len(v)+1)
		rawItems = append(rawItems, claudeCodeBlockRaw)
		prefixedNext := false
		for idx, item := range v {
			rawItem := resolveRawSystemItem(idx, item)
			if m, ok := item.(map[string]any); ok {
				if text, ok := m["text"].(string); ok && strings.TrimSpace(text) == strings.TrimSpace(claudeCodeSystemPrompt) {
					continue
				}
				if !prefixedNext {
					if blockType, _ := m["type"].(string); blockType == "text" {
						if text, ok := m["text"].(string); ok && strings.TrimSpace(text) != "" && !strings.HasPrefix(text, claudeCodePrefix) {
							if rawItem != "" {
								if next, err := sjson.SetBytes([]byte(rawItem), "text", claudeCodePrefix+"\n\n"+text); err == nil {
									rawItem = string(next)
								}
							} else {
								m["text"] = claudeCodePrefix + "\n\n" + text
								if marshaled, err := json.Marshal(m); err == nil {
									rawItem = string(marshaled)
								}
							}
							prefixedNext = true
						}
					}
				}
			}
			if rawItem != "" {
				rawItems = append(rawItems, rawItem)
			}
		}
		return setSystemRaw("[" + strings.Join(rawItems, ",") + "]")
	default:
		return setSystemRaw("[" + claudeCodeBlockRaw + "]")
	}
}
func enforceCacheControlLimit(body []byte) []byte {
	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return body
	}
	removeCacheControlFromThinkingBlocks(data)
	count := countCacheControlBlocks(data)
	if count <= maxCacheControlBlocks {
		return body
	}
	for count > maxCacheControlBlocks {
		if removeCacheControlFromMessages(data) {
			count--
			continue
		}
		if removeCacheControlFromSystem(data) {
			count--
			continue
		}
		break
	}
	result := body
	var err error
	if system, ok := data["system"]; ok {
		result, err = sjson.SetBytes(result, "system", system)
		if err != nil {
			return body
		}
	}
	if messages, ok := data["messages"]; ok {
		result, err = sjson.SetBytes(result, "messages", messages)
		if err != nil {
			return body
		}
	}
	return result
}
func countCacheControlBlocks(data map[string]any) int {
	count := 0
	if system, ok := data["system"].([]any); ok {
		for _, item := range system {
			if m, ok := item.(map[string]any); ok {
				if blockType, _ := m["type"].(string); blockType == "thinking" {
					continue
				}
				if _, has := m["cache_control"]; has {
					count++
				}
			}
		}
	}
	if messages, ok := data["messages"].([]any); ok {
		for _, msg := range messages {
			if msgMap, ok := msg.(map[string]any); ok {
				if content, ok := msgMap["content"].([]any); ok {
					for _, item := range content {
						if m, ok := item.(map[string]any); ok {
							if blockType, _ := m["type"].(string); blockType == "thinking" {
								continue
							}
							if _, has := m["cache_control"]; has {
								count++
							}
						}
					}
				}
			}
		}
	}
	return count
}
func removeCacheControlFromMessages(data map[string]any) bool {
	messages, ok := data["messages"].([]any)
	if !ok {
		return false
	}
	for _, msg := range messages {
		msgMap, ok := msg.(map[string]any)
		if !ok {
			continue
		}
		content, ok := msgMap["content"].([]any)
		if !ok {
			continue
		}
		for _, item := range content {
			if m, ok := item.(map[string]any); ok {
				if blockType, _ := m["type"].(string); blockType == "thinking" {
					continue
				}
				if _, has := m["cache_control"]; has {
					delete(m, "cache_control")
					return true
				}
			}
		}
	}
	return false
}
func removeCacheControlFromSystem(data map[string]any) bool {
	system, ok := data["system"].([]any)
	if !ok {
		return false
	}
	for i := len(system) - 1; i >= 0; i-- {
		if m, ok := system[i].(map[string]any); ok {
			if blockType, _ := m["type"].(string); blockType == "thinking" {
				continue
			}
			if _, has := m["cache_control"]; has {
				delete(m, "cache_control")
				return true
			}
		}
	}
	return false
}
func removeCacheControlFromThinkingBlocks(data map[string]any) {
	if system, ok := data["system"].([]any); ok {
		for _, item := range system {
			if m, ok := item.(map[string]any); ok {
				if blockType, _ := m["type"].(string); blockType == "thinking" {
					if _, has := m["cache_control"]; has {
						delete(m, "cache_control")
						logger.LegacyPrintf("service.gateway", "[Warning] Removed illegal cache_control from thinking block in system")
					}
				}
			}
		}
	}
	if messages, ok := data["messages"].([]any); ok {
		for msgIdx, msg := range messages {
			if msgMap, ok := msg.(map[string]any); ok {
				if content, ok := msgMap["content"].([]any); ok {
					for contentIdx, item := range content {
						if m, ok := item.(map[string]any); ok {
							if blockType, _ := m["type"].(string); blockType == "thinking" {
								if _, has := m["cache_control"]; has {
									delete(m, "cache_control")
									logger.LegacyPrintf("service.gateway", "[Warning] Removed illegal cache_control from thinking block in messages[%d].content[%d]", msgIdx, contentIdx)
								}
							}
						}
					}
				}
			}
		}
	}
}
func (s *GatewayService) buildUpstreamRequest(ctx context.Context, c *gin.Context, account *Account, body []byte, token, tokenType, modelID string, reqStream bool, mimicClaudeCode bool) (*http.Request, error) {
	targetURL, err := s.resolveAnthropicTargetURL(account, anthropicMessagesPath, claudeAPIURL)
	if err != nil {
		return nil, err
	}
	clientHeaders := http.Header{}
	if c != nil && c.Request != nil {
		clientHeaders = c.Request.Header
	}
	var fingerprint *Fingerprint
	if mimicClaudeCode && s.identityService != nil {
		fp, err := s.identityService.GetOrCreateFingerprint(ctx, account.ID, clientHeaders)
		if err != nil {
			logger.LegacyPrintf("service.gateway", "Warning: failed to get fingerprint for account %d: %v", account.ID, err)
		} else {
			fingerprint = fp
			accountUUID := account.GetExtraString("account_uuid")
			if accountUUID != "" && fp.ClientID != "" {
				if newBody, err := s.identityService.RewriteUserIDWithMasking(ctx, body, account, accountUUID, fp.ClientID, fp.UserAgent); err == nil && len(newBody) > 0 {
					body = newBody
				}
			}
		}
	}
	req, err := http.NewRequestWithContext(ctx, "POST", targetURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if tokenType == "oauth" {
		req.Header.Set("authorization", "Bearer "+token)
	} else {
		req.Header.Set("x-api-key", token)
	}
	for key, values := range clientHeaders {
		lowerKey := strings.ToLower(key)
		if allowedHeaders[lowerKey] {
			for _, v := range values {
				req.Header.Add(key, v)
			}
		}
	}
	if fingerprint != nil {
		s.identityService.ApplyFingerprint(req, fingerprint)
	}
	if req.Header.Get("content-type") == "" {
		req.Header.Set("content-type", "application/json")
	}
	if req.Header.Get("anthropic-version") == "" {
		req.Header.Set("anthropic-version", "2023-06-01")
	}
	if tokenType == "oauth" || mimicClaudeCode {
		applyClaudeOAuthHeaderDefaults(req, reqStream)
	}
	policyFilterSet := s.getBetaPolicyFilterSet(ctx, c, account)
	effectiveDropSet := mergeDropSets(policyFilterSet)
	effectiveDropWithClaudeCodeSet := mergeDropSets(policyFilterSet, claude.BetaClaudeCode)
	if tokenType == "oauth" {
		if mimicClaudeCode {
			applyClaudeCodeMimicHeaders(req, reqStream)
			incomingBeta := req.Header.Get("anthropic-beta")
			requiredBetas := []string{claude.BetaOAuth, claude.BetaInterleavedThinking}
			req.Header.Set("anthropic-beta", mergeAnthropicBetaDropping(requiredBetas, incomingBeta, effectiveDropWithClaudeCodeSet))
		} else {
			clientBetaHeader := req.Header.Get("anthropic-beta")
			req.Header.Set("anthropic-beta", stripBetaTokensWithSet(s.getBetaHeader(modelID, clientBetaHeader), effectiveDropSet))
		}
	} else {
		if mimicClaudeCode {
			applyClaudeCodeMimicHeaders(req, reqStream)
			incomingBeta := req.Header.Get("anthropic-beta")
			requiredBetas := []string{claude.BetaClaudeCode, claude.BetaInterleavedThinking, claude.BetaFineGrainedToolStreaming}
			req.Header.Set("anthropic-beta", mergeAnthropicBetaDropping(requiredBetas, incomingBeta, effectiveDropWithClaudeCodeSet))
		} else if existingBeta := req.Header.Get("anthropic-beta"); existingBeta != "" {
			req.Header.Set("anthropic-beta", stripBetaTokensWithSet(existingBeta, effectiveDropSet))
		} else if s.cfg != nil && s.cfg.Gateway.InjectBetaForAPIKey {
			if requestNeedsBetaFeatures(body) {
				if beta := defaultAPIKeyBetaHeader(body); beta != "" {
					req.Header.Set("anthropic-beta", beta)
				}
			}
		}
	}
	if c != nil && (tokenType == "oauth" || mimicClaudeCode) {
		c.Set(claudeMimicDebugInfoKey, buildClaudeMimicDebugLine(req, body, account, tokenType, mimicClaudeCode))
	}
	syncClaudeCodeSessionHeader(req, body)
	if s.debugClaudeMimicEnabled() {
		logClaudeMimicDebug(req, body, account, tokenType, mimicClaudeCode)
	}
	return req, nil
}
func applyClaudeOAuthHeaderDefaults(req *http.Request, isStream bool) {
	if req == nil {
		return
	}
	if req.Header.Get("accept") == "" {
		req.Header.Set("accept", "application/json")
	}
	for key, value := range claude.DefaultHeaders {
		if value == "" {
			continue
		}
		if req.Header.Get(key) == "" {
			req.Header.Set(key, value)
		}
	}
	if isStream && req.Header.Get("x-stainless-helper-method") == "" {
		req.Header.Set("x-stainless-helper-method", "stream")
	}
}
func mergeAnthropicBeta(required []string, incoming string) string {
	seen := make(map[string]struct{}, len(required)+8)
	out := make([]string, 0, len(required)+8)
	add := func(v string) {
		v = strings.TrimSpace(v)
		if v == "" {
			return
		}
		if _, ok := seen[v]; ok {
			return
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	for _, r := range required {
		add(r)
	}
	for _, p := range strings.Split(incoming, ",") {
		add(p)
	}
	return strings.Join(out, ",")
}
