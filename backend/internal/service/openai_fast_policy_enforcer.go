package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/tidwall/sjson"
	"go.uber.org/zap"
)

type openAIFastPolicyDecision struct {
	matched          bool
	serviceTier      string
	scope            string
	action           string
	modelWhitelisted bool
	usedFallback     bool
	rule             OpenAIFastPolicyRule
}

type openAIFastPolicyBlockedError struct {
	ServiceTier string
	Model       string
	Scope       string
}

func (e *openAIFastPolicyBlockedError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("openai fast policy blocked: tier=%s model=%s scope=%s", strings.TrimSpace(e.ServiceTier), strings.TrimSpace(e.Model), strings.TrimSpace(e.Scope))
}

func openAIFastPolicyScopeForAccount(account *Account) string {
	if account == nil {
		return OpenAIFastPolicyScopeAll
	}
	if account.Type == AccountTypeAPIKey {
		return OpenAIFastPolicyScopeAPIKey
	}
	return OpenAIFastPolicyScopeOAuth
}

func normalizeOpenAIFastPolicyServiceTier(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "fast" {
		return "priority"
	}
	return value
}

func openAIFastPolicyModelWhitelisted(model string, whitelist []string) bool {
	if len(whitelist) == 0 {
		return false
	}
	value := strings.TrimSpace(model)
	if value == "" {
		return false
	}
	for _, allowed := range whitelist {
		if strings.EqualFold(strings.TrimSpace(allowed), value) {
			return true
		}
	}
	return false
}

func openAIFastPolicyScopeMatches(scope string, account *Account) bool {
	scope = strings.ToLower(strings.TrimSpace(scope))
	switch scope {
	case OpenAIFastPolicyScopeAll:
		return true
	case OpenAIFastPolicyScopeOAuth:
		return account != nil && account.Type != AccountTypeAPIKey
	case OpenAIFastPolicyScopeAPIKey:
		return account != nil && account.Type == AccountTypeAPIKey
	default:
		return true
	}
}

func resolveOpenAIFastPolicyDecision(settings *OpenAIFastPolicySettings, account *Account, serviceTier string, model string) openAIFastPolicyDecision {
	decision := openAIFastPolicyDecision{
		serviceTier: normalizeOpenAIFastPolicyServiceTier(serviceTier),
		scope:       openAIFastPolicyScopeForAccount(account),
		action:      OpenAIFastPolicyActionPass,
	}
	if settings == nil || len(settings.Rules) == 0 || decision.serviceTier == "" {
		return decision
	}

	for _, rule := range settings.Rules {
		if normalizeOpenAIFastPolicyServiceTier(rule.ServiceTier) != decision.serviceTier {
			continue
		}
		if !openAIFastPolicyScopeMatches(rule.Scope, account) {
			continue
		}
		decision.matched = true
		decision.rule = rule
		decision.scope = strings.ToLower(strings.TrimSpace(rule.Scope))

		effectiveAction := strings.ToLower(strings.TrimSpace(rule.Action))
		modelWhitelisted := openAIFastPolicyModelWhitelisted(model, rule.ModelWhitelist)
		if len(rule.ModelWhitelist) > 0 && !modelWhitelisted {
			effectiveAction = strings.ToLower(strings.TrimSpace(rule.FallbackAction))
			decision.usedFallback = true
		}

		if !isOpenAIFastPolicyAction(effectiveAction) {
			effectiveAction = OpenAIFastPolicyActionFilter
		}

		decision.action = effectiveAction
		decision.modelWhitelisted = modelWhitelisted
		return decision
	}

	return decision
}

func (s *OpenAIGatewayService) getOpenAIFastPolicySettings(ctx context.Context) *OpenAIFastPolicySettings {
	if s == nil || s.settingService == nil {
		return DefaultOpenAIFastPolicySettings()
	}
	settings, _ := s.settingService.GetOpenAIFastPolicySettings(ctx)
	if settings == nil {
		return DefaultOpenAIFastPolicySettings()
	}
	return settings
}

func (s *OpenAIGatewayService) evaluateOpenAIFastPolicy(ctx context.Context, account *Account, serviceTier string, model string) openAIFastPolicyDecision {
	return resolveOpenAIFastPolicyDecision(s.getOpenAIFastPolicySettings(ctx), account, serviceTier, model)
}

