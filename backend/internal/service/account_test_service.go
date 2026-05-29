package service

import (
	"context"
	"errors"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

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

type AccountTestService struct {
	accountRepo                  AccountRepository
	accountModelImportService    *AccountModelImportService
	userRepo                     UserRepository
	apiKeyRepo                   APIKeyRepository
	usageLogRepo                 UsageLogRepository
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
	opsService                   *OpsService
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

func (s *AccountTestService) SetUsageLogDependencies(userRepo UserRepository, apiKeyRepo APIKeyRepository, usageLogRepo UsageLogRepository) {
	s.userRepo = userRepo
	s.apiKeyRepo = apiKeyRepo
	s.usageLogRepo = usageLogRepo
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

func (s *AccountTestService) validateUpstreamBaseURL(raw string) (string, error) {
	if s.cfg == nil {
		return "", errors.New("config is not available")
	}
	return validateUpstreamBaseURLWithConfig(s.cfg, raw)
}

// generateSessionString generates a Claude Code style session string.
// The output format is determined by the UA version in claude.DefaultHeaders,
// ensuring consistency between the user_id format and the UA sent to upstream.
