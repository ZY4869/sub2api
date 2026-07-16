package service

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/xai"
	"golang.org/x/sync/singleflight"
)

const (
	grokQuotaUpstreamTimeout = 20 * time.Second
	grokQuotaProbeInput      = "."
)

type GrokQuotaProbeResult struct {
	Source            string              `json:"source"`
	Model             string              `json:"model,omitempty"`
	Billing           *xai.BillingSummary `json:"billing,omitempty"`
	Snapshot          *xai.QuotaSnapshot  `json:"snapshot,omitempty"`
	LocalUsage24h     *WindowStats        `json:"local_usage_24h,omitempty"`
	LocalUsage7d      *WindowStats        `json:"local_usage_7d,omitempty"`
	LocalUsageMonthly *WindowStats        `json:"local_usage_monthly,omitempty"`
	StatusCode        int                 `json:"status_code,omitempty"`
	HeadersObserved   bool                `json:"headers_observed"`
	ResetSupported    bool                `json:"reset_supported"`
	FetchedAt         int64               `json:"fetched_at"`
	Persisted         bool                `json:"persisted"`
	ProbeError        string              `json:"probe_error,omitempty"`
}

type GrokQuotaResetResult struct {
	Supported bool   `json:"supported"`
	Code      string `json:"code"`
	Message   string `json:"message"`
}

type GrokQuotaService struct {
	accountRepo   AccountRepository
	proxyRepo     ProxyRepository
	tokenProvider *GrokTokenProvider
	httpUpstream  HTTPUpstream
	usageLogRepo  UsageLogRepository
	probeFlight   singleflight.Group
}

func NewGrokQuotaService(
	accountRepo AccountRepository,
	proxyRepo ProxyRepository,
	tokenProvider *GrokTokenProvider,
	httpUpstream HTTPUpstream,
	usageLogRepo UsageLogRepository,
) *GrokQuotaService {
	return &GrokQuotaService{
		accountRepo:   accountRepo,
		proxyRepo:     proxyRepo,
		tokenProvider: tokenProvider,
		httpUpstream:  httpUpstream,
		usageLogRepo:  usageLogRepo,
	}
}

func (s *GrokQuotaService) QueryQuota(ctx context.Context, accountID int64) (*GrokQuotaProbeResult, error) {
	billingResult, billingErr := s.ProbeBilling(ctx, accountID)
	if billingErr == nil && billingResult != nil && grokBillingHasAuthoritativeQuota(billingResult.Billing) {
		return billingResult, nil
	}
	probeResult, probeErr := s.ProbeUsage(ctx, accountID)
	if probeErr != nil {
		if billingResult != nil && billingResult.Billing != nil {
			billingResult.ProbeError = probeErr.Error()
			return billingResult, nil
		}
		return nil, probeErr
	}
	if probeResult == nil {
		if billingErr != nil {
			return nil, billingErr
		}
		return nil, infraerrors.New(http.StatusBadGateway, "GROK_QUOTA_PROBE_EMPTY", "Grok quota probe returned no result")
	}
	if billingResult != nil {
		probeResult.Source = "hybrid_probe"
		probeResult.Billing = billingResult.Billing
		probeResult.LocalUsage24h = billingResult.LocalUsage24h
		probeResult.LocalUsage7d = billingResult.LocalUsage7d
		probeResult.LocalUsageMonthly = billingResult.LocalUsageMonthly
		probeResult.Persisted = probeResult.Persisted || billingResult.Persisted
	}
	return probeResult, nil
}

func grokBillingHasAuthoritativeQuota(billing *xai.BillingSummary) bool {
	if billing == nil {
		return false
	}
	return billing.UsagePercent != nil ||
		billing.UsedPercent != nil ||
		(billing.MonthlyLimitCents != nil && *billing.MonthlyLimitCents > 0) ||
		strings.TrimSpace(billing.Plan) != ""
}

func (s *GrokQuotaService) ProbeUsage(ctx context.Context, accountID int64) (*GrokQuotaProbeResult, error) {
	return s.runProbeFlight(ctx, "active:"+strconv.FormatInt(accountID, 10), func(sharedCtx context.Context) (*GrokQuotaProbeResult, error) {
		return s.probeUsage(sharedCtx, accountID)
	})
}

