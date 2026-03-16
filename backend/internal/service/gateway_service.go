package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/util/responseheaders"
	"github.com/Wei-Shaw/sub2api/internal/util/urlvalidator"
	"github.com/gin-gonic/gin"
	gocache "github.com/patrickmn/go-cache"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"golang.org/x/sync/singleflight"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync/atomic"
	"time"
)

const (
	claudeAPIURL                 = "https://api.anthropic.com/v1/messages?beta=true"
	claudeAPICountTokensURL      = "https://api.anthropic.com/v1/messages/count_tokens?beta=true"
	stickySessionTTL             = time.Hour
	defaultMaxLineSize           = 500 * 1024 * 1024
	claudeCodeSystemPrompt       = "You are Claude Code, Anthropic's official CLI for Claude."
	maxCacheControlBlocks        = 4
	defaultUserGroupRateCacheTTL = 30 * time.Second
	defaultModelsListCacheTTL    = 15 * time.Second
)
const (
	claudeMimicDebugInfoKey = "claude_mimic_debug_info"
)

type forceCacheBillingKeyType struct{}
type accountWithLoad struct {
	account  *Account
	loadInfo *AccountLoadInfo
}

var ForceCacheBillingContextKey = forceCacheBillingKeyType{}

func cloneStringSlice(src []string) []string {
	if len(src) == 0 {
		return nil
	}
	dst := make([]string, len(src))
	copy(dst, src)
	return dst
}

var (
	sseDataRe                = regexp.MustCompile(`^data:\s*`)
	sessionIDRegex           = regexp.MustCompile(`session_([a-f0-9-]{36})`)
	claudeCliUserAgentRe     = regexp.MustCompile(`^claude-cli/\d+\.\d+\.\d+`)
	claudeCodePromptPrefixes = []string{"You are Claude Code, Anthropic's official CLI for Claude", "You are a Claude agent, built on Anthropic's Claude Agent SDK", "You are a file search specialist for Claude Code", "You are a helpful AI assistant tasked with summarizing conversations"}
)
var systemBlockFilterPrefixes = []string{"x-anthropic-billing-header"}
var ErrClaudeCodeOnly = errors.New("this group only allows Claude Code clients")
var ErrNoAvailableAccounts = errors.New("no available accounts")
var allowedHeaders = map[string]bool{"accept": true, "x-stainless-retry-count": true, "x-stainless-timeout": true, "x-stainless-lang": true, "x-stainless-package-version": true, "x-stainless-os": true, "x-stainless-arch": true, "x-stainless-runtime": true, "x-stainless-runtime-version": true, "x-stainless-helper-method": true, "anthropic-dangerous-direct-browser-access": true, "anthropic-version": true, "x-app": true, "anthropic-beta": true, "accept-language": true, "sec-fetch-mode": true, "user-agent": true, "content-type": true}

type GatewayCache interface {
	GetSessionAccountID(ctx context.Context, groupID int64, sessionHash string) (int64, error)
	SetSessionAccountID(ctx context.Context, groupID int64, sessionHash string, accountID int64, ttl time.Duration) error
	RefreshSessionTTL(ctx context.Context, groupID int64, sessionHash string, ttl time.Duration) error
	DeleteSessionAccountID(ctx context.Context, groupID int64, sessionHash string) error
}

func derefGroupID(groupID *int64) int64 {
	if groupID == nil {
		return 0
	}
	return *groupID
}
func prefetchedStickyGroupIDFromContext(ctx context.Context) (int64, bool) {
	return PrefetchedStickyGroupIDFromContext(ctx)
}
func prefetchedStickyAccountIDFromContext(ctx context.Context, groupID *int64) int64 {
	prefetchedGroupID, ok := prefetchedStickyGroupIDFromContext(ctx)
	if !ok || prefetchedGroupID != derefGroupID(groupID) {
		return 0
	}
	if accountID, ok := PrefetchedStickyAccountIDFromContext(ctx); ok && accountID > 0 {
		return accountID
	}
	return 0
}
func shouldClearStickySession(account *Account, requestedModel string) bool {
	if account == nil {
		return false
	}
	if account.Status == StatusError || account.Status == StatusDisabled || !account.Schedulable {
		return true
	}
	if account.TempUnschedulableUntil != nil && time.Now().Before(*account.TempUnschedulableUntil) {
		return true
	}
	if remaining := account.GetRateLimitRemainingTimeWithContext(context.Background(), requestedModel); remaining > 0 {
		return true
	}
	return false
}

