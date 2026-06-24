package service

import (
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID                              int64
	Email                           string
	Username                        string
	Notes                           string
	PasswordHash                    string
	Role                            string
	Balance                         float64
	Balances                        map[string]float64
	Concurrency                     int
	Status                          string
	Deleted                         bool
	AdminFreeBilling                bool
	RequestDetailsReview            bool
	UsageModelDisplayMode           string
	UsageViewPreferences            UsageViewPreferences
	GlobalRealtimeCountdownEnabled  bool
	AccountRealtimeCountdownEnabled bool
	VisualPresetPreference          string
	AccountVisualPresetOverride     string
	AccountTodayStatsWindows        []string
	AccountTodayStatsCycleMode      string
	AccountGroupDisplayMode         string
	AccountStatusDisplayMode        string
	APIKeyModelBindingMode          string
	ExternalModelCatalogViewMode    string
	APIKeyAccessTimePolicy          *TimeAccessPolicy
	AllowedGroups                   []int64
	TokenVersion                    int64 // Incremented on password change to invalidate existing tokens
	CreatedAt                       time.Time
	UpdatedAt                       time.Time

	// GroupRates 用户专属分组倍率配置
	// map[groupID]rateMultiplier
	GroupRates map[int64]float64

	// TOTP 双因素认证字段
	TotpSecretEncrypted *string    // AES-256-GCM 加密的 TOTP 密钥
	TotpEnabled         bool       // 是否启用 TOTP
	TotpEnabledAt       *time.Time // TOTP 启用时间

	APIKeys       []APIKey
	Subscriptions []UserSubscription
}

func (u *User) IsAdmin() bool {
	return u != nil && u.Role == RoleAdmin
}

func (u *User) IsActive() bool {
	return u.Status == StatusActive
}

func (u *User) IsAdminFreeBillingEnabled() bool {
	return u != nil && u.Role == RoleAdmin && u.AdminFreeBilling
}

func (u *User) CanReviewRequestDetails() bool {
	return u != nil && (u.IsAdmin() || u.RequestDetailsReview)
}

func (u *User) HasUsableBillingBalance() bool {
	if u == nil {
		return false
	}
	for currency, amount := range u.Balances {
		if normalizeBillingCurrency(currency) != "" && amount > 0 {
			return true
		}
	}
	return u.Balance > 0
}

func (u *User) EffectiveUsageModelDisplayMode() string {
	if u == nil {
		return UsageModelDisplayModeModelOnly
	}
	return NormalizeUserUsageModelDisplayMode(u.UsageModelDisplayMode)
}

func (u *User) EffectiveVisualPreset(siteDefault string) string {
	if u == nil {
		return NormalizeVisualPreset(siteDefault)
	}
	return ResolveVisualPreset(siteDefault, u.VisualPresetPreference, u.AccountVisualPresetOverride)
}

const (
	APIKeyModelBindingModeModelRequired = "model_required"
	APIKeyModelBindingModeGroupAllowed  = "group_allowed"
)

const (
	ExternalModelCatalogViewModeFollowKeyBinding = "follow_key_binding"
	ExternalModelCatalogViewModeGroupFirst       = "group_first"
	ExternalModelCatalogViewModeModelOnly        = "model_only"
)

func NormalizeAPIKeyModelBindingMode(value string) string {
	switch strings.TrimSpace(value) {
	case APIKeyModelBindingModeGroupAllowed:
		return APIKeyModelBindingModeGroupAllowed
	default:
		return APIKeyModelBindingModeModelRequired
	}
}

func (u *User) EffectiveAPIKeyModelBindingMode() string {
	if u == nil {
		return APIKeyModelBindingModeModelRequired
	}
	return NormalizeAPIKeyModelBindingMode(u.APIKeyModelBindingMode)
}

func NormalizeExternalModelCatalogViewMode(value string) string {
	switch strings.TrimSpace(value) {
	case ExternalModelCatalogViewModeGroupFirst:
		return ExternalModelCatalogViewModeGroupFirst
	case ExternalModelCatalogViewModeModelOnly:
		return ExternalModelCatalogViewModeModelOnly
	default:
		return ExternalModelCatalogViewModeFollowKeyBinding
	}
}

func ValidateExternalModelCatalogViewMode(value string) error {
	switch strings.TrimSpace(value) {
	case ExternalModelCatalogViewModeFollowKeyBinding,
		ExternalModelCatalogViewModeGroupFirst,
		ExternalModelCatalogViewModeModelOnly:
		return nil
	default:
		return infraerrors.BadRequest(
			"EXTERNAL_MODEL_CATALOG_VIEW_MODE_INVALID",
			"external_model_catalog_view_mode must be follow_key_binding, group_first, or model_only",
		)
	}
}

func (u *User) EffectiveExternalModelCatalogViewMode() string {
	if u == nil {
		return ExternalModelCatalogViewModeModelOnly
	}
	switch NormalizeExternalModelCatalogViewMode(u.ExternalModelCatalogViewMode) {
	case ExternalModelCatalogViewModeGroupFirst:
		return ExternalModelCatalogViewModeGroupFirst
	case ExternalModelCatalogViewModeModelOnly:
		return ExternalModelCatalogViewModeModelOnly
	default:
		if u.EffectiveAPIKeyModelBindingMode() == APIKeyModelBindingModeGroupAllowed {
			return ExternalModelCatalogViewModeGroupFirst
		}
		return ExternalModelCatalogViewModeModelOnly
	}
}

func EffectiveExternalModelCatalogViewMode(user *User) string {
	if user == nil {
		return ExternalModelCatalogViewModeModelOnly
	}
	return user.EffectiveExternalModelCatalogViewMode()
}

// CanBindGroup checks whether a user can bind to a given group.
// For standard groups:
// - Public groups (non-exclusive): all users can bind
// - Exclusive groups: only users with the group in AllowedGroups can bind
func (u *User) CanBindGroup(groupID int64, isExclusive bool) bool {
	// 公开分组（非专属）：所有用户都可以绑定
	if !isExclusive {
		return true
	}
	// 专属分组：需要在 AllowedGroups 中
	for _, id := range u.AllowedGroups {
		if id == groupID {
			return true
		}
	}
	return false
}

func (u *User) SetPassword(password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(hash)
	return nil
}

func (u *User) CheckPassword(password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) == nil
}
