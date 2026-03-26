package service

import (
	"context"
	"fmt"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

func (s *adminServiceImpl) validateAccountGroupBindings(ctx context.Context, groupIDs []int64, accountPlatform string, extra map[string]any) error {
	if len(groupIDs) == 0 || s.groupRepo == nil {
		return nil
	}
	allowed := allowedGroupPlatformsForAccount(accountPlatform, extra)
	if len(allowed) == 0 {
		return nil
	}
	for _, groupID := range groupIDs {
		group, err := s.groupRepo.GetByID(ctx, groupID)
		if err != nil {
			return fmt.Errorf("get group: %w", err)
		}
		if group == nil {
			return fmt.Errorf("get group: %w", ErrGroupNotFound)
		}
		if _, ok := allowed[group.Platform]; ok {
			continue
		}
		accountPlatforms := strings.Join(RoutingPlatformsFromValues(accountPlatform, extra), ",")
		if accountPlatforms == "" {
			accountPlatforms = RoutingPlatformFromValues(accountPlatform, extra)
		}
		return infraerrors.BadRequest(
			"INVALID_GROUP_BINDING",
			fmt.Sprintf("account platform %s cannot bind group platform %s", accountPlatforms, group.Platform),
		)
	}
	return nil
}

func allowedGroupPlatformsForAccount(accountPlatform string, extra map[string]any) map[string]struct{} {
	platforms := RoutingPlatformsFromValues(accountPlatform, extra)
	if len(platforms) == 0 {
		return nil
	}
	allowed := make(map[string]struct{}, len(platforms)+2)
	for _, platform := range platforms {
		allowed[platform] = struct{}{}
	}
	if strings.EqualFold(accountPlatform, PlatformAntigravity) && extraBool(extra, "mixed_scheduling") {
		allowed[PlatformAnthropic] = struct{}{}
		allowed[PlatformGemini] = struct{}{}
	}
	return allowed
}

func extraBool(extra map[string]any, key string) bool {
	if len(extra) == 0 {
		return false
	}
	value, ok := extra[key].(bool)
	return ok && value
}