func (s *GrokQuotaService) probeUsage(ctx context.Context, accountID int64) (*GrokQuotaProbeResult, error) {
	account, token, proxyURL, err := s.prepareProbe(ctx, accountID)
	if err != nil {
		return nil, err
	}
	probeModel := firstNonEmptyString(DefaultGrokBuildTextModelID(), "grok-4.5")
	body, err := json.Marshal(map[string]any{
		"model":             probeModel,
		"input":             grokQuotaProbeInput,
		"max_output_tokens": 1,
		"store":             false,
	})
	if err != nil {
		return nil, infraerrors.New(http.StatusBadRequest, "GROK_QUOTA_PROBE_BODY_ERROR", "failed to build probe body").WithCause(err)
	}
	targetURL, err := grokJoinVersionedEndpoint(defaultGrokCLIBaseURL, grokEndpointResponses)
	if err != nil {
		return nil, infraerrors.New(http.StatusBadRequest, "GROK_QUOTA_BASE_URL_INVALID", "invalid Grok base_url").WithCause(err)
	}
	callCtx, cancel := context.WithTimeout(ctx, grokQuotaUpstreamTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(callCtx, http.MethodPost, targetURL, bytes.NewReader(body))
	if err != nil {
		return nil, infraerrors.New(http.StatusInternalServerError, "GROK_QUOTA_PROBE_REQUEST_BUILD_FAILED", "failed to build upstream request").WithCause(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	applyGrokCLIHeaders(req.Header)
	resp, err := s.httpUpstream.Do(req, proxyURL, account.ID, maxInt(account.Concurrency, 1))
	if err != nil {
		return nil, infraerrors.New(http.StatusBadGateway, "GROK_QUOTA_PROBE_REQUEST_FAILED", "upstream probe failed").WithCause(err)
	}
	defer func() { _ = resp.Body.Close() }()

	snapshot := xai.ObserveQuotaHeaders(resp.Header, resp.StatusCode, "active_probe")
	resetAt, limited := grokRateLimitResetAtForAccount(account, snapshot, time.Now())
	if limited {
		normalizeGrokExhaustedWindowResets(snapshot, resetAt, time.Now())
	}
	persistErr := s.accountRepo.UpdateExtra(ctx, account.ID, map[string]any{grokQuotaSnapshotExtraKey: snapshot})
	if limited {
		persistGrokRateLimit(ctx, s.accountRepo, account, resetAt)
	} else if isSuccessfulGrokRateLimitRecovery(account, snapshot) {
		clearGrokRateLimitAfterRecovery(ctx, s.accountRepo, account)
	}
	result := &GrokQuotaProbeResult{
		Source:          "active_probe",
		Model:           probeModel,
		Snapshot:        snapshot,
		StatusCode:      resp.StatusCode,
		HeadersObserved: snapshot.HeadersObserved,
		ResetSupported:  false,
		FetchedAt:       time.Now().Unix(),
		Persisted:       persistErr == nil,
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return result, nil
	}
	if resp.StatusCode >= 400 {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4<<10))
		slog.Warn("grok_quota_probe_failed", "account_id", account.ID, "status", resp.StatusCode)
		return nil, infraerrors.Newf(grokQuotaHTTPStatus(resp.StatusCode), "GROK_QUOTA_PROBE_UPSTREAM_ERROR", "upstream returned %d for Grok quota probe", resp.StatusCode)
	}
	return result, nil
}

func (s *GrokQuotaService) ProbeBilling(ctx context.Context, accountID int64) (*GrokQuotaProbeResult, error) {
	return s.runProbeFlight(ctx, "billing:"+strconv.FormatInt(accountID, 10), func(sharedCtx context.Context) (*GrokQuotaProbeResult, error) {
		return s.probeBilling(sharedCtx, accountID)
	})
}

func (s *GrokQuotaService) probeBilling(ctx context.Context, accountID int64) (*GrokQuotaProbeResult, error) {
	account, token, proxyURL, err := s.prepareProbe(ctx, accountID)
	if err != nil {
		return nil, err
	}
	probeCtx, cancel := context.WithTimeout(ctx, grokQuotaUpstreamTimeout)
	defer cancel()
	type billingResult struct {
		summary *xai.BillingSummary
		status  int
		err     error
	}
	var weekly, monthly billingResult
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		weekly.summary, weekly.status, weekly.err = s.fetchBilling(probeCtx, account, token, proxyURL, true)
	}()
	go func() {
		defer wg.Done()
		monthly.summary, monthly.status, monthly.err = s.fetchBilling(probeCtx, account, token, proxyURL, false)
	}()
	wg.Wait()

	weeklyOK := weekly.summary != nil
	monthlyOK := monthly.summary != nil
	if !weeklyOK && !monthlyOK {
		return nil, mergeGrokBillingProbeErrors(weekly.status, monthly.status, weekly.err, monthly.err)
	}
	statusCode := preferSuccessfulBillingStatus(weekly.status, monthly.status, weeklyOK, monthlyOK)
	previous, _ := grokBillingSnapshotFromExtra(account.Extra)
	billing := xai.MergeBillingProbeResult(previous, weekly.summary, monthly.summary, weeklyOK, monthlyOK)
	billing = xai.StampBillingSummary(billing, statusCode, "billing_probe")
	persistErr := s.accountRepo.UpdateExtra(ctx, account.ID, map[string]any{grokBillingExtraKey: billing})
	if persistErr != nil {
		slog.Warn("grok_billing_persist_failed", "account_id", account.ID, "error", persistErr)
	}
	now := time.Now().UTC()
	localUsage24h, localUsage7d, localUsageMonthly := grokLocalUsageForQuota(ctx, s.usageLogRepo, account.ID, billing, now)
	return &GrokQuotaProbeResult{
		Source:            "billing_probe",
		Billing:           billing,
		LocalUsage24h:     localUsage24h,
		LocalUsage7d:      localUsage7d,
		LocalUsageMonthly: localUsageMonthly,
		StatusCode:        statusCode,
		FetchedAt:         now.Unix(),
		Persisted:         persistErr == nil,
	}, nil
}

