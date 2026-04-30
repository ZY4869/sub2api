package admin

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const (
	bulkUpdateAccountsFiltersPageSize = 1000
	bulkUpdateAccountsFiltersMaxPages = 10000
)

func (h *AccountHandler) resolveBulkUpdateAccountIDsByFilters(c *gin.Context, filters *BulkUpdateAccountsFilters) ([]int64, error) {
	if h == nil || h.adminService == nil {
		return nil, fmt.Errorf("admin service is nil")
	}
	if c == nil || c.Request == nil {
		return nil, fmt.Errorf("request context is nil")
	}
	if filters == nil {
		return nil, nil
	}

	platform := strings.TrimSpace(filters.Platform)
	accountType := strings.TrimSpace(filters.Type)
	status := service.NormalizeAdminAccountStatusInput(filters.Status)
	lifecycle := service.NormalizeAccountLifecycleInput(firstNonEmptyString(filters.Lifecycle, service.AccountLifecycleNormal))
	privacyMode := strings.TrimSpace(filters.PrivacyMode)
	limitedView := service.NormalizeAccountLimitedViewInput(firstNonEmptyString(filters.LimitedView, service.AccountLimitedViewAll))
	limitedReason := service.NormalizeAccountRateLimitReasonInput(filters.LimitedReason)
	runtimeView := service.NormalizeAccountRuntimeViewInput(firstNonEmptyString(filters.RuntimeView, service.AccountRuntimeViewAll))

	search := strings.TrimSpace(filters.Search)
	if len(search) > 100 {
		search = search[:100]
	}

	groupID := int64(0)
	if groupIDStr := strings.TrimSpace(filters.Group); groupIDStr != "" {
		if groupIDStr == accountListGroupUngroupedQueryValue {
			groupID = service.AccountListGroupUngrouped
		} else {
			parsedGroupID, err := strconv.ParseInt(groupIDStr, 10, 64)
			if err != nil || parsedGroupID < 0 {
				return nil, infraerrors.BadRequest("ADMIN_ACCOUNT_INVALID_GROUP_FILTER", "Invalid group filter")
			}
			groupID = parsedGroupID
		}
	}

	requestCtx := service.WithAccountLimitedFilters(c.Request.Context(), limitedView, limitedReason)
	requestCtx, _, err := h.buildRuntimeQueryContext(requestCtx, runtimeView)
	if err != nil {
		return nil, err
	}

	ids := make([]int64, 0, 32)
	page := 1
	var total int64
	for {
		accounts, t, err := h.adminService.ListAccounts(
			requestCtx,
			page,
			bulkUpdateAccountsFiltersPageSize,
			platform,
			accountType,
			status,
			search,
			groupID,
			lifecycle,
			privacyMode,
		)
		if err != nil {
			return nil, err
		}
		total = t
		if len(accounts) == 0 {
			break
		}
		for i := range accounts {
			ids = append(ids, accounts[i].ID)
		}
		if int64(len(ids)) >= total {
			break
		}
		page++
		if page > bulkUpdateAccountsFiltersMaxPages {
			return nil, fmt.Errorf("too many pages while resolving bulk update targets")
		}
	}

	logger.FromContext(c.Request.Context()).With(
		zap.String("component", "handler.admin.account.bulk_update"),
		zap.String("filters_sha256", hashBulkUpdateAccountsFilters(filters)),
		zap.String("platform", platform),
		zap.String("type", accountType),
		zap.String("status", status),
		zap.String("group", strings.TrimSpace(filters.Group)),
		zap.String("lifecycle", lifecycle),
		zap.String("privacy_mode", privacyMode),
		zap.String("limited_view", limitedView),
		zap.String("limited_reason", limitedReason),
		zap.String("runtime_view", runtimeView),
		zap.Int("resolved_ids", len(ids)),
		zap.Int64("total", total),
	).Info("bulk update targets resolved from filters")

	return ids, nil
}

func hashBulkUpdateAccountsFilters(filters *BulkUpdateAccountsFilters) string {
	if filters == nil {
		return ""
	}
	raw, err := json.Marshal(filters)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
