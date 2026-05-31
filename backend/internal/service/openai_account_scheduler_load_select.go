package service

import "context"

func (s *defaultOpenAIAccountScheduler) tryOpenAILoadBalanceAcquire(
	ctx context.Context,
	req OpenAIAccountScheduleRequest,
	order []openAIAccountCandidateScore,
	candidateCount int,
	topK int,
	loadSkew float64,
	phase string,
) (*AccountSelectionResult, bool, error) {
	for _, candidate := range order {
		if isContextDoneError(ctx, nil) {
			return nil, false, ctx.Err()
		}
		fresh := s.resolveOpenAILoadBalanceFreshAccount(ctx, req, candidate)
		if fresh == nil {
			if isContextDoneError(ctx, nil) {
				return nil, false, ctx.Err()
			}
			continue
		}
		result, acquireErr := s.service.tryAcquireAccountSlot(ctx, fresh.ID, fresh.Concurrency)
		if acquireErr != nil {
			return nil, false, acquireErr
		}
		if result != nil && result.Acquired {
			s.logLoadBalanceSelection(phase, req, candidate, candidateCount, topK, loadSkew)
			if req.SessionHash != "" {
				_ = s.service.BindStickySession(ctx, req.GroupID, req.SessionHash, fresh.ID)
			}
			return &AccountSelectionResult{
				Account:     fresh,
				Acquired:    true,
				ReleaseFunc: result.ReleaseFunc,
			}, true, nil
		}
	}
	return nil, false, nil
}

func (s *defaultOpenAIAccountScheduler) buildOpenAILoadBalanceWaitSelection(
	ctx context.Context,
	req OpenAIAccountScheduleRequest,
	order []openAIAccountCandidateScore,
	candidateCount int,
	topK int,
	loadSkew float64,
) (*AccountSelectionResult, bool, error) {
	cfg := s.service.schedulingConfig()
	for _, candidate := range order {
		if isContextDoneError(ctx, nil) {
			return nil, false, ctx.Err()
		}
		fresh := s.resolveOpenAILoadBalanceFreshAccount(ctx, req, candidate)
		if fresh == nil {
			if isContextDoneError(ctx, nil) {
				return nil, false, ctx.Err()
			}
			continue
		}
		s.logLoadBalanceSelection("wait", req, candidate, candidateCount, topK, loadSkew)
		return &AccountSelectionResult{
			Account: fresh,
			WaitPlan: &AccountWaitPlan{
				AccountID:      fresh.ID,
				MaxConcurrency: fresh.Concurrency,
				Timeout:        cfg.FallbackWaitTimeout,
				MaxWaiting:     cfg.FallbackMaxWaiting,
			},
		}, true, nil
	}
	return nil, false, nil
}

func (s *defaultOpenAIAccountScheduler) resolveOpenAILoadBalanceFreshAccount(
	ctx context.Context,
	req OpenAIAccountScheduleRequest,
	candidate openAIAccountCandidateScore,
) *Account {
	fresh := s.service.resolveFreshSchedulableOpenAIAccount(ctx, candidate.account, req.RequestedModel)
	if fresh == nil {
		return nil
	}
	if !s.isAccountTransportCompatible(fresh, req.RequiredTransport) {
		return nil
	}
	if !s.isAccountEndpointCapabilityCompatible(fresh, req.RequiredCapability) {
		return nil
	}
	fresh = s.service.recheckSelectedOpenAIAccountFromDB(ctx, fresh, req.RequestedModel)
	if fresh == nil || !s.isAccountTransportCompatible(fresh, req.RequiredTransport) || !s.isAccountEndpointCapabilityCompatible(fresh, req.RequiredCapability) {
		return nil
	}
	return fresh
}
