package service

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
)

func normalizeKiroRuntimeModelID(modelID string) string {
	if normalized := claude.NormalizeModelID(modelID); normalized != "" {
		return normalized
	}
	return strings.TrimSpace(modelID)
}
