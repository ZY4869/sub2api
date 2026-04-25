package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"strings"
	"time"
)

func readLimited(r io.Reader, limit int64) ([]byte, error) {
	limited := io.LimitReader(r, limit+1)
	b, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > limit {
		return nil, errors.New("too large")
	}
	return b, nil
}

func truncateText(v string, max int) string {
	trimmed := strings.TrimSpace(v)
	if trimmed == "" {
		return ""
	}
	if max <= 0 {
		return ""
	}
	runes := []rune(trimmed)
	if len(runes) <= max {
		return trimmed
	}
	return string(runes[:max])
}

func randomChallenge() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err == nil {
		return "cm-" + hex.EncodeToString(b[:])
	}
	return "cm-" + hex.EncodeToString([]byte(time.Now().Format("150405")))
}
