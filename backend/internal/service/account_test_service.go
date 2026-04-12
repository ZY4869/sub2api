package service

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/geminicli"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/Wei-Shaw/sub2api/internal/pkg/protocolruntime"
	"github.com/Wei-Shaw/sub2api/internal/util/urlvalidator"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// sseDataPrefix matches SSE data lines with optional whitespace after colon.
// Some upstream APIs return non-standard "data:" without space (should be "data: ").
var sseDataPrefix = regexp.MustCompile(`^data:\s*`)

const (
	testClaudeAPIURL = "https://api.anthropic.com/v1/messages?beta=true"
)

// TestEvent represents a SSE event for account testing
type TestEvent struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	Model    string `json:"model,omitempty"`
	Status   string `json:"status,omitempty"`
	Code     string `json:"code,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
	Data     any    `json:"data,omitempty"`
	Success  bool   `json:"success,omitempty"`
	Error    string `json:"error,omitempty"`
}

const (
	defaultGeminiTextTestPrompt  = "hi"
	defaultGeminiImageTestPrompt = "Generate a cute orange cat astronaut sticker on a clean pastel background."
)

// AccountTestService handles account testing operations
type AccountTestService struct {
	accountRepo                  AccountRepository
	accountModelImportService    *AccountModelImportService
	claudeTokenProvider          *ClaudeTokenProvider
	modelRegistryService         *ModelRegistryService
	gatewayService               *GatewayService
	grokGatewayService           *GrokGatewayService
	openAIGatewayService         *OpenAIGatewayService
	geminiCompatService          *GeminiMessagesCompatService
	openAITokenProvider          *OpenAITokenProvider
	geminiTokenProvider          *GeminiTokenProvider
	antigravityGatewayService    *AntigravityGatewayService
	httpUpstream                 HTTPUpstream
	tlsFingerprintProfileService *TLSFingerprintProfileService
	cfg                          *config.Config
	backgroundRunner             func(func())
}

// NewAccountTestService creates a new AccountTestService
func NewAccountTestService(
	accountRepo AccountRepository,
	accountModelImportService *AccountModelImportService,
	geminiTokenProvider *GeminiTokenProvider,
	antigravityGatewayService *AntigravityGatewayService,
	httpUpstream HTTPUpstream,
	cfg *config.Config,
) *AccountTestService {
	return &AccountTestService{
		accountRepo:               accountRepo,
		accountModelImportService: accountModelImportService,
		geminiTokenProvider:       geminiTokenProvider,
		antigravityGatewayService: antigravityGatewayService,
		httpUpstream:              httpUpstream,
		cfg:                       cfg,
	}
}

func (s *AccountTestService) SetModelRegistryService(modelRegistryService *ModelRegistryService) {
	s.modelRegistryService = modelRegistryService
}

func (s *AccountTestService) SetGatewayService(gatewayService *GatewayService) {
	s.gatewayService = gatewayService
}

func (s *AccountTestService) SetGrokGatewayService(grokGatewayService *GrokGatewayService) {
	s.grokGatewayService = grokGatewayService
}

func (s *AccountTestService) SetOpenAIGatewayService(openAIGatewayService *OpenAIGatewayService) {
	s.openAIGatewayService = openAIGatewayService
}

func (s *AccountTestService) SetGeminiCompatService(geminiCompatService *GeminiMessagesCompatService) {
	s.geminiCompatService = geminiCompatService
}

func (s *AccountTestService) SetClaudeTokenProvider(claudeTokenProvider *ClaudeTokenProvider) {
	s.claudeTokenProvider = claudeTokenProvider
}

func (s *AccountTestService) SetOpenAITokenProvider(openAITokenProvider *OpenAITokenProvider) {
	s.openAITokenProvider = openAITokenProvider
}

func (s *AccountTestService) SetTLSFingerprintProfileService(tlsFingerprintProfileService *TLSFingerprintProfileService) {
	s.tlsFingerprintProfileService = tlsFingerprintProfileService
}

func (s *AccountTestService) runBackgroundTask(fn func()) {
	if fn == nil {
		return
	}
	if s.backgroundRunner != nil {
		s.backgroundRunner(fn)
		return
	}
	go fn()
}

func (s *AccountTestService) resolveTestModelID(ctx context.Context, account *Account, modelID string) string {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return modelID
	}
	if s.modelRegistryService == nil {
		return modelID
	}
	if canonicalID, ok, err := s.modelRegistryService.ResolveModel(ctx, modelID); err == nil && ok && canonicalID != "" {
		modelID = canonicalID
	}
	if account != nil {
		if protocolID, ok, err := s.modelRegistryService.ResolveProtocolModel(ctx, modelID, registryRouteForAccount(account)); err == nil && ok && protocolID != "" {
			return protocolID
		}
	}
	return modelID
}

func cloneAccountForBackgroundProbe(account *Account) *Account {
	if account == nil {
		return nil
	}

	cloned := *account
	cloned.Credentials = cloneStringAnyMap(account.Credentials)
	cloned.Extra = cloneStringAnyMap(account.Extra)
	cloned.Proxy = nil
	if account.Proxy != nil {
		proxy := *account.Proxy
		cloned.Proxy = &proxy
	}

	cloned.modelMappingCache = nil
	cloned.modelMappingCacheReady = false
	cloned.modelMappingCacheCredentialsPtr = 0
	cloned.modelMappingCacheRawPtr = 0
	cloned.modelMappingCacheRawLen = 0
	cloned.modelMappingCacheRawSig = 0

	return &cloned
}

func (s *AccountTestService) refreshOpenAIKnownModelsSnapshot(account *Account) {
	if s == nil || s.accountRepo == nil || s.accountModelImportService == nil || account == nil || !account.IsOpenAIOAuth() {
		return
	}

	cloned := cloneAccountForBackgroundProbe(account)
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	probeResult, err := s.accountModelImportService.ProbeAccountModels(ctx, cloned)
	if err != nil {
		slog.Warn(
			"openai_known_models_probe_failed",
			"account_id", account.ID,
			"duration_ms", time.Since(start).Milliseconds(),
			"error", err,
		)
		return
	}
	if probeResult == nil || len(probeResult.DetectedModels) == 0 {
		slog.Info(
			"openai_known_models_probe_empty",
			"account_id", account.ID,
			"duration_ms", time.Since(start).Milliseconds(),
		)
		return
	}

	updatedAt := time.Now().UTC()
	updates := MergeStringAnyMap(
		BuildOpenAIKnownModelsExtra(
			probeResult.DetectedModels,
			updatedAt,
			OpenAIKnownModelsSourceTestProbe,
		),
		BuildAccountModelProbeSnapshotExtra(
			probeResult.DetectedModels,
			updatedAt,
			AccountModelProbeSnapshotSourceTestProbe,
			probeResult.ProbeSource,
		),
	)
	if err := s.accountRepo.UpdateExtra(ctx, account.ID, updates); err != nil {
		slog.Warn(
			"openai_known_models_snapshot_update_failed",
			"account_id", account.ID,
			"duration_ms", time.Since(start).Milliseconds(),
			"error", err,
		)
		return
	}
	mergeAccountExtra(account, updates)

	slog.Info(
		"openai_known_models_snapshot_updated",
		"account_id", account.ID,
		"model_count", len(probeResult.DetectedModels),
		"probe_source", probeResult.ProbeSource,
		"duration_ms", time.Since(start).Milliseconds(),
	)
}

func (s *AccountTestService) validateUpstreamBaseURL(raw string) (string, error) {
	if s.cfg == nil {
		return "", errors.New("config is not available")
	}
	if !s.cfg.Security.URLAllowlist.Enabled {
		return urlvalidator.ValidateURLFormat(raw, s.cfg.Security.URLAllowlist.AllowInsecureHTTP)
	}
	normalized, err := urlvalidator.ValidateHTTPSURL(raw, urlvalidator.ValidationOptions{
		AllowedHosts:     s.cfg.Security.URLAllowlist.UpstreamHosts,
		RequireAllowlist: true,
		AllowPrivate:     s.cfg.Security.URLAllowlist.AllowPrivateHosts,
	})
	if err != nil {
		return "", err
	}
	return normalized, nil
}

// generateSessionString generates a Claude Code style session string.
// The output format is determined by the UA version in claude.DefaultHeaders,
// ensuring consistency between the user_id format and the UA sent to upstream.
func generateSessionString() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	hex64 := hex.EncodeToString(b)
	sessionUUID := uuid.New().String()
	uaVersion := ExtractCLIVersion(claude.DefaultHeaders["User-Agent"])
	return FormatMetadataUserID(hex64, "", sessionUUID, uaVersion), nil
}

// createTestPayload creates a Claude Code style test request payload
func createTestPayload(modelID string) (map[string]any, error) {
	sessionID, err := generateSessionString()
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"model": modelID,
		"messages": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{
						"type": "text",
						"text": "hi",
						"cache_control": map[string]string{
							"type": "ephemeral",
						},
					},
				},
			},
		},
		"system": []map[string]any{
			{
				"type": "text",
				"text": claudeCodeSystemPrompt,
				"cache_control": map[string]string{
					"type": "ephemeral",
				},
			},
		},
		"metadata": map[string]string{
			"user_id": sessionID,
		},
		"max_tokens":  1024,
		"temperature": 1,
		"stream":      true,
	}, nil
}

func createAnthropicStandardTestPayload(modelID string) map[string]any {
	return map[string]any{
		"model": modelID,
		"messages": []map[string]any{
			{
				"role":    "user",
				"content": "hi",
			},
		},
		"max_tokens": 64,
		"stream":     true,
	}
}

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
	case PlatformOpenAI, PlatformGrok, PlatformCopilot:
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

func ambiguousGatewayProbeResolutionError() error {
	return infraerrors.BadRequest("TEST_PROBE_RESOLUTION_FAILED", "mixed protocol gateway test could not resolve a unique protocol")
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
func (s *AccountTestService) TestAccountConnection(c *gin.Context, accountID int64, modelID string, prompt string, sourceProtocol string, targetProvider string, targetModelID string, testMode string) error {
	ctx := c.Request.Context()

	// Get account
	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return s.sendErrorAndEnd(c, "Account not found")
	}

	resolvedTarget, err := s.resolveGatewayTestTarget(ctx, account, modelID, sourceProtocol, targetProvider, targetModelID)
	if err != nil {
		reason := infraerrors.Reason(err)
		if reason == "TEST_PROBE_RESOLUTION_FAILED" {
			protocolruntime.RecordAccountProbeResolutionFailed(reason)
			slog.Warn(
				"account_probe_resolution_failed",
				"account_id", accountID,
				"requested_model_id", strings.TrimSpace(modelID),
				"source_protocol", normalizeTestSourceProtocol(sourceProtocol),
				"target_provider", NormalizeModelProvider(targetProvider),
				"target_model_id", strings.TrimSpace(targetModelID),
				"reason", reason,
				"error", err,
			)
		} else {
			protocolruntime.RecordAccountTestResolutionFailed(reason)
		}
		slog.Warn(
			"account_test_resolution_failed",
			"account_id", accountID,
			"requested_model_id", strings.TrimSpace(modelID),
			"source_protocol", normalizeTestSourceProtocol(sourceProtocol),
			"target_provider", NormalizeModelProvider(targetProvider),
			"target_model_id", strings.TrimSpace(targetModelID),
			"reason", reason,
			"error", err,
		)
		return err
	}
	if resolvedTarget.SourceProtocol != "" {
		account = ResolveProtocolGatewayInboundAccount(account, resolvedTarget.SourceProtocol)
	}
	if resolvedTarget.ModelID != "" {
		modelID = resolvedTarget.ModelID
	}
	simulatedClient := s.resolveGatewayTestSimulatedClient(ctx, account, resolvedTarget.SourceProtocol, modelID)
	normalizedTestMode := normalizeAccountTestMode(testMode)
	runtimeMeta := buildAccountTestRuntimeMeta(
		account,
		normalizedTestMode,
		resolvedTarget.SourceProtocol,
		resolvedTarget.TargetProvider,
		resolvedTarget.TargetModelID,
		modelID,
		simulatedClient,
	)
	s.setResolvedTestRuntimeMeta(c, runtimeMeta)
	slog.Info(
		"account_test_start",
		"account_id", accountID,
		"test_mode", string(normalizedTestMode),
		"inbound_endpoint", runtimeMeta.InboundEndpoint,
		"source_protocol", runtimeMeta.SourceProtocol,
		"target_provider", runtimeMeta.TargetProvider,
		"target_model_id", runtimeMeta.TargetModelID,
		"resolved_model_id", runtimeMeta.ResolvedModelID,
		"compat_path", runtimeMeta.CompatPath,
		"runtime_platform", runtimeMeta.RuntimePlatform,
		"simulated_client", runtimeMeta.SimulatedClient,
	)

	var testErr error
	if normalizedTestMode == AccountTestModeRealForward {
		testErr = s.testAccountConnectionRealForward(c, account, modelID, prompt, resolvedTarget.SourceProtocol, simulatedClient)
	} else {
		testErr = s.testAccountConnectionHealthCheck(c, account, modelID, prompt, resolvedTarget.SourceProtocol, simulatedClient)
	}
	if testErr != nil {
		slog.Warn(
			"account_test_complete",
			"account_id", accountID,
			"status", "failed",
			"test_mode", string(normalizedTestMode),
			"inbound_endpoint", runtimeMeta.InboundEndpoint,
			"source_protocol", runtimeMeta.SourceProtocol,
			"target_provider", runtimeMeta.TargetProvider,
			"target_model_id", runtimeMeta.TargetModelID,
			"resolved_model_id", runtimeMeta.ResolvedModelID,
			"compat_path", runtimeMeta.CompatPath,
			"runtime_platform", runtimeMeta.RuntimePlatform,
			"error", testErr,
		)
		return testErr
	}
	slog.Info(
		"account_test_complete",
		"account_id", accountID,
		"status", "success",
		"test_mode", string(normalizedTestMode),
		"inbound_endpoint", runtimeMeta.InboundEndpoint,
		"source_protocol", runtimeMeta.SourceProtocol,
		"target_provider", runtimeMeta.TargetProvider,
		"target_model_id", runtimeMeta.TargetModelID,
		"resolved_model_id", runtimeMeta.ResolvedModelID,
		"compat_path", runtimeMeta.CompatPath,
		"runtime_platform", runtimeMeta.RuntimePlatform,
	)
	return nil
}

func (s *AccountTestService) testAccountConnectionHealthCheck(c *gin.Context, account *Account, modelID string, prompt string, resolvedSourceProtocol string, simulatedClient string) error {
	if account == nil {
		return s.sendErrorAndEnd(c, "Account not found")
	}

	if account.IsOpenAI() {
		return s.testOpenAIAccountConnection(c, account, modelID, resolvedSourceProtocol, simulatedClient)
	}

	if account.IsGrok() {
		return s.testGrokAccountConnection(c, account, modelID)
	}

	if account.IsGemini() {
		return s.testGeminiAccountConnection(c, account, modelID, prompt, resolvedSourceProtocol, simulatedClient)
	}

	if RoutingPlatformForAccount(account) == PlatformAntigravity {
		return s.routeAntigravityTest(c, account, modelID, prompt)
	}

	return s.testClaudeAccountConnection(c, account, modelID, resolvedSourceProtocol, simulatedClient)
}

// testClaudeAccountConnection tests an Anthropic Claude account's connection
func (s *AccountTestService) testClaudeAccountConnection(c *gin.Context, account *Account, modelID string, sourceProtocol string, simulatedClient string) error {
	if account != nil && RoutingPlatformForAccount(account) == PlatformKiro {
		return s.testKiroAccountConnection(c, account, modelID)
	}
	ctx := c.Request.Context()
	shouldMimicClaudeClient := IsClaudeClientMimicEnabled(account, sourceProtocol)

	// Determine the model to use
	testModelID := modelID
	if testModelID == "" {
		testModelID = claude.DefaultTestModel
	}

	// API Key 账号测试连接时也需要应用通配符模型映射。
	if account.Type == "apikey" {
		testModelID = account.GetMappedModel(testModelID)
	}

	// Bedrock accounts use a separate test path
	if account.IsBedrock() {
		return s.testBedrockAccountConnection(c, ctx, account, testModelID)
	}
	testModelID = s.resolveTestModelID(ctx, account, testModelID)

	// Determine authentication method and API URL
	var authToken string
	var useBearer bool
	var apiURL string
	var err error

	if account.IsOAuth() {
		// OAuth or Setup Token - use Bearer token
		useBearer = true
		apiURL = testClaudeAPIURL
		if account.Type == AccountTypeOAuth && s.claudeTokenProvider != nil {
			authToken, err = s.claudeTokenProvider.GetAccessToken(ctx, account)
			if err != nil {
				return s.sendErrorAndEnd(c, fmt.Sprintf("Failed to get access token: %s", err.Error()))
			}
		} else {
			authToken = account.GetCredential("access_token")
		}
		if authToken == "" {
			return s.sendErrorAndEnd(c, "No access token available")
		}
	} else if account.Type == "apikey" {
		// API Key - use x-api-key header
		useBearer = false
		authToken = account.GetCredential("api_key")
		if authToken == "" {
			return s.sendErrorAndEnd(c, "No API key available")
		}

		baseURL := account.GetBaseURL()
		if baseURL == "" {
			baseURL = "https://api.anthropic.com"
		}
		normalizedBaseURL, err := s.validateUpstreamBaseURL(baseURL)
		if err != nil {
			return s.sendErrorAndEnd(c, fmt.Sprintf("Invalid base URL: %s", err.Error()))
		}
		apiURL = strings.TrimSuffix(normalizedBaseURL, "/") + "/v1/messages?beta=true"
	} else {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Unsupported account type: %s", account.Type))
	}

	// Set SSE headers
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.Flush()

	var payloadBytes []byte
	if shouldMimicClaudeClient || useBearer {
		payload, payloadErr := createTestPayload(testModelID)
		if payloadErr != nil {
			return s.sendErrorAndEnd(c, "Failed to create test payload")
		}
		payloadBytes, _ = json.Marshal(payload)
	} else {
		payloadBytes, _ = json.Marshal(createAnthropicStandardTestPayload(testModelID))
	}

	// Send test_start event
	s.sendEvent(c, TestEvent{Type: "test_start", Model: testModelID})
	s.sendResolvedTestRuntimeMetaEvents(c)

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return s.sendErrorAndEnd(c, "Failed to create request")
	}

	// Set common headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-version", "2023-06-01")

	if shouldMimicClaudeClient || useBearer {
		for key, value := range claude.DefaultHeaders {
			req.Header.Set(key, value)
		}
	}

	// Set authentication header
	if useBearer {
		req.Header.Set("anthropic-beta", claude.DefaultBetaHeader)
		req.Header.Set("Authorization", "Bearer "+authToken)
	} else {
		if shouldMimicClaudeClient {
			req.Header.Set("anthropic-beta", claude.APIKeyBetaHeader)
		}
		req.Header.Set("x-api-key", authToken)
	}

	// Get proxy URL
	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	tlsProfile := resolveAccountTLSFingerprintProfile(account, s.tlsFingerprintProfileService)

	resp, err := s.httpUpstream.DoWithTLS(req, proxyURL, account.ID, account.Concurrency, tlsProfile)
	if err != nil {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Request failed: %s", err.Error()))
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return s.sendFailedTestResponse(c, ctx, account, resp.StatusCode, body, "API returned")
	}

	// Process SSE stream
	return s.processClaudeStream(c, resp.Body)
}

func (s *AccountTestService) testKiroAccountConnection(c *gin.Context, account *Account, modelID string) error {
	ctx := c.Request.Context()
	testModelID := modelID
	if testModelID == "" {
		testModelID = claude.DefaultTestModel
	}
	testModelID = s.resolveTestModelID(ctx, account, testModelID)

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.Flush()

	payload, err := createTestPayload(testModelID)
	if err != nil {
		return s.sendErrorAndEnd(c, "Failed to create test payload")
	}
	payloadBytes, _ := json.Marshal(payload)
	s.sendEvent(c, TestEvent{Type: "test_start", Model: testModelID})

	runtime := NewKiroRuntimeService(s.accountRepo, s.httpUpstream, s.claudeTokenProvider)
	runtime.SetTLSFingerprintProfileService(s.tlsFingerprintProfileService)
	result, err := runtime.ExecuteClaude(ctx, account, KiroRuntimeExecuteInput{
		Body:           payloadBytes,
		ModelID:        testModelID,
		Stream:         true,
		RequestHeaders: c.Request.Header,
	})
	if err != nil {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Request failed: %s", err.Error()))
	}
	if result == nil || result.Response == nil {
		return s.sendErrorAndEnd(c, "Kiro runtime returned empty response")
	}

	resp := result.Response
	defer func() { _ = resp.Body.Close() }()
	s.sendKiroRuntimeMetaEvents(c, result)
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return s.sendFailedTestResponse(c, ctx, account, resp.StatusCode, body, "API returned")
	}

	return s.processClaudeStream(c, resp.Body)
}

func (s *AccountTestService) sendKiroRuntimeMetaEvents(c *gin.Context, result *KiroRuntimeExecuteResult) {
	if c == nil || result == nil {
		return
	}
	emitMeta := func(text string, data map[string]any) {
		s.sendEvent(c, TestEvent{
			Type: "content",
			Text: text,
			Data: data,
		})
	}

	emitMeta(
		fmt.Sprintf("Kiro runtime region: %s", result.Region),
		map[string]any{
			"kind":   "runtime_meta",
			"key":    "resolved_region",
			"value":  result.Region,
			"source": "kiro_runtime",
		},
	)
	emitMeta(
		fmt.Sprintf("Kiro runtime endpoint: %s (%s)", result.Endpoint.Name, result.Endpoint.URL),
		map[string]any{
			"kind":   "runtime_meta",
			"key":    "endpoint",
			"value":  result.Endpoint.URL,
			"label":  result.Endpoint.Name,
			"source": "kiro_runtime",
		},
	)
	emitMeta(
		fmt.Sprintf("Kiro endpoint fallback: %t", result.FallbackUsed),
		map[string]any{
			"kind":   "runtime_meta",
			"key":    "fallback",
			"value":  result.FallbackUsed,
			"source": "kiro_runtime",
		},
	)
	emitMeta(
		fmt.Sprintf("Kiro profile ARN present: %t", strings.TrimSpace(result.ProfileARN) != ""),
		map[string]any{
			"kind":   "runtime_meta",
			"key":    "profile_arn_present",
			"value":  strings.TrimSpace(result.ProfileARN) != "",
			"source": "kiro_runtime",
		},
	)
}

// testBedrockAccountConnection tests a Bedrock (SigV4 or API Key) account using non-streaming invoke
func (s *AccountTestService) testBedrockAccountConnection(c *gin.Context, ctx context.Context, account *Account, testModelID string) error {
	region := bedrockRuntimeRegion(account)
	resolvedModelID, ok := ResolveBedrockModelID(account, testModelID)
	if !ok {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Unsupported Bedrock model: %s", testModelID))
	}
	testModelID = resolvedModelID

	// Set SSE headers (test UI expects SSE)
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.Flush()

	// Create a minimal Bedrock-compatible payload (no stream, no cache_control)
	bedrockPayload := map[string]any{
		"anthropic_version": "bedrock-2023-05-31",
		"messages": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{
						"type": "text",
						"text": "hi",
					},
				},
			},
		},
		"max_tokens":  256,
		"temperature": 1,
	}
	bedrockBody, _ := json.Marshal(bedrockPayload)

	// Use non-streaming endpoint (response is standard Claude JSON)
	apiURL := BuildBedrockURL(region, testModelID, false)

	s.sendEvent(c, TestEvent{Type: "test_start", Model: testModelID})

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(bedrockBody))
	if err != nil {
		return s.sendErrorAndEnd(c, "Failed to create request")
	}
	req.Header.Set("Content-Type", "application/json")

	// Sign or set auth based on account type
	if account.IsBedrockAPIKey() {
		apiKey := account.GetCredential("api_key")
		if apiKey == "" {
			return s.sendErrorAndEnd(c, "No API key available")
		}
		req.Header.Set("Authorization", "Bearer "+apiKey)
	} else {
		signer, err := NewBedrockSignerFromAccount(account)
		if err != nil {
			return s.sendErrorAndEnd(c, fmt.Sprintf("Failed to create Bedrock signer: %s", err.Error()))
		}
		if err := signer.SignRequest(ctx, req, bedrockBody); err != nil {
			return s.sendErrorAndEnd(c, fmt.Sprintf("Failed to sign request: %s", err.Error()))
		}
	}

	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}

	resp, err := s.httpUpstream.DoWithTLS(req, proxyURL, account.ID, account.Concurrency, nil)
	if err != nil {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Request failed: %s", err.Error()))
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return s.sendFailedTestResponse(c, ctx, account, resp.StatusCode, body, "API returned")
	}

	// Bedrock non-streaming response is standard Claude JSON, extract the text
	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Failed to parse response: %s", err.Error()))
	}

	text := ""
	if len(result.Content) > 0 {
		text = result.Content[0].Text
	}
	if text == "" {
		text = "(empty response)"
	}

	s.sendEvent(c, TestEvent{Type: "content", Text: text})
	s.sendEvent(c, TestEvent{Type: "test_complete", Success: true})
	return nil
}

// testOpenAIAccountConnection tests an OpenAI account's connection
func (s *AccountTestService) testOpenAIAccountConnection(c *gin.Context, account *Account, modelID string, sourceProtocol string, simulatedClient string) error {
	ctx := c.Request.Context()
	requestFormat := ResolveOpenAITextRequestFormatForAccount(account, "")

	testModelID := modelID
	if testModelID == "" {
		testModelID = defaultOpenAIOAuthTestModelID(ctx, account, s.modelRegistryService)
	}

	// For API Key accounts with model mapping, map the model
	if account.Type == "apikey" {
		mapping := account.GetModelMapping()
		if len(mapping) > 0 {
			if mappedModel, exists := mapping[testModelID]; exists {
				testModelID = mappedModel
			}
		}
	}

	// Determine authentication method and API URL
	var authToken string
	var apiURL string
	var isOAuth bool
	var err error
	useChatGPTOAuth := isChatGPTOpenAIOAuthAccount(account)
	var chatgptAccountID string

	if account.IsOAuth() {
		isOAuth = true
		if s.openAITokenProvider != nil {
			authToken, err = s.openAITokenProvider.GetAccessToken(ctx, account)
			if err != nil {
				return s.sendErrorAndEnd(c, fmt.Sprintf("Failed to get access token: %s", err.Error()))
			}
		} else {
			authToken = account.GetOpenAIAccessToken()
		}
		if authToken == "" {
			return s.sendErrorAndEnd(c, "No access token available")
		}

		apiURL, err = resolveOpenAITargetURLForRequestFormat(account, requestFormat, s.validateUpstreamBaseURL)
		if err != nil {
			return s.sendErrorAndEnd(c, fmt.Sprintf("Invalid base URL: %s", err.Error()))
		}
		if useChatGPTOAuth {
			chatgptAccountID = account.GetChatGPTAccountID()
		}
	} else if account.Type == "apikey" {
		// API Key - use Platform API
		authToken = account.GetOpenAIApiKey()
		if authToken == "" {
			return s.sendErrorAndEnd(c, "No API key available")
		}

		baseURL := account.GetOpenAIBaseURL()
		if baseURL == "" {
			baseURL = "https://api.openai.com"
		}
		normalizedBaseURL, err := s.validateUpstreamBaseURL(baseURL)
		if err != nil {
			return s.sendErrorAndEnd(c, fmt.Sprintf("Invalid base URL: %s", err.Error()))
		}
		apiURL = buildOpenAIResponsesURLForPlatform(normalizedBaseURL, account.Platform)
		if requestFormat == GatewayOpenAIRequestFormatChatCompletions {
			apiURL = buildOpenAIChatCompletionsURLForPlatform(normalizedBaseURL, account.Platform)
		}
	} else {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Unsupported account type: %s", account.Type))
	}

	// Set SSE headers
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.Flush()

	payload := createOpenAITestPayloadForRequestFormat(testModelID, requestFormat, useChatGPTOAuth)
	payloadBytes, _ := json.Marshal(payload)

	// Send test_start event
	s.sendEvent(c, TestEvent{Type: "test_start", Model: testModelID})
	s.sendResolvedTestRuntimeMetaEvents(c)

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return s.sendErrorAndEnd(c, "Failed to create request")
	}

	// Set common headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+authToken)
	if simulatedClient == GatewayClientProfileCodex {
		req.Header.Set("Originator", resolveOpenAIUpstreamOriginator(c, true))
		req.Header.Set("User-Agent", codexCLIUserAgent)
	}

	if isCopilotOAuthAccount(account) {
		req.Header.Set("Accept", "text/event-stream")
		applyCopilotDefaultHeaders(req.Header, account)
	}

	// Set OAuth-specific headers for ChatGPT internal API
	if useChatGPTOAuth {
		req.Host = "chatgpt.com"
		req.Header.Set("accept", "text/event-stream")
		if chatgptAccountID != "" {
			req.Header.Set("chatgpt-account-id", chatgptAccountID)
		}
	}

	// Get proxy URL
	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	tlsProfile := resolveAccountTLSFingerprintProfile(account, s.tlsFingerprintProfileService)

	resp, err := s.httpUpstream.DoWithTLS(req, proxyURL, account.ID, account.Concurrency, tlsProfile)
	if err != nil {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Request failed: %s", err.Error()))
	}
	defer func() { _ = resp.Body.Close() }()

	if useChatGPTOAuth && s.accountRepo != nil {
		if updates, err := extractOpenAICodexProbeUpdates(resp); err == nil && len(updates) > 0 {
			_ = s.accountRepo.UpdateExtra(ctx, account.ID, updates)
			mergeAccountExtra(account, updates)
		}
		if snapshot := ParseCodexRateLimitHeaders(resp.Header); snapshot != nil {
			if resetAt := codexRateLimitResetAtFromSnapshot(snapshot, time.Now()); resetAt != nil {
				_ = setAccountRateLimited(ctx, s.accountRepo, account.ID, *resetAt, codexRateLimitReasonFromSnapshot(snapshot))
				account.RateLimitResetAt = resetAt
			}
		}
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if useChatGPTOAuth && s.accountRepo != nil {
			if resetAt := (&RateLimitService{}).calculateOpenAI429ResetTime(resp.Header); resetAt != nil {
				_ = setAccountRateLimited(ctx, s.accountRepo, account.ID, *resetAt, codexRateLimitReasonFromSnapshot(ParseCodexRateLimitHeaders(resp.Header)))
				account.RateLimitResetAt = resetAt
			}
		}
		if resp.StatusCode == http.StatusUnauthorized && s.accountRepo != nil {
			errMsg := fmt.Sprintf("Authentication failed (401): %s", string(body))
			_ = s.accountRepo.SetError(ctx, account.ID, errMsg)
		}
		return s.sendFailedTestResponse(c, ctx, account, resp.StatusCode, body, "API returned")
	}

	if isOAuth && s.accountModelImportService != nil && s.accountRepo != nil {
		accountForProbe := cloneAccountForBackgroundProbe(account)
		s.runBackgroundTask(func() {
			s.refreshOpenAIKnownModelsSnapshot(accountForProbe)
		})
	}

	// Process SSE stream
	if requestFormat == GatewayOpenAIRequestFormatChatCompletions {
		return s.processOpenAIChatCompletionsStream(c, resp.Body)
	}
	return s.processOpenAIStream(c, resp.Body)
}

func defaultGeminiTestModelID(account *Account) string {
	if account != nil && account.IsGeminiVertexSource() {
		return defaultGeminiVertexValidationModel
	}
	return geminicli.DefaultTestModel
}

// testGeminiAccountConnection tests a Gemini account's connection
func (s *AccountTestService) testGeminiAccountConnection(c *gin.Context, account *Account, modelID string, prompt string, sourceProtocol string, simulatedClient string) error {
	ctx := c.Request.Context()
	shouldMimicGeminiCLI := simulatedClient == GatewayClientProfileGeminiCLI

	// Determine the model to use
	testModelID := modelID
	if testModelID == "" {
		testModelID = defaultGeminiTestModelID(account)
	}

	// For API Key accounts with model mapping, map the model
	if account.Type == AccountTypeAPIKey {
		mapping := account.GetModelMapping()
		if len(mapping) > 0 {
			if mappedModel, exists := mapping[testModelID]; exists {
				testModelID = mappedModel
			}
		}
	}
	testModelID = s.resolveTestModelID(ctx, account, testModelID)

	// Set SSE headers
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.Flush()

	// Create test payload (Gemini format)
	payload := createGeminiTestPayload(testModelID, prompt)

	// Build request based on account type
	var req *http.Request
	var err error

	switch account.Type {
	case AccountTypeAPIKey:
		req, err = s.buildGeminiAPIKeyRequest(ctx, account, testModelID, payload, shouldMimicGeminiCLI)
	case AccountTypeOAuth:
		req, err = s.buildGeminiOAuthRequest(ctx, account, testModelID, payload, shouldMimicGeminiCLI)
	default:
		return s.sendErrorAndEnd(c, fmt.Sprintf("Unsupported account type: %s", account.Type))
	}

	if err != nil {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Failed to build request: %s", err.Error()))
	}

	// Send test_start event
	s.sendEvent(c, TestEvent{Type: "test_start", Model: testModelID})
	s.sendResolvedTestRuntimeMetaEvents(c)

	// Get proxy and execute request
	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	tlsProfile := resolveAccountTLSFingerprintProfile(account, s.tlsFingerprintProfileService)

	resp, err := s.httpUpstream.DoWithTLS(req, proxyURL, account.ID, account.Concurrency, tlsProfile)
	if err != nil {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Request failed: %s", err.Error()))
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return s.sendFailedTestResponse(c, ctx, account, resp.StatusCode, body, "API returned")
	}

	// Process SSE stream
	return s.processGeminiStream(c, resp.Body)
}

// routeAntigravityTest 路由 Antigravity 账号的测试请求。
// APIKey 类型走原生协议（与 gateway_handler 路由一致），OAuth/Upstream 走 CRS 中转。
func (s *AccountTestService) routeAntigravityTest(c *gin.Context, account *Account, modelID string, prompt string) error {
	if account.Type == AccountTypeAPIKey {
		if strings.HasPrefix(modelID, "gemini-") {
			return s.testGeminiAccountConnection(c, account, modelID, prompt, "", "")
		}
		return s.testClaudeAccountConnection(c, account, modelID, "", "")
	}
	return s.testAntigravityAccountConnection(c, account, modelID)
}

// testAntigravityAccountConnection tests an Antigravity account's connection
// 支持 Claude 和 Gemini 两种协议，使用非流式请求
func (s *AccountTestService) testAntigravityAccountConnection(c *gin.Context, account *Account, modelID string) error {
	ctx := c.Request.Context()

	// 默认模型：Claude 使用 claude-sonnet-4-5，Gemini 使用 gemini-3-pro-preview
	testModelID := modelID
	if testModelID == "" {
		testModelID = "claude-sonnet-4-5"
	}

	if s.antigravityGatewayService == nil {
		return s.sendErrorAndEnd(c, "Antigravity gateway service not configured")
	}

	// Set SSE headers
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.Flush()

	// Send test_start event
	s.sendEvent(c, TestEvent{Type: "test_start", Model: testModelID})

	// 调用 AntigravityGatewayService.TestConnection（复用协议转换逻辑）
	result, err := s.antigravityGatewayService.TestConnection(ctx, account, testModelID)
	if err != nil {
		return s.sendErrorAndEnd(c, err.Error())
	}

	// 发送响应内容
	if result.Text != "" {
		s.sendEvent(c, TestEvent{Type: "content", Text: result.Text})
	}

	s.sendEvent(c, TestEvent{Type: "test_complete", Success: true})
	return nil
}

// buildGeminiAPIKeyRequest builds request for Gemini API Key accounts
func (s *AccountTestService) buildGeminiAPIKeyRequest(ctx context.Context, account *Account, modelID string, payload []byte, mimicGeminiCLI bool) (*http.Request, error) {
	apiKey := account.GetCredential("api_key")
	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("no API key available")
	}

	baseURL := account.GetCredential("base_url")
	if baseURL == "" {
		if account.IsGeminiVertexExpress() {
			baseURL = geminicli.VertexAIBaseURL
		} else {
			baseURL = geminicli.AIStudioBaseURL
		}
	}
	normalizedBaseURL, err := s.validateUpstreamBaseURL(baseURL)
	if err != nil {
		return nil, err
	}

	fullURL := fmt.Sprintf("%s/v1beta/models/%s:streamGenerateContent?alt=sse",
		strings.TrimRight(normalizedBaseURL, "/"), modelID)
	if account.IsGeminiVertexExpress() {
		modelID = normalizeVertexUpstreamModelID(modelID)
		actionPath, err := account.GeminiVertexExpressModelActionPath(modelID, "streamGenerateContent")
		if err != nil {
			return nil, err
		}
		fullURL = strings.TrimRight(normalizedBaseURL, "/") + actionPath
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fullURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if account.IsGeminiVertexExpress() {
		query := req.URL.Query()
		query.Set("key", apiKey)
		query.Set("alt", "sse")
		req.URL.RawQuery = query.Encode()
	} else {
		req.Header.Set("x-goog-api-key", apiKey)
	}
	if mimicGeminiCLI {
		req.Header.Set("User-Agent", geminicli.GeminiCLIUserAgent)
	}

	return req, nil
}

// buildGeminiOAuthRequest builds request for Gemini OAuth accounts
func (s *AccountTestService) buildGeminiOAuthRequest(ctx context.Context, account *Account, modelID string, payload []byte, mimicGeminiCLI bool) (*http.Request, error) {
	if s.geminiTokenProvider == nil {
		return nil, fmt.Errorf("gemini token provider not configured")
	}

	// Get access token (auto-refreshes if needed)
	accessToken, err := s.geminiTokenProvider.GetAccessToken(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	if account.IsGeminiVertexAI() {
		modelID = normalizeVertexUpstreamModelID(modelID)
		baseURL := account.GetGeminiVertexBaseURL(geminicli.VertexAIBaseURL)
		normalizedBaseURL, err := s.validateUpstreamBaseURL(baseURL)
		if err != nil {
			return nil, err
		}
		actionPath, err := account.GeminiVertexModelActionPath(modelID, "streamGenerateContent")
		if err != nil {
			return nil, err
		}
		fullURL := fmt.Sprintf("%s%s?alt=sse", strings.TrimRight(normalizedBaseURL, "/"), actionPath)

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, bytes.NewReader(payload))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+accessToken)
		if mimicGeminiCLI {
			req.Header.Set("User-Agent", geminicli.GeminiCLIUserAgent)
		}
		return req, nil
	}

	projectID := strings.TrimSpace(account.GetCredential("project_id"))
	if projectID == "" {
		// AI Studio OAuth mode (no project_id): call generativelanguage API directly with Bearer token.
		baseURL := account.GetCredential("base_url")
		if strings.TrimSpace(baseURL) == "" {
			baseURL = geminicli.AIStudioBaseURL
		}
		normalizedBaseURL, err := s.validateUpstreamBaseURL(baseURL)
		if err != nil {
			return nil, err
		}
		fullURL := fmt.Sprintf("%s/v1beta/models/%s:streamGenerateContent?alt=sse", strings.TrimRight(normalizedBaseURL, "/"), modelID)

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, bytes.NewReader(payload))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+accessToken)
		if mimicGeminiCLI {
			req.Header.Set("User-Agent", geminicli.GeminiCLIUserAgent)
		}
		return req, nil
	}

	// Code Assist mode (with project_id)
	return s.buildCodeAssistRequest(ctx, accessToken, projectID, modelID, payload)
}

// buildCodeAssistRequest builds request for Google Code Assist API (used by Gemini CLI and Antigravity)
func (s *AccountTestService) buildCodeAssistRequest(ctx context.Context, accessToken, projectID, modelID string, payload []byte) (*http.Request, error) {
	var inner map[string]any
	if err := json.Unmarshal(payload, &inner); err != nil {
		return nil, err
	}

	wrapped := map[string]any{
		"model":   modelID,
		"project": projectID,
		"request": inner,
	}
	wrappedBytes, _ := json.Marshal(wrapped)

	normalizedBaseURL, err := s.validateUpstreamBaseURL(geminicli.GeminiCliBaseURL)
	if err != nil {
		return nil, err
	}
	fullURL := fmt.Sprintf("%s/v1internal:streamGenerateContent?alt=sse", normalizedBaseURL)

	req, err := http.NewRequestWithContext(ctx, "POST", fullURL, bytes.NewReader(wrappedBytes))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("User-Agent", geminicli.GeminiCLIUserAgent)

	return req, nil
}

// createGeminiTestPayload creates a minimal test payload for Gemini API.
// Image models use the image-generation path so the frontend can preview the returned image.
func createGeminiTestPayload(modelID string, prompt string) []byte {
	if isImageGenerationModel(modelID) {
		imagePrompt := strings.TrimSpace(prompt)
		if imagePrompt == "" {
			imagePrompt = defaultGeminiImageTestPrompt
		}

		payload := map[string]any{
			"contents": []map[string]any{
				{
					"role": "user",
					"parts": []map[string]any{
						{"text": imagePrompt},
					},
				},
			},
			"generationConfig": map[string]any{
				"responseModalities": []string{"TEXT", "IMAGE"},
				"imageConfig": map[string]any{
					"aspectRatio": "1:1",
				},
			},
		}
		bytes, _ := json.Marshal(payload)
		return bytes
	}

	textPrompt := strings.TrimSpace(prompt)
	if textPrompt == "" {
		textPrompt = defaultGeminiTextTestPrompt
	}

	payload := map[string]any{
		"contents": []map[string]any{
			{
				"role": "user",
				"parts": []map[string]any{
					{"text": textPrompt},
				},
			},
		},
		"systemInstruction": map[string]any{
			"parts": []map[string]any{
				{"text": "You are a helpful AI assistant."},
			},
		},
	}
	bytes, _ := json.Marshal(payload)
	return bytes
}

// processGeminiStream processes SSE stream from Gemini API
func (s *AccountTestService) processGeminiStream(c *gin.Context, body io.Reader) error {
	reader := bufio.NewReader(body)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				s.sendEvent(c, TestEvent{Type: "test_complete", Success: true})
				return nil
			}
			return s.sendErrorAndEnd(c, fmt.Sprintf("Stream read error: %s", err.Error()))
		}

		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, "data: ") {
			continue
		}

		jsonStr := strings.TrimPrefix(line, "data: ")
		if jsonStr == "[DONE]" {
			s.sendEvent(c, TestEvent{Type: "test_complete", Success: true})
			return nil
		}

		var data map[string]any
		if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
			continue
		}

		// Support two Gemini response formats:
		// - AI Studio: {"candidates": [...]}
		// - Gemini CLI: {"response": {"candidates": [...]}}
		if resp, ok := data["response"].(map[string]any); ok && resp != nil {
			data = resp
		}
		if candidates, ok := data["candidates"].([]any); ok && len(candidates) > 0 {
			if candidate, ok := candidates[0].(map[string]any); ok {
				// Extract content first (before checking completion)
				if content, ok := candidate["content"].(map[string]any); ok {
					if parts, ok := content["parts"].([]any); ok {
						for _, part := range parts {
							if partMap, ok := part.(map[string]any); ok {
								if text, ok := partMap["text"].(string); ok && text != "" {
									s.sendEvent(c, TestEvent{Type: "content", Text: text})
								}
								if inlineData, ok := partMap["inlineData"].(map[string]any); ok {
									mimeType, _ := inlineData["mimeType"].(string)
									data, _ := inlineData["data"].(string)
									if strings.HasPrefix(strings.ToLower(mimeType), "image/") && data != "" {
										s.sendEvent(c, TestEvent{
											Type:     "image",
											ImageURL: fmt.Sprintf("data:%s;base64,%s", mimeType, data),
											MimeType: mimeType,
										})
									}
								}
							}
						}
					}
				}

				// Check for completion after extracting content
				if finishReason, ok := candidate["finishReason"].(string); ok && finishReason != "" {
					s.sendEvent(c, TestEvent{Type: "test_complete", Success: true})
					return nil
				}
			}
		}

		// Handle errors
		if errData, ok := data["error"].(map[string]any); ok {
			errorMsg := "Unknown error"
			if msg, ok := errData["message"].(string); ok {
				errorMsg = msg
			}
			return s.sendErrorAndEnd(c, errorMsg)
		}
	}
}

func createOpenAITestPayloadForRequestFormat(modelID string, requestFormat string, isOAuth bool) map[string]any {
	if NormalizeGatewayOpenAIRequestFormat(requestFormat) == GatewayOpenAIRequestFormatChatCompletions {
		return createOpenAIChatCompletionsTestPayload(modelID)
	}
	return createOpenAITestPayload(modelID, isOAuth)
}

// createOpenAITestPayload creates a test payload for OpenAI Responses API
func createOpenAITestPayload(modelID string, isOAuth bool) map[string]any {
	payload := map[string]any{
		"model": modelID,
		"input": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{
						"type": "input_text",
						"text": "hi",
					},
				},
			},
		},
		"stream": true,
	}

	// OAuth accounts using ChatGPT internal API require store: false
	if isOAuth {
		payload["store"] = false
	}

	// All accounts require instructions for Responses API
	payload["instructions"] = openai.DefaultInstructions

	return payload
}

func createOpenAIChatCompletionsTestPayload(modelID string) map[string]any {
	return map[string]any{
		"model": modelID,
		"messages": []map[string]any{
			{
				"role":    "user",
				"content": "hi",
			},
		},
		"stream": true,
		"stream_options": map[string]any{
			"include_usage": true,
		},
	}
}

// processClaudeStream processes the SSE stream from Claude API
func (s *AccountTestService) processClaudeStream(c *gin.Context, body io.Reader) error {
	reader := bufio.NewReader(body)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				s.sendEvent(c, TestEvent{Type: "test_complete", Success: true})
				return nil
			}
			return s.sendErrorAndEnd(c, fmt.Sprintf("Stream read error: %s", err.Error()))
		}

		line = strings.TrimSpace(line)
		if line == "" || !sseDataPrefix.MatchString(line) {
			continue
		}

		jsonStr := sseDataPrefix.ReplaceAllString(line, "")
		if jsonStr == "[DONE]" {
			s.sendEvent(c, TestEvent{Type: "test_complete", Success: true})
			return nil
		}

		var data map[string]any
		if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
			continue
		}

		eventType, _ := data["type"].(string)

		switch eventType {
		case "content_block_delta":
			if delta, ok := data["delta"].(map[string]any); ok {
				if text, ok := delta["text"].(string); ok {
					s.sendEvent(c, TestEvent{Type: "content", Text: text})
				}
			}
		case "message_stop":
			s.sendEvent(c, TestEvent{Type: "test_complete", Success: true})
			return nil
		case "error":
			errorMsg := "Unknown error"
			if errData, ok := data["error"].(map[string]any); ok {
				if msg, ok := errData["message"].(string); ok {
					errorMsg = msg
				}
			}
			return s.sendErrorAndEnd(c, errorMsg)
		}
	}
}

// processOpenAIStream processes the SSE stream from OpenAI Responses API
func (s *AccountTestService) processOpenAIStream(c *gin.Context, body io.Reader) error {
	reader := bufio.NewReader(body)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				s.sendEvent(c, TestEvent{Type: "test_complete", Success: true})
				return nil
			}
			return s.sendErrorAndEnd(c, fmt.Sprintf("Stream read error: %s", err.Error()))
		}

		line = strings.TrimSpace(line)
		if line == "" || !sseDataPrefix.MatchString(line) {
			continue
		}

		jsonStr := sseDataPrefix.ReplaceAllString(line, "")
		if jsonStr == "[DONE]" {
			s.sendEvent(c, TestEvent{Type: "test_complete", Success: true})
			return nil
		}

		var data map[string]any
		if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
			continue
		}

		eventType, _ := data["type"].(string)

		switch eventType {
		case "response.output_text.delta":
			// OpenAI Responses API uses "delta" field for text content
			if delta, ok := data["delta"].(string); ok && delta != "" {
				s.sendEvent(c, TestEvent{Type: "content", Text: delta})
			}
		case "response.completed":
			s.sendEvent(c, TestEvent{Type: "test_complete", Success: true})
			return nil
		case "error":
			errorMsg := "Unknown error"
			if errData, ok := data["error"].(map[string]any); ok {
				if msg, ok := errData["message"].(string); ok {
					errorMsg = msg
				}
			}
			return s.sendErrorAndEnd(c, errorMsg)
		}
	}
}

func (s *AccountTestService) processOpenAIChatCompletionsStream(c *gin.Context, body io.Reader) error {
	reader := bufio.NewReader(body)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				s.sendEvent(c, TestEvent{Type: "test_complete", Success: true})
				return nil
			}
			return s.sendErrorAndEnd(c, fmt.Sprintf("Stream read error: %s", err.Error()))
		}

		line = strings.TrimSpace(line)
		if line == "" || !sseDataPrefix.MatchString(line) {
			continue
		}

		jsonStr := sseDataPrefix.ReplaceAllString(line, "")
		if jsonStr == "[DONE]" {
			s.sendEvent(c, TestEvent{Type: "test_complete", Success: true})
			return nil
		}

		var data map[string]any
		if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
			continue
		}

		if choices, ok := data["choices"].([]any); ok && len(choices) > 0 {
			if choice, ok := choices[0].(map[string]any); ok {
				if delta, ok := choice["delta"].(map[string]any); ok {
					if text, ok := delta["content"].(string); ok && text != "" {
						s.sendEvent(c, TestEvent{Type: "content", Text: text})
					}
				}
			}
		}

		if errData, ok := data["error"].(map[string]any); ok {
			errorMsg := "Unknown error"
			if msg, ok := errData["message"].(string); ok {
				errorMsg = msg
			}
			return s.sendErrorAndEnd(c, errorMsg)
		}
	}
}

// sendEvent sends a SSE event to the client
func (s *AccountTestService) sendEvent(c *gin.Context, event TestEvent) {
	eventJSON, _ := json.Marshal(event)
	if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", eventJSON); err != nil {
		log.Printf("failed to write SSE event: %v", err)
		return
	}
	c.Writer.Flush()
}

// sendErrorAndEnd sends an error event and ends the stream
func (s *AccountTestService) sendErrorAndEnd(c *gin.Context, errorMsg string) error {
	log.Printf("Account test error: %s", errorMsg)
	s.sendEvent(c, TestEvent{Type: "error", Error: errorMsg})
	return fmt.Errorf("%s", errorMsg)
}

func (s *AccountTestService) sendBlacklistAdviceEvent(c *gin.Context, advice *BlacklistAdvice) {
	if c == nil || advice == nil {
		return
	}
	s.sendEvent(c, TestEvent{
		Type: "blacklist_advice",
		Data: advice,
	})
}

func (s *AccountTestService) sendFailedTestResponse(c *gin.Context, ctx context.Context, account *Account, statusCode int, body []byte, prefix string) error {
	message, advice := s.formatFailedTestResponse(ctx, account, statusCode, body, prefix)
	if advice != nil {
		s.sendBlacklistAdviceEvent(c, advice)
	}
	return s.sendErrorAndEnd(c, message)
}

func (s *AccountTestService) formatFailedTestResponse(ctx context.Context, account *Account, statusCode int, body []byte, prefix string) (string, *BlacklistAdvice) {
	if strings.TrimSpace(prefix) == "" {
		prefix = "API returned"
	}
	message := fmt.Sprintf("%s %d: %s", prefix, statusCode, string(body))
	advice := BuildBlacklistAdvice(account, statusCode, body)
	if s == nil || s.accountRepo == nil || account == nil {
		return message, advice
	}
	if match := DetectHardBannedAccount(statusCode, body); match != nil {
		s.tryAutoBlacklistFailedTest(ctx, account, advice, match.ReasonCode, match.ReasonMessage, body)
		return message, advice
	}
	if advice != nil &&
		(statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden) &&
		advice.Decision == BlacklistAdviceRecommendBlacklist {
		s.tryAutoBlacklistFailedTest(ctx, account, advice, advice.ReasonCode, advice.ReasonMessage, body)
		return message, advice
	}
	if statusCode == http.StatusForbidden {
		_ = s.accountRepo.SetError(ctx, account.ID, message)
	}
	return message, advice
}

func (s *AccountTestService) tryAutoBlacklistFailedTest(ctx context.Context, account *Account, advice *BlacklistAdvice, reasonCode string, reasonMessage string, body []byte) {
	if s == nil || s.accountRepo == nil || account == nil {
		return
	}
	now := time.Now()
	purgeAt := now.Add(AccountBlacklistRetention)
	if err := s.accountRepo.MarkBlacklisted(ctx, account.ID, reasonCode, reasonMessage, now, purgeAt); err != nil {
		slog.Warn("account_test_mark_blacklisted_failed", "account_id", account.ID, "reason_code", reasonCode, "error", err)
		return
	}
	if advice == nil {
		return
	}
	advice.Decision = BlacklistAdviceAutoBlacklisted
	advice.ReasonCode = firstNonEmptyHardBanString(reasonCode, advice.ReasonCode)
	advice.ReasonMessage = firstNonEmptyHardBanString(reasonMessage, advice.ReasonMessage, string(body))
	advice.AlreadyBlacklisted = true
	advice.CollectFeedback = false
}

type parsedBackgroundTestOutput struct {
	ResponseText            string
	ErrorMessage            string
	ResolvedModelID         string
	ResolvedPlatform        string
	ResolvedSourceProtocol  string
	BlacklistAdviceDecision string
}

// RunTestBackgroundDetailed executes an account test in-memory (no real HTTP client),
// captures the SSE output, and returns a structured result for admin actions.
func (s *AccountTestService) RunTestBackgroundDetailed(ctx context.Context, input ScheduledTestExecutionInput) (*BackgroundAccountTestResult, error) {
	startedAt := time.Now()

	w := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(w)
	ginCtx.Request = (&http.Request{}).WithContext(ctx)

	testMode := string(normalizeAccountTestMode(input.TestMode))
	testErr := s.TestAccountConnection(
		ginCtx,
		input.AccountID,
		strings.TrimSpace(input.ModelID),
		strings.TrimSpace(input.Prompt),
		normalizeTestSourceProtocol(input.SourceProtocol),
		NormalizeModelProvider(input.TargetProvider),
		strings.TrimSpace(input.TargetModelID),
		testMode,
	)

	finishedAt := time.Now()
	parsed := parseTestSSEOutputDetailed(w.Body.String())

	status := "success"
	errMsg := parsed.ErrorMessage
	if testErr != nil || errMsg != "" {
		status = "failed"
		if errMsg == "" && testErr != nil {
			errMsg = testErr.Error()
		}
	}

	currentLifecycleState := ""
	if s != nil && s.accountRepo != nil {
		if account, err := s.accountRepo.GetByID(ctx, input.AccountID); err == nil && account != nil {
			currentLifecycleState = account.LifecycleState
		}
	}

	return &BackgroundAccountTestResult{
		Status:                  status,
		ResponseText:            parsed.ResponseText,
		ErrorMessage:            errMsg,
		LatencyMs:               finishedAt.Sub(startedAt).Milliseconds(),
		StartedAt:               startedAt,
		FinishedAt:              finishedAt,
		ResolvedModelID:         parsed.ResolvedModelID,
		ResolvedPlatform:        parsed.ResolvedPlatform,
		ResolvedSourceProtocol:  parsed.ResolvedSourceProtocol,
		BlacklistAdviceDecision: parsed.BlacklistAdviceDecision,
		CurrentLifecycleState:   currentLifecycleState,
	}, nil
}

// RunTestBackground preserves the legacy scheduled-test result shape.
func (s *AccountTestService) RunTestBackground(ctx context.Context, input ScheduledTestExecutionInput) (*ScheduledTestResult, error) {
	result, err := s.RunTestBackgroundDetailed(ctx, input)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	return &ScheduledTestResult{
		Status:       result.Status,
		ResponseText: result.ResponseText,
		ErrorMessage: result.ErrorMessage,
		LatencyMs:    result.LatencyMs,
		StartedAt:    result.StartedAt,
		FinishedAt:   result.FinishedAt,
	}, nil
}

// parseTestSSEOutputDetailed extracts key execution details from captured SSE output.
func parseTestSSEOutputDetailed(body string) parsedBackgroundTestOutput {
	result := parsedBackgroundTestOutput{}
	var texts []string
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if !sseDataPrefix.MatchString(line) {
			continue
		}
		jsonStr := sseDataPrefix.ReplaceAllString(line, "")
		var event TestEvent
		if err := json.Unmarshal([]byte(jsonStr), &event); err != nil {
			continue
		}
		switch event.Type {
		case "test_start":
			if event.Model != "" {
				result.ResolvedModelID = strings.TrimSpace(event.Model)
			}
		case "content":
			if event.Text != "" {
				texts = append(texts, event.Text)
			}
			runtimeMeta, ok := event.Data.(map[string]any)
			if !ok || strings.TrimSpace(fmt.Sprint(runtimeMeta["kind"])) != "runtime_meta" {
				continue
			}
			key := strings.TrimSpace(fmt.Sprint(runtimeMeta["key"]))
			value := strings.TrimSpace(fmt.Sprint(runtimeMeta["value"]))
			switch key {
			case "resolved_platform":
				result.ResolvedPlatform = value
			case "resolved_protocol":
				result.ResolvedSourceProtocol = normalizeTestSourceProtocol(value)
			}
		case "blacklist_advice":
			if advice, ok := event.Data.(map[string]any); ok {
				result.BlacklistAdviceDecision = strings.TrimSpace(fmt.Sprint(advice["decision"]))
			}
		case "error":
			result.ErrorMessage = event.Error
		}
	}
	result.ResponseText = strings.Join(texts, "")
	return result
}
