package repository

import (
	"encoding/json"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func timeAccessPolicyFromMap(raw map[string]any) *service.TimeAccessPolicy {
	if len(raw) == 0 {
		return nil
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var policy service.TimeAccessPolicy
	if err := json.Unmarshal(data, &policy); err != nil {
		return nil
	}
	normalized, err := service.NormalizeTimeAccessPolicy(&policy)
	if err != nil {
		return &policy
	}
	return normalized
}

func timeAccessPolicyToMap(policy *service.TimeAccessPolicy) map[string]any {
	normalized, err := service.NormalizeTimeAccessPolicy(policy)
	if err != nil || normalized == nil {
		return map[string]any{}
	}
	data, err := json.Marshal(normalized)
	if err != nil {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return map[string]any{}
	}
	return out
}
