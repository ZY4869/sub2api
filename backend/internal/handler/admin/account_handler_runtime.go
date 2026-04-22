package admin

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type trackedActiveAccountReader interface {
	GetTrackedActiveAccountIDs(ctx context.Context) ([]int64, error)
}

type accountRuntimeSummaryResponse struct {
	InUse int64 `json:"in_use"`
}

func (h *AccountHandler) GetRuntimeSummary(c *gin.Context) {
	search := strings.TrimSpace(c.Query("search"))
	if len(search) > 100 {
		search = search[:100]
	}

	var groupID int64
	if groupIDStr := c.Query("group"); groupIDStr != "" {
		if groupIDStr == accountListGroupUngroupedQueryValue {
			groupID = service.AccountListGroupUngrouped
		} else {
			parsedGroupID, err := strconv.ParseInt(groupIDStr, 10, 64)
			if err != nil || parsedGroupID < 0 {
				response.BadRequestKey(c, "admin.account.invalid_group_filter", "Invalid group filter")
				return
			}
			groupID = parsedGroupID
		}
	}

	filters := service.AccountStatusSummaryFilters{
		Platform:      c.Query("platform"),
		AccountType:   c.Query("type"),
		Search:        search,
		GroupID:       groupID,
		Lifecycle:     c.DefaultQuery("lifecycle", service.AccountLifecycleNormal),
		PrivacyMode:   strings.TrimSpace(c.Query("privacy_mode")),
		LimitedView:   service.NormalizeAccountLimitedViewInput(c.DefaultQuery("limited_view", service.AccountLimitedViewAll)),
		LimitedReason: service.NormalizeAccountRateLimitReasonInput(c.Query("limited_reason")),
		RuntimeView:   service.NormalizeAccountRuntimeViewInput(c.DefaultQuery("runtime_view", service.AccountRuntimeViewAll)),
	}

	inUseCount, err := h.countInUseAccounts(c.Request.Context(), filters)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if filters.RuntimeView == service.AccountRuntimeViewAvailableOnly {
		inUseCount = 0
	}

	payload := accountRuntimeSummaryResponse{InUse: inUseCount}
	etag := buildAccountRuntimeSummaryETag(payload, filters)
	if etag != "" {
		c.Header("ETag", etag)
		c.Header("Vary", "If-None-Match")
		if ifNoneMatchMatched(c.GetHeader("If-None-Match"), etag) {
			c.Status(http.StatusNotModified)
			return
		}
	}

	response.Success(c, payload)
}

func (h *AccountHandler) buildRuntimeQueryContext(ctx context.Context, runtimeView string) (context.Context, []int64, error) {
	normalizedView := service.NormalizeAccountRuntimeViewInput(runtimeView)
	if normalizedView != service.AccountRuntimeViewInUseOnly && normalizedView != service.AccountRuntimeViewAvailableOnly {
		return service.WithAccountRuntimeFilters(ctx, normalizedView, nil), nil, nil
	}

	candidateIDs, err := h.resolveInUseAccountIDs(ctx)
	if err != nil {
		return nil, nil, err
	}
	return service.WithAccountRuntimeFilters(ctx, normalizedView, candidateIDs), candidateIDs, nil
}

func (h *AccountHandler) resolveInUseAccountIDs(ctx context.Context) ([]int64, error) {
	inUse := make(map[int64]struct{})

	if h.concurrencyService != nil {
		trackedIDs, err := h.concurrencyService.GetTrackedActiveAccountIDs(ctx)
		if err != nil {
			return nil, err
		}
		if len(trackedIDs) > 0 {
			counts, err := h.concurrencyService.GetAccountConcurrencyBatch(ctx, trackedIDs)
			if err != nil {
				return nil, err
			}
			for accountID, count := range counts {
				if count > 0 {
					inUse[accountID] = struct{}{}
				}
			}
		}
	}

	if h.sessionLimitCache != nil {
		reader, ok := h.sessionLimitCache.(trackedActiveAccountReader)
		if ok {
			trackedIDs, err := reader.GetTrackedActiveAccountIDs(ctx)
			if err != nil {
				return nil, err
			}
			if len(trackedIDs) > 0 {
				idleTimeouts := make(map[int64]time.Duration, len(trackedIDs))
				accounts, err := h.adminService.GetAccountsByIDs(ctx, trackedIDs)
				if err != nil {
					return nil, err
				}
				for _, account := range accounts {
					if account == nil {
						continue
					}
					if timeoutMinutes := account.GetSessionIdleTimeoutMinutes(); timeoutMinutes > 0 {
						idleTimeouts[account.ID] = time.Duration(timeoutMinutes) * time.Minute
					}
				}
				counts, err := h.sessionLimitCache.GetActiveSessionCountBatch(ctx, trackedIDs, idleTimeouts)
				if err != nil {
					return nil, err
				}
				for accountID, count := range counts {
					if count > 0 {
						inUse[accountID] = struct{}{}
					}
				}
			}
		}
	}

	if len(inUse) == 0 {
		return nil, nil
	}

	ids := make([]int64, 0, len(inUse))
	for accountID := range inUse {
		ids = append(ids, accountID)
	}
	slices.Sort(ids)
	return ids, nil
}

