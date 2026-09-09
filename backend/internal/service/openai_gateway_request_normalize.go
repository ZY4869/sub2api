package service

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"strings"
)

func getOpenAIReasoningEffortFromReqBody(reqBody map[string]any) (value string, present bool) {
	if reqBody == nil {
		return "", false
	}
	if reasoning, ok := reqBody["reasoning"].(map[string]any); ok {
		if effort, ok := reasoning["effort"].(string); ok {
			if normalized := normalizeOpenAIReasoningEffortRaw(effort); normalized != nil {
				return *normalized, true
			}
			return "", true
		}
	}
	if effort, ok := reqBody["reasoning_effort"].(string); ok {
		if normalized := normalizeOpenAIReasoningEffortRaw(effort); normalized != nil {
			return *normalized, true
		}
		return "", true
	}
	return "", false
}
func deriveOpenAIReasoningEffortFromModel(model string) string {
	if strings.TrimSpace(model) == "" {
		return ""
	}
	modelID := strings.TrimSpace(model)
	if strings.Contains(modelID, "/") {
		parts := strings.Split(modelID, "/")
		modelID = parts[len(parts)-1]
	}
	parts := strings.FieldsFunc(strings.ToLower(modelID), func(r rune) bool {
		switch r {
		case '-', '_', ' ':
			return true
		default:
			return false
		}
	})
	if len(parts) == 0 {
		return ""
	}
	if normalized := normalizeOpenAIReasoningEffortRaw(parts[len(parts)-1]); normalized != nil {
		return *normalized
	}
	return ""
}

func deriveOpenAIReasoningEffortFromModelCandidates(models ...string) string {
	for _, model := range models {
		if value := deriveOpenAIReasoningEffortFromModel(model); value != "" {
			return value
		}
	}
	return ""
}

func extractOpenAIRequestMetaFromBody(body []byte) (model string, stream bool, promptCacheKey string) {
	if len(body) == 0 {
		return "", false, ""
	}
	model = strings.TrimSpace(gjson.GetBytes(body, "model").String())
	stream = gjson.GetBytes(body, "stream").Bool()
	promptCacheKey = strings.TrimSpace(gjson.GetBytes(body, "prompt_cache_key").String())
	return model, stream, promptCacheKey
}

// NormalizeStatelessNativeResponsesBody enforces the stateless contract used by
// native DeepSeek/Kimi Responses endpoints. It deliberately preserves caller
// instructions, but removes conversation state and persistence flags that the
// native endpoints do not support.
func NormalizeStatelessNativeResponsesBody(body []byte, platform string) ([]byte, bool, error) {
	if len(body) == 0 || (platform != PlatformDeepSeek && platform != PlatformKimi) {
		return body, false, nil
	}
	normalized := body
	changed := false
	store := gjson.GetBytes(normalized, "store")
	if !store.Exists() || store.Type != gjson.False {
		next, err := sjson.SetBytes(normalized, "store", false)
		if err != nil {
			return body, false, fmt.Errorf("normalize stateless responses store=false: %w", err)
		}
		normalized = next
		changed = true
	}
	if gjson.GetBytes(normalized, "previous_response_id").Exists() {
		next, err := sjson.DeleteBytes(normalized, "previous_response_id")
		if err != nil {
			return body, false, fmt.Errorf("normalize stateless responses previous_response_id: %w", err)
		}
		normalized = next
		changed = true
	}
	return normalized, changed, nil
}

func shouldNormalizeStatelessNativeResponses(c *gin.Context, account *Account) bool {
	if account == nil || (RoutingPlatformForAccount(account) != PlatformDeepSeek && RoutingPlatformForAccount(account) != PlatformKimi) {
		return false
	}
	if c == nil || c.Request == nil || c.Request.URL == nil {
		return false
	}
	path := strings.ToLower(strings.TrimSpace(c.Request.URL.Path))
	return strings.Contains(path, "/responses")
}

