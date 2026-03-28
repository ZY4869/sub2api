package service

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func (s *GrokGatewayService) ProbeSSOAccount(ctx context.Context, account *Account, requestedModel string) (*GrokSSOProbeResult, error) {
	if s == nil {
		return nil, fmt.Errorf("grok gateway service is not configured")
	}
	if account == nil || !account.IsGrokSSO() {
		return nil, fmt.Errorf("grok sso account is required")
	}
	if strings.TrimSpace(account.GetGrokSSOToken()) == "" {
		return nil, fmt.Errorf("grok sso account is missing sso_token")
	}

	result := &GrokSSOProbeResult{
		Tier:             ResolveGrokTier(account.Extra),
		Capabilities:     ResolveGrokCapabilities(account.Extra),
		CapabilityModels: GrokCapabilityModelIDsForAccount(account),
		VisibleModels:    GrokVisibleModelIDsForAccount(account),
	}

	probeModel := strings.TrimSpace(requestedModel)
	if probeModel == "" {
		if len(result.VisibleModels) > 0 {
			probeModel = result.VisibleModels[0]
		} else if len(result.CapabilityModels) > 0 {
			probeModel = result.CapabilityModels[0]
		}
	}
	if probeModel == "" {
		return result, fmt.Errorf("no Grok models are enabled for this account")
	}
	result.RequestedModel = probeModel

	validation, err := s.validateSSORequest(account, probeModel, "", nil)
	if err != nil {
		return result, err
	}
	result.MappedModel = validation.MappedModel

	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
	}

	exec, err := s.executeSSOReverseRequest(ctx, account, validation.MappedModel, "", "health check", map[string]any{})
	if err != nil {
		return result, err
	}
	result.ResponseID = strings.TrimSpace(exec.ResponseID)
	result.ConversationID = strings.TrimSpace(exec.ConversationID)
	return result, nil
}
