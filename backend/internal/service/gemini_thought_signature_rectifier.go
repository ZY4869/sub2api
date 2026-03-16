package service

import "encoding/json"

func rectifyGeminiThoughtSignatures(body []byte) ([]byte, bool, error) {
	if len(body) == 0 {
		return body, false, nil
	}
	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, false, err
	}
	changed := replaceAllGeminiThoughtSignatures(payload, geminiDummyThoughtSignature)
	if !changed {
		return body, false, nil
	}
	out, err := json.Marshal(payload)
	if err != nil {
		return nil, false, err
	}
	return out, true, nil
}

func replaceAllGeminiThoughtSignatures(value any, replacement string) bool {
	changed := false
	switch v := value.(type) {
	case map[string]any:
		for key, item := range v {
			if key == "thoughtSignature" {
				if item != replacement {
					v[key] = replacement
					changed = true
				}
				continue
			}
			if replaceAllGeminiThoughtSignatures(item, replacement) {
				changed = true
			}
		}
	case []any:
		for _, item := range v {
			if replaceAllGeminiThoughtSignatures(item, replacement) {
				changed = true
			}
		}
	}
	return changed
}

