package service

import (
	"context"
	"net/url"
	"strings"
	"sync/atomic"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

var (
	vertexGlobalSelectedTotal        atomic.Int64
	vertexRegionalFallbackTotal      atomic.Int64
	vertexNoCandidateAfterQuotaTotal atomic.Int64
)

func GeminiVertexRoutingStats() (globalSelected, regionalFallback, noCandidateAfterQuota int64) {
	return vertexGlobalSelectedTotal.Load(), vertexRegionalFallbackTotal.Load(), vertexNoCandidateAfterQuotaTotal.Load()
}

func geminiVertexRoutingLocation(account *Account) string {
	if account == nil || !account.IsGeminiVertexSource() {
		return ""
	}
	if raw := strings.TrimSpace(account.GetCredential("vertex_location")); raw != "" {
		return normalizeVertexLocation(raw)
	}
	if inferred := inferVertexLocationFromBaseURL(geminiVertexRoutingBaseURL(account)); inferred != "" {
		return inferred
	}
	return "global"
}

func geminiVertexRoutingBaseURL(account *Account) string {
	if account == nil {
		return ""
	}
	if account.IsGeminiVertexAI() {
		return account.GetGeminiVertexBaseURL("")
	}
	if account.IsGeminiVertexExpress() {
		return account.GetGeminiVertexExpressBaseURL("")
	}
	return ""
}

func inferVertexLocationFromBaseURL(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return ""
	}
	host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	switch {
	case host == "", host == "aiplatform.googleapis.com":
		return "global"
	case strings.HasSuffix(host, "-aiplatform.googleapis.com"):
		return strings.TrimSuffix(host, "-aiplatform.googleapis.com")
	default:
		return ""
	}
}

func geminiVertexEndpointKind(account *Account) string {
	if account == nil || !account.IsGeminiVertexSource() {
		return ""
	}
	if geminiVertexRoutingLocation(account) == "global" {
		return "global"
	}
	return "regional"
}

func geminiRegionalPenalty(account *Account, preferOAuth bool) int {
	if !preferOAuth || account == nil {
		return 0
	}
	if geminiVertexEndpointKind(account) == "regional" {
		return 1
	}
	return 0
}

func isPreferredAccountBySelectionOrder(candidate, current *Account, preferOAuth bool) bool {
	if current == nil {
		return candidate != nil
	}
	if candidate == nil {
		return false
	}
	if candidate.Priority != current.Priority {
		return candidate.Priority < current.Priority
	}
	candidatePenalty := geminiRegionalPenalty(candidate, preferOAuth)
	currentPenalty := geminiRegionalPenalty(current, preferOAuth)
	if candidatePenalty != currentPenalty {
		return candidatePenalty < currentPenalty
	}
	switch {
	case candidate.LastUsedAt == nil && current.LastUsedAt != nil:
		return true
	case candidate.LastUsedAt != nil && current.LastUsedAt == nil:
		return false
	case candidate.LastUsedAt == nil && current.LastUsedAt == nil:
		return preferOAuth && candidate.Type == AccountTypeOAuth && current.Type != AccountTypeOAuth
	default:
		return candidate.LastUsedAt.Before(*current.LastUsedAt)
	}
}

type geminiVertexRoutingObservation struct {
	noCandidateAfterQuota    bool
	samePriorityGlobalReason string
}

func (s *GatewayService) observeGeminiVertexRouting(ctx context.Context, accounts []Account, groupID *int64, requestedModel, platform string, useMixed bool, excludedIDs map[int64]struct{}, selected *Account, phase string) {
	if platform != PlatformGemini {
		return
	}
	observation := s.summarizeGeminiVertexRouting(ctx, accounts, groupID, requestedModel, platform, useMixed, excludedIDs, selected)
	if selected == nil {
		if observation.noCandidateAfterQuota {
			vertexNoCandidateAfterQuotaTotal.Add(1)
			logger.LegacyPrintf("service.gateway", "[VertexRouting] selection_phase=%s model=%s vertex_no_candidate_after_quota=true fallback_reason=quota_exceeded", phase, requestedModel)
		}
		return
	}
	if !selected.IsGeminiVertexSource() {
		return
	}
	location := geminiVertexRoutingLocation(selected)
	endpointKind := geminiVertexEndpointKind(selected)
	fallbackReason := ""
	switch endpointKind {
	case "global":
		vertexGlobalSelectedTotal.Add(1)
	case "regional":
		fallbackReason = observation.samePriorityGlobalReason
		if fallbackReason != "" {
			vertexRegionalFallbackTotal.Add(1)
		}
	}
	logger.LegacyPrintf(
		"service.gateway",
		"[VertexRouting] selection_phase=%s model=%s account_id=%d vertex_location=%s vertex_endpoint_kind=%s fallback_reason=%s",
		phase,
		requestedModel,
		selected.ID,
		location,
		endpointKind,
		fallbackReason,
	)
}

