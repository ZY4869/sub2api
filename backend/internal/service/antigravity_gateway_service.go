package service

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	antigravityStickySessionTTL                    = time.Hour
	antigravityMaxRetries                          = 3
	antigravityRetryBaseDelay                      = 1 * time.Second
	antigravityRetryMaxDelay                       = 16 * time.Second
	antigravityRateLimitThreshold                  = 7 * time.Second
	antigravitySmartRetryMinWait                   = 1 * time.Second
	antigravitySmartRetryMaxAttempts               = 1
	antigravityDefaultRateLimitDuration            = 30 * time.Second
	antigravityModelCapacityRetryMaxAttempts       = 60
	antigravityModelCapacityRetryWait              = 1 * time.Second
	googleRPCStatusResourceExhausted               = "RESOURCE_EXHAUSTED"
	googleRPCStatusUnavailable                     = "UNAVAILABLE"
	googleRPCTypeRetryInfo                         = "type.googleapis.com/google.rpc.RetryInfo"
	googleRPCTypeErrorInfo                         = "type.googleapis.com/google.rpc.ErrorInfo"
	googleRPCReasonModelCapacityExhausted          = "MODEL_CAPACITY_EXHAUSTED"
	googleRPCReasonRateLimitExceeded               = "RATE_LIMIT_EXCEEDED"
	antigravitySingleAccountSmartRetryMaxAttempts  = 3
	antigravitySingleAccountSmartRetryMaxWait      = 15 * time.Second
	antigravitySingleAccountSmartRetryTotalMaxWait = 30 * time.Second
	antigravityModelCapacityCooldown               = 10 * time.Second
)

var antigravityPassthroughErrorMessages = []string{"prompt is too long"}

var (
	modelCapacityExhaustedMu    sync.RWMutex
	modelCapacityExhaustedUntil = make(map[string]time.Time)
)

const (
	antigravityForwardBaseURLEnv  = "GATEWAY_ANTIGRAVITY_FORWARD_BASE_URL"
	antigravityFallbackSecondsEnv = "GATEWAY_ANTIGRAVITY_FALLBACK_COOLDOWN_SECONDS"
)

type AntigravityAccountSwitchError struct {
	OriginalAccountID int64
	RateLimitedModel  string
	IsStickySession   bool
}

func (e *AntigravityAccountSwitchError) Error() string {
	return fmt.Sprintf("account %d model %s rate limited, need switch", e.OriginalAccountID, e.RateLimitedModel)
}

func IsAntigravityAccountSwitchError(err error) (*AntigravityAccountSwitchError, bool) {
	var switchErr *AntigravityAccountSwitchError
	if errors.As(err, &switchErr) {
		return switchErr, true
	}
	return nil, false
}

type PromptTooLongError struct {
	StatusCode int
	RequestID  string
	Body       []byte
}

func (e *PromptTooLongError) Error() string {
	return fmt.Sprintf("prompt too long: status=%d", e.StatusCode)
}

func shouldRetryAntigravityError(statusCode int) bool {
	switch statusCode {
	case 429, 500, 502, 503, 504, 529:
		return true
	default:
		return false
	}
}

func isURLLevelRateLimit(body []byte) bool {
	bodyStr := string(body)
	return strings.Contains(bodyStr, "Resource has been exhausted") && !strings.Contains(bodyStr, "capacity on this model")
}

func isAntigravityConnectionError(err error) bool {
	if err == nil {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	var opErr *net.OpError
	return errors.As(err, &opErr)
}

func shouldAntigravityFallbackToNextURL(err error, statusCode int) bool {
	if isAntigravityConnectionError(err) {
		return true
	}
	return statusCode == http.StatusTooManyRequests
}

func getSessionID(c *gin.Context) string {
	if c == nil {
		return ""
	}
	return c.GetHeader("session_id")
}

func logPrefix(sessionID, accountName string) string {
	if sessionID != "" {
		return fmt.Sprintf("[antigravity-Forward] session=%s account=%s", sessionID, accountName)
	}
	return fmt.Sprintf("[antigravity-Forward] account=%s", accountName)
}

type AntigravityGatewayService struct {
	accountRepo       AccountRepository
	tokenProvider     *AntigravityTokenProvider
	rateLimitService  *RateLimitService
	httpUpstream      HTTPUpstream
	settingService    *SettingService
	cache             GatewayCache
	schedulerSnapshot *SchedulerSnapshotService
}

func NewAntigravityGatewayService(accountRepo AccountRepository, cache GatewayCache, schedulerSnapshot *SchedulerSnapshotService, tokenProvider *AntigravityTokenProvider, rateLimitService *RateLimitService, httpUpstream HTTPUpstream, settingService *SettingService) *AntigravityGatewayService {
	return &AntigravityGatewayService{
		accountRepo:       accountRepo,
		tokenProvider:     tokenProvider,
		rateLimitService:  rateLimitService,
		httpUpstream:      httpUpstream,
		settingService:    settingService,
		cache:             cache,
		schedulerSnapshot: schedulerSnapshot,
	}
}

func (s *AntigravityGatewayService) GetTokenProvider() *AntigravityTokenProvider {
	return s.tokenProvider
}

func (s *AntigravityGatewayService) getDefaultRateLimitDuration() time.Duration {
	defaultDur := antigravityDefaultRateLimitDuration
	if s.settingService != nil && s.settingService.cfg != nil && s.settingService.cfg.Gateway.AntigravityFallbackCooldownMinutes > 0 {
		defaultDur = time.Duration(s.settingService.cfg.Gateway.AntigravityFallbackCooldownMinutes) * time.Minute
	}
	if override, ok := antigravityFallbackCooldownSeconds(); ok {
		defaultDur = override
	}
	return defaultDur
}

func (s *AntigravityGatewayService) resolveResetTime(resetAt *int64, defaultDur time.Duration) time.Time {
	if resetAt != nil {
		return time.Unix(*resetAt, 0)
	}
	return time.Now().Add(defaultDur)
}
