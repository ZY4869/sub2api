package service

import (
	"fmt"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

var deepSeekChatPrefixBetaModels = map[string]struct{}{
	"deepseek-v4-flash": {},
	"deepseek-v4-pro":   {},
}

var deepSeekFIMCompletionModels = map[string]struct{}{
	"deepseek-v4-flash": {},
	"deepseek-v4-pro":   {},
}

type deepSeekChatRequestError struct {
	reason  string
	message string
}

func (e *deepSeekChatRequestError) Error() string {
	if e == nil {
		return ""
	}
	return e.message
}

type preparedDeepSeekNativeChatRequest struct {
	body                  []byte
	originalModel         string
	mappedModel           string
	stream                bool
	clientRequestedUsage  bool
	explicitBetaSpecified bool
	explicitBetaValue     bool
	autoBetaRequested     bool
	betaEnabled           bool
	betaStripped          bool
}

func normalizeDeepSeekModelID(model string) string {
	return strings.TrimSpace(strings.ToLower(model))
}

func isDeepSeekChatPrefixBetaModel(model string) bool {
	_, ok := deepSeekChatPrefixBetaModels[normalizeDeepSeekModelID(model)]
	return ok
}

func isDeepSeekFIMCompletionModel(model string) bool {
	_, ok := deepSeekFIMCompletionModels[normalizeDeepSeekModelID(model)]
	return ok
}

func prepareDeepSeekNativeChatRequestBody(account *Account, body []byte, defaultMappedModel string) (*preparedDeepSeekNativeChatRequest, error) {
	prepared := &preparedDeepSeekNativeChatRequest{}
	prepared.originalModel = strings.TrimSpace(gjson.GetBytes(body, "model").String())
	if prepared.originalModel == "" {
		return nil, &deepSeekChatRequestError{
			reason:  "missing_model",
			message: "model is required",
		}
	}

	prepared.mappedModel = resolveOpenAIForwardModel(account, prepared.originalModel, defaultMappedModel)
	prepared.stream = gjson.GetBytes(body, "stream").Bool()
	prepared.clientRequestedUsage = gjson.GetBytes(body, "stream_options.include_usage").Bool()
	var err error

	if prepared.mappedModel != "" && prepared.mappedModel != prepared.originalModel {
		body, err = sjson.SetBytes(body, "model", prepared.mappedModel)
		if err != nil {
			return nil, fmt.Errorf("rewrite deepseek model: %w", err)
		}
	}
	if prepared.stream && !prepared.clientRequestedUsage {
		body, err = sjson.SetBytes(body, "stream_options.include_usage", true)
		if err != nil {
			return nil, fmt.Errorf("inject deepseek stream usage: %w", err)
		}
	}

	body, prepared.explicitBetaSpecified, prepared.explicitBetaValue, err = stripDeepSeekExplicitBetaField(body)
	if err != nil {
		return nil, err
	}

	prepared.autoBetaRequested = deepSeekChatPrefixBetaRequested(body)
	switch {
	case prepared.explicitBetaSpecified && prepared.explicitBetaValue:
		if !isDeepSeekChatPrefixBetaModel(prepared.mappedModel) {
			return nil, &deepSeekChatRequestError{
				reason:  "deepseek_chat_beta_model_unsupported",
				message: "DeepSeek beta chat/completions currently only supports deepseek-v4-flash or deepseek-v4-pro",
			}
		}
		prepared.betaEnabled = true
	case prepared.explicitBetaSpecified && !prepared.explicitBetaValue:
		if prepared.autoBetaRequested {
			body, prepared.betaStripped, err = stripDeepSeekChatPrefixBetaFields(body)
			if err != nil {
				return nil, err
			}
		}
	default:
		prepared.betaEnabled = prepared.autoBetaRequested && isDeepSeekChatPrefixBetaModel(prepared.mappedModel)
		if prepared.autoBetaRequested && !prepared.betaEnabled {
			body, prepared.betaStripped, err = stripDeepSeekChatPrefixBetaFields(body)
			if err != nil {
				return nil, err
			}
		}
	}

	prepared.body = body
	return prepared, nil
}

func stripDeepSeekExplicitBetaField(body []byte) ([]byte, bool, bool, error) {
	betaResult := gjson.GetBytes(body, "beta")
	if !betaResult.Exists() {
		return body, false, false, nil
	}
	if betaResult.Type != gjson.True && betaResult.Type != gjson.False {
		return nil, false, false, &deepSeekChatRequestError{
			reason:  "invalid_beta",
			message: "beta must be a boolean",
		}
	}

	updated, err := sjson.DeleteBytes(body, "beta")
	if err != nil {
		return nil, false, false, fmt.Errorf("strip deepseek explicit beta field: %w", err)
	}
	return updated, true, betaResult.Bool(), nil
}

func deepSeekChatPrefixBetaRequested(body []byte) bool {
	for _, message := range gjson.GetBytes(body, "messages").Array() {
		if !strings.EqualFold(strings.TrimSpace(message.Get("role").String()), "assistant") {
			continue
		}
		if message.Get("prefix").Bool() {
			return true
		}
		if strings.TrimSpace(message.Get("reasoning_content").String()) != "" {
			return true
		}
	}
	return false
}

func stripDeepSeekChatPrefixBetaFields(body []byte) ([]byte, bool, error) {
	updated := body
	changed := false
	for idx, message := range gjson.GetBytes(body, "messages").Array() {
		if !strings.EqualFold(strings.TrimSpace(message.Get("role").String()), "assistant") {
			continue
		}
		if message.Get("prefix").Exists() {
			nextBody, err := sjson.DeleteBytes(updated, fmt.Sprintf("messages.%d.prefix", idx))
			if err != nil {
				return nil, false, fmt.Errorf("strip deepseek prefix beta field: %w", err)
			}
			updated = nextBody
			changed = true
		}
		if strings.TrimSpace(message.Get("reasoning_content").String()) != "" {
			nextBody, err := sjson.DeleteBytes(updated, fmt.Sprintf("messages.%d.reasoning_content", idx))
			if err != nil {
				return nil, false, fmt.Errorf("strip deepseek reasoning_content beta field: %w", err)
			}
			updated = nextBody
			changed = true
		}
	}
	return updated, changed, nil
}
