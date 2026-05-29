package service

import (
	"context"
	"math"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	userPlatformQuotaDailyWindow   = 24 * time.Hour
	userPlatformQuotaWeeklyWindow  = 7 * 24 * time.Hour
	userPlatformQuotaMonthlyWindow = 30 * 24 * time.Hour
)

var ErrUserPlatformQuotaExceeded = infraerrors.TooManyRequests("USER_PLATFORM_QUOTA_EXCEEDED", "user platform quota exceeded")

type UserPlatformQuota struct {
	ID                 int64
	UserID             int64
	Platform           string
	DailyLimitUSD      *float64
	WeeklyLimitUSD     *float64
	MonthlyLimitUSD    *float64
	DailyUsageUSD      float64
	WeeklyUsageUSD     float64
	MonthlyUsageUSD    float64
	DailyWindowStart   *time.Time
	WeeklyWindowStart  *time.Time
	MonthlyWindowStart *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type UserPlatformQuotaCycle struct {
	LimitUSD    *float64   `json:"limit"`
	UsageUSD    float64    `json:"used"`
	WindowStart *time.Time `json:"window_start,omitempty"`
	ResetAt     *time.Time `json:"reset_at,omitempty"`
}

type UserPlatformQuotaView struct {
	Platform string                 `json:"platform"`
	Daily    UserPlatformQuotaCycle `json:"daily"`
	Weekly   UserPlatformQuotaCycle `json:"weekly"`
	Monthly  UserPlatformQuotaCycle `json:"monthly"`
}

type UserPlatformQuotaInput struct {
	Platform        string   `json:"platform"`
	DailyLimitUSD   *float64 `json:"daily_limit_usd"`
	WeeklyLimitUSD  *float64 `json:"weekly_limit_usd"`
	MonthlyLimitUSD *float64 `json:"monthly_limit_usd"`
}

type UserPlatformQuotaRepository interface {
	ListByUser(ctx context.Context, userID int64) ([]UserPlatformQuota, error)
	ReplaceForUser(ctx context.Context, userID int64, items []UserPlatformQuotaInput) ([]UserPlatformQuota, error)
}

type UserPlatformQuotaService struct {
	repo UserPlatformQuotaRepository
}

func NewUserPlatformQuotaService(repo UserPlatformQuotaRepository) *UserPlatformQuotaService {
	return &UserPlatformQuotaService{repo: repo}
}

func (s *UserPlatformQuotaService) ListUserQuotas(ctx context.Context, userID int64) ([]UserPlatformQuotaView, error) {
	if s == nil || s.repo == nil || userID <= 0 {
		return []UserPlatformQuotaView{}, nil
	}
	items, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return BuildUserPlatformQuotaViews(items, time.Now().UTC()), nil
}

func (s *UserPlatformQuotaService) ReplaceUserQuotas(ctx context.Context, userID int64, input []UserPlatformQuotaInput) ([]UserPlatformQuotaView, error) {
	if s == nil || s.repo == nil || userID <= 0 {
		return []UserPlatformQuotaView{}, nil
	}
	normalized, err := NormalizeUserPlatformQuotaInputs(input)
	if err != nil {
		return nil, err
	}
	items, err := s.repo.ReplaceForUser(ctx, userID, normalized)
	if err != nil {
		return nil, err
	}
	return BuildUserPlatformQuotaViews(items, time.Now().UTC()), nil
}

func (s *UserPlatformQuotaService) CheckUserPlatformQuotaAllowed(ctx context.Context, userID int64, platform string) error {
	if s == nil || s.repo == nil || userID <= 0 {
		return nil
	}
	items, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return err
	}
	return CheckUserPlatformQuotaAllowed(items, platform, time.Now().UTC())
}

func NormalizeUserPlatformQuotaInputs(input []UserPlatformQuotaInput) ([]UserPlatformQuotaInput, error) {
	seen := make(map[string]int, len(input))
	out := make([]UserPlatformQuotaInput, 0, len(input))
	for _, item := range input {
		platform := CanonicalizePlatformValue(item.Platform)
		if platform == "" {
			return nil, infraerrors.BadRequest("INVALID_PLATFORM", "platform is required")
		}
		normalized := UserPlatformQuotaInput{
			Platform:        platform,
			DailyLimitUSD:   normalizeQuotaLimitPtr(item.DailyLimitUSD),
			WeeklyLimitUSD:  normalizeQuotaLimitPtr(item.WeeklyLimitUSD),
			MonthlyLimitUSD: normalizeQuotaLimitPtr(item.MonthlyLimitUSD),
		}
		if err := validateQuotaLimit(normalized.DailyLimitUSD); err != nil {
			return nil, err
		}
		if err := validateQuotaLimit(normalized.WeeklyLimitUSD); err != nil {
			return nil, err
		}
		if err := validateQuotaLimit(normalized.MonthlyLimitUSD); err != nil {
			return nil, err
		}
		if !hasAnyPlatformQuotaLimit(normalized) {
			continue
		}
		if idx, ok := seen[platform]; ok {
			out[idx] = normalized
			continue
		}
		seen[platform] = len(out)
		out = append(out, normalized)
	}
	return out, nil
}

