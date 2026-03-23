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

type BatchArchiveAccountsRequest struct {
	AccountIDs []int64 `json:"account_ids" binding:"required,min=1"`
	GroupName  string  `json:"group_name"`
}

type BatchArchiveAccountsResult struct {
	ArchivedCount    int     `json:"archived_count"`
	FailedCount      int     `json:"failed_count"`
	ArchiveGroupID   int64   `json:"archive_group_id"`
	ArchiveGroupName string  `json:"archive_group_name"`
	SuccessIDs       []int64 `json:"success_ids,omitempty"`
	FailedIDs        []int64 `json:"failed_ids,omitempty"`
}

func (h *AccountHandler) BatchArchiveAccounts(c *gin.Context) {
	var req BatchArchiveAccountsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if strings.TrimSpace(req.GroupName) == "" {
		response.BadRequest(c, "group_name is required")
		return
	}

	executeAdminIdempotentJSON(c, "admin.accounts.batch_archive", req, service.DefaultWriteIdempotencyTTL(), func(ctx context.Context) (any, error) {
		return h.executeBatchArchiveAccounts(ctx, &req)
	})
}

func (h *AccountHandler) executeBatchArchiveAccounts(ctx context.Context, req *BatchArchiveAccountsRequest) (*BatchArchiveAccountsResult, error) {
	accountIDs := uniqueBatchArchiveAccountIDs(req.AccountIDs)
	accounts, err := h.adminService.GetAccountsByIDs(ctx, accountIDs)
	if err != nil {
		return nil, err
	}
	if len(accounts) != len(accountIDs) {
		return nil, infraerrors.BadRequest("ACCOUNT_BATCH_ARCHIVE_NOT_FOUND", "some selected accounts no longer exist")
	}

	platform, err := resolveBatchArchivePlatform(accounts)
	if err != nil {
		return nil, err
	}

	archiveGroup, err := h.resolveBatchCreateArchiveGroup(ctx, platform, req.GroupName)
	if err != nil {
		return nil, err
	}

	groupIDs := []int64{archiveGroup.ID}
	slog.Info("admin_account_batch_archive_started",
		"platform", platform,
		"account_count", len(accountIDs),
		"archive_group_name", archiveGroup.Name,
	)

	updateResult, err := h.adminService.BulkUpdateAccounts(ctx, &service.BulkUpdateAccountsInput{
		AccountIDs:              accountIDs,
		Status:                  service.StatusDisabled,
		GroupIDs:                &groupIDs,
		LifecycleState:          service.AccountLifecycleArchived,
		LifecycleReasonCode:     "batch_archive",
		LifecycleReasonMessage:  "Archived via batch archive",
	})
	if err != nil {
		return nil, err
	}

	result := &BatchArchiveAccountsResult{
		ArchivedCount:    updateResult.Success,
		FailedCount:      updateResult.Failed,
		ArchiveGroupID:   archiveGroup.ID,
		ArchiveGroupName: archiveGroup.Name,
		SuccessIDs:       append([]int64(nil), updateResult.SuccessIDs...),
		FailedIDs:        append([]int64(nil), updateResult.FailedIDs...),
	}

	slog.Info("admin_account_batch_archive_completed",
		"platform", platform,
		"archived_count", result.ArchivedCount,
		"failed_count", result.FailedCount,
		"archive_group_name", result.ArchiveGroupName,
	)

	return result, nil
}

func uniqueBatchArchiveAccountIDs(ids []int64) []int64 {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[int64]struct{}, len(ids))
	unique := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	return unique
}

func resolveBatchArchivePlatform(accounts []*service.Account) (string, error) {
	if len(accounts) == 0 {
		return "", infraerrors.BadRequest("ACCOUNT_BATCH_ARCHIVE_EMPTY", "no accounts selected")
	}

	platform := strings.TrimSpace(accounts[0].Platform)
	for _, account := range accounts[1:] {
		if !strings.EqualFold(platform, strings.TrimSpace(account.Platform)) {
			return "", infraerrors.BadRequest(
				"ACCOUNT_BATCH_ARCHIVE_MIXED_PLATFORM",
				fmt.Sprintf("batch archive only supports accounts from the same platform: %s vs %s", platform, account.Platform),
			)
		}
	}
	return platform, nil
}