type AccountWaitPlan struct {
	AccountID      int64
	MaxConcurrency int
	Timeout        time.Duration
	MaxWaiting     int
}
type AccountSelectionResult struct {
	Account     *Account
	Acquired    bool
	ReleaseFunc func()
	WaitPlan    *AccountWaitPlan
}
type ClaudeUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
	CacheCreation5mTokens    int
	CacheCreation1hTokens    int
}
type ForwardResult struct {
	RequestID        string
	Usage            ClaudeUsage
	Model            string
	ReasoningEffort  *string
	Stream           bool
	Duration         time.Duration
	FirstTokenMs     *int
	ClientDisconnect bool
	ImageCount       int
	ImageSize        string
	MediaType        string
	MediaURL         string
}
type UpstreamFailoverError struct {
	StatusCode             int
	ResponseBody           []byte
	ResponseHeaders        http.Header
	ForceCacheBilling      bool
	RetryableOnSameAccount bool
}

func (e *UpstreamFailoverError) Error() string {
	return fmt.Sprintf("upstream error: %d (failover)", e.StatusCode)
}
func (s *GatewayService) TempUnscheduleRetryableError(ctx context.Context, accountID int64, failoverErr *UpstreamFailoverError) {
	if failoverErr == nil || !failoverErr.RetryableOnSameAccount {
		return
	}
	switch failoverErr.StatusCode {
	case http.StatusBadRequest:
		tempUnscheduleGoogleConfigError(ctx, s.accountRepo, accountID, "[handler]")
	case http.StatusBadGateway:
		tempUnscheduleEmptyResponse(ctx, s.accountRepo, accountID, "[handler]")
	}
}

type GatewayService struct {
	accountRepo           AccountRepository
	groupRepo             GroupRepository
	usageLogRepo          UsageLogRepository
	usageBillingRepo      UsageBillingRepository
	userRepo              UserRepository
	userSubRepo           UserSubscriptionRepository
	userGroupRateRepo     UserGroupRateRepository
	cache                 GatewayCache
	digestStore           *DigestSessionStore
	cfg                   *config.Config
	schedulerSnapshot     *SchedulerSnapshotService
	billingService        *BillingService
	rateLimitService      *RateLimitService
	billingCacheService   *BillingCacheService
	identityService       *IdentityService
	httpUpstream          HTTPUpstream
	deferredService       *DeferredService
	concurrencyService    *ConcurrencyService
	claudeTokenProvider   *ClaudeTokenProvider
	sessionLimitCache     SessionLimitCache
	rpmCache              RPMCache
	userGroupRateResolver *userGroupRateResolver
	userGroupRateCache    *gocache.Cache
	userGroupRateSF       singleflight.Group
	modelsListCache       *gocache.Cache
	modelsListCacheTTL    time.Duration
	settingService        *SettingService
	modelRegistryService  *ModelRegistryService
	responseHeaderFilter  *responseheaders.CompiledHeaderFilter
	debugModelRouting     atomic.Bool
	debugClaudeMimic      atomic.Bool
}

