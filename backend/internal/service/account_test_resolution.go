package service

import (
	"context"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

func gatewayTestSourceProtocolLabel(sourceProtocol string) string {
	if descriptor, ok := ProtocolGatewayDescriptorByID(sourceProtocol); ok {
		return descriptor.DisplayName
	}
	return sourceProtocol
}

func gatewayTestSimulatedClientLabel(simulatedClient string) string {
	switch strings.TrimSpace(simulatedClient) {
	case GatewayClientProfileCodex:
		return "Codex"
	case GatewayClientProfileGeminiCLI:
		return "Gemini CLI"
	case "claude_client_mimic":
		return "Claude Client Mimic"
	default:
		return simulatedClient
	}
}

type resolvedGatewayTestTarget struct {
	ModelID        string
	SourceProtocol string
	TargetProvider string
	TargetModelID  string
}

func containsGatewayProtocol(values []string, target string) bool {
	target = normalizeTestSourceProtocol(target)
	for _, value := range values {
		if normalizeTestSourceProtocol(value) == target {
			return true
		}
	}
	return false
}

func supportedProtocolsForProvider(provider string) []string {
	switch NormalizeModelProvider(provider) {
	case PlatformOpenAI, PlatformGrok, PlatformDeepSeek:
		return []string{PlatformOpenAI}
	case PlatformAnthropic, PlatformKiro:
		return []string{PlatformAnthropic}
	case PlatformGemini:
		return []string{PlatformGemini}
	case PlatformAntigravity:
		return []string{PlatformAnthropic, PlatformGemini}
	default:
		return nil
	}
}

func firstNonEmptyGatewayTestValue(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func matchGatewayTestProtocols(models []AvailableTestModel, acceptedProtocols []string, provider string, modelID string) []string {
	provider = NormalizeModelProvider(provider)
	trimmedModelID := strings.TrimSpace(modelID)
	normalizedModelID := normalizeRegistryID(trimmedModelID)

	matched := make([]string, 0, len(acceptedProtocols))
	seen := make(map[string]struct{}, len(acceptedProtocols))
	for _, candidate := range models {
		protocol := normalizeTestSourceProtocol(candidate.SourceProtocol)
		if protocol == "" || !containsGatewayProtocol(acceptedProtocols, protocol) {
			continue
		}
		if provider != "" && NormalizeModelProvider(candidate.Provider) != provider {
			continue
		}
		if trimmedModelID != "" {
			if candidate.ID != trimmedModelID && normalizeRegistryID(candidate.CanonicalID) != normalizedModelID {
				continue
			}
		}
		if _, ok := seen[protocol]; ok {
			continue
		}
		seen[protocol] = struct{}{}
		matched = append(matched, protocol)
	}
	if len(matched) > 0 || provider == "" || trimmedModelID != "" {
		return matched
	}
	for _, protocol := range supportedProtocolsForProvider(provider) {
		if !containsGatewayProtocol(acceptedProtocols, protocol) {
			continue
		}
		if _, ok := seen[protocol]; ok {
			continue
		}
		seen[protocol] = struct{}{}
		matched = append(matched, protocol)
	}
	return matched
}

func resolveGatewayDefaultTestModelID(models []AvailableTestModel, provider string) string {
	normalizedProvider := NormalizeModelProvider(provider)
	for _, model := range models {
		if NormalizeModelProvider(model.Provider) == normalizedProvider {
			return strings.TrimSpace(model.ID)
		}
	}
	return ""
}

func invalidGatewaySourceProtocolError() error {
	return infraerrors.BadRequest("TEST_SOURCE_PROTOCOL_INVALID", "selected source_protocol is not accepted by this account")
}

func invalidGatewayTargetProviderError() error {
	return infraerrors.BadRequest("TEST_TARGET_PROVIDER_INVALID", "selected target_provider is not available for this account")
}

func incompatibleGatewayTargetProviderError() error {
	return infraerrors.BadRequest("TEST_TARGET_PROVIDER_INCOMPATIBLE", "selected target_provider is not compatible with source_protocol")
}

func ambiguousGatewayTargetProviderError() error {
	return infraerrors.BadRequest("TEST_TARGET_PROVIDER_REQUIRED", "mixed protocol gateway test requires target_provider or source_protocol")
}

func invalidGatewayTargetModelError() error {
	return infraerrors.BadRequest("TEST_TARGET_MODEL_INVALID", "selected target_model_id is not available for this account")
}

func missingGatewayDefaultTargetModelError() error {
	return infraerrors.BadRequest("TEST_TARGET_MODEL_REQUIRED", "selected target_provider does not have a default test model")
}

func testModelNotAllowedError() error {
	return infraerrors.BadRequest("TEST_MODEL_NOT_ALLOWED", "selected model is not allowed for this account")
}

func ambiguousGatewayProbeResolutionError() error {
	return infraerrors.BadRequest("TEST_PROBE_RESOLUTION_FAILED", "mixed protocol gateway test could not resolve a unique protocol")
}

func (s *AccountTestService) resolveRestrictedDefaultTestModel(ctx context.Context, account *Account) (string, string) {
	if account == nil || !accountHasExplicitModelRestrictions(account) {
		return "", ""
	}
	models := BuildAvailableTestModels(ctx, account, s.modelRegistryService)
	if len(models) == 0 {
		return "", ""
	}
	return strings.TrimSpace(models[0].ID), normalizeTestSourceProtocol(models[0].SourceProtocol)
}

func (s *AccountTestService) ensureAllowedTestModel(ctx context.Context, account *Account, modelID string) error {
	if account == nil || strings.TrimSpace(modelID) == "" || !accountHasExplicitModelRestrictions(account) {
		return nil
	}
	if isRequestedModelSupportedByAccount(ctx, s.modelRegistryService, account, modelID) {
		return nil
	}
	return testModelNotAllowedError()
}

func (s *AccountTestService) resolveGatewayTestTarget(ctx context.Context, account *Account, modelID string, requested string, targetProvider string, targetModelID string) (resolvedGatewayTestTarget, error) {
	resolution := resolvedGatewayTestTarget{
		ModelID:        strings.TrimSpace(modelID),
		SourceProtocol: normalizeTestSourceProtocol(requested),
		TargetProvider: NormalizeModelProvider(targetProvider),
		TargetModelID:  strings.TrimSpace(targetModelID),
	}
	if resolution.ModelID == "" && resolution.TargetModelID != "" {
		resolution.ModelID = resolution.TargetModelID
	}
	if !IsProtocolGatewayAccount(account) {
		return resolution, nil
	}

	acceptedProtocols := GetAccountGatewayAcceptedProtocols(account)
	if len(acceptedProtocols) == 0 {
		return resolution, nil
	}

	availableModels := BuildAvailableTestModels(ctx, account, s.modelRegistryService)
	defaultProvider := ""
	defaultTargetModelID := ""
	if resolution.SourceProtocol == "" && resolution.TargetProvider == "" && resolution.TargetModelID == "" {
		defaultProvider = GetAccountGatewayTestProvider(account)
		defaultTargetModelID = GetAccountGatewayTestModelID(account)
	}

	resolveDefaultModelForProvider := func(provider string) (string, error) {
		if provider == "" {
			return "", nil
		}
		if defaultModelID := resolveGatewayDefaultTestModelID(availableModels, provider); defaultModelID != "" {
			return defaultModelID, nil
		}
		return "", missingGatewayDefaultTargetModelError()
	}

	resolveProtocol := func(provider string, explicitModelID string) ([]string, error) {
		if provider != "" && strings.TrimSpace(explicitModelID) != "" {
			if matches := matchGatewayTestProtocols(availableModels, acceptedProtocols, provider, ""); len(matches) == 0 {
				return nil, invalidGatewayTargetProviderError()
			}
		}
		matched := matchGatewayTestProtocols(availableModels, acceptedProtocols, provider, explicitModelID)
		if len(matched) == 0 {
			if explicitModelID != "" {
				return nil, invalidGatewayTargetModelError()
			}
			if provider != "" {
				return nil, invalidGatewayTargetProviderError()
			}
			return nil, ambiguousGatewayTargetProviderError()
		}
		return matched, nil
	}

	if len(acceptedProtocols) == 1 {
		resolution.SourceProtocol = acceptedProtocols[0]
		if normalizedRequested := normalizeTestSourceProtocol(requested); normalizedRequested != "" && normalizedRequested != resolution.SourceProtocol {
			return resolvedGatewayTestTarget{}, invalidGatewaySourceProtocolError()
		}
		providerToValidate := firstNonEmptyGatewayTestValue(resolution.TargetProvider, defaultProvider)
		if providerToValidate != "" {
			matchedProtocols, err := resolveProtocol(providerToValidate, resolution.ModelID)
			if err != nil {
				return resolvedGatewayTestTarget{}, err
			}
			if !containsGatewayProtocol(matchedProtocols, resolution.SourceProtocol) {
				return resolvedGatewayTestTarget{}, incompatibleGatewayTargetProviderError()
			}
			if resolution.ModelID == "" {
				defaultModelID, err := resolveDefaultModelForProvider(providerToValidate)
				if err != nil {
					return resolvedGatewayTestTarget{}, err
				}
				resolution.ModelID = defaultModelID
			}
		}
		if resolution.TargetModelID == "" {
			resolution.TargetModelID = defaultTargetModelID
		}
		if resolution.TargetProvider == "" {
			resolution.TargetProvider = defaultProvider
		}
		if resolution.ModelID == "" && resolution.TargetModelID != "" {
			resolution.ModelID = resolution.TargetModelID
		}
		return resolution, nil
	}

	if resolution.SourceProtocol != "" {
		if !containsGatewayProtocol(acceptedProtocols, resolution.SourceProtocol) {
			return resolvedGatewayTestTarget{}, invalidGatewaySourceProtocolError()
		}
		if resolution.TargetProvider != "" {
			matchedProtocols, err := resolveProtocol(resolution.TargetProvider, resolution.ModelID)
			if err != nil {
				return resolvedGatewayTestTarget{}, err
			}
			if !containsGatewayProtocol(matchedProtocols, resolution.SourceProtocol) {
				return resolvedGatewayTestTarget{}, incompatibleGatewayTargetProviderError()
			}
			if resolution.ModelID == "" {
				defaultModelID, err := resolveDefaultModelForProvider(resolution.TargetProvider)
				if err != nil {
					return resolvedGatewayTestTarget{}, err
				}
				resolution.ModelID = defaultModelID
			}
		}
		return resolution, nil
	}

	if resolution.TargetProvider != "" {
		matchedProtocols, err := resolveProtocol(resolution.TargetProvider, resolution.ModelID)
		if err != nil {
			return resolvedGatewayTestTarget{}, err
		}
		if len(matchedProtocols) > 1 {
			return resolvedGatewayTestTarget{}, ambiguousGatewayProbeResolutionError()
		}
		resolution.SourceProtocol = matchedProtocols[0]
		if resolution.ModelID == "" {
			defaultModelID, err := resolveDefaultModelForProvider(resolution.TargetProvider)
			if err != nil {
				return resolvedGatewayTestTarget{}, err
			}
			resolution.ModelID = defaultModelID
		}
		return resolution, nil
	}

	if resolution.TargetModelID != "" {
		matchedProtocols, err := resolveProtocol("", resolution.TargetModelID)
		if err != nil {
			return resolvedGatewayTestTarget{}, err
		}
		if len(matchedProtocols) > 1 {
			return resolvedGatewayTestTarget{}, ambiguousGatewayTargetProviderError()
		}
		resolution.SourceProtocol = matchedProtocols[0]
		if resolution.ModelID == "" {
			resolution.ModelID = resolution.TargetModelID
		}
		return resolution, nil
	}

	if defaultProvider != "" {
		matchedProtocols, err := resolveProtocol(defaultProvider, firstNonEmptyGatewayTestValue(resolution.ModelID, defaultTargetModelID))
		if err != nil {
			return resolvedGatewayTestTarget{}, err
		}
		if len(matchedProtocols) > 1 {
			return resolvedGatewayTestTarget{}, ambiguousGatewayProbeResolutionError()
		}
		resolution.SourceProtocol = matchedProtocols[0]
		resolution.TargetProvider = defaultProvider
		resolution.TargetModelID = defaultTargetModelID
		if resolution.ModelID == "" {
			if resolution.TargetModelID != "" {
				resolution.ModelID = resolution.TargetModelID
			} else {
				defaultModelID, err := resolveDefaultModelForProvider(defaultProvider)
				if err != nil {
					return resolvedGatewayTestTarget{}, err
				}
				resolution.ModelID = defaultModelID
			}
		}
		return resolution, nil
	}

	if defaultTargetModelID != "" {
		matchedProtocols, err := resolveProtocol("", defaultTargetModelID)
		if err != nil {
			return resolvedGatewayTestTarget{}, err
		}
		if len(matchedProtocols) > 1 {
			return resolvedGatewayTestTarget{}, ambiguousGatewayTargetProviderError()
		}
		resolution.SourceProtocol = matchedProtocols[0]
		resolution.TargetModelID = defaultTargetModelID
		if resolution.ModelID == "" {
			resolution.ModelID = defaultTargetModelID
		}
		return resolution, nil
	}

	if resolution.ModelID != "" {
		matchedProtocols, err := resolveProtocol("", resolution.ModelID)
		if err != nil {
			return resolvedGatewayTestTarget{}, err
		}
		if len(matchedProtocols) > 1 {
			return resolvedGatewayTestTarget{}, ambiguousGatewayTargetProviderError()
		}
		resolution.SourceProtocol = matchedProtocols[0]
		return resolution, nil
	}

	matchedProtocols, err := resolveProtocol("", "")
	if err != nil {
		return resolvedGatewayTestTarget{}, err
	}
	if len(matchedProtocols) != 1 {
		return resolvedGatewayTestTarget{}, ambiguousGatewayTargetProviderError()
	}
	resolution.SourceProtocol = matchedProtocols[0]
	return resolution, nil
}

func (s *AccountTestService) resolveGatewayTestSimulatedClient(ctx context.Context, account *Account, sourceProtocol string, modelID string) string {
	if account == nil || !IsProtocolGatewayAccount(account) {
		return ""
	}
	if normalizedProtocol := normalizeTestSourceProtocol(sourceProtocol); normalizedProtocol == "" {
		return ""
	}
	trimmedModelID := strings.TrimSpace(modelID)
	if trimmedModelID != "" {
		if route := MatchGatewayClientRoute(account, sourceProtocol, trimmedModelID); route != nil {
			return route.ClientProfile
		}
		if s.modelRegistryService != nil {
			if canonicalModelID, ok, err := s.modelRegistryService.ResolveModel(ctx, trimmedModelID); err == nil && ok && canonicalModelID != "" {
				if route := MatchGatewayClientRoute(account, sourceProtocol, canonicalModelID); route != nil {
					return route.ClientProfile
				}
				if protocolModelID := s.resolveTestModelID(ctx, account, canonicalModelID); protocolModelID != "" {
					if route := MatchGatewayClientRoute(account, sourceProtocol, protocolModelID); route != nil {
						return route.ClientProfile
					}
				}
			}
		}
	}
	if IsClaudeClientMimicEnabled(account, sourceProtocol) {
		return "claude_client_mimic"
	}
	return ""
}

// TestAccountConnection tests an account's connection by sending a test request
// All account types use full Claude Code client characteristics, only auth header differs
// modelID is optional - if empty, defaults to claude.DefaultTestModel
