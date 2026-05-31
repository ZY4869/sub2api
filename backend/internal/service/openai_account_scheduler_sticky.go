package service

import (
	"context"
	"log/slog"
	"strings"
)

func (s *defaultOpenAIAccountScheduler) selectBySessionHash(
	ctx context.Context,
	req OpenAIAccountScheduleRequest,
) (*AccountSelectionResult, *AccountSelectionResult, error) {
	sessionHash := strings.TrimSpace(req.SessionHash)
	if sessionHash == "" || s == nil || s.service == nil || s.service.cache == nil {
		return nil, nil, nil
	}

	accountID := req.StickyAccountID
	if accountID <= 0 {
		var err error
		accountID, err = s.service.getStickySessionAccountID(ctx, req.GroupID, sessionHash)
		if err != nil || accountID <= 0 {
			return nil, nil, nil
		}
	}
	if accountID <= 0 {
		return nil, nil, nil
	}
	if req.ExcludedIDs != nil {
		if _, excluded := req.ExcludedIDs[accountID]; excluded {
			return nil, nil, nil
		}
	}

	account, err := s.service.getSchedulableAccount(ctx, accountID)
	if err != nil {
		if isContextDoneError(ctx, err) {
			return nil, nil, err
		}
		_ = s.service.deleteStickySessionAccountID(ctx, req.GroupID, sessionHash)
		return nil, nil, nil
	}
	if account == nil {
		_ = s.service.deleteStickySessionAccountID(ctx, req.GroupID, sessionHash)
		return nil, nil, nil
	}
	if shouldClearStickySession(account, req.RequestedModel) || !isOpenAITextRuntimeAccount(account) || !account.IsSchedulable() {
		_ = s.service.deleteStickySessionAccountID(ctx, req.GroupID, sessionHash)
		return nil, nil, nil
	}
	if req.RequestedModel != "" && !s.service.isModelSupportedByAccountWithContext(ctx, account, req.RequestedModel) {
		return nil, nil, nil
	}
	if !account.IsSchedulableForModelWithContext(ctx, req.RequestedModel) {
		_ = s.service.deleteStickySessionAccountID(ctx, req.GroupID, sessionHash)
		return nil, nil, nil
	}
	if !s.isAccountTransportCompatible(account, req.RequiredTransport) {
		_ = s.service.deleteStickySessionAccountID(ctx, req.GroupID, sessionHash)
		return nil, nil, nil
	}
	if !s.isAccountEndpointCapabilityCompatible(account, req.RequiredCapability) {
		_ = s.service.deleteStickySessionAccountID(ctx, req.GroupID, sessionHash)
		return nil, nil, nil
	}
	account = s.service.recheckSelectedOpenAIAccountFromDB(ctx, account, req.RequestedModel)
	if account == nil {
		_ = s.service.deleteStickySessionAccountID(ctx, req.GroupID, sessionHash)
		return nil, nil, nil
	}
	if !s.isAccountEndpointCapabilityCompatible(account, req.RequiredCapability) {
		_ = s.service.deleteStickySessionAccountID(ctx, req.GroupID, sessionHash)
		return nil, nil, nil
	}

	result, acquireErr := s.service.tryAcquireAccountSlot(ctx, accountID, account.Concurrency)
	if acquireErr == nil && result != nil && result.Acquired {
		_ = s.service.refreshStickySessionTTL(ctx, req.GroupID, sessionHash, s.service.openAIWSSessionStickyTTL())
		return &AccountSelectionResult{
			Account:     account,
			Acquired:    true,
			ReleaseFunc: result.ReleaseFunc,
		}, nil, nil
	}
	if acquireErr != nil {
		slog.DebugContext(ctx, "openai_account_scheduler_sticky_acquire_failed", "account_id", accountID, "error", acquireErr)
		slog.DebugContext(ctx, "openai_account_scheduler_sticky_busy_try_load_balance", "account_id", accountID, "session", shortSessionHash(sessionHash))
		return nil, nil, nil
	}
	stickyWait := s.service.buildOpenAIStickyWaitSelection(ctx, req.GroupID, sessionHash, req.RequestedModel, account, s.service.schedulingConfig())
	slog.DebugContext(ctx, "openai_account_scheduler_sticky_busy_try_load_balance", "account_id", accountID, "session", shortSessionHash(sessionHash))
	return nil, stickyWait, nil
}

func (s *defaultOpenAIAccountScheduler) isAccountTransportCompatible(account *Account, requiredTransport OpenAIUpstreamTransport) bool {
	if requiredTransport == OpenAIUpstreamTransportAny || requiredTransport == OpenAIUpstreamTransportHTTPSSE {
		return true
	}
	if s == nil || s.service == nil || account == nil {
		return false
	}
	return s.service.getOpenAIWSProtocolResolver().Resolve(account).Transport == requiredTransport
}

func (s *defaultOpenAIAccountScheduler) isAccountEndpointCapabilityCompatible(account *Account, requiredCapability OpenAIEndpointCapability) bool {
	return SupportsOpenAIEndpointCapability(account, requiredCapability)
}