func NewGatewayService(accountRepo AccountRepository, groupRepo GroupRepository, usageLogRepo UsageLogRepository, usageBillingRepo UsageBillingRepository, userRepo UserRepository, userSubRepo UserSubscriptionRepository, userGroupRateRepo UserGroupRateRepository, cache GatewayCache, cfg *config.Config, schedulerSnapshot *SchedulerSnapshotService, concurrencyService *ConcurrencyService, billingService *BillingService, rateLimitService *RateLimitService, billingCacheService *BillingCacheService, identityService *IdentityService, httpUpstream HTTPUpstream, deferredService *DeferredService, claudeTokenProvider *ClaudeTokenProvider, sessionLimitCache SessionLimitCache, rpmCache RPMCache, digestStore *DigestSessionStore, settingService *SettingService) *GatewayService {
	userGroupRateTTL := resolveUserGroupRateCacheTTL(cfg)
	modelsListTTL := resolveModelsListCacheTTL(cfg)
	svc := &GatewayService{accountRepo: accountRepo, groupRepo: groupRepo, usageLogRepo: usageLogRepo, usageBillingRepo: usageBillingRepo, userRepo: userRepo, userSubRepo: userSubRepo, userGroupRateRepo: userGroupRateRepo, cache: cache, digestStore: digestStore, cfg: cfg, schedulerSnapshot: schedulerSnapshot, concurrencyService: concurrencyService, billingService: billingService, rateLimitService: rateLimitService, billingCacheService: billingCacheService, identityService: identityService, httpUpstream: httpUpstream, deferredService: deferredService, claudeTokenProvider: claudeTokenProvider, sessionLimitCache: sessionLimitCache, rpmCache: rpmCache, userGroupRateCache: gocache.New(userGroupRateTTL, time.Minute), settingService: settingService, modelsListCache: gocache.New(modelsListTTL, time.Minute), modelsListCacheTTL: modelsListTTL, responseHeaderFilter: compileResponseHeaderFilter(cfg)}
	svc.userGroupRateResolver = newUserGroupRateResolver(userGroupRateRepo, svc.userGroupRateCache, userGroupRateTTL, &svc.userGroupRateSF, "service.gateway")
	svc.debugModelRouting.Store(parseDebugEnvBool(os.Getenv("SUB2API_DEBUG_MODEL_ROUTING")))
	svc.debugClaudeMimic.Store(parseDebugEnvBool(os.Getenv("SUB2API_DEBUG_CLAUDE_MIMIC")))
	return svc
}

func (s *GatewayService) SetModelRegistryService(modelRegistryService *ModelRegistryService) {
	s.modelRegistryService = modelRegistryService
}
func (s *GatewayService) GetAccessToken(ctx context.Context, account *Account) (string, string, error) {
	switch account.Type {
	case AccountTypeOAuth, AccountTypeSetupToken:
		return s.getOAuthToken(ctx, account)
	case AccountTypeAPIKey:
		apiKey := account.GetCredential("api_key")
		if apiKey == "" {
			return "", "", errors.New("api_key not found in credentials")
		}
		return apiKey, "apikey", nil
	default:
		return "", "", fmt.Errorf("unsupported account type: %s", account.Type)
	}
}
func (s *GatewayService) getOAuthToken(ctx context.Context, account *Account) (string, string, error) {
	if account.Platform == PlatformAnthropic && account.Type == AccountTypeOAuth && s.claudeTokenProvider != nil {
		accessToken, err := s.claudeTokenProvider.GetAccessToken(ctx, account)
		if err != nil {
			return "", "", err
		}
		return accessToken, "oauth", nil
	}
	accessToken := account.GetCredential("access_token")
	if accessToken == "" {
		return "", "", errors.New("access_token not found in credentials")
	}
	return accessToken, "oauth", nil
}

const (
	maxRetryAttempts = 5
	retryBaseDelay   = 300 * time.Millisecond
	retryMaxDelay    = 3 * time.Second
	maxRetryElapsed  = 10 * time.Second
)

