package admin

import (
	"context"
	"fmt"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type CodexSessionImportRequest struct {
	Content                 string         `json:"content"`
	Contents                []string       `json:"contents"`
	Name                    string         `json:"name"`
	Notes                   *string        `json:"notes"`
	GroupIDs                []int64        `json:"group_ids"`
	ProxyID                 *int64         `json:"proxy_id"`
	Concurrency             *int           `json:"concurrency"`
	Priority                *int           `json:"priority"`
	RateMultiplier          *float64       `json:"rate_multiplier"`
	LoadFactor              *int           `json:"load_factor"`
	ExpiresAt               *int64         `json:"expires_at"`
	AutoPauseOnExpired      *bool          `json:"auto_pause_on_expired"`
	CredentialExtras        map[string]any `json:"credential_extras"`
	Extra                   map[string]any `json:"extra"`
	UpdateExisting          *bool          `json:"update_existing"`
	SkipDefaultGroupBind    *bool          `json:"skip_default_group_bind"`
	ConfirmMixedChannelRisk *bool          `json:"confirm_mixed_channel_risk"`
}

type CodexSessionImportResult struct {
	Total    int                         `json:"total"`
	Created  int                         `json:"created"`
	Updated  int                         `json:"updated"`
	Skipped  int                         `json:"skipped"`
	Failed   int                         `json:"failed"`
	Items    []CodexSessionImportItem    `json:"items,omitempty"`
	Warnings []CodexSessionImportMessage `json:"warnings,omitempty"`
	Errors   []CodexSessionImportMessage `json:"errors,omitempty"`
}

type CodexSessionImportItem struct {
	Index     int    `json:"index"`
	Name      string `json:"name,omitempty"`
	Action    string `json:"action"`
	AccountID int64  `json:"account_id,omitempty"`
	Message   string `json:"message,omitempty"`
}

type CodexSessionImportMessage struct {
	Index   int    `json:"index"`
	Name    string `json:"name,omitempty"`
	Message string `json:"message"`
}

func (h *AccountHandler) ImportCodexSession(c *gin.Context) {
	var req CodexSessionImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequestKey(c, "admin.account.invalid_request", "Invalid request: %s", err.Error())
		return
	}
	if err := validateCodexImportRequest(req); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	entries, err := parseCodexSessionImportEntries(req)
	if err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("CODEX_SESSION_IMPORT_PARSE_FAILED", err.Error()))
		return
	}
	if len(entries) == 0 {
		response.BadRequestKey(c, "admin.account.invalid_request", "请输入 accessToken 或 Codex session JSON")
		return
	}

	audit := newCodexSessionImportAudit(c, req, len(entries))
	executeAdminIdempotentJSON(c, "admin.accounts.codex_sessions.import", req, service.DefaultWriteIdempotencyTTL(), func(ctx context.Context) (any, error) {
		audit.logStarted(ctx)
		result, err := h.importCodexSessions(ctx, req, entries)
		audit.logFinished(ctx, result, err)
		return result, err
	})
}

func validateCodexImportRequest(req CodexSessionImportRequest) error {
	if req.Concurrency != nil && *req.Concurrency < 0 {
		return infraerrors.BadRequest("CODEX_SESSION_IMPORT_INVALID_CONCURRENCY", "concurrency must be >= 0")
	}
	if req.Priority != nil && *req.Priority < 0 {
		return infraerrors.BadRequest("CODEX_SESSION_IMPORT_INVALID_PRIORITY", "priority must be >= 0")
	}
	if req.RateMultiplier != nil && *req.RateMultiplier < 0 {
		return infraerrors.BadRequest("CODEX_SESSION_IMPORT_INVALID_RATE_MULTIPLIER", "rate_multiplier must be >= 0")
	}
	if req.LoadFactor != nil && (*req.LoadFactor < 0 || *req.LoadFactor > 10000) {
		return infraerrors.BadRequest("CODEX_SESSION_IMPORT_INVALID_LOAD_FACTOR", "load_factor must be between 0 and 10000")
	}
	return nil
}