func CheckUserPlatformQuotaAllowed(items []UserPlatformQuota, platform string, now time.Time) error {
	platform = NormalizeUserPlatformQuotaPlatform(platform)
	if platform == "" || len(items) == 0 {
		return nil
	}
	for _, item := range items {
		if NormalizeUserPlatformQuotaPlatform(item.Platform) != platform {
			continue
		}
		if exceeded, cycle := userPlatformQuotaExceeded(item, now); exceeded {
			return ErrUserPlatformQuotaExceeded.WithMetadata(map[string]string{
				"platform": platform,
				"cycle":    cycle,
			})
		}
		return nil
	}
	return nil
}

func BuildUserPlatformQuotaViews(items []UserPlatformQuota, now time.Time) []UserPlatformQuotaView {
	now = now.UTC()
	out := make([]UserPlatformQuotaView, 0, len(items))
	for _, item := range items {
		out = append(out, UserPlatformQuotaView{
			Platform: item.Platform,
			Daily:    quotaCycle(item.DailyLimitUSD, item.DailyUsageUSD, item.DailyWindowStart, userPlatformQuotaDailyWindow, now),
			Weekly:   quotaCycle(item.WeeklyLimitUSD, item.WeeklyUsageUSD, item.WeeklyWindowStart, userPlatformQuotaWeeklyWindow, now),
			Monthly:  quotaCycle(item.MonthlyLimitUSD, item.MonthlyUsageUSD, item.MonthlyWindowStart, userPlatformQuotaMonthlyWindow, now),
		})
	}
	return out
}

func NormalizeUserPlatformQuotaPlatform(platform string) string {
	return CanonicalizePlatformValue(strings.TrimSpace(platform))
}

func UserPlatformQuotaCostUSD(cost *CostBreakdown, isSubscriptionBill bool) float64 {
	if cost == nil {
		return 0
	}
	currency := normalizeBillingCurrency(cost.Currency)
	if isSubscriptionBill {
		if cost.TotalCostUSDEquivalent > 0 {
			return cost.TotalCostUSDEquivalent
		}
		return costUSDEquivalent(cost.TotalCost, currency, cost.USDToCNYRate)
	}
	if cost.ActualCostUSDEquivalent > 0 {
		return cost.ActualCostUSDEquivalent
	}
	return costUSDEquivalent(cost.ActualCost, currency, cost.USDToCNYRate)
}

func UserPlatformQuotaPlatformForGroup(group *Group) string {
	if group == nil {
		return ""
	}
	return NormalizeUserPlatformQuotaPlatform(group.Platform)
}

func UserPlatformQuotaPlatformForAccount(account *Account) string {
	return NormalizeUserPlatformQuotaPlatform(RoutingPlatformForAccount(account))
}

func normalizeQuotaLimitPtr(value *float64) *float64 {
	if value == nil {
		return nil
	}
	v := *value
	if v == 0 {
		return nil
	}
	normalized := math.Round(v*1e10) / 1e10
	return &normalized
}

func validateQuotaLimit(value *float64) error {
	if value == nil {
		return nil
	}
	if math.IsNaN(*value) || math.IsInf(*value, 0) || *value < 0 {
		return infraerrors.BadRequest("INVALID_QUOTA_LIMIT", "quota limit must be a non-negative number")
	}
	return nil
}

func hasAnyPlatformQuotaLimit(item UserPlatformQuotaInput) bool {
	return item.DailyLimitUSD != nil || item.WeeklyLimitUSD != nil || item.MonthlyLimitUSD != nil
}

func quotaCycle(limit *float64, usage float64, start *time.Time, window time.Duration, now time.Time) UserPlatformQuotaCycle {
	resetStart, resetUsage := resetWindowUsage(usage, start, window, now)
	var resetAt *time.Time
	if resetStart != nil {
		next := resetStart.Add(window).UTC()
		resetAt = &next
	}
	return UserPlatformQuotaCycle{
		LimitUSD:    cloneFloat64Ptr(limit),
		UsageUSD:    resetUsage,
		WindowStart: resetStart,
		ResetAt:     resetAt,
	}
}

func userPlatformQuotaExceeded(item UserPlatformQuota, now time.Time) (bool, string) {
	if cycleLimitExceeded(item.DailyLimitUSD, item.DailyUsageUSD, item.DailyWindowStart, userPlatformQuotaDailyWindow, now) {
		return true, "daily"
	}
	if cycleLimitExceeded(item.WeeklyLimitUSD, item.WeeklyUsageUSD, item.WeeklyWindowStart, userPlatformQuotaWeeklyWindow, now) {
		return true, "weekly"
	}
	if cycleLimitExceeded(item.MonthlyLimitUSD, item.MonthlyUsageUSD, item.MonthlyWindowStart, userPlatformQuotaMonthlyWindow, now) {
		return true, "monthly"
	}
	return false, ""
}

func cycleLimitExceeded(limit *float64, usage float64, start *time.Time, window time.Duration, now time.Time) bool {
	if limit == nil || *limit <= 0 {
		return false
	}
	_, resetUsage := resetWindowUsage(usage, start, window, now)
	return resetUsage >= *limit
}

func resetWindowUsage(usage float64, start *time.Time, window time.Duration, now time.Time) (*time.Time, float64) {
	if start == nil || start.IsZero() {
		return nil, 0
	}
	normalized := start.UTC()
	if normalized.Add(window).After(now) {
		return &normalized, usage
	}
	return nil, 0
}

func cloneFloat64Ptr(value *float64) *float64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