func (h *AccountHandler) countInUseAccounts(ctx context.Context, filters service.AccountStatusSummaryFilters) (int64, error) {
	candidateIDs, err := h.resolveInUseAccountIDs(ctx)
	if err != nil {
		return 0, err
	}
	if len(candidateIDs) == 0 {
		return 0, nil
	}

	accounts, err := h.adminService.GetAccountsByIDs(ctx, candidateIDs)
	if err != nil {
		return 0, err
	}

	now := time.Now()
	var count int64
	for _, account := range accounts {
		if account == nil {
			continue
		}
		if accountMatchesRuntimeSummaryFilters(account, filters, now) {
			count++
		}
	}
	return count, nil
}

func accountMatchesRuntimeSummaryFilters(account *service.Account, filters service.AccountStatusSummaryFilters, now time.Time) bool {
	if account == nil {
		return false
	}
	if platform := strings.TrimSpace(filters.Platform); platform != "" && !strings.EqualFold(account.Platform, platform) {
		return false
	}
	if accountType := strings.TrimSpace(filters.AccountType); accountType != "" && !strings.EqualFold(account.Type, accountType) {
		return false
	}
	privacyMode := strings.TrimSpace(filters.PrivacyMode)
	if privacyMode != "" {
		accountPrivacyMode := strings.TrimSpace(account.GetExtraString("privacy_mode"))
		if privacyMode == "unset" {
			if accountPrivacyMode != "" {
				return false
			}
		} else if accountPrivacyMode != privacyMode {
			return false
		}
	}
	search := strings.TrimSpace(strings.ToLower(filters.Search))
	if search != "" && !strings.Contains(strings.ToLower(account.Name), search) {
		return false
	}

	lifecycle := service.NormalizeAccountLifecycleInput(filters.Lifecycle)
	accountLifecycle := service.NormalizeAccountLifecycleInput(account.LifecycleState)
	if lifecycle != service.AccountLifecycleAll && accountLifecycle != lifecycle {
		return false
	}

	if filters.GroupID == service.AccountListGroupUngrouped {
		if len(account.GroupIDs) > 0 || len(account.Groups) > 0 {
			return false
		}
	} else if filters.GroupID > 0 {
		matched := false
		for _, groupID := range account.GroupIDs {
			if groupID == filters.GroupID {
				matched = true
				break
			}
		}
		if !matched {
			for _, group := range account.Groups {
				if group.ID == filters.GroupID {
					matched = true
					break
				}
			}
		}
		if !matched {
			return false
		}
	}

	displayRateLimit := service.AccountDisplayRateLimitState(account, now)
	isLimited := displayRateLimit.Limited
	switch service.NormalizeAccountLimitedViewInput(filters.LimitedView) {
	case service.AccountLimitedViewNormalOnly:
		if isLimited {
			return false
		}
	case service.AccountLimitedViewLimitedOnly:
		if !isLimited {
			return false
		}
	}
	if reason := service.NormalizeAccountRateLimitReasonInput(filters.LimitedReason); reason != "" {
		if !isLimited || displayRateLimit.Reason != reason {
			return false
		}
	}

	return true
}

func buildAccountRuntimeSummaryETag(summary accountRuntimeSummaryResponse, filters service.AccountStatusSummaryFilters) string {
	payload := struct {
		Filters service.AccountStatusSummaryFilters `json:"filters"`
		Summary accountRuntimeSummaryResponse       `json:"summary"`
	}{
		Filters: filters,
		Summary: summary,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(raw)
	return "\"" + hex.EncodeToString(sum[:]) + "\""
}