func normalizeOpenAIPassthroughOAuthBody(body []byte, compact bool) ([]byte, bool, error) {
	if len(body) == 0 {
		return body, false, nil
	}
	normalized := body
	changed := false
	if compact {
		if store := gjson.GetBytes(normalized, "store"); store.Exists() {
			next, err := sjson.DeleteBytes(normalized, "store")
			if err != nil {
				return body, false, fmt.Errorf("normalize passthrough body delete store: %w", err)
			}
			normalized = next
			changed = true
		}
		if stream := gjson.GetBytes(normalized, "stream"); stream.Exists() {
			next, err := sjson.DeleteBytes(normalized, "stream")
			if err != nil {
				return body, false, fmt.Errorf("normalize passthrough body delete stream: %w", err)
			}
			normalized = next
			changed = true
		}
	} else {
		if store := gjson.GetBytes(normalized, "store"); !store.Exists() || store.Type != gjson.False {
			next, err := sjson.SetBytes(normalized, "store", false)
			if err != nil {
				return body, false, fmt.Errorf("normalize passthrough body store=false: %w", err)
			}
			normalized = next
			changed = true
		}
		if stream := gjson.GetBytes(normalized, "stream"); !stream.Exists() || stream.Type != gjson.True {
			next, err := sjson.SetBytes(normalized, "stream", true)
			if err != nil {
				return body, false, fmt.Errorf("normalize passthrough body stream=true: %w", err)
			}
			normalized = next
			changed = true
		}
	}
	return normalized, changed, nil
}
func detectOpenAIPassthroughInstructionsRejectReason(reqModel string, body []byte) string {
	model := strings.ToLower(strings.TrimSpace(reqModel))
	if !strings.Contains(model, "codex") {
		return ""
	}
	instructions := gjson.GetBytes(body, "instructions")
	if !instructions.Exists() {
		return "instructions_missing"
	}
	if instructions.Type != gjson.String {
		return "instructions_not_string"
	}
	if strings.TrimSpace(instructions.String()) == "" {
		return "instructions_empty"
	}
	return ""
}
func extractOpenAIReasoningEffortResolutionFromBody(body []byte, requestedModel string, modelCandidates ...string) GatewayEffortResolution {
	candidates := appendOpenAIReasoningModelCandidates([]string{requestedModel}, strings.TrimSpace(gjson.GetBytes(body, "model").String()))
	candidates = appendOpenAIReasoningModelCandidates(candidates, modelCandidates...)
	reasoningEffort := strings.TrimSpace(gjson.GetBytes(body, "reasoning.effort").String())
	if reasoningEffort != "" {
		return ResolveOpenAIEffortForModels(reasoningEffort, "", effortSourceOpenAIField, candidates...)
	}
	reasoningEffort = strings.TrimSpace(gjson.GetBytes(body, "reasoning_effort").String())
	if reasoningEffort != "" {
		return ResolveOpenAIEffortForModels(reasoningEffort, "", effortSourceOpenAIAlias, candidates...)
	}
	value := deriveOpenAIReasoningEffortFromModelCandidates(candidates...)
	if value == "" {
		return GatewayEffortResolution{}
	}
	return GatewayEffortResolution{
		Raw:       &value,
		Effective: NormalizeOpenAIReasoningEffortEffectiveForModels(value, candidates...),
		Source:    effortSourceModelSuffix,
	}
}

