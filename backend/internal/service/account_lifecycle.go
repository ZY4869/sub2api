package service

import (
	"strings"
	"time"
)

const (
	AccountLifecycleNormal      = "normal"
	AccountLifecycleArchived    = "archived"
	AccountLifecycleBlacklisted = "blacklisted"
	AccountLifecycleAll         = "all"
)

const AccountBlacklistRetention = 72 * time.Hour

func NormalizeAccountLifecycleInput(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", AccountLifecycleNormal:
		return AccountLifecycleNormal
	case AccountLifecycleArchived:
		return AccountLifecycleArchived
	case AccountLifecycleBlacklisted:
		return AccountLifecycleBlacklisted
	case AccountLifecycleAll:
		return AccountLifecycleAll
	default:
		return AccountLifecycleNormal
	}
}

func IsAccountLifecycleSchedulable(lifecycle string) bool {
	return NormalizeAccountLifecycleInput(lifecycle) == AccountLifecycleNormal
}
