package admin

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type AccountBillingProbeRequest struct {
	Force bool `json:"force"`
}

type BatchAccountBillingProbeRequest struct {
	AccountIDs  []int64 `json:"account_ids"`
	Platform    string  `json:"platform"`
	Type        string  `json:"type"`
	Status      string  `json:"status"`
	Search      string  `json:"search"`
	GroupID     int64   `json:"group_id"`
	Lifecycle   string  `json:"lifecycle"`
	PrivacyMode string  `json:"privacy_mode"`
	Limit       int     `json:"limit"`
}

type AccountBillingProbeResult struct {
	AccountID   int64                         `json:"account_id"`
	AccountName string                        `json:"account_name,omitempty"`
	Platform    string                        `json:"platform,omitempty"`
	Supported   bool                          `json:"supported"`
	Status      string                        `json:"status"`
	Source      string                        `json:"source,omitempty"`
	FetchedAt   int64                         `json:"fetched_at,omitempty"`
	Persisted   bool                          `json:"persisted"`
	Result      *service.GrokQuotaProbeResult `json:"result,omitempty"`
	Error       string                        `json:"error,omitempty"`
}

type BatchAccountBillingProbeResponse struct {
	Items       []AccountBillingProbeResult `json:"items"`
	Total       int                         `json:"total"`
	Succeeded   int                         `json:"succeeded"`
	Failed      int                         `json:"failed"`
	Unsupported int                         `json:"unsupported"`
}

func (h *AccountHandler) ProbeBilling(c *gin.Context) {
	var req AccountBillingProbeRequest
	_ = c.ShouldBindJSON(&req)

	settings, ok := h.requireBillingProbeEnabled(c)
	if !ok {
		return
	}

	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || accountID <= 0 {
		response.BadRequestKey(c, "admin.account.invalid_id", "Invalid account ID")
		return
	}

	started := time.Now()
	result, err := h.probeAccountBilling(c.Request.Context(), accountID, settings)
	if err != nil {
		h.recordAdminAudit(c, "admin.accounts.billing_probe", "account", strconv.FormatInt(accountID, 10), service.AuditStatusFailure, map[string]any{
			"duration_ms": time.Since(started).Milliseconds(),
			"error":       err.Error(),
		})
		response.ErrorFrom(c, err)
		return
	}
	status := service.AuditStatusSuccess
	if !result.Supported || result.Status != "success" {
		status = service.AuditStatusFailure
	}
	h.recordAdminAudit(c, "admin.accounts.billing_probe", "account", strconv.FormatInt(accountID, 10), status, map[string]any{
		"duration_ms": time.Since(started).Milliseconds(),
		"platform":    result.Platform,
		"supported":   result.Supported,
		"status":      result.Status,
	})
	slog.Info("admin_billing_probe_done", "account_id", accountID, "platform", result.Platform, "status", result.Status, "duration_ms", time.Since(started).Milliseconds())
	response.Success(c, result)
}

func (h *AccountHandler) BatchProbeBilling(c *gin.Context) {
	var req BatchAccountBillingProbeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	settings, ok := h.requireBillingProbeEnabled(c)
	if !ok {
		return
	}

	accountIDs, err := h.resolveBillingProbeAccountIDs(c.Request.Context(), req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if len(accountIDs) == 0 {
		response.Success(c, BatchAccountBillingProbeResponse{Items: []AccountBillingProbeResult{}})
		return
	}

	started := time.Now()
	items := h.runBillingProbeBatch(c.Request.Context(), accountIDs, settings)
	out := summarizeBillingProbeBatch(items)
	h.recordAdminAudit(c, "admin.accounts.billing_probe.batch", "account", "", service.AuditStatusSuccess, map[string]any{
		"duration_ms": time.Since(started).Milliseconds(),
		"total":       out.Total,
		"succeeded":   out.Succeeded,
		"failed":      out.Failed,
		"unsupported": out.Unsupported,
	})
	slog.Info("admin_billing_probe_batch_done", "total", out.Total, "succeeded", out.Succeeded, "failed", out.Failed, "unsupported", out.Unsupported, "duration_ms", time.Since(started).Milliseconds())
	response.Success(c, out)
}

func (h *AccountHandler) requireBillingProbeEnabled(c *gin.Context) (*service.UpstreamBillingProbeSettings, bool) {
	settings := service.DefaultUpstreamBillingProbeSettings()
	if h != nil && h.settingService != nil {
		loaded, err := h.settingService.GetUpstreamBillingProbeSettings(c.Request.Context())
		if err != nil {
			response.ErrorFrom(c, err)
			return nil, false
		}
		settings = loaded
	}
	if settings == nil || !settings.Enabled {
		response.ErrorWithDetails(c, http.StatusForbidden, "Upstream billing probe is disabled", "UPSTREAM_BILLING_PROBE_DISABLED", map[string]string{
			"setting": "upstream_billing_probe",
		})
		return nil, false
	}
	return settings, true
}

func (h *AccountHandler) resolveBillingProbeAccountIDs(ctx context.Context, req BatchAccountBillingProbeRequest) ([]int64, error) {
	if len(req.AccountIDs) > 0 {
		return normalizeBillingProbeAccountIDs(req.AccountIDs, billingProbeBatchLimit(req.Limit)), nil
	}
	limit := billingProbeBatchLimit(req.Limit)
	platform := strings.TrimSpace(req.Platform)
	if platform == "" {
		platform = service.PlatformGrok
	}
	lifecycle := strings.TrimSpace(req.Lifecycle)
	if lifecycle == "" {
		lifecycle = service.AccountLifecycleAll
	}
	accounts, _, err := h.adminService.ListAccounts(
		ctx,
		1,
		limit,
		platform,
		strings.TrimSpace(req.Type),
		strings.TrimSpace(req.Status),
		strings.TrimSpace(req.Search),
		req.GroupID,
		lifecycle,
		strings.TrimSpace(req.PrivacyMode),
	)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(accounts))
	for i := range accounts {
		ids = append(ids, accounts[i].ID)
	}
	return ids, nil
}

