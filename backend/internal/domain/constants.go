package domain

// Status constants.
const (
	StatusActive   = "active"
	StatusDisabled = "disabled"
	StatusError    = "error"
	StatusUnused   = "unused"
	StatusUsed     = "used"
	StatusExpired  = "expired"
)

// Role constants.
const (
	RoleAdmin = "admin"
	RoleUser  = "user"
)

// Platform constants.
const (
	PlatformAnthropic       = "anthropic"
	PlatformOpenAI          = "openai"
	PlatformGemini          = "gemini"
	PlatformProtocolGateway = "protocol_gateway"
	PlatformAntigravity     = "antigravity"
	PlatformSora            = "sora"
	PlatformKiro            = "kiro"
	PlatformCopilot         = "copilot"
	PlatformGrok            = "grok"
)

// Account type constants.
const (
	AccountTypeOAuth      = "oauth"
	AccountTypeSetupToken = "setup-token"
	AccountTypeAPIKey     = "apikey"
	AccountTypeUpstream   = "upstream"
	AccountTypeBedrock    = "bedrock"
	AccountTypeSSO        = "sso"
)

// Redeem type constants.
const (
	RedeemTypeBalance      = "balance"
	RedeemTypeConcurrency  = "concurrency"
	RedeemTypeSubscription = "subscription"
	RedeemTypeInvitation   = "invitation"
)

// PromoCode status constants.
const (
	PromoCodeStatusActive   = "active"
	PromoCodeStatusDisabled = "disabled"
)

// Admin adjustment type constants.
const (
	AdjustmentTypeAdminBalance     = "admin_balance"
	AdjustmentTypeAdminConcurrency = "admin_concurrency"
)

// Group subscription type constants.
const (
	SubscriptionTypeStandard     = "standard"
	SubscriptionTypeSubscription = "subscription"
)

// Subscription status constants.
const (
	SubscriptionStatusActive    = "active"
	SubscriptionStatusExpired   = "expired"
	SubscriptionStatusSuspended = "suspended"
)

// DefaultAntigravityModelMapping is used when account model_mapping is unset.
var DefaultAntigravityModelMapping = map[string]string{
	"claude-opus-4-6-thinking":       "claude-opus-4-6-thinking",
	"claude-opus-4-6":                "claude-opus-4-6-thinking",
	"claude-opus-4-5-thinking":       "claude-opus-4-6-thinking",
	"claude-sonnet-4-6":              "claude-sonnet-4-6",
	"claude-sonnet-4.5":              "claude-sonnet-4-5",
	"claude-sonnet-4-5":              "claude-sonnet-4-5",
	"claude-sonnet-4-5-thinking":     "claude-sonnet-4-5-thinking",
	"claude-opus-4-5-20251101":       "claude-opus-4-6-thinking",
	"claude-sonnet-4-5-20250929":     "claude-sonnet-4-5",
	"claude-haiku-4.5":               "claude-sonnet-4-5",
	"claude-haiku-4-5":               "claude-sonnet-4-5",
	"claude-haiku-4-5-20251001":      "claude-sonnet-4-5",
	"gemini-2.5-flash":               "gemini-2.5-flash",
	"gemini-2.5-flash-image":         "gemini-2.5-flash-image",
	"gemini-2.5-flash-image-preview": "gemini-2.5-flash-image",
	"gemini-2.5-flash-lite":          "gemini-2.5-flash-lite",
	"gemini-2.5-flash-thinking":      "gemini-2.5-flash-thinking",
	"gemini-2.5-pro":                 "gemini-2.5-pro",
	"gemini-3-flash":                 "gemini-3-flash",
	"gemini-3-pro-high":              "gemini-3-pro-high",
	"gemini-3-pro-low":               "gemini-3-pro-low",
	"gemini-3-flash-preview":         "gemini-3-flash",
	"gemini-3-pro-preview":           "gemini-3-pro-high",
	"gemini-3.1-pro-high":            "gemini-3.1-pro-high",
	"gemini-3.1-pro-low":             "gemini-3.1-pro-low",
	"gemini-3.1-pro-preview":         "gemini-3.1-pro-high",
	"gemini-3.1-flash-image":         "gemini-3.1-flash-image",
	"gemini-3.1-flash-image-preview": "gemini-3.1-flash-image",
	"gemini-3-pro-image":             "gemini-3.1-flash-image",
	"gemini-3-pro-image-preview":     "gemini-3.1-flash-image",
	"gpt-oss-120b-medium":            "gpt-oss-120b-medium",
	"tab_flash_lite_preview":         "tab_flash_lite_preview",
}

// DefaultBedrockModelMapping maps Anthropic model ids to Bedrock model ids.
var DefaultBedrockModelMapping = map[string]string{
	"claude-opus-4-6-thinking":   "us.anthropic.claude-opus-4-6-v1",
	"claude-opus-4-6":            "us.anthropic.claude-opus-4-6-v1",
	"claude-opus-4-5-thinking":   "us.anthropic.claude-opus-4-5-20251101-v1:0",
	"claude-opus-4-5-20251101":   "us.anthropic.claude-opus-4-5-20251101-v1:0",
	"claude-opus-4-1":            "us.anthropic.claude-opus-4-1-20250805-v1:0",
	"claude-opus-4-20250514":     "us.anthropic.claude-opus-4-20250514-v1:0",
	"claude-sonnet-4-6-thinking": "us.anthropic.claude-sonnet-4-6",
	"claude-sonnet-4-6":          "us.anthropic.claude-sonnet-4-6",
	"claude-sonnet-4-5":          "us.anthropic.claude-sonnet-4-5-20250929-v1:0",
	"claude-sonnet-4-5-thinking": "us.anthropic.claude-sonnet-4-5-20250929-v1:0",
	"claude-sonnet-4-5-20250929": "us.anthropic.claude-sonnet-4-5-20250929-v1:0",
	"claude-sonnet-4-20250514":   "us.anthropic.claude-sonnet-4-20250514-v1:0",
	"claude-haiku-4-5":           "us.anthropic.claude-haiku-4-5-20251001-v1:0",
	"claude-haiku-4-5-20251001":  "us.anthropic.claude-haiku-4-5-20251001-v1:0",
}
