package handler

import (
	"encoding/json"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type parsedOpsError struct {
	ErrorType string
	Message   string
	Code      string
}

func parseOpsErrorResponse(body []byte) parsedOpsError {
	if len(body) == 0 {
		return parsedOpsError{}
	}

	// Fast path: attempt to decode into a generic map.
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		return parsedOpsError{Message: truncateString(string(body), 1024)}
	}

	// Claude/OpenAI-style gateway error: { type:"error", error:{ type, message } }
	if errObj, ok := m["error"].(map[string]any); ok {
		t, _ := errObj["type"].(string)
		msg, _ := errObj["message"].(string)
		// Gemini googleError also uses "error": { code, message, status }
		if msg == "" {
			if v, ok := errObj["message"]; ok {
				msg, _ = v.(string)
			}
		}
		if t == "" {
			// Gemini error does not have "type" field.
			t = "api_error"
		}
		// For gemini error, capture numeric code as string for business-limited mapping if needed.
		var code string
		if v, ok := errObj["code"]; ok {
			switch n := v.(type) {
			case float64:
				code = strconvItoa(int(n))
			case int:
				code = strconvItoa(n)
			}
		}
		return parsedOpsError{ErrorType: t, Message: msg, Code: code}
	}

	// APIKeyAuth-style: { code:"INSUFFICIENT_BALANCE", message:"..." }
	code, _ := m["code"].(string)
	msg, _ := m["message"].(string)
	if code != "" || msg != "" {
		return parsedOpsError{ErrorType: "api_error", Message: msg, Code: code}
	}

	return parsedOpsError{Message: truncateString(string(body), 1024)}
}

func resolveOpsPlatform(apiKey *service.APIKey, fallback string) string {
	if apiKey != nil && apiKey.Group != nil && apiKey.Group.Platform != "" {
		return apiKey.Group.Platform
	}
	return fallback
}

func guessPlatformFromPath(path string) string {
	p := strings.ToLower(path)
	switch {
	case strings.HasPrefix(p, "/antigravity/"):
		return service.PlatformAntigravity
	case strings.HasPrefix(p, "/v1beta/"):
		return service.PlatformGemini
	case strings.Contains(p, "/responses"):
		return service.PlatformOpenAI
	default:
		return ""
	}
}
