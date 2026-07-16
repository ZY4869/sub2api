package service

import (
	"context"
	"fmt"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// AccountUsageService 账号使用量查询服务
type AccountUsageService struct {
	accountRepo                           AccountRepository
	usageLogRepo                          UsageLogRepository
	usageFetcher                          ClaudeUsageFetcher
	geminiQuotaService                    *GeminiQuotaService
	antigravityQuotaFetcher               *AntigravityQuotaFetcher
	grokQuotaService                      *GrokQuotaService
	grokQuotaFetcher                      *GrokQuotaFetcher
	cache                                 *UsageCache
	identityCache                         IdentityCache
	openAICodexProbe                      func(ctx context.Context, account *Account) (map[string]any, *time.Time, error)
	openAICodexScopeProbe                 func(ctx context.Context, account *Account, modelID string) (map[string]any, *time.Time, error)
	openAICodexScopeProbeHTTP             func(ctx context.Context, account *Account, modelID string) (map[string]any, *time.Time, error)
	openAICodexWSProbeDialer              openAIWSClientDialer
	openAICodexWSProbeReadTimeoutOverride time.Duration
	openAIResetCreditService              OpenAIResetCreditReader
	tlsFingerprintProfileService          *TLSFingerprintProfileService
}

// NewAccountUsageService 创建AccountUsageService实例
func NewAccountUsageService(
	accountRepo AccountRepository,
	usageLogRepo UsageLogRepository,
	usageFetcher ClaudeUsageFetcher,
	geminiQuotaService *GeminiQuotaService,
	antigravityQuotaFetcher *AntigravityQuotaFetcher,
	cache *UsageCache,
	identityCache IdentityCache,
) *AccountUsageService {
	return &AccountUsageService{
		accountRepo:             accountRepo,
		usageLogRepo:            usageLogRepo,
		usageFetcher:            usageFetcher,
		geminiQuotaService:      geminiQuotaService,
		antigravityQuotaFetcher: antigravityQuotaFetcher,
		cache:                   cache,
		identityCache:           identityCache,
	}
}

func (s *AccountUsageService) SetGrokQuotaDependencies(quotaService *GrokQuotaService, quotaFetcher *GrokQuotaFetcher) {
	if s == nil {
		return
	}
	s.grokQuotaService = quotaService
	s.grokQuotaFetcher = quotaFetcher
}

func (s *AccountUsageService) SetTLSFingerprintProfileService(tlsFingerprintProfileService *TLSFingerprintProfileService) {
	s.tlsFingerprintProfileService = tlsFingerprintProfileService
}

func (s *AccountUsageService) SetOpenAIResetCreditService(resetCreditService OpenAIResetCreditReader) {
	s.openAIResetCreditService = resetCreditService
}

// GetUsage 获取账号使用量
// OAuth账号: 调用Anthropic API获取真实数据（需要profile scope），API响应缓存10分钟，窗口统计缓存1分钟
// Setup Token账号: 根据session_window推算5h窗口，7d数据不可用（没有profile scope）
// API Key账号: 不支持usage查询
func (s *AccountUsageService) GetUsage(ctx context.Context, accountID int64, force bool) (*UsageInfo, error) {
	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("get account failed: %w", err)
	}
	runtimePlatform := EffectiveProtocol(account)

	if runtimePlatform == PlatformOpenAI && account.Type == AccountTypeOAuth {
		usage, err := s.getOpenAIUsage(ctx, account, force)
		if err == nil {
			s.tryClearRecoverableAccountError(ctx, account)
		}
		return usage, err
	}

	if runtimePlatform == PlatformGemini {
		usage, err := s.getGeminiUsage(ctx, account)
		if err == nil {
			s.tryClearRecoverableAccountError(ctx, account)
		}
		return usage, err
	}

	// Antigravity 平台：使用 AntigravityQuotaFetcher 获取额度
	if account.Platform == PlatformAntigravity {
		usage, err := s.getAntigravityUsage(ctx, account, force)
		if err == nil {
			s.tryClearRecoverableAccountError(ctx, account)
		}
		return usage, err
	}

	if runtimePlatform == PlatformGrok && account.Type == AccountTypeOAuth {
		usage, err := s.getGrokUsage(ctx, account, force)
		if err == nil {
			s.tryClearRecoverableAccountError(ctx, account)
		}
		return usage, err
	}

	// 只有oauth类型账号可以通过API获取usage（有profile scope）
	if account.CanGetUsage() {
		return s.getAnthropicActiveUsage(ctx, account, force)
	}

	// Setup Token账号：根据session_window推算（没有profile scope，无法调用usage API）
	if account.Type == AccountTypeSetupToken {
		usage := s.estimateSetupTokenUsageWithContext(ctx, account)
		// 添加窗口统计
		s.addWindowStats(ctx, account, usage, force)
		return usage, nil
	}

	if runtimePlatform == PlatformKiro && account.Type == AccountTypeOAuth {
		return nil, infraerrors.BadRequest("ACCOUNT_USAGE_UNSUPPORTED", "kiro oauth accounts do not support active usage query")
	}

	// API Key账号不支持usage查询
	return nil, infraerrors.BadRequest(
		"ACCOUNT_USAGE_UNSUPPORTED",
		fmt.Sprintf("account type %s does not support active usage query", account.Type),
	)
}

func (s *AccountUsageService) getGrokUsage(ctx context.Context, account *Account, force bool) (*UsageInfo, error) {
	if s == nil || account == nil {
		return nil, fmt.Errorf("account is nil")
	}
	if s.grokQuotaService != nil {
		var result *GrokQuotaProbeResult
		var err error
		if force {
			result, err = s.grokQuotaService.QueryQuota(ctx, account.ID)
		} else {
			result, err = s.grokQuotaService.ProbeBilling(ctx, account.ID)
		}
		if err == nil && result != nil {
			account, _ = s.accountRepo.GetByID(ctx, account.ID)
		} else if force {
			return nil, err
		}
	}
	if s.grokQuotaFetcher == nil {
		s.grokQuotaFetcher = NewGrokQuotaFetcher()
	}
	usage := s.grokQuotaFetcher.BuildUsageInfo(account)
	usage.Source = "active"
	if usage.GrokBilling != nil {
		usage.GrokLocalUsage24h, usage.GrokLocalUsage7d, usage.GrokLocalUsageMonthly = grokLocalUsageForQuota(ctx, s.usageLogRepo, account.ID, usage.GrokBilling, time.Now().UTC())
	}
	return usage, nil
}