func (s *GrokQuotaService) runProbeFlight(ctx context.Context, key string, probe func(context.Context) (*GrokQuotaProbeResult, error)) (*GrokQuotaProbeResult, error) {
	if s == nil {
		return nil, infraerrors.New(http.StatusInternalServerError, "GROK_QUOTA_NOT_CONFIGURED", "grok quota service is not configured")
	}
	resultCh := s.probeFlight.DoChan(key, func() (any, error) {
		sharedCtx, cancel := context.WithTimeout(context.Background(), grokQuotaUpstreamTimeout+5*time.Second)
		defer cancel()
		return probe(sharedCtx)
	})
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case flightResult := <-resultCh:
		if flightResult.Err != nil {
			return nil, flightResult.Err
		}
		result, ok := flightResult.Val.(*GrokQuotaProbeResult)
		if !ok || result == nil {
			return nil, infraerrors.New(http.StatusInternalServerError, "GROK_QUOTA_PROBE_RESULT_INVALID", "invalid Grok quota probe result")
		}
		cloned := *result
		return &cloned, nil
	}
}

func (s *GrokQuotaService) fetchBilling(ctx context.Context, account *Account, token string, proxyURL string, weekly bool) (*xai.BillingSummary, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, xai.BuildBillingURL(weekly), nil)
	if err != nil {
		return nil, 0, infraerrors.New(http.StatusInternalServerError, "GROK_QUOTA_PROBE_REQUEST_BUILD_FAILED", "failed to build billing request").WithCause(err)
	}
	xai.ApplyCLIBillingHeaders(req, token)
	resp, err := s.httpUpstream.Do(req, proxyURL, account.ID, maxInt(account.Concurrency, 2))
	if err != nil {
		return nil, 0, infraerrors.New(http.StatusBadGateway, "GROK_QUOTA_PROBE_REQUEST_FAILED", "billing request failed").WithCause(err)
	}
	defer func() { _ = resp.Body.Close() }()
	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, resp.StatusCode, nil
	}
	if resp.StatusCode >= 400 {
		slog.Warn("grok_quota_billing_failed", "account_id", account.ID, "weekly", weekly, "status", resp.StatusCode)
		return nil, resp.StatusCode, infraerrors.Newf(grokQuotaHTTPStatus(resp.StatusCode), "GROK_QUOTA_PROBE_UPSTREAM_ERROR", "billing returned %d", resp.StatusCode)
	}
	payload, err := xai.ParseBillingPayload(bodyBytes)
	if err != nil {
		return nil, resp.StatusCode, infraerrors.New(http.StatusBadGateway, "GROK_QUOTA_BILLING_PARSE_ERROR", "failed to parse billing body").WithCause(err)
	}
	return xai.BuildBillingSummary(payload.Config), resp.StatusCode, nil
}

func mergeGrokBillingProbeErrors(weeklyStatus, monthlyStatus int, weeklyErr, monthlyErr error) error {
	if weeklyErr != nil {
		return weeklyErr
	}
	if monthlyErr != nil {
		return monthlyErr
	}
	if weeklyStatus == http.StatusTooManyRequests || monthlyStatus == http.StatusTooManyRequests {
		return infraerrors.New(http.StatusTooManyRequests, "GROK_QUOTA_PROBE_UPSTREAM_ERROR", "billing rate limited")
	}
	if weeklyStatus != 0 || monthlyStatus != 0 {
		return infraerrors.New(http.StatusBadGateway, "GROK_QUOTA_PROBE_UPSTREAM_ERROR", "xAI billing endpoints returned no quota data")
	}
	return infraerrors.New(http.StatusBadGateway, "GROK_QUOTA_BILLING_EMPTY", "xAI billing endpoints returned no quota data")
}

func preferSuccessfulBillingStatus(weeklyStatus, monthlyStatus int, weeklyOK, monthlyOK bool) int {
	if weeklyOK && weeklyStatus >= 200 && weeklyStatus < 300 {
		return weeklyStatus
	}
	if monthlyOK && monthlyStatus >= 200 && monthlyStatus < 300 {
		return monthlyStatus
	}
	if weeklyStatus != 0 {
		return weeklyStatus
	}
	return monthlyStatus
}

