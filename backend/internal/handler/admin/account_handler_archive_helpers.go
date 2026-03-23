package admin

import (
	"context"
	"errors"
	"fmt"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (h *AccountHandler) resolveArchiveGroup(ctx context.Context, platform string, groupName string) (*service.Group, error) {
	trimmedName := strings.TrimSpace(groupName)
	group, err := h.adminService.GetGroupByName(ctx, trimmedName)
	if err == nil && group != nil {
		if !strings.EqualFold(group.Platform, platform) {
			return nil, infraerrors.BadRequest("ACCOUNT_BATCH_CREATE_ARCHIVE_GROUP_PLATFORM_CONFLICT", fmt.Sprintf("group %q already exists under platform %s", trimmedName, group.Platform))
		}
		return group, nil
	}
	if err != nil && !errors.Is(err, service.ErrGroupNotFound) {
		return nil, err
	}

	created, createErr := h.adminService.CreateGroup(ctx, &service.CreateGroupInput{
		Name:             trimmedName,
		Description:      "Archive group for archived accounts",
		Platform:         platform,
		RateMultiplier:   1,
		IsExclusive:      false,
		SubscriptionType: service.SubscriptionTypeStandard,
	})
	if createErr == nil {
		return created, nil
	}

	group, err = h.adminService.GetGroupByName(ctx, trimmedName)
	if err == nil && group != nil {
		if !strings.EqualFold(group.Platform, platform) {
			return nil, infraerrors.BadRequest("ACCOUNT_BATCH_CREATE_ARCHIVE_GROUP_PLATFORM_CONFLICT", fmt.Sprintf("group %q already exists under platform %s", trimmedName, group.Platform))
		}
		return group, nil
	}
	return nil, createErr
}