func (s *GatewayService) shouldRetryUpstreamError(account *Account, statusCode int) bool {
	if account.IsOAuth() {
		return statusCode == 403
	}
	return !account.ShouldHandleErrorCode(statusCode)
}
func (s *GatewayService) shouldFailoverUpstreamError(statusCode int) bool {
	switch statusCode {
	case 401, 403, 429, 529:
		return true
	default:
		return statusCode >= 500
	}
}
func retryBackoffDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return retryBaseDelay
	}
	delay := retryBaseDelay * time.Duration(1<<(attempt-1))
	if delay > retryMaxDelay {
		return retryMaxDelay
	}
	return delay
}
func sleepWithContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer func() {
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
func (s *GatewayService) isThinkingBlockSignatureError(respBody []byte) bool {
	msg := strings.ToLower(strings.TrimSpace(extractUpstreamErrorMessage(respBody)))
	if msg == "" {
		return false
	}
	logger.LegacyPrintf("service.gateway", "[SignatureCheck] Checking error message: %s", msg)
	if strings.Contains(msg, "signature") {
		logger.LegacyPrintf("service.gateway", "[SignatureCheck] Detected signature error")
		return true
	}
	if strings.Contains(msg, "expected") && (strings.Contains(msg, "thinking") || strings.Contains(msg, "redacted_thinking")) {
		logger.LegacyPrintf("service.gateway", "[SignatureCheck] Detected thinking block type error")
		return true
	}
	if strings.Contains(msg, "cannot be modified") && (strings.Contains(msg, "thinking") || strings.Contains(msg, "redacted_thinking")) {
		logger.LegacyPrintf("service.gateway", "[SignatureCheck] Detected thinking block modification error")
		return true
	}
	if strings.Contains(msg, "non-empty content") || strings.Contains(msg, "empty content") {
		logger.LegacyPrintf("service.gateway", "[SignatureCheck] Detected empty content error")
		return true
	}
	return false
}
func (s *GatewayService) shouldFailoverOn400(respBody []byte) bool {
	msg := strings.ToLower(strings.TrimSpace(extractUpstreamErrorMessage(respBody)))
	if msg == "" {
		return false
	}
	if strings.Contains(msg, "anthropic-beta") || strings.Contains(msg, "beta feature") || strings.Contains(msg, "requires beta") {
		return true
	}
	if strings.Contains(msg, "thinking") || strings.Contains(msg, "thought_signature") || strings.Contains(msg, "signature") {
		return true
	}
	if strings.Contains(msg, "tool_use") || strings.Contains(msg, "tool_result") || strings.Contains(msg, "tools") {
		return true
	}
	return false
}
func ExtractUpstreamErrorMessage(body []byte) string {
	return extractUpstreamErrorMessage(body)
}
func extractUpstreamErrorMessage(body []byte) string {
	if m := gjson.GetBytes(body, "error.message").String(); strings.TrimSpace(m) != "" {
		inner := strings.TrimSpace(m)
		if strings.HasPrefix(inner, "{") {
			if innerMsg := gjson.Get(inner, "error.message").String(); strings.TrimSpace(innerMsg) != "" {
				return innerMsg
			}
		}
		return m
	}
	if d := gjson.GetBytes(body, "detail").String(); strings.TrimSpace(d) != "" {
		return d
	}
	return gjson.GetBytes(body, "message").String()
}

func extractUpstreamErrorCode(body []byte) string {
	if code := strings.TrimSpace(gjson.GetBytes(body, "error.code").String()); code != "" {
		return code
	}

	inner := strings.TrimSpace(gjson.GetBytes(body, "error.message").String())
	if !strings.HasPrefix(inner, "{") {
		return ""
	}

	if code := strings.TrimSpace(gjson.Get(inner, "error.code").String()); code != "" {
		return code
	}

	// Be defensive: some upstreams may append extra text after the JSON object.
	if lastBrace := strings.LastIndex(inner, "}"); lastBrace >= 0 {
		if code := strings.TrimSpace(gjson.Get(inner[:lastBrace+1], "error.code").String()); code != "" {
			return code
		}
	}

	return ""
}
func isCountTokensUnsupported404(statusCode int, body []byte) bool {
	if statusCode != http.StatusNotFound {
		return false
	}
	msg := strings.ToLower(strings.TrimSpace(extractUpstreamErrorMessage(body)))
	if msg == "" {
		return false
	}
	if strings.Contains(msg, "/v1/messages/count_tokens") {
		return true
	}
	return strings.Contains(msg, "count_tokens") && strings.Contains(msg, "not found")
}
func (s *GatewayService) replaceModelInResponseBody(body []byte, fromModel, toModel string) []byte {
	if m := gjson.GetBytes(body, "model"); m.Exists() && m.Str == fromModel {
		newBody, err := sjson.SetBytes(body, "model", toModel)
		if err != nil {
			return body
		}
		return newBody
	}
	return body
}
func (s *GatewayService) countTokensError(c *gin.Context, status int, errType, message string) {
	c.JSON(status, gin.H{"type": "error", "error": gin.H{"type": errType, "message": message}})
}
func (s *GatewayService) validateUpstreamBaseURL(raw string) (string, error) {
	if s.cfg != nil && !s.cfg.Security.URLAllowlist.Enabled {
		normalized, err := urlvalidator.ValidateURLFormat(raw, s.cfg.Security.URLAllowlist.AllowInsecureHTTP)
		if err != nil {
			return "", fmt.Errorf("invalid base_url: %w", err)
		}
		return normalized, nil
	}
	normalized, err := urlvalidator.ValidateHTTPSURL(raw, urlvalidator.ValidationOptions{AllowedHosts: s.cfg.Security.URLAllowlist.UpstreamHosts, RequireAllowlist: true, AllowPrivate: s.cfg.Security.URLAllowlist.AllowPrivateHosts})
	if err != nil {
		return "", fmt.Errorf("invalid base_url: %w", err)
	}
	return normalized, nil
}