func (s *GatewayService) summarizeGeminiVertexRouting(ctx context.Context, accounts []Account, groupID *int64, requestedModel, platform string, useMixed bool, excludedIDs map[int64]struct{}, selected *Account) geminiVertexRoutingObservation {
	observation := geminiVertexRoutingObservation{}
	if platform != PlatformGemini {
		return observation
	}
	selectedRegional := selected != nil && geminiVertexEndpointKind(selected) == "regional"
	hasPreQuotaEligibleVertex := false
	hasPostQuotaEligibleVertex := false
	for i := range accounts {
		account := &accounts[i]
		if !account.IsGeminiVertexSource() || !s.isAccountAllowedForPlatformWithContext(ctx, account, platform, useMixed) {
			continue
		}
		if requestedModel != "" && !s.isModelSupportedByAccountWithContext(ctx, account, requestedModel) {
			continue
		}
		preQuotaEligible := s.isAccountSchedulableForSelection(account) &&
			s.isAccountSchedulableForModelSelection(ctx, account, requestedModel) &&
			s.isAccountSchedulableForWindowCost(ctx, account, false) &&
			s.isAccountSchedulableForRPM(ctx, account, false)
		if preQuotaEligible {
			hasPreQuotaEligibleVertex = true
			if s.isAccountSchedulableForQuota(account) {
				hasPostQuotaEligibleVertex = true
			}
		}
	}
	if selectedRegional {
		if reason, found := s.findGeminiVertexFallbackReason(ctx, accounts, groupID, requestedModel, platform, useMixed, excludedIDs, selected); found {
			observation.samePriorityGlobalReason = reason
		} else if broaderAccounts, err := s.loadGeminiVertexObservationAccounts(ctx, groupID); err == nil {
			if reason, found := s.findGeminiVertexFallbackReason(ctx, broaderAccounts, groupID, requestedModel, platform, useMixed, excludedIDs, selected); found {
				observation.samePriorityGlobalReason = reason
			}
		}
	}
	observation.noCandidateAfterQuota = selected == nil && hasPreQuotaEligibleVertex && !hasPostQuotaEligibleVertex
	return observation
}

func (s *GatewayService) loadGeminiVertexObservationAccounts(ctx context.Context, groupID *int64) ([]Account, error) {
	if s == nil || s.accountRepo == nil {
		return nil, nil
	}
	if groupID != nil {
		return s.accountRepo.ListByGroup(ctx, *groupID)
	}
	return s.accountRepo.ListByPlatform(ctx, PlatformGemini)
}

func (s *GatewayService) findGeminiVertexFallbackReason(ctx context.Context, accounts []Account, groupID *int64, requestedModel, platform string, useMixed bool, excludedIDs map[int64]struct{}, selected *Account) (string, bool) {
	if selected == nil || geminiVertexEndpointKind(selected) != "regional" {
		return "", false
	}
	foundGlobal := false
	for i := range accounts {
		account := &accounts[i]
		if !account.IsGeminiVertexSource() || geminiVertexEndpointKind(account) != "global" {
			continue
		}
		if account.Priority != selected.Priority {
			continue
		}
		if groupID == nil && !s.isAccountInGroup(account, nil) {
			continue
		}
		if !s.isAccountAllowedForPlatformWithContext(ctx, account, platform, useMixed) {
			continue
		}
		foundGlobal = true
		if requestedModel != "" && !s.isModelSupportedByAccountWithContext(ctx, account, requestedModel) {
			return "model_unsupported", true
		}
		if reason := s.geminiVertexSkipReason(ctx, account, requestedModel, platform, useMixed, excludedIDs); reason != "" {
			return reason, true
		}
	}
	if foundGlobal {
		return "unavailable", true
	}
	return "", false
}

func (s *GatewayService) geminiVertexSkipReason(ctx context.Context, account *Account, requestedModel, platform string, useMixed bool, excludedIDs map[int64]struct{}) string {
	if account == nil || !account.IsGeminiVertexSource() {
		return ""
	}
	if excludedIDs != nil {
		if _, excluded := excludedIDs[account.ID]; excluded {
			return "excluded"
		}
	}
	if !s.isAccountSchedulableForSelection(account) {
		switch {
		case account.IsRateLimited():
			return "rate_limited"
		case account.IsOverloaded():
			return "overloaded"
		default:
			return "unschedulable"
		}
	}
	if !s.isAccountAllowedForPlatformWithContext(ctx, account, platform, useMixed) {
		return "platform_mismatch"
	}
	if requestedModel != "" && !s.isModelSupportedByAccountWithContext(ctx, account, requestedModel) {
		return "model_unsupported"
	}
	if !s.isAccountSchedulableForModelSelection(ctx, account, requestedModel) {
		if account.GetRateLimitRemainingTimeWithContext(ctx, requestedModel) > 0 {
			return "model_rate_limited"
		}
		return "precheck_blocked"
	}
	if !s.isAccountSchedulableForQuota(account) {
		return "quota_exceeded"
	}
	if !s.isAccountSchedulableForWindowCost(ctx, account, false) {
		return "window_cost_blocked"
	}
	if !s.isAccountSchedulableForRPM(ctx, account, false) {
		return "rpm_blocked"
	}
	return ""
}
