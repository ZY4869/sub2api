package admin

import (
	"log/slog"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const accountModelPolicyBackfillDefaultPageSize = 100

func (h *AccountHandler) BackfillModelPolicies(c *gin.Context) {
	slog.Info(
		"admin_account_model_policy_backfill_start",
		"page_size", accountModelPolicyBackfillDefaultPageSize,
	)

	result, err := h.adminService.BackfillAccountModelPolicies(
		c.Request.Context(),
		h.modelRegistryService,
		accountModelPolicyBackfillDefaultPageSize,
	)
	if err != nil {
		slog.Error(
			"admin_account_model_policy_backfill_failed",
			"page_size", accountModelPolicyBackfillDefaultPageSize,
			"error", err,
		)
		response.ErrorFrom(c, err)
		return
	}
	if result == nil {
		result = &service.AccountModelPolicyBackfillResult{}
	}

	slog.Info(
		"admin_account_model_policy_backfill_complete",
		"page_size", accountModelPolicyBackfillDefaultPageSize,
		"scanned", result.Scanned,
		"updated", result.Updated,
		"scope_normalized", result.ScopeNormalized,
		"snapshot_refreshed", result.SnapshotRefreshed,
	)
	response.Success(c, result)
}