func (s *OpenAIGatewayService) applyOpenAIFastPolicyToRequestBodyMap(ctx context.Context, account *Account, reqBody map[string]any) (bool, error) {
	if reqBody == nil {
		return false, nil
	}
	tierPtr := extractOpenAIServiceTier(reqBody)
	if tierPtr == nil {
		return false, nil
	}
	model, _ := reqBody["model"].(string)
	model = strings.TrimSpace(model)

	decision := s.evaluateOpenAIFastPolicy(ctx, account, *tierPtr, model)
	if !decision.matched || decision.action == OpenAIFastPolicyActionPass {
		return false, nil
	}

	if decision.action == OpenAIFastPolicyActionBlock {
		s.logOpenAIFastPolicyDecision(ctx, account, model, *tierPtr, decision, "http_map")
		return false, &openAIFastPolicyBlockedError{ServiceTier: *tierPtr, Model: model, Scope: decision.scope}
	}

	// filter
	delete(reqBody, "service_tier")
	s.logOpenAIFastPolicyDecision(ctx, account, model, *tierPtr, decision, "http_map")
	return true, nil
}

func (s *OpenAIGatewayService) applyOpenAIFastPolicyToJSONBody(ctx context.Context, account *Account, body []byte, serviceTier string, model string) ([]byte, openAIFastPolicyDecision, error) {
	decision := s.evaluateOpenAIFastPolicy(ctx, account, serviceTier, model)
	if !decision.matched || decision.action == OpenAIFastPolicyActionPass {
		return body, decision, nil
	}
	s.logOpenAIFastPolicyDecision(ctx, account, model, serviceTier, decision, "http_bytes")
	if decision.action == OpenAIFastPolicyActionBlock {
		return body, decision, &openAIFastPolicyBlockedError{ServiceTier: serviceTier, Model: model, Scope: decision.scope}
	}

	next, err := deleteTopLevelJSONKeyBytes(body, "service_tier")
	if err != nil {
		return body, decision, err
	}
	return next, decision, nil
}

func deleteTopLevelJSONKeyBytes(body []byte, key string) ([]byte, error) {
	key = strings.TrimSpace(key)
	if len(body) == 0 || key == "" {
		return body, nil
	}
	next, err := sjson.DeleteBytes(body, key)
	if err == nil {
		return next, nil
	}
	// Fallback for malformed JSON paths or unexpected payload shapes.
	var payload map[string]any
	if unmarshalErr := json.Unmarshal(body, &payload); unmarshalErr != nil {
		return nil, err
	}
	delete(payload, key)
	rebuilt, marshalErr := json.Marshal(payload)
	if marshalErr != nil {
		return nil, marshalErr
	}
	return rebuilt, nil
}

func (s *OpenAIGatewayService) logOpenAIFastPolicyDecision(ctx context.Context, account *Account, model string, serviceTier string, decision openAIFastPolicyDecision, source string) {
	if ctx == nil {
		ctx = context.Background()
	}
	if !decision.matched {
		return
	}

	accountID := int64(0)
	accountType := ""
	if account != nil {
		accountID = account.ID
		accountType = string(account.Type)
	}

	level := zap.DebugLevel
	if decision.action == OpenAIFastPolicyActionFilter || decision.action == OpenAIFastPolicyActionBlock {
		level = zap.InfoLevel
	}

	log := logger.FromContext(ctx).With(
		zap.String("component", "service.openai_fast_policy"),
		zap.String("source", strings.TrimSpace(source)),
		zap.Int64("account_id", accountID),
		zap.String("account_type", strings.TrimSpace(accountType)),
		zap.String("service_tier", normalizeOpenAIFastPolicyServiceTier(serviceTier)),
		zap.String("model", strings.TrimSpace(model)),
		zap.String("action", strings.TrimSpace(decision.action)),
		zap.String("rule_action", strings.TrimSpace(decision.rule.Action)),
		zap.String("rule_scope", strings.TrimSpace(decision.rule.Scope)),
		zap.Bool("model_whitelisted", decision.modelWhitelisted),
		zap.Bool("used_fallback", decision.usedFallback),
	)
	if level == zap.InfoLevel {
		log.Info("openai fast policy applied")
	} else {
		log.Debug("openai fast policy evaluated")
	}
}