func extractOpenAIReasoningEffortFromBody(body []byte, requestedModel string) *string {
	return extractOpenAIReasoningEffortResolutionFromBody(body, requestedModel).Effective
}
func extractOpenAIServiceTier(reqBody map[string]any) *string {
	if reqBody == nil {
		return nil
	}
	raw, ok := reqBody["service_tier"].(string)
	if !ok {
		return nil
	}
	return normalizeOpenAIServiceTier(raw)
}
func extractOpenAIServiceTierFromBody(body []byte) *string {
	if len(body) == 0 {
		return nil
	}
	return normalizeOpenAIServiceTier(gjson.GetBytes(body, "service_tier").String())
}
func normalizeOpenAIServiceTier(raw string) *string {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return nil
	}
	if value == "fast" {
		value = "priority"
	}
	switch value {
	case "priority", "flex":
		return &value
	default:
		return nil
	}
}
func getOpenAIRequestBodyMap(c *gin.Context, body []byte) (map[string]any, error) {
	if c != nil {
		if cached, ok := c.Get(OpenAIParsedRequestBodyKey); ok {
			if reqBody, ok := cached.(map[string]any); ok && reqBody != nil {
				return reqBody, nil
			}
		}
	}
	var reqBody map[string]any
	if err := json.Unmarshal(body, &reqBody); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}
	if c != nil {
		c.Set(OpenAIParsedRequestBodyKey, reqBody)
	}
	return reqBody, nil
}
func extractOpenAIReasoningEffortResolution(reqBody map[string]any, requestedModel string, modelCandidates ...string) GatewayEffortResolution {
	candidates := []string{requestedModel}
	if reqBody != nil {
		if model, ok := reqBody["model"].(string); ok {
			candidates = append(candidates, model)
		}
	}
	candidates = appendOpenAIReasoningModelCandidates(candidates, modelCandidates...)
	if value, present := getOpenAIReasoningEffortFromReqBody(reqBody); present {
		if value == "" {
			return GatewayEffortResolution{}
		}
		source := effortSourceOpenAIAlias
		if reasoning, ok := reqBody["reasoning"].(map[string]any); ok {
			if _, ok := reasoning["effort"]; ok {
				source = effortSourceOpenAIField
			}
		}
		return ResolveOpenAIEffortForModels(value, "", source, candidates...)
	}
	value := deriveOpenAIReasoningEffortFromModelCandidates(candidates...)
	if value == "" {
		return GatewayEffortResolution{}
	}
	return GatewayEffortResolution{
		Raw:       &value,
		Effective: NormalizeOpenAIReasoningEffortEffectiveForModels(value, candidates...),
		Source:    effortSourceModelSuffix,
	}
}

func applyOpenAIEffortResolutionToReqBody(reqBody map[string]any, effortResolution GatewayEffortResolution) {
	if reqBody == nil || effortResolution.Effective == nil {
		return
	}
	switch effortResolution.Source {
	case effortSourceOpenAIField, effortSourceOpenAIAlias, effortSourceTopLevel, effortSourceAnthropicField, effortSourceModelSuffix:
		reasoning, _ := reqBody["reasoning"].(map[string]any)
		if reasoning == nil {
			reasoning = map[string]any{}
			reqBody["reasoning"] = reasoning
		}
		reasoning["effort"] = *effortResolution.Effective
		delete(reqBody, "reasoning_effort")
		if effortResolution.Source == effortSourceTopLevel {
			delete(reqBody, "effortLevel")
		}
	}
}

func normalizeOpenAIRequestBodyEffort(reqBody map[string]any, requestedModel string, modelCandidates ...string) GatewayEffortResolution {
	effortResolution := extractOpenAIReasoningEffortResolution(reqBody, requestedModel, modelCandidates...)
	applyOpenAIEffortResolutionToReqBody(reqBody, effortResolution)
	return effortResolution
}

func applyOpenAIEffortResolutionToBodyBytes(body []byte, effortResolution GatewayEffortResolution) ([]byte, error) {
	if effortResolution.Effective == nil {
		return body, nil
	}
	normalized := body
	var err error
	switch effortResolution.Source {
	case effortSourceOpenAIField, effortSourceOpenAIAlias, effortSourceTopLevel, effortSourceAnthropicField, effortSourceModelSuffix:
		normalized, err = sjson.SetBytes(normalized, "reasoning.effort", *effortResolution.Effective)
		if err != nil {
			return body, err
		}
		if gjson.GetBytes(normalized, "reasoning_effort").Exists() {
			if nextBody, delErr := sjson.DeleteBytes(normalized, "reasoning_effort"); delErr == nil {
				normalized = nextBody
			}
		}
		if effortResolution.Source == effortSourceTopLevel && gjson.GetBytes(normalized, "effortLevel").Exists() {
			if nextBody, delErr := sjson.DeleteBytes(normalized, "effortLevel"); delErr == nil {
				normalized = nextBody
			}
		}
	}
	return normalized, nil
}

func normalizeOpenAIRequestBodyEffortBytes(body []byte, requestedModel string, modelCandidates ...string) ([]byte, GatewayEffortResolution, error) {
	effortResolution := extractOpenAIReasoningEffortResolutionFromBody(body, requestedModel, modelCandidates...)
	normalized, err := applyOpenAIEffortResolutionToBodyBytes(body, effortResolution)
	if err != nil {
		return body, effortResolution, err
	}
	return normalized, effortResolution, nil
}

func appendOpenAIReasoningModelCandidates(base []string, candidates ...string) []string {
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate) != "" {
			base = append(base, candidate)
		}
	}
	return base
}
