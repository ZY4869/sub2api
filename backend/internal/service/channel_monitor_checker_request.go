package service

import (
	"errors"
	"net/url"
	"strings"
)

func buildChannelMonitorRequest(monitor *ChannelMonitor, modelID string, challenge string) (path string, payload map[string]any, requireChallenge bool, err error) {
	if monitor == nil {
		return "", nil, false, errors.New("monitor is required")
	}
	protocol := strings.TrimSpace(strings.ToLower(monitor.RequestProtocol))
	if protocol == "" {
		protocol = inferChannelMonitorRequestProtocol(monitor.Provider)
	}
	mode := monitor.BodyOverrideMode
	override := monitor.BodyOverride
	if mode == "" {
		mode = ChannelMonitorBodyOverrideModeOff
	}

	prompt := buildChannelMonitorPrompt(monitor.TestPromptTemplate, challenge)

	switch protocol {
	case ChannelMonitorRequestProtocolOpenAI:
		path = "/v1/chat/completions"
		base := buildOpenAIStyleChannelMonitorPayload(modelID, prompt, false)
		if normalizeChannelMonitorOpenAIAPIMode(protocol, monitor.OpenAIAPIMode) == ChannelMonitorOpenAIAPIModeResponses {
			path = "/v1/responses"
			base = buildOpenAIStyleChannelMonitorPayload(modelID, prompt, true)
		}
		payload, requireChallenge, err = applyBodyOverride(protocol, mode, base, override)
		return path, payload, requireChallenge, err
	case ChannelMonitorRequestProtocolAnthropic:
		path = "/v1/messages"
		base := map[string]any{
			"model":      modelID,
			"max_tokens": 32,
			"messages": []any{
				map[string]any{"role": "user", "content": prompt},
			},
			"temperature": 0,
		}
		payload, requireChallenge, err = applyBodyOverride(protocol, mode, base, override)
		return path, payload, requireChallenge, err
	case ChannelMonitorRequestProtocolGemini:
		escaped := url.PathEscape(modelID)
		path = "/v1beta/models/" + escaped + ":generateContent"
		base := map[string]any{
			"contents": []any{
				map[string]any{
					"role": "user",
					"parts": []any{
						map[string]any{"text": prompt},
					},
				},
			},
			"generationConfig": map[string]any{
				"temperature":     0,
				"maxOutputTokens": 32,
			},
		}
		payload, requireChallenge, err = applyBodyOverride(protocol, mode, base, override)
		return path, payload, requireChallenge, err
	default:
		return "", nil, false, errors.New("invalid request protocol")
	}
}

func buildChannelMonitorPrompt(template string, challenge string) string {
	challenge = strings.TrimSpace(challenge)
	tpl := strings.TrimSpace(template)
	if tpl == "" {
		return "Please reply with exactly: " + challenge
	}
	if strings.Contains(tpl, "{{challenge}}") {
		return strings.ReplaceAll(tpl, "{{challenge}}", challenge)
	}
	return tpl + "\n\nPlease reply with exactly: " + challenge
}

func buildOpenAIStyleChannelMonitorPayload(modelID string, prompt string, responses bool) map[string]any {
	if responses {
		return map[string]any{
			"model":             modelID,
			"input":             prompt,
			"temperature":       0,
			"max_output_tokens": 32,
			"stream":            false,
		}
	}
	return map[string]any{
		"model": modelID,
		"messages": []any{
			map[string]any{"role": "user", "content": prompt},
		},
		"temperature": 0,
		"max_tokens":  32,
		"stream":      false,
	}
}

func applyBodyOverride(protocol string, mode string, base map[string]any, override map[string]any) (map[string]any, bool, error) {
	switch mode {
	case ChannelMonitorBodyOverrideModeOff:
		return base, true, nil
	case ChannelMonitorBodyOverrideModeMerge:
		merged := map[string]any{}
		for k, v := range base {
			merged[k] = v
		}
		for k, v := range override {
			if isChannelMonitorBodyOverrideKeyBlocked(protocol, k) {
				continue
			}
			merged[k] = v
		}
		return merged, true, nil
	case ChannelMonitorBodyOverrideModeReplace:
		if len(override) == 0 {
			return nil, false, ErrChannelMonitorInvalidBodyOverride
		}
		return override, false, nil
	default:
		return nil, false, ErrChannelMonitorInvalidOverrideMode
	}
}

func isChannelMonitorBodyOverrideKeyBlocked(protocol string, key string) bool {
	k := strings.ToLower(strings.TrimSpace(key))
	if k == "" {
		return true
	}
	switch protocol {
	case ChannelMonitorRequestProtocolOpenAI:
		return k == "model" || k == "messages" || k == "input" || k == "stream"
	case ChannelMonitorRequestProtocolAnthropic:
		return k == "model" || k == "messages"
	case ChannelMonitorRequestProtocolGemini:
		return k == "contents"
	default:
		return true
	}
}
