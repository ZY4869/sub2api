package service

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/modelregistry"
)

const (
	ModelRegistryScheduleActive      = "active"
	ModelRegistryScheduleScheduled   = "scheduled"
	ModelRegistryScheduleExpired     = "expired"
	ModelRegistryScheduleOutOfWindow = "out_of_window"
	ModelRegistryScheduleInvalid     = "invalid"
)

func modelEntryTimeAccessPolicy(entry modelregistry.ModelEntry) *TimeAccessPolicy {
	policy := timeAccessPolicyFromAnyMap(entry.AccessTimePolicy)
	if strings.TrimSpace(entry.AvailableFrom) != "" {
		if t, err := time.Parse(time.RFC3339, strings.TrimSpace(entry.AvailableFrom)); err == nil {
			if policy == nil {
				policy = &TimeAccessPolicy{}
			}
			policy.NotBefore = &t
		}
	}
	if strings.TrimSpace(entry.AvailableUntil) != "" {
		if t, err := time.Parse(time.RFC3339, strings.TrimSpace(entry.AvailableUntil)); err == nil {
			if policy == nil {
				policy = &TimeAccessPolicy{}
			}
			policy.NotAfter = &t
		}
	}
	if policy != nil {
		policy.Enabled = true
	}
	return policy
}

func modelRegistryScheduleStatus(entry modelregistry.ModelEntry, now time.Time) string {
	eval := EvaluateTimeAccessPolicy(modelEntryTimeAccessPolicy(entry), now)
	if eval.Allowed {
		return ModelRegistryScheduleActive
	}
	switch eval.Reason {
	case TimeAccessDecisionNotBefore:
		return ModelRegistryScheduleScheduled
	case TimeAccessDecisionNotAfter:
		return ModelRegistryScheduleExpired
	case TimeAccessDecisionOutsideWindow:
		return ModelRegistryScheduleOutOfWindow
	default:
		return ModelRegistryScheduleInvalid
	}
}

func modelRegistryEntryCurrentlyAvailable(entry modelregistry.ModelEntry, now time.Time) bool {
	return modelRegistryScheduleStatus(entry, now) == ModelRegistryScheduleActive
}

func timeAccessPolicyFromAnyMap(raw map[string]any) *TimeAccessPolicy {
	if len(raw) == 0 {
		return nil
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var policy TimeAccessPolicy
	if err := json.Unmarshal(data, &policy); err != nil {
		return nil
	}
	normalized, err := NormalizeTimeAccessPolicy(&policy)
	if err != nil {
		return &policy
	}
	return normalized
}

func timeAccessPolicyToAnyMap(policy *TimeAccessPolicy) map[string]any {
	normalized, err := NormalizeTimeAccessPolicy(policy)
	if err != nil || normalized == nil {
		return nil
	}
	data, err := json.Marshal(normalized)
	if err != nil {
		return nil
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil
	}
	return out
}
