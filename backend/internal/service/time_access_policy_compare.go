package service

import "encoding/json"

func timeAccessPoliciesEqual(a, b *TimeAccessPolicy) bool {
	na, errA := NormalizeTimeAccessPolicy(a)
	nb, errB := NormalizeTimeAccessPolicy(b)
	if errA != nil || errB != nil {
		return a == b
	}
	ba, _ := json.Marshal(na)
	bb, _ := json.Marshal(nb)
	return string(ba) == string(bb)
}
