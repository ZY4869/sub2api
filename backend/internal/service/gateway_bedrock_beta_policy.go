package service

import (
	"context"
	"strings"
)

func filterBetaTokens(tokens []string, filterSet map[string]struct{}) []string {
	if tokens == nil {
		return nil
	}
	if len(filterSet) == 0 {
		return tokens
	}
	out := make([]string, 0, len(tokens))
	for _, t := range tokens {
		if t == "" {
			continue
		}
		if _, drop := filterSet[t]; drop {
			continue
		}
		out = append(out, t)
	}
	return out
}

func bedrockBetaPolicyScopeMatches(scope string, account *Account) bool {
	switch scope {
	case BetaPolicyScopeAll:
		return true
	case BetaPolicyScopeBedrock:
		return account != nil && account.Type == AccountTypeBedrock
	case BetaPolicyScopeOAuth:
		return account != nil && account.IsOAuth()
	case BetaPolicyScopeAPIKey:
		// Bedrock is neither OAuth nor APIKey, but policy "apikey" is still safer to treat
		// as "non-OAuth" to avoid surprising bypasses.
		return account == nil || !account.IsOAuth()
	default:
		return true
	}
}

func (s *GatewayService) resolveBedrockBetaTokensForRequest(ctx context.Context, account *Account, clientBetaHeader string, body []byte, modelID string) ([]string, error) {
	// 1) Read policy settings (optional).
	var (
		blockRules []BetaPolicyRule
		filterSet  map[string]struct{}
	)
	if s != nil && s.settingService != nil {
		settings, err := s.settingService.GetBetaPolicySettings(ctx)
		if err == nil && settings != nil {
			for _, rule := range settings.Rules {
				if !bedrockBetaPolicyScopeMatches(rule.Scope, account) {
					continue
				}
				switch rule.Action {
				case BetaPolicyActionBlock:
					blockRules = append(blockRules, rule)
				case BetaPolicyActionFilter:
					if filterSet == nil {
						filterSet = make(map[string]struct{})
					}
					filterSet[strings.TrimSpace(rule.BetaToken)] = struct{}{}
				}
			}
		}
	}

	// 2) Block on the original Anthropic beta header tokens before Bedrock transforms.
	// (e.g. advanced-tool-use-2025-11-20 -> tool-search-tool-2025-10-19)
	for _, rule := range blockRules {
		if rule.BetaToken == "" {
			continue
		}
		if clientBetaHeader != "" && containsBetaToken(clientBetaHeader, rule.BetaToken) {
			msg := strings.TrimSpace(rule.ErrorMessage)
			if msg == "" {
				msg = "beta feature " + rule.BetaToken + " is not allowed"
			}
			return nil, &BetaBlockedError{Message: msg}
		}
	}

	// 3) Resolve Bedrock beta token list (parse + auto inject + Bedrock transforms + whitelist filter).
	betaTokens := ResolveBedrockBetaTokens(clientBetaHeader, body, modelID)

	// 4) Block based on the effective Bedrock token list (includes body auto-injected tokens).
	if len(blockRules) > 0 && len(betaTokens) > 0 {
		tokenSet := make(map[string]struct{}, len(betaTokens))
		for _, t := range betaTokens {
			tokenSet[t] = struct{}{}
		}
		for _, rule := range blockRules {
			if rule.BetaToken == "" {
				continue
			}
			if _, ok := tokenSet[rule.BetaToken]; ok {
				msg := strings.TrimSpace(rule.ErrorMessage)
				if msg == "" {
					msg = "beta feature " + rule.BetaToken + " is not allowed"
				}
				return nil, &BetaBlockedError{Message: msg}
			}
		}
	}

	// 5) Apply filter rules.
	betaTokens = filterBetaTokens(betaTokens, filterSet)
	return betaTokens, nil
}

