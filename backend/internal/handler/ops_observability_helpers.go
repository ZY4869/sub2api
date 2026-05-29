package handler

import (
	"encoding/json"
	"strconv"
	"unicode/utf8"
)

func truncateString(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if len(s) <= max {
		return s
	}
	cut := s[:max]
	// Ensure truncation does not split multi-byte characters.
	for len(cut) > 0 && !utf8.ValidString(cut) {
		cut = cut[:len(cut)-1]
	}
	return cut
}

func strconvItoa(v int) string {
	return strconv.Itoa(v)
}

// shouldSkipOpsErrorLog determines if an error should be skipped from logging based on settings.
// Returns true for errors that should be filtered according to OpsAdvancedSettings.

func strconvFloat(value float64) string {
	raw, _ := json.Marshal(value)
	return string(raw)
}