func normalizeBillingProbeAccountIDs(ids []int64, limit int) []int64 {
	seen := make(map[int64]struct{}, len(ids))
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
		if len(out) >= limit {
			break
		}
	}
	return out
}

func billingProbeBatchLimit(limit int) int {
	if limit <= 0 {
		return 50
	}
	if limit > 100 {
		return 100
	}
	return limit
}

func (h *AccountHandler) runBillingProbeBatch(ctx context.Context, accountIDs []int64, settings *service.UpstreamBillingProbeSettings) []AccountBillingProbeResult {
	concurrency := settings.BatchConcurrency
	if concurrency <= 0 {
		concurrency = 2
	}
	if concurrency > len(accountIDs) {
		concurrency = len(accountIDs)
	}
	items := make([]AccountBillingProbeResult, len(accountIDs))
	jobs := make(chan int)
	var wg sync.WaitGroup
	for worker := 0; worker < concurrency; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobs {
				result, err := h.probeAccountBilling(ctx, accountIDs[idx], settings)
				if err != nil {
					items[idx] = AccountBillingProbeResult{
						AccountID: accountIDs[idx],
						Supported: false,
						Status:    "failed",
						Error:     err.Error(),
					}
					continue
				}
				items[idx] = *result
			}
		}()
	}
	for i := range accountIDs {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	return items
}

func (h *AccountHandler) probeAccountBilling(ctx context.Context, accountID int64, settings *service.UpstreamBillingProbeSettings) (*AccountBillingProbeResult, error) {
	account, err := h.adminService.GetAccount(ctx, accountID)
	if err != nil {
		return nil, err
	}
	result := &AccountBillingProbeResult{
		AccountID:   account.ID,
		AccountName: account.Name,
		Platform:    account.Platform,
		Supported:   false,
		Status:      "unsupported",
	}
	if !strings.EqualFold(strings.TrimSpace(account.Platform), service.PlatformGrok) {
		result.Error = "billing probe is not supported for this platform"
		return result, nil
	}
	if h.grokQuotaService == nil {
		result.Status = "failed"
		result.Error = "grok quota service is not configured"
		return result, nil
	}
	timeout := time.Duration(settings.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	probe, err := h.grokQuotaService.ProbeBilling(probeCtx, account.ID)
	if err != nil {
		result.Status = "failed"
		result.Error = err.Error()
		return result, nil
	}
	result.Supported = true
	result.Status = "success"
	result.Source = probe.Source
	result.FetchedAt = probe.FetchedAt
	result.Persisted = probe.Persisted
	result.Result = probe
	return result, nil
}

func summarizeBillingProbeBatch(items []AccountBillingProbeResult) BatchAccountBillingProbeResponse {
	out := BatchAccountBillingProbeResponse{
		Items: items,
		Total: len(items),
	}
	for i := range items {
		switch {
		case !items[i].Supported && items[i].Status == "unsupported":
			out.Unsupported++
		case items[i].Status == "success":
			out.Succeeded++
		default:
			out.Failed++
		}
	}
	return out
}
