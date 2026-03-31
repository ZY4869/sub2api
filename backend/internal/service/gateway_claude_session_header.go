package service

import (
	"net/http"
	"strings"

	"github.com/tidwall/gjson"
)

func syncClaudeCodeSessionHeader(req *http.Request, body []byte) {
	if req == nil {
		return
	}
	if strings.TrimSpace(req.Header.Get("X-Claude-Code-Session-Id")) == "" {
		return
	}
	userID := strings.TrimSpace(gjson.GetBytes(body, "metadata.user_id").String())
	if userID == "" {
		return
	}
	parsed := ParseMetadataUserID(userID)
	if parsed == nil || strings.TrimSpace(parsed.SessionID) == "" {
		return
	}
	req.Header.Set("X-Claude-Code-Session-Id", parsed.SessionID)
}
