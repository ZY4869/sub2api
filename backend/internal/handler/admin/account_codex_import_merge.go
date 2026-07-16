package admin

import (
	"context"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type codexImportOptions struct {
	CredentialExtras     map[string]any
	UpdateExisting       bool
	Concurrency          int
	Priority             int
	UpdateConcurrency    *int
	UpdatePriority       *int
	SkipDefaultGroupBind bool
	SkipMixedChannel     bool
	ProxyID              *int64
	GroupIDs             []int64
	RateMultiplier       *float64
	LoadFactor           *int
	Notes                *string
}

func newCodexImportOptions(req CodexSessionImportRequest) codexImportOptions {
	options := codexImportOptions{
		CredentialExtras:  sanitizeCodexImportCredentialExtras(req.CredentialExtras),
		UpdateExisting:    true,
		Concurrency:       3,
		Priority:          50,
		UpdateConcurrency: req.Concurrency,
		UpdatePriority:    req.Priority,
		ProxyID:           req.ProxyID,
		GroupIDs:          append([]int64(nil), req.GroupIDs...),
		RateMultiplier:    req.RateMultiplier,
		LoadFactor:        req.LoadFactor,
		Notes:             req.Notes,
		SkipMixedChannel:  req.ConfirmMixedChannelRisk != nil && *req.ConfirmMixedChannelRisk,
	}
	if req.UpdateExisting != nil {
		options.UpdateExisting = *req.UpdateExisting
	}
	if req.Concurrency != nil {
		options.Concurrency = *req.Concurrency
	}
	if req.Priority != nil {
		options.Priority = *req.Priority
	}
	if req.SkipDefaultGroupBind != nil {
		options.SkipDefaultGroupBind = *req.SkipDefaultGroupBind
	}
	return options
}

func (h *AccountHandler) importCodexExisting(ctx context.Context, options codexImportOptions, index int, name string, item *codexImportAccount, credentials, extra map[string]any, existing *service.Account, matchedKey string, expiresAt *int64, autoPause *bool, accountIndex *codexAccountIndex, result *CodexSessionImportResult) error {
	if !options.UpdateExisting {
		appendCodexImportSkip(result, index, name, "已存在匹配账号，按 update_existing=false 跳过")
		return nil
	}
	if strings.HasPrefix(matchedKey, "account:") && item.UserID != "" &&
		codexCredentialString(existing.Credentials, "chatgpt_user_id") == "" {
		result.Warnings = append(result.Warnings, CodexSessionImportMessage{Index: index, Name: name, Message: "已有账号未记录 chatgpt_user_id，已按共享的 chatgpt_account_id 匹配并回填，请确认两者属于同一用户"})
	}
	if item.RefreshToken == "" && codexCredentialString(existing.Credentials, "refresh_token") != "" {
		result.Warnings = append(result.Warnings, CodexSessionImportMessage{Index: index, Name: name, Message: "已有账号包含 refresh_token，本次 accessToken-only 导入已保留自动续期凭据"})
		expiresAt = nil
		autoPause = nil
	}

	mergedCredentials := mergeCodexImportCredentials(existing.Credentials, credentials, item)
	mergedExtra := mergeCodexImportMap(existing.Extra, extra)
	input := &service.UpdateAccountInput{
		Credentials:        mergedCredentials,
		Extra:              mergedExtra,
		Concurrency:        options.UpdateConcurrency,
		Priority:           options.UpdatePriority,
		RateMultiplier:     options.RateMultiplier,
		LoadFactor:         options.LoadFactor,
		ExpiresAt:          expiresAt,
		AutoPauseOnExpired: autoPause,
	}
	if options.ProxyID != nil {
		input.ProxyID = options.ProxyID
	}
	if len(options.GroupIDs) > 0 {
		groupIDs := append([]int64(nil), options.GroupIDs...)
		input.GroupIDs = &groupIDs
		input.SkipMixedChannelCheck = options.SkipMixedChannel
	}
	updated, err := h.adminService.UpdateAccount(ctx, existing.ID, input)
	if err != nil {
		appendCodexImportFailure(result, index, name, err)
		return nil
	}
	h.invalidateCodexImportTokenCache(ctx, updated)
	accountID := existing.ID
	if updated != nil {
		accountID = updated.ID
		accountIndex.Add(*updated)
	}
	result.Updated++
	result.Items = append(result.Items, CodexSessionImportItem{Index: index, Name: name, Action: "updated", AccountID: accountID})
	return nil
}

func (h *AccountHandler) importCodexNew(ctx context.Context, options codexImportOptions, index int, name string, credentials, extra map[string]any, expiresAt *int64, autoPause *bool, accountIndex *codexAccountIndex, result *CodexSessionImportResult) error {
	account, err := h.adminService.CreateAccount(ctx, &service.CreateAccountInput{
		Name:                  name,
		Notes:                 options.Notes,
		Platform:              service.PlatformOpenAI,
		Type:                  service.AccountTypeOAuth,
		Credentials:           credentials,
		Extra:                 extra,
		ProxyID:               options.ProxyID,
		Concurrency:           options.Concurrency,
		Priority:              options.Priority,
		RateMultiplier:        options.RateMultiplier,
		LoadFactor:            options.LoadFactor,
		GroupIDs:              options.GroupIDs,
		ExpiresAt:             expiresAt,
		AutoPauseOnExpired:    autoPause,
		SkipDefaultGroupBind:  options.SkipDefaultGroupBind,
		SkipMixedChannelCheck: options.SkipMixedChannel,
	})
	if err != nil {
		appendCodexImportFailure(result, index, name, err)
		return nil
	}
	accountID := int64(0)
	if account != nil {
		accountID = account.ID
		accountIndex.Add(*account)
	}
	result.Created++
	result.Items = append(result.Items, CodexSessionImportItem{Index: index, Name: name, Action: "created", AccountID: accountID})
	return nil
}

func sanitizeCodexImportCredentialExtras(input map[string]any) map[string]any {
	if len(input) == 0 {
		return nil
	}
	protected := map[string]struct{}{
		"access_token": {}, "refresh_token": {}, "id_token": {}, "expires_at": {},
		"email": {}, "chatgpt_account_id": {}, "chatgpt_user_id": {}, "organization_id": {},
		"plan_type": {}, "client_id": {}, "auth_mode": {}, "openai_auth_mode": {},
		"token_type": {}, "chatgpt_account_is_fedramp": {},
		"agent_runtime_id": {}, "agent_private_key": {}, "task_id": {},
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		normalizedKey := strings.TrimSpace(key)
		if normalizedKey == "" {
			continue
		}
		if _, ok := protected[strings.ToLower(normalizedKey)]; ok {
			continue
		}
		out[normalizedKey] = value
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func mergeCodexImportMap(existing, incoming map[string]any) map[string]any {
	out := make(map[string]any, len(existing)+len(incoming))
	for key, value := range existing {
		out[key] = value
	}
	for key, value := range incoming {
		out[key] = value
	}
	return out
}

func mergeCodexImportCredentials(existing, incoming map[string]any, item *codexImportAccount) map[string]any {
	out := mergeCodexImportMap(existing, incoming)
	if item == nil {
		return out
	}
	if strings.TrimSpace(item.RefreshToken) == "" {
		if codexCredentialString(existing, "refresh_token") == "" {
			delete(out, "refresh_token")
			delete(out, "client_id")
		} else {
			out["refresh_token"] = existing["refresh_token"]
			if clientID, ok := existing["client_id"]; ok {
				out["client_id"] = clientID
			}
		}
	}
	if strings.TrimSpace(item.IDToken) == "" {
		delete(out, "id_token")
	}
	if item.IsAgentIdentity {
		delete(out, "access_token")
		delete(out, "refresh_token")
		delete(out, "id_token")
		delete(out, "expires_at")
		delete(out, "client_id")
	}
	return out
}