func (s *GrokQuotaService) ResetQuota(ctx context.Context, accountID int64) (*GrokQuotaResetResult, error) {
	if _, err := s.loadGrokOAuthAccount(ctx, accountID); err != nil {
		return nil, err
	}
	return &GrokQuotaResetResult{
		Supported: false,
		Code:      "GROK_QUOTA_RESET_UNSUPPORTED",
		Message:   "xAI does not expose a Grok subscription quota reset endpoint for OAuth accounts",
	}, nil
}

func (s *GrokQuotaService) prepareProbe(ctx context.Context, accountID int64) (*Account, string, string, error) {
	if s == nil || s.tokenProvider == nil || s.httpUpstream == nil {
		return nil, "", "", infraerrors.New(http.StatusInternalServerError, "GROK_QUOTA_NOT_CONFIGURED", "grok quota service is not configured")
	}
	account, err := s.loadGrokOAuthAccount(ctx, accountID)
	if err != nil {
		return nil, "", "", err
	}
	token, err := s.tokenProvider.GetAccessToken(ctx, account)
	if err != nil {
		return nil, "", "", infraerrors.New(http.StatusBadGateway, "GROK_QUOTA_TOKEN_UNAVAILABLE", "failed to acquire access token").WithCause(err)
	}
	if strings.TrimSpace(token) == "" {
		return nil, "", "", infraerrors.New(http.StatusBadGateway, "GROK_QUOTA_TOKEN_UNAVAILABLE", "access token is empty")
	}
	return account, token, s.resolveProxyURL(ctx, account), nil
}

func (s *GrokQuotaService) resolveProxyURL(ctx context.Context, account *Account) string {
	if account == nil || account.ProxyID == nil {
		return ""
	}
	if account.Proxy != nil {
		return account.Proxy.URL()
	}
	if s != nil && s.proxyRepo != nil {
		if proxy, err := s.proxyRepo.GetByID(ctx, *account.ProxyID); err == nil && proxy != nil {
			account.Proxy = proxy
			return proxy.URL()
		}
	}
	return ""
}

func (s *GrokQuotaService) loadGrokOAuthAccount(ctx context.Context, accountID int64) (*Account, error) {
	if s == nil || s.accountRepo == nil {
		return nil, infraerrors.New(http.StatusInternalServerError, "GROK_QUOTA_NOT_CONFIGURED", "grok quota service is not configured")
	}
	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, infraerrors.New(http.StatusNotFound, "GROK_QUOTA_ACCOUNT_NOT_FOUND", "account not found").WithCause(err)
	}
	if account == nil {
		return nil, infraerrors.New(http.StatusNotFound, "GROK_QUOTA_ACCOUNT_NOT_FOUND", "account not found")
	}
	if account.Platform != PlatformGrok {
		return nil, infraerrors.New(http.StatusBadRequest, "GROK_QUOTA_INVALID_PLATFORM", "account is not a Grok account")
	}
	if account.Type != AccountTypeOAuth {
		return nil, infraerrors.New(http.StatusBadRequest, "GROK_QUOTA_INVALID_TYPE", "account is not an OAuth account")
	}
	return account, nil
}

func (s *GrokGatewayService) observeGrokQuotaHeaders(ctx context.Context, account *Account, headers http.Header, statusCode int, source string) {
	if s == nil || s.accountRepo == nil || account == nil || !account.IsGrokOAuth() {
		return
	}
	snapshot := xai.ObserveQuotaHeaders(headers, statusCode, source)
	if snapshot == nil || (!snapshot.HasObservedHeaders() && statusCode < http.StatusBadRequest) {
		return
	}
	if resetAt, limited := grokRateLimitResetAtForAccount(account, snapshot, time.Now()); limited {
		normalizeGrokExhaustedWindowResets(snapshot, resetAt, time.Now())
		persistGrokRateLimit(ctx, s.accountRepo, account, resetAt)
	} else if isSuccessfulGrokRateLimitRecovery(account, snapshot) {
		clearGrokRateLimitAfterRecovery(ctx, s.accountRepo, account)
	}
	if err := s.accountRepo.UpdateExtra(ctx, account.ID, map[string]any{grokQuotaSnapshotExtraKey: snapshot}); err != nil {
		slog.Warn("grok_quota_passive_persist_failed", "account_id", account.ID, "status", statusCode, "error", err)
	}
}

func grokQuotaHTTPStatus(statusCode int) int {
	switch statusCode {
	case http.StatusUnauthorized, http.StatusForbidden, http.StatusTooManyRequests:
		return statusCode
	default:
		if statusCode >= http.StatusInternalServerError {
			return http.StatusBadGateway
		}
		return http.StatusBadGateway
	}
}
