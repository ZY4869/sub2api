package service

import (
	"context"
	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"strings"
)

func (s *GatewayService) getBetaHeader(modelID string, clientBetaHeader string) string {
	if clientBetaHeader != "" {
		if strings.Contains(clientBetaHeader, claude.BetaOAuth) {
			return clientBetaHeader
		}
		parts := strings.Split(clientBetaHeader, ",")
		for i, p := range parts {
			parts[i] = strings.TrimSpace(p)
		}
		claudeCodeIdx := -1
		for i, p := range parts {
			if p == claude.BetaClaudeCode {
				claudeCodeIdx = i
				break
			}
		}
		if claudeCodeIdx >= 0 {
			newParts := make([]string, 0, len(parts)+1)
			newParts = append(newParts, parts[:claudeCodeIdx+1]...)
			newParts = append(newParts, claude.BetaOAuth)
			newParts = append(newParts, parts[claudeCodeIdx+1:]...)
			return strings.Join(newParts, ",")
		}
		return claude.BetaOAuth + "," + clientBetaHeader
	}
	if strings.Contains(strings.ToLower(modelID), "haiku") {
		return claude.HaikuBetaHeader
	}
	return claude.DefaultBetaHeader
}
func requestNeedsBetaFeatures(body []byte) bool {
	tools := gjson.GetBytes(body, "tools")
	if tools.Exists() && tools.IsArray() && len(tools.Array()) > 0 {
		return true
	}
	thinkingType := gjson.GetBytes(body, "thinking.type").String()
	if strings.EqualFold(thinkingType, "enabled") || strings.EqualFold(thinkingType, "adaptive") {
		return true
	}
	return false
}
func defaultAPIKeyBetaHeader(body []byte) string {
	modelID := gjson.GetBytes(body, "model").String()
	if strings.Contains(strings.ToLower(modelID), "haiku") {
		return claude.APIKeyHaikuBetaHeader
	}
	return claude.APIKeyBetaHeader
}
func mergeAnthropicBetaDropping(required []string, incoming string, drop map[string]struct{}) string {
	merged := mergeAnthropicBeta(required, incoming)
	if merged == "" || len(drop) == 0 {
		return merged
	}
	out := make([]string, 0, 8)
	for _, p := range strings.Split(merged, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, ok := drop[p]; ok {
			continue
		}
		out = append(out, p)
	}
	return strings.Join(out, ",")
}
func stripBetaTokens(header string, tokens []string) string {
	if header == "" || len(tokens) == 0 {
		return header
	}
	return stripBetaTokensWithSet(header, buildBetaTokenSet(tokens))
}
func stripBetaTokensWithSet(header string, drop map[string]struct{}) string {
	if header == "" || len(drop) == 0 {
		return header
	}
	parts := strings.Split(header, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, ok := drop[p]; ok {
			continue
		}
		out = append(out, p)
	}
	if len(out) == len(parts) {
		return header
	}
	return strings.Join(out, ",")
}

type BetaBlockedError struct{ Message string }

func (e *BetaBlockedError) Error() string {
	return e.Message
}

type betaPolicyResult struct {
	blockErr  *BetaBlockedError
	filterSet map[string]struct{}
}

func (s *GatewayService) evaluateBetaPolicy(ctx context.Context, betaHeader string, account *Account) betaPolicyResult {
	if s.settingService == nil {
		return betaPolicyResult{}
	}
	settings, err := s.settingService.GetBetaPolicySettings(ctx)
	if err != nil || settings == nil {
		return betaPolicyResult{}
	}
	isOAuth := account.IsOAuth()
	var result betaPolicyResult
	for _, rule := range settings.Rules {
		if !betaPolicyScopeMatches(rule.Scope, isOAuth) {
			continue
		}
		switch rule.Action {
		case BetaPolicyActionBlock:
			if result.blockErr == nil && betaHeader != "" && containsBetaToken(betaHeader, rule.BetaToken) {
				msg := rule.ErrorMessage
				if msg == "" {
					msg = "beta feature " + rule.BetaToken + " is not allowed"
				}
				result.blockErr = &BetaBlockedError{Message: msg}
			}
		case BetaPolicyActionFilter:
			if result.filterSet == nil {
				result.filterSet = make(map[string]struct{})
			}
			result.filterSet[rule.BetaToken] = struct{}{}
		}
	}
	return result
}
func mergeDropSets(policySet map[string]struct{}, extra ...string) map[string]struct{} {
	if len(policySet) == 0 && len(extra) == 0 {
		return defaultDroppedBetasSet
	}
	m := make(map[string]struct{}, len(defaultDroppedBetasSet)+len(policySet)+len(extra))
	for t := range defaultDroppedBetasSet {
		m[t] = struct{}{}
	}
	for t := range policySet {
		m[t] = struct{}{}
	}
	for _, t := range extra {
		m[t] = struct{}{}
	}
	return m
}

const betaPolicyFilterSetKey = "betaPolicyFilterSet"

func (s *GatewayService) getBetaPolicyFilterSet(ctx context.Context, c *gin.Context, account *Account) map[string]struct{} {
	if c != nil {
		if v, ok := c.Get(betaPolicyFilterSetKey); ok {
			if fs, ok := v.(map[string]struct{}); ok {
				return fs
			}
		}
	}
	return s.evaluateBetaPolicy(ctx, "", account).filterSet
}
func betaPolicyScopeMatches(scope string, isOAuth bool) bool {
	switch scope {
	case BetaPolicyScopeAll:
		return true
	case BetaPolicyScopeOAuth:
		return isOAuth
	case BetaPolicyScopeAPIKey:
		return !isOAuth
	default:
		return true
	}
}
func droppedBetaSet(extra ...string) map[string]struct{} {
	m := make(map[string]struct{}, len(defaultDroppedBetasSet)+len(extra))
	for t := range defaultDroppedBetasSet {
		m[t] = struct{}{}
	}
	for _, t := range extra {
		m[t] = struct{}{}
	}
	return m
}
func containsBetaToken(header, token string) bool {
	if header == "" || token == "" {
		return false
	}
	for _, p := range strings.Split(header, ",") {
		if strings.TrimSpace(p) == token {
			return true
		}
	}
	return false
}
func buildBetaTokenSet(tokens []string) map[string]struct{} {
	m := make(map[string]struct{}, len(tokens))
	for _, t := range tokens {
		if t == "" {
			continue
		}
		m[t] = struct{}{}
	}
	return m
}

var defaultDroppedBetasSet = buildBetaTokenSet(claude.DroppedBetas)
