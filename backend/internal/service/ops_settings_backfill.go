package service

import (
	"encoding/json"
	"strings"
)

type opsAdvancedSettingsPresenceProbe struct {
	RequestDetailCleanupEnabled  json.RawMessage `json:"request_detail_cleanup_enabled"`
	RequestDetailCleanupSchedule json.RawMessage `json:"request_detail_cleanup_schedule"`
}

func backfillOpsAdvancedSettingsRequestDetailCleanup(raw string, cfg *OpsAdvancedSettings) {
	if cfg == nil || strings.TrimSpace(raw) == "" {
		return
	}

	var probe opsAdvancedSettingsPresenceProbe
	if err := json.Unmarshal([]byte(raw), &probe); err != nil {
		return
	}

	if probe.RequestDetailCleanupEnabled == nil {
		cfg.RequestDetailCleanupEnabled = cfg.DataRetention.CleanupEnabled
	}
	if probe.RequestDetailCleanupSchedule == nil {
		cfg.RequestDetailCleanupSchedule = cfg.DataRetention.CleanupSchedule
	}
}
