package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/handler/admin"
)

// AdminHandlers contains all admin-related HTTP handlers
type AdminHandlers struct {
	Dashboard              *admin.DashboardHandler
	User                   *admin.UserHandler
	Group                  *admin.GroupHandler
	Channel                *admin.ChannelHandler
	ChannelMonitor         *admin.ChannelMonitorHandler
	ChannelMonitorTemplate *admin.ChannelMonitorTemplateHandler
	Account                *admin.AccountHandler
	Affiliate              *admin.AffiliateHandler
	Docs                   *admin.DocsHandler
	Announcement           *admin.AnnouncementHandler
	DataManagement         *admin.DataManagementHandler
	Backup                 *admin.BackupHandler
	OAuth                  *admin.OAuthHandler
	OpenAIOAuth            *admin.OpenAIOAuthHandler
	KiroOAuth              *admin.KiroOAuthHandler
	GeminiOAuth            *admin.GeminiOAuthHandler
	AntigravityOAuth       *admin.AntigravityOAuthHandler
	Proxy                  *admin.ProxyHandler
	Redeem                 *admin.RedeemHandler
	Promo                  *admin.PromoHandler
	Setting                *admin.SettingHandler
	Ops                    *admin.OpsHandler
	System                 *admin.SystemHandler
	Subscription           *admin.SubscriptionHandler
	Usage                  *admin.UsageHandler
	UserAttribute          *admin.UserAttributeHandler
	ErrorPassthrough       *admin.ErrorPassthroughHandler
	APIKey                 *admin.AdminAPIKeyHandler
	ModelCatalog           *admin.ModelCatalogHandler
	ModelRegistry          *admin.ModelRegistryHandler
	ScheduledTest          *admin.ScheduledTestHandler
	TLSFingerprintProfile  *admin.TLSFingerprintProfileHandler
}

// Handlers contains all HTTP handlers
type Handlers struct {
	Auth             *AuthHandler
	User             *UserHandler
	Meta             *MetaHandler
	APIKey           *APIKeyHandler
	AvailableChannel *AvailableChannelHandler
	ChannelMonitor   *ChannelMonitorHandler
	Usage            *UsageHandler
	Redeem           *RedeemHandler
	Subscription     *SubscriptionHandler
	Announcement     *AnnouncementHandler
	Docs             *DocsHandler
	Admin            *AdminHandlers
	Gateway          *GatewayHandler
	OpenAIGateway    *OpenAIGatewayHandler
	GrokGateway      *GrokGatewayHandler
	DocumentAI       *DocumentAIHandler
	Setting          *SettingHandler
	Totp             *TotpHandler
}

// BuildInfo contains build-time information
type BuildInfo struct {
	Version   string
	BuildType string // "source" for manual builds, "release" for CI builds
}
