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
func extractOpenAIRequestMetaFromBody(body []byte) (model string, stream bool, promptCacheKey string) {
	if len(body) == 0 {
		return "", false, ""
	}
	model = strings.TrimSpace(gjson.GetBytes(body, "model").String())
	stream = gjson.GetBytes(body, "stream").Bool()
	promptCacheKey = strings.TrimSpace(gjson.GetBytes(body, "prompt_cache_key").String())
	return model, stream, promptCacheKey
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
func extractOpenAIReasoningEffortResolutionFromBody(body []byte, requestedModel string) GatewayEffortResolution {
	reasoningEffort := strings.TrimSpace(gjson.GetBytes(body, "reasoning.effort").String())
	if reasoningEffort != "" {
		return ResolveOpenAIEffort(reasoningEffort, "", effortSourceOpenAIField)
	}
	reasoningEffort = strings.TrimSpace(gjson.GetBytes(body, "reasoning_effort").String())
	if reasoningEffort != "" {
		return ResolveOpenAIEffort(reasoningEffort, "", effortSourceOpenAIAlias)
	}
	value := deriveOpenAIReasoningEffortFromModel(requestedModel)
	if value == "" {
		return GatewayEffortResolution{}
	}
	return GatewayEffortResolution{
		Raw:       &value,
		Effective: NormalizeOpenAIReasoningEffortEffective(value),
		Source:    "model_suffix",
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
func extractOpenAIReasoningEffortResolution(reqBody map[string]any, requestedModel string) GatewayEffortResolution {
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
		return ResolveOpenAIEffort(value, "", source)
	}
	value := deriveOpenAIReasoningEffortFromModel(requestedModel)
	if value == "" {
		return GatewayEffortResolution{}
	}
	return GatewayEffortResolution{
		Raw:       &value,
		Effective: NormalizeOpenAIReasoningEffortEffective(value),
		Source:    "model_suffix",
	}
}

func applyOpenAIEffortResolutionToReqBody(reqBody map[string]any, effortResolution GatewayEffortResolution) {
	if reqBody == nil || effortResolution.Effective == nil {
		return
	}
	switch effortResolution.Source {
	case effortSourceOpenAIField, effortSourceOpenAIAlias, effortSourceTopLevel, effortSourceAnthropicField:
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

func normalizeOpenAIRequestBodyEffort(reqBody map[string]any, requestedModel string) GatewayEffortResolution {
	effortResolution := extractOpenAIReasoningEffortResolution(reqBody, requestedModel)
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
	case effortSourceOpenAIField, effortSourceOpenAIAlias, effortSourceTopLevel, effortSourceAnthropicField:
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

func normalizeOpenAIRequestBodyEffortBytes(body []byte, requestedModel string) ([]byte, GatewayEffortResolution, error) {
	effortResolution := extractOpenAIReasoningEffortResolutionFromBody(body, requestedModel)
	normalized, err := applyOpenAIEffortResolutionToBodyBytes(body, effortResolution)
	if err != nil {
		return body, effortResolution, err
	}
	return normalized, effortResolution, nil
}
