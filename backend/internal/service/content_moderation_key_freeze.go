package service

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	contentModerationAuthFreezeDuration    = 10 * time.Minute
	contentModerationRateFreezeDuration    = time.Minute
	contentModerationTransientFreezeWindow = 10 * time.Second
)

var (
	contentModerationFreezeMu sync.RWMutex
	contentModerationFreeze   = map[string]contentModerationFreezeState{}
)

type contentModerationFreezeState struct {
	until     time.Time
	lastError string
}

func RegisterContentModerationKeyFailure(hash, reason string, statusCode int, err error, now time.Time) {
	hash = strings.TrimSpace(strings.ToLower(hash))
	if hash == "" {
		return
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	duration := contentModerationFreezeDuration(statusCode, err)
	if duration <= 0 {
		return
	}
	contentModerationFreezeMu.Lock()
	contentModerationFreeze[hash] = contentModerationFreezeState{
		until:     now.Add(duration),
		lastError: truncateModerationErrorReason(reason),
	}
	contentModerationFreezeMu.Unlock()
}

func ClearContentModerationKeyFreeze(hash string) {
	hash = strings.TrimSpace(strings.ToLower(hash))
	if hash == "" {
		return
	}
	contentModerationFreezeMu.Lock()
	delete(contentModerationFreeze, hash)
	contentModerationFreezeMu.Unlock()
}

func getContentModerationFreeze(hash string, now time.Time) (contentModerationFreezeState, bool) {
	hash = strings.TrimSpace(strings.ToLower(hash))
	if hash == "" {
		return contentModerationFreezeState{}, false
	}
	contentModerationFreezeMu.RLock()
	state, ok := contentModerationFreeze[hash]
	contentModerationFreezeMu.RUnlock()
	if !ok {
		return contentModerationFreezeState{}, false
	}
	if now.Before(state.until) {
		return state, true
	}
	ClearContentModerationKeyFreeze(hash)
	return contentModerationFreezeState{}, false
}

func contentModerationFreezeDuration(statusCode int, err error) time.Duration {
	switch {
	case statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden:
		return contentModerationAuthFreezeDuration
	case statusCode == http.StatusTooManyRequests:
		return contentModerationRateFreezeDuration
	case statusCode >= 500:
		return contentModerationTransientFreezeWindow
	case err != nil:
		return contentModerationTransientFreezeWindow
	default:
		return 0
	}
}
