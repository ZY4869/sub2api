package service

import (
	"context"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

const (
	defaultGrokOAuthReconcilePageSize = 50
	maxGrokOAuthReconcilePageSize     = 500
	maxGrokOAuthReconcileWindow       = 24 * time.Hour

	GrokOAuthReconcileReasonMissingRefreshToken = "missing_refresh_token"
	GrokOAuthReconcileReasonMissingAccessToken  = "missing_access_token"
	GrokOAuthReconcileReasonMissingExpiry       = "missing_expiry"
	GrokOAuthReconcileReasonInvalidExpiry       = "invalid_expiry"
	GrokOAuthReconcileReasonNearExpiry          = "near_expiry"

	GrokOAuthReconcileActionBlock   = "block_account"
	GrokOAuthReconcileActionRefresh = "refresh_credentials"

	GrokOAuthReconcileOutcomePlanned = "planned"
	GrokOAuthReconcileOutcomeApplied = "applied"
	GrokOAuthReconcileOutcomeSkipped = "skipped"
	GrokOAuthReconcileOutcomeFailed  = "failed"
)

var (
	ErrGrokOAuthReconcileMode   = infraerrors.BadRequest("GROK_OAUTH_RECONCILE_MODE_INVALID", "apply requires dry_run=false and apply=true")
	ErrGrokOAuthReconcileLimit  = infraerrors.BadRequest("GROK_OAUTH_RECONCILE_LIMIT_INVALID", "limit is outside the allowed reconciliation page range")
	ErrGrokOAuthReconcileWindow = infraerrors.BadRequest("GROK_OAUTH_RECONCILE_WINDOW_INVALID", "refresh_window_seconds is outside the allowed range")
)

type GrokOAuthReconciler interface {
	ReconcileGrokOAuth(ctx context.Context, input GrokOAuthReconcileInput) (*GrokOAuthReconcileResult, error)
}

type GrokOAuthConditionalErrorRepository interface {
	SetGrokOAuthErrorIfCredentialsUnchanged(ctx context.Context, id int64, expectedCredentials map[string]any, errorMsg string) (bool, error)
}

type GrokOAuthConditionalCredentialsRepository interface {
	UpdateGrokOAuthCredentialsIfCredentialsUnchanged(ctx context.Context, id int64, expectedCredentials map[string]any, credentials map[string]any) (bool, error)
}

type GrokOAuthReconcileAccountLister interface {
	ListActiveGrokOAuthAfterID(ctx context.Context, afterID int64, limit int) ([]Account, error)
}

type GrokOAuthReconcileInput struct {
	DryRun        bool
	Apply         bool
	AfterID       int64
	Limit         int
	RefreshWindow time.Duration
}

type GrokOAuthReconcileItem struct {
	AccountID int64  `json:"account_id"`
	Reason    string `json:"reason"`
	Action    string `json:"action"`
	Outcome   string `json:"outcome"`
}

type GrokOAuthReconcileResult struct {
	DryRun       bool                     `json:"dry_run"`
	Scanned      int                      `json:"scanned"`
	Actionable   int                      `json:"actionable"`
	WouldBlock   int                      `json:"would_block"`
	WouldRefresh int                      `json:"would_refresh"`
	Blocked      int                      `json:"blocked"`
	Refreshed    int                      `json:"refreshed"`
	Skipped      int                      `json:"skipped"`
	Failed       int                      `json:"failed"`
	Items        []GrokOAuthReconcileItem `json:"items"`
	NextAfterID  int64                    `json:"next_after_id"`
	HasMore      bool                     `json:"has_more"`
}

func (s *TokenRefreshService) ReconcileGrokOAuth(ctx context.Context, input GrokOAuthReconcileInput) (*GrokOAuthReconcileResult, error) {
	if input.Apply && input.DryRun {
		return nil, ErrGrokOAuthReconcileMode
	}
	limit := input.Limit
	if limit == 0 {
		limit = defaultGrokOAuthReconcilePageSize
	}
	if limit < 1 || limit > maxGrokOAuthReconcilePageSize {
		return nil, ErrGrokOAuthReconcileLimit
	}
	refreshWindow := input.RefreshWindow
	if refreshWindow == 0 {
		refreshWindow = grokTokenRefreshSkew
	}
	if refreshWindow < 0 || refreshWindow > maxGrokOAuthReconcileWindow {
		return nil, ErrGrokOAuthReconcileWindow
	}
	if refreshWindow < grokTokenRefreshSkew {
		refreshWindow = grokTokenRefreshSkew
	}
	dryRun := !input.Apply
	accounts, err := s.listGrokOAuthReconcileAccounts(ctx, input.AfterID, limit)
	if err != nil {
		return nil, err
	}
	result := &GrokOAuthReconcileResult{
		DryRun: dryRun,
		Items:  make([]GrokOAuthReconcileItem, 0, len(accounts)),
	}
	conditionalRepo, _ := s.accountRepo.(GrokOAuthConditionalErrorRepository)
	grokRefresher := NewGrokTokenRefresher(s.grokOAuthService())
	for i := range accounts {
		account := &accounts[i]
		result.Scanned++
		result.NextAfterID = account.ID
		if !dryRun {
			if _, err := EnsureGrokBuildModelScopePersisted(ctx, s.accountRepo, account, nil, time.Now().UTC()); err != nil {
				return nil, err
			}
		}
		reason, action, actionable := classifyGrokOAuthReconcileAccount(account, refreshWindow)
		if !actionable {
			result.Skipped++
			continue
		}
		result.Actionable++
		item := GrokOAuthReconcileItem{
			AccountID: account.ID,
			Reason:    reason,
			Action:    action,
			Outcome:   GrokOAuthReconcileOutcomePlanned,
		}
		if action == GrokOAuthReconcileActionBlock {
			result.WouldBlock++
		} else {
			result.WouldRefresh++
		}
		if dryRun {
			result.Items = append(result.Items, item)
			continue
		}
		switch action {
		case GrokOAuthReconcileActionBlock:
			if conditionalRepo == nil {
				item.Outcome = GrokOAuthReconcileOutcomeFailed
				result.Failed++
				break
			}
			applied, err := conditionalRepo.SetGrokOAuthErrorIfCredentialsUnchanged(ctx, account.ID, account.Credentials, "Grok OAuth credential reconciliation: missing refresh token")
			if err != nil {
				item.Outcome = GrokOAuthReconcileOutcomeFailed
				result.Failed++
				break
			}
			if !applied {
				item.Outcome = GrokOAuthReconcileOutcomeSkipped
				result.Skipped++
				break
			}
			if s.cacheInvalidator != nil {
				_ = s.cacheInvalidator.InvalidateToken(ctx, account)
			}
			item.Outcome = GrokOAuthReconcileOutcomeApplied
			result.Blocked++
		case GrokOAuthReconcileActionRefresh:
			if s.refreshAPI == nil {
				item.Outcome = GrokOAuthReconcileOutcomeFailed
				result.Failed++
				break
			}
			refreshResult, err := s.refreshAPI.RefreshIfNeededWithExpectedCredentials(ctx, account, grokRefresher, refreshWindow, account.Credentials)
			if err != nil {
				item.Outcome = GrokOAuthReconcileOutcomeFailed
				result.Failed++
				break
			}
			if refreshResult == nil || refreshResult.CASMiss || refreshResult.LockHeld || !refreshResult.Refreshed {
				item.Outcome = GrokOAuthReconcileOutcomeSkipped
				result.Skipped++
				break
			}
			item.Outcome = GrokOAuthReconcileOutcomeApplied
			result.Refreshed++
		}
		result.Items = append(result.Items, item)
	}
	result.HasMore = len(accounts) == limit
	return result, nil
}

func (s *TokenRefreshService) listGrokOAuthReconcileAccounts(ctx context.Context, afterID int64, limit int) ([]Account, error) {
	if lister, ok := s.accountRepo.(GrokOAuthReconcileAccountLister); ok && lister != nil {
		return lister.ListActiveGrokOAuthAfterID(ctx, afterID, limit)
	}
	accounts, _, err := s.accountRepo.ListWithFilters(
		ctx,
		pagination.PaginationParams{Page: 1, PageSize: limit},
		PlatformGrok,
		AccountTypeOAuth,
		StatusActive,
		"",
		0,
		AccountLifecycleNormal,
		"",
	)
	if err != nil {
		return nil, err
	}
	filtered := make([]Account, 0, len(accounts))
	for i := len(accounts) - 1; i >= 0; i-- {
		if accounts[i].ID > afterID {
			filtered = append(filtered, accounts[i])
		}
	}
	return filtered, nil
}

func (s *TokenRefreshService) grokOAuthService() *GrokOAuthService {
	for _, refresher := range s.refreshers {
		if grokRefresher, ok := refresher.(*GrokTokenRefresher); ok {
			return grokRefresher.grokOAuthService
		}
	}
	return nil
}

func classifyGrokOAuthReconcileAccount(account *Account, refreshWindow time.Duration) (reason, action string, actionable bool) {
	if account == nil || !account.IsGrokOAuth() || account.Status != StatusActive {
		return "", "", false
	}
	if strings.TrimSpace(account.GetCredential("refresh_token")) == "" {
		return GrokOAuthReconcileReasonMissingRefreshToken, GrokOAuthReconcileActionBlock, true
	}
	if strings.TrimSpace(account.GetGrokOAuthAccessToken()) == "" {
		return GrokOAuthReconcileReasonMissingAccessToken, GrokOAuthReconcileActionRefresh, true
	}
	rawExpiry := strings.TrimSpace(account.GetCredential("expires_at"))
	if rawExpiry == "" {
		return GrokOAuthReconcileReasonMissingExpiry, GrokOAuthReconcileActionRefresh, true
	}
	expiresAt := account.GetCredentialAsTime("expires_at")
	if expiresAt == nil {
		return GrokOAuthReconcileReasonInvalidExpiry, GrokOAuthReconcileActionRefresh, true
	}
	if time.Until(*expiresAt) <= refreshWindow {
		return GrokOAuthReconcileReasonNearExpiry, GrokOAuthReconcileActionRefresh, true
	}
	return "", "", false
}

func ParseGrokOAuthReconcileRefreshWindowSeconds(raw string) (time.Duration, error) {
	if strings.TrimSpace(raw) == "" {
		return 0, nil
	}
	seconds, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return 0, ErrGrokOAuthReconcileWindow
	}
	return time.Duration(seconds) * time.Second, nil
}
