package service

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

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
	s.captureUpstreamFailure(c, statusCode, body)
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
	if IsUpstreamRedirectBlockedResponse(nil, body) {
		return UpstreamRedirectBlockedMessage, nil
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
		if ShouldAccountNeedReauth(account, advice.ReasonCode) {
			_, expired := MarkAccountNeedsReauth(ctx, s.accountRepo, account, firstNonEmptyHardBanString(advice.ReasonMessage, message), time.Now())
			if expired {
				advice.Decision = BlacklistAdviceAutoBlacklisted
				advice.ReasonCode = AccountReauthDeadlineExpiredCode
				advice.AlreadyBlacklisted = true
				advice.CollectFeedback = false
			}
			return message, advice
		}
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
