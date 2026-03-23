package admin

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type ArchiveGroupAccountsRequest struct {
	SourceGroupID int64  `json:"source_group_id" binding:"required"`
	GroupName     string `json:"group_name" binding:"required"`
}

type ArchiveGroupAccountsResult struct {
	SourceGroupID      int64   `json:"source_group_id"`
	SourceGroupName    string  `json:"source_group_name"`
	ArchivedCount      int     `json:"archived_count"`
	FailedCount        int     `json:"failed_count"`
	ArchiveGroupID     int64   `json:"archive_group_id"`
	ArchiveGroupName   string  `json:"archive_group_name"`
	ArchivedAccountIDs []int64 `json:"archived_account_ids,omitempty"`
	FailedAccountIDs   []int64 `json:"failed_account_ids,omitempty"`
}

const archiveGroupListPageSize = 200

func (h *AccountHandler) ArchiveGroupAccounts(c *gin.Context) {
	var req ArchiveGroupAccountsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if req.SourceGroupID <= 0 {
		response.BadRequest(c, "source_group_id must be greater than 0")
		return
	}
	if strings.TrimSpace(req.GroupName) == "" {
		response.BadRequest(c, "group_name is required")
		return
	}

	executeAdminIdempotentJSON(c, "admin.accounts.archive_group", req, service.DefaultWriteIdempotencyTTL(), func(ctx context.Context) (any, error) {
		return h.executeArchiveGroupAccounts(ctx, &req)
	})
}

func (h *AccountHandler) executeArchiveGroupAccounts(ctx context.Context, req *ArchiveGroupAccountsRequest) (*ArchiveGroupAccountsResult, error) {
	sourceGroup, err := h.adminService.GetGroup(ctx, req.SourceGroupID)
	if err != nil {
		return nil, err
	}

	accounts, err := h.listAllAccountsForArchiveGroup(ctx, req.SourceGroupID)
	if err != nil {
		return nil, err
	}
	if len(accounts) == 0 {
		return nil, infraerrors.BadRequest(
			"ACCOUNT_GROUP_ARCHIVE_EMPTY",
			fmt.Sprintf("group %q has no accounts to archive", sourceGroup.Name),
		)
	}

	platform, err := resolveBatchArchivePlatform(accounts)
	if err != nil {
		return nil, err
	}
	if groupPlatform := strings.TrimSpace(sourceGroup.Platform); groupPlatform != "" && !strings.EqualFold(groupPlatform, platform) {
		return nil, infraerrors.BadRequest(
			"ACCOUNT_GROUP_ARCHIVE_PLATFORM_MISMATCH",
			fmt.Sprintf("group %q belongs to platform %s but contains %s accounts", sourceGroup.Name, groupPlatform, platform),
		)
	}

	archiveGroup, err := h.resolveBatchCreateArchiveGroup(ctx, platform, req.GroupName)
	if err != nil {
		return nil, err
	}

	accountIDs := make([]int64, 0, len(accounts))
	for _, account := range accounts {
		if account == nil || account.ID <= 0 {
			continue
		}
		accountIDs = append(accountIDs, account.ID)
	}

	groupIDs := []int64{archiveGroup.ID}
	slog.Info("admin_account_group_archive_started",
		"platform", platform,
		"source_group_id", sourceGroup.ID,
		"source_group_name", sourceGroup.Name,
		"account_count", len(accountIDs),
		"archive_group_name", archiveGroup.Name,
	)

	updateResult, err := h.adminService.BulkUpdateAccounts(ctx, &service.BulkUpdateAccountsInput{
		AccountIDs:             accountIDs,
		Status:                 service.StatusDisabled,
		GroupIDs:               &groupIDs,
		LifecycleState:         service.AccountLifecycleArchived,
		LifecycleReasonCode:    "archive_group",
		LifecycleReasonMessage: fmt.Sprintf("Archived from group %s", sourceGroup.Name),
	})
	if err != nil {
		return nil, err
	}

	result := &ArchiveGroupAccountsResult{
		SourceGroupID:      sourceGroup.ID,
		SourceGroupName:    sourceGroup.Name,
		ArchivedCount:      updateResult.Success,
		FailedCount:        updateResult.Failed,
		ArchiveGroupID:     archiveGroup.ID,
		ArchiveGroupName:   archiveGroup.Name,
		ArchivedAccountIDs: append([]int64(nil), updateResult.SuccessIDs...),
		FailedAccountIDs:   append([]int64(nil), updateResult.FailedIDs...),
	}

	slog.Info("admin_account_group_archive_completed",
		"platform", platform,
		"source_group_id", sourceGroup.ID,
		"source_group_name", sourceGroup.Name,
		"archived_count", result.ArchivedCount,
		"failed_count", result.FailedCount,
		"archive_group_name", result.ArchiveGroupName,
	)

	return result, nil
}

func (h *AccountHandler) listAllAccountsForArchiveGroup(ctx context.Context, groupID int64) ([]*service.Account, error) {
	collected := make([]*service.Account, 0)
	for page := 1; ; page++ {
		items, total, err := h.adminService.ListAccounts(ctx, page, archiveGroupListPageSize, "", "", "", "", groupID, service.AccountLifecycleNormal)
		if err != nil {
			return nil, err
		}
		if len(items) == 0 {
			break
		}
		for i := range items {
			account := items[i]
			collected = append(collected, &account)
		}
		if int64(len(collected)) >= total {
			break
		}
	}
	return collected, nil
}
