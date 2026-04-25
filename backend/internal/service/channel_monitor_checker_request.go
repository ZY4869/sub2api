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
	provider := monitor.Provider
	mode := monitor.BodyOverrideMode
	override := monitor.BodyOverride
	if mode == "" {
		mode = ChannelMonitorBodyOverrideModeOff
	}

	prompt := "Please reply with exactly: " + challenge

	switch provider {
	case ChannelMonitorProviderOpenAI, ChannelMonitorProviderGrok:
		path = "/v1/chat/completions"
		if provider == ChannelMonitorProviderGrok {
			path = "/grok/v1/chat/completions"
		}
		base := map[string]any{
			"model": modelID,
			"messages": []any{
				map[string]any{"role": "user", "content": prompt},
			},
			"temperature": 0,
			"max_tokens":  32,
			"stream":      false,
		}
		payload, requireChallenge, err = applyBodyOverride(provider, mode, base, override)
		return path, payload, requireChallenge, err
	case ChannelMonitorProviderAnthropic, ChannelMonitorProviderAntigravity:
		path = "/v1/messages"
		if provider == ChannelMonitorProviderAntigravity {
			path = "/antigravity/v1/messages"
		}
		base := map[string]any{
			"model":      modelID,
			"max_tokens": 32,
			"messages": []any{
				map[string]any{"role": "user", "content": prompt},
			},
			"temperature": 0,
		}
		payload, requireChallenge, err = applyBodyOverride(provider, mode, base, override)
		return path, payload, requireChallenge, err
	case ChannelMonitorProviderGemini:
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
		payload, requireChallenge, err = applyBodyOverride(provider, mode, base, override)
		return path, payload, requireChallenge, err
	default:
		return "", nil, false, errors.New("invalid provider")
	}
}

func applyBodyOverride(provider string, mode string, base map[string]any, override map[string]any) (map[string]any, bool, error) {
	switch mode {
	case ChannelMonitorBodyOverrideModeOff:
		return base, true, nil
	case ChannelMonitorBodyOverrideModeMerge:
		merged := map[string]any{}
		for k, v := range base {
			merged[k] = v
		}
		for k, v := range override {
			if isChannelMonitorBodyOverrideKeyBlocked(provider, k) {
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

func isChannelMonitorBodyOverrideKeyBlocked(provider string, key string) bool {
	k := strings.ToLower(strings.TrimSpace(key))
	if k == "" {
		return true
	}
	switch provider {
	case ChannelMonitorProviderOpenAI, ChannelMonitorProviderGrok:
		return k == "model" || k == "messages" || k == "stream"
	case ChannelMonitorProviderAnthropic, ChannelMonitorProviderAntigravity:
		return k == "model" || k == "messages"
	case ChannelMonitorProviderGemini:
		return k == "contents"
	default:
		return true
	}
}
