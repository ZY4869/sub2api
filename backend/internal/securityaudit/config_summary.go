package securityaudit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

func preservedTokenCiphertext(next storageEndpoint, current storageConfig) string {
	for _, endpoint := range current.Endpoints {
		if endpoint.ID == next.ID && endpoint.BaseURL == next.BaseURL {
			return endpoint.TokenCiphertext
		}
	}
	return ""
}

func changeSummary(cfg storageConfig) string {
	rawGroups, _ := json.Marshal(cfg.GroupIDs)
	digest := sha256.Sum256(rawGroups)
	summary := map[string]any{
		"enabled": cfg.Enabled, "blocking_enabled": cfg.BlockingEnabled,
		"store_pass_events": cfg.StorePassEvents, "endpoint_count": len(cfg.Endpoints),
		"scanner_count": len(cfg.Scanners), "all_groups": cfg.AllGroups,
		"group_count": len(cfg.GroupIDs), "group_hash": hex.EncodeToString(digest[:]),
	}
	raw, _ := json.Marshal(summary)
	return string(raw)
}
