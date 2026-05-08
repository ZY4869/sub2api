package service

import (
	"strings"
	"time"
)

type ContentModerationAPIKeyStatus struct {
	Hash        string `json:"hash"`
	Masked      string `json:"masked"`
	FrozenUntil string `json:"frozen_until,omitempty"`
	LastError   string `json:"last_error,omitempty"`
}

func ContentModerationAPIKeyStatuses(items []ContentModerationAPIKey, now time.Time) []ContentModerationAPIKeyStatus {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	normalized := normalizeContentModerationAPIKeys(items, now)
	statuses := make([]ContentModerationAPIKeyStatus, 0, len(normalized))
	for _, item := range normalized {
		status := ContentModerationAPIKeyStatus{
			Hash:   item.Hash,
			Masked: maskContentModerationAPIKey(item.Key),
		}
		if freeze, ok := getContentModerationFreeze(item.Hash, now); ok {
			status.FrozenUntil = freeze.until.UTC().Format(time.RFC3339)
			status.LastError = freeze.lastError
		}
		statuses = append(statuses, status)
	}
	return statuses
}

func SelectContentModerationAPIKey(items []ContentModerationAPIKey, now time.Time) (ContentModerationAPIKey, bool) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	normalized := normalizeContentModerationAPIKeys(items, now)
	for _, item := range normalized {
		if _, frozen := getContentModerationFreeze(item.Hash, now); !frozen {
			return item, true
		}
	}
	return ContentModerationAPIKey{}, false
}

func maskContentModerationAPIKey(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	if len(key) <= 10 {
		return strings.Repeat("*", len(key))
	}
	return key[:6] + "..." + key[len(key)-4:]
}
