package claude

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/modelregistry"
)

func NormalizeModelID(modelID string) string {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return ""
	}
	if resolved, ok := modelregistry.ResolveToCanonicalID(modelID); ok && resolved != "" {
		return resolved
	}
	return modelregistry.NormalizeID(modelID)
}