func (h *AccountHandler) importCodexSessions(ctx context.Context, req CodexSessionImportRequest, entries []codexImportEntry) (CodexSessionImportResult, error) {
	result := CodexSessionImportResult{Total: len(entries), Items: make([]CodexSessionImportItem, 0, len(entries))}
	existingAccounts, err := h.listCodexImportExistingAccounts(ctx)
	if err != nil {
		return result, err
	}

	index := buildCodexAccountIndex(existingAccounts)
	seen := map[string]codexSeenIdentity{}
	options := newCodexImportOptions(req)
	for _, entry := range entries {
		if err := h.importCodexSessionEntry(ctx, req, options, entry, len(entries), index, seen, &result); err != nil {
			return result, err
		}
	}
	return result, nil
}

func (h *AccountHandler) listCodexImportExistingAccounts(ctx context.Context) ([]service.Account, error) {
	const pageSize = 200
	var out []service.Account
	for page := 1; ; page++ {
		items, total, err := h.adminService.ListAccounts(ctx, page, pageSize, service.PlatformOpenAI, service.AccountTypeOAuth, "", "", 0, service.AccountLifecycleAll, "")
		if err != nil {
			return nil, err
		}
		out = append(out, items...)
		if len(out) >= int(total) || len(items) == 0 {
			return out, nil
		}
	}
}

func (h *AccountHandler) importCodexSessionEntry(ctx context.Context, req CodexSessionImportRequest, options codexImportOptions, entry codexImportEntry, total int, index *codexAccountIndex, seen map[string]codexSeenIdentity, result *CodexSessionImportResult) error {
	item, err := normalizeCodexImportEntry(entry)
	if err != nil {
		appendCodexImportFailure(result, entry.Index, "", err)
		return nil
	}
	name := buildCodexCreateAccountName(req.Name, item, entry.Index, total)
	effectiveExpiresAt, credentialExpiresAt, autoPause, warnings, err := resolveCodexImportExpiry(req, item)
	if err != nil {
		appendCodexImportFailure(result, entry.Index, name, err)
		return nil
	}
	item.WarningTexts = append(item.WarningTexts, warnings...)
	if credentialExpiresAt != nil {
		item.Credentials["expires_at"] = credentialExpiresAt.Format(codexImportTimeLayout)
	}
	for _, warning := range item.WarningTexts {
		result.Warnings = append(result.Warnings, CodexSessionImportMessage{Index: entry.Index, Name: name, Message: warning})
	}
	if duplicate, ok := firstSeenCodexIdentity(seen, item.IdentityKeys, item.UserID); ok {
		message := fmt.Sprintf("与第 %d 条导入项重复，已跳过", duplicate)
		appendCodexImportSkip(result, entry.Index, name, message)
		return nil
	}
	markCodexIdentitySeen(seen, item.IdentityKeys, entry.Index, item.UserID)

	credentials := mergeCodexImportMap(item.Credentials, options.CredentialExtras)
	extra := mergeCodexImportMap(req.Extra, item.Extra)
	existing, matchedKey := index.Find(item.IdentityKeys, item.UserID)
	if existing != nil {
		return h.importCodexExisting(ctx, options, entry.Index, name, item, credentials, extra, existing, matchedKey, effectiveExpiresAt, autoPause, index, result)
	}
	return h.importCodexNew(ctx, options, entry.Index, name, credentials, extra, effectiveExpiresAt, autoPause, index, result)
}

func appendCodexImportFailure(result *CodexSessionImportResult, index int, name string, err error) {
	message := err.Error()
	result.Failed++
	result.Items = append(result.Items, CodexSessionImportItem{Index: index, Name: name, Action: "failed", Message: message})
	result.Errors = append(result.Errors, CodexSessionImportMessage{Index: index, Name: name, Message: message})
}

func appendCodexImportSkip(result *CodexSessionImportResult, index int, name, message string) {
	result.Skipped++
	result.Items = append(result.Items, CodexSessionImportItem{Index: index, Name: name, Action: "skipped", Message: message})
	result.Warnings = append(result.Warnings, CodexSessionImportMessage{Index: index, Name: name, Message: message})
}
