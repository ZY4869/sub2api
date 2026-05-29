package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

func (s *RateLimitService) GetTempUnschedStatus(ctx context.Context, accountID int64) (*TempUnschedState, error) {
	now := time.Now().Unix()
	if s.tempUnschedCache != nil {
		state, err := s.tempUnschedCache.GetTempUnsched(ctx, accountID)
		if err != nil {
			return nil, err
		}
		if state != nil && state.UntilUnix > now {
			return state, nil
		}
	}

	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if account.TempUnschedulableUntil == nil {
		return nil, nil
	}
	if account.TempUnschedulableUntil.Unix() <= now {
		return nil, nil
	}

	state := &TempUnschedState{
		UntilUnix: account.TempUnschedulableUntil.Unix(),
	}

	if account.TempUnschedulableReason != "" {
		var parsed TempUnschedState
		if err := json.Unmarshal([]byte(account.TempUnschedulableReason), &parsed); err == nil {
			if parsed.UntilUnix == 0 {
				parsed.UntilUnix = state.UntilUnix
			}
			state = &parsed
		} else {
			state.ErrorMessage = account.TempUnschedulableReason
		}
	}

	if s.tempUnschedCache != nil {
		if err := s.tempUnschedCache.SetTempUnsched(ctx, accountID, state); err != nil {
			slog.Warn("temp_unsched_cache_set_failed", "account_id", accountID, "error", err)
		}
	}

	return state, nil
}

func (s *RateLimitService) HandleTempUnschedulable(ctx context.Context, account *Account, statusCode int, responseBody []byte) bool {
	if account == nil {
		return false
	}
	if !account.ShouldHandleErrorCode(statusCode) {
		return false
	}
	return s.tryTempUnschedulable(ctx, account, statusCode, responseBody)
}

const tempUnschedBodyMaxBytes = 64 << 10
const tempUnschedMessageMaxBytes = 2048

func (s *RateLimitService) tryTempUnschedulable(ctx context.Context, account *Account, statusCode int, responseBody []byte) bool {
	if account == nil {
		return false
	}
	if !account.IsTempUnschedulableEnabled() {
		return false
	}
	// 401 首次命中可临时不可调度（给 token 刷新窗口）；
	// 若历史上已因 401 进入过临时不可调度，则本次应升级为 error（返回 false 交由默认错误逻辑处理）。
	// Antigravity 跳过：其 401 由 applyErrorPolicy 的 temp_unschedulable_rules 自行控制，无需升级逻辑。
	if statusCode == http.StatusUnauthorized && EffectiveProtocol(account) != PlatformAntigravity {
		reason := account.TempUnschedulableReason
		// 缓存可能没有 reason，从 DB 回退读取
		if reason == "" {
			if dbAcc, err := s.accountRepo.GetByID(ctx, account.ID); err == nil && dbAcc != nil {
				reason = dbAcc.TempUnschedulableReason
			}
		}
		if wasTempUnschedByStatusCode(reason, statusCode) {
			slog.Info("401_escalated_to_error", "account_id", account.ID,
				"reason", "previous temp-unschedulable was also 401")
			return false
		}
	}
	rules := account.GetTempUnschedulableRules()
	if len(rules) == 0 {
		return false
	}
	if statusCode <= 0 || len(responseBody) == 0 {
		return false
	}

	body := responseBody
	if len(body) > tempUnschedBodyMaxBytes {
		body = body[:tempUnschedBodyMaxBytes]
	}
	bodyLower := strings.ToLower(string(body))

	for idx, rule := range rules {
		if rule.ErrorCode != statusCode || len(rule.Keywords) == 0 {
			continue
		}
		matchedKeyword := matchTempUnschedKeyword(bodyLower, rule.Keywords)
		if matchedKeyword == "" {
			continue
		}

		if s.triggerTempUnschedulable(ctx, account, rule, idx, statusCode, matchedKeyword, responseBody) {
			return true
		}
	}

	return false
}
