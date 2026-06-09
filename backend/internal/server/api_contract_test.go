//go:build unit

package server_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	adminhandler "github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAPIContracts(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		setup      func(t *testing.T, deps *contractDeps)
		method     string
		path       string
		body       string
		headers    map[string]string
		wantStatus int
		wantJSON   string
	}{
		{
			name:       "GET /api/v1/auth/me",
			method:     http.MethodGet,
			path:       "/api/v1/auth/me",
			wantStatus: http.StatusOK,
			wantJSON: `{
				"code": 0,
				"message": "success",
				"data": {
					"id": 1,
					"email": "alice@example.com",
					"username": "alice",
					"role": "user",
					"balance": 12.5,
					"concurrency": 5,
					"status": "active",
					"usage_model_display_mode": "model_only",
					"global_realtime_countdown_enabled": false,
					"account_realtime_countdown_enabled": true,
					"visual_preset_preference": "inherit",
					"account_visual_preset_override": "inherit",
					"account_today_stats_windows": ["today", "weekly", "total"],
					"account_group_display_mode": "full",
					"api_key_model_binding_mode": "model_required",
					"api_key_access_time_policy": null,
					"allowed_groups": null,
 					"created_at": "2025-01-02T03:04:05Z",
 					"updated_at": "2025-01-02T03:04:05Z",
					"run_mode": "standard",
					"admin_free_billing": false,
					"request_details_review": false
 				}
 			}`,
		},
		{
			name: "GET /api/v1/settings/public",
			setup: func(t *testing.T, deps *contractDeps) {
				t.Helper()
				deps.settingRepo.SetAll(map[string]string{
					service.SettingKeyRegistrationEnabled:              "true",
					service.SettingKeyEmailVerifyEnabled:               "false",
					service.SettingKeyRegistrationEmailSuffixWhitelist: "[]",
					service.SettingKeyPromoCodeEnabled:                 "true",

					service.SettingKeyTurnstileEnabled: "true",
					service.SettingKeyTurnstileSiteKey: "site-key",

					service.SettingKeySiteName:                       "Sub2API",
					service.SettingKeySiteLogo:                       "",
					service.SettingKeySiteSubtitle:                   "Subtitle",
					service.SettingKeyVisualPresetDefault:            "airy",
					service.SettingKeyAccountAiryWhiteSurfaceEnabled: "true",
					service.SettingKeyAPIBaseURL:                     "https://api.example.com",
					service.SettingKeyContactInfo:                    "support",
					service.SettingKeyDocURL:                         "https://docs.example.com",
					service.SettingKeyCustomMenuItems:                `[{"id":"page-public","label":"Guide","icon_svg":"","url":"","visibility":"user","sort_order":0,"page_mode":"markdown","page_slug":"guide","page_content":"# hidden","page_published":true},{"id":"page-draft","label":"Draft","icon_svg":"","url":"","visibility":"user","sort_order":1,"page_mode":"markdown","page_slug":"draft","page_content":"# hidden","page_published":false},{"id":"page-admin","label":"Admin","icon_svg":"","url":"https://admin.example.com","visibility":"admin","sort_order":2,"page_mode":"iframe"}]`,
					service.SettingKeyLoginAgreementEnabled:          "true",
					service.SettingKeyLoginAgreementMode:             "checkbox",
					service.SettingKeyLoginAgreementUpdatedAt:        "2026-05-07",
					service.SettingKeyLoginAgreementDocuments:        `[{"id":"terms","title":"Terms","page_slug":"terms"}]`,
				})
			},
			method:     http.MethodGet,
			path:       "/api/v1/settings/public",
			wantStatus: http.StatusOK,
			wantJSON: `{
				"code": 0,
				"message": "success",
				"data": {
					"registration_enabled": true,
					"email_verify_enabled": false,
					"registration_email_suffix_whitelist": [],
					"promo_code_enabled": true,
					"password_reset_enabled": false,
					"invitation_code_enabled": false,
					"totp_enabled": false,
					"turnstile_enabled": true,
					"turnstile_site_key": "site-key",
					"site_name": "Sub2API",
					"site_logo": "",
					"site_subtitle": "Subtitle",
					"visual_preset_default": "airy",
					"account_airy_white_surface_enabled": true,
					"api_base_url": "https://api.example.com",
					"contact_info": "support",
					"doc_url": "https://docs.example.com",
					"home_content": "",
					"hide_ccs_import_button": false,
					"available_channels_enabled": false,
					"channel_monitor_enabled": false,
					"public_model_catalog_enabled": true,
					"affiliate_enabled": false,
					"purchase_subscription_enabled": false,
					"purchase_subscription_url": "",
					"payment_provider_airwallex_enabled": false,
					"payment_allowed_currencies": ["USD", "CNY", "HKD"],
					"payment_default_currency": "USD",
					"payment_min_topup_amount": 1,
					"payment_max_topup_amount": 5000,
					"payment_mobile_force_qrcode_enabled": false,
					"payment_subscription_plans": [],
					"custom_menu_items": [
						{
							"id": "page-public",
							"label": "Guide",
							"icon_svg": "",
							"url": "",
							"visibility": "user",
							"sort_order": 0,
							"page_mode": "markdown",
							"page_slug": "guide",
							"page_published": true
						}
					],
					"login_agreement_enabled": true,
					"login_agreement_mode": "checkbox",
					"login_agreement_updated_at": "2026-05-07",
					"login_agreement_documents": [
						{
							"id": "terms",
							"title": "Terms",
							"page_slug": "terms"
						}
					],
					"linuxdo_oauth_enabled": false,
					"github_oauth_enabled": false,
					"google_oauth_enabled": false,
					"dingtalk_oauth_enabled": false,
					"backend_mode_enabled": false,
					"maintenance_mode_enabled": false,
					"version": "0.0.0-test"
				}
			}`,
		},
		{
			name: "GET /api/v1/settings/public exposes effective Airwallex flag without webhook secret",
			setup: func(t *testing.T, deps *contractDeps) {
				t.Helper()
				deps.settingRepo.SetAll(map[string]string{
					service.SettingKeyPurchaseSubscriptionEnabled:     "true",
					service.SettingKeyPaymentProviderAirwallexEnabled: "true",
					service.SettingKeyAirwallexClientID:               "client-id",
					service.SettingKeyAirwallexAPIKey:                 "secret-api-key",
					service.SettingKeyAirwallexWebhookSecret:          "secret-webhook",
					service.SettingKeyPaymentAllowedCurrencies:        `["USD","CNY","HKD"]`,
					service.SettingKeyPaymentDefaultCurrency:          "USD",
					service.SettingKeyPaymentMinTopupAmount:           "1",
					service.SettingKeyPaymentMaxTopupAmount:           "5000",
					service.SettingKeyPaymentSubscriptionPlans:        "[]",
				})
			},
			method:     http.MethodGet,
			path:       "/api/v1/settings/public",
			wantStatus: http.StatusOK,
			wantJSON: `{
				"code": 0,
				"message": "success",
				"data": {
					"registration_enabled": false,
					"email_verify_enabled": false,
					"registration_email_suffix_whitelist": [],
					"promo_code_enabled": true,
					"password_reset_enabled": false,
					"invitation_code_enabled": false,
					"totp_enabled": false,
					"turnstile_enabled": false,
					"turnstile_site_key": "",
					"site_name": "Sub2API",
					"site_logo": "",
					"site_subtitle": "Subscription to API Conversion Platform",
					"visual_preset_default": "classic",
					"account_airy_white_surface_enabled": false,
					"api_base_url": "",
					"contact_info": "",
					"doc_url": "",
					"home_content": "",
					"hide_ccs_import_button": false,
					"available_channels_enabled": false,
					"channel_monitor_enabled": false,
					"public_model_catalog_enabled": true,
					"affiliate_enabled": false,
					"purchase_subscription_enabled": true,
					"purchase_subscription_url": "",
					"payment_provider_airwallex_enabled": true,
					"payment_allowed_currencies": ["USD", "CNY", "HKD"],
					"payment_default_currency": "USD",
					"payment_min_topup_amount": 1,
					"payment_max_topup_amount": 5000,
					"payment_mobile_force_qrcode_enabled": false,
					"payment_subscription_plans": [],
					"custom_menu_items": [],
					"login_agreement_enabled": false,
					"login_agreement_mode": "checkbox",
					"login_agreement_updated_at": "",
					"login_agreement_documents": [],
					"linuxdo_oauth_enabled": false,
					"github_oauth_enabled": false,
					"google_oauth_enabled": false,
					"dingtalk_oauth_enabled": false,
					"backend_mode_enabled": false,
					"maintenance_mode_enabled": false,
					"version": "0.0.0-test"
				}
			}`,
		},
		{
			name: "GET /api/v1/pages/:slug",
			setup: func(t *testing.T, deps *contractDeps) {
				t.Helper()
				deps.settingRepo.SetAll(map[string]string{
					service.SettingKeyCustomMenuItems: `[{"id":"page-guide","label":"Guide","icon_svg":"","url":"","visibility":"user","sort_order":0,"page_mode":"markdown","page_slug":"guide","page_content":"# Guide\nWindows path: C:\\temp\\file.txt","page_published":true}]`,
				})
			},
			method:     http.MethodGet,
			path:       "/api/v1/pages/guide",
			wantStatus: http.StatusOK,
			wantJSON: `{
				"code": 0,
				"message": "success",
				"data": {
					"id": "page-guide",
					"slug": "guide",
					"label": "Guide",
					"visibility": "user",
					"page_mode": "markdown",
					"content": "# Guide\nWindows path: C:\\temp\\file.txt"
				}
			}`,
		},
		{
			name: "GET /api/v1/pages/:slug admin page without auth returns not found",
			setup: func(t *testing.T, deps *contractDeps) {
				t.Helper()
				deps.settingRepo.SetAll(map[string]string{
					service.SettingKeyCustomMenuItems: `[{"id":"page-admin","label":"Admin Guide","icon_svg":"","url":"","visibility":"admin","sort_order":0,"page_mode":"markdown","page_slug":"admin-guide","page_content":"# Admin Guide","page_published":true}]`,
				})
			},
			method:     http.MethodGet,
			path:       "/api/v1/pages/admin-guide",
			wantStatus: http.StatusNotFound,
			wantJSON: `{
				"code": 404,
				"message": "custom page not found",
				"reason": "CUSTOM_PAGE_NOT_FOUND"
			}`,
		},
		{
			name: "GET /api/v1/pages/:slug admin page with admin auth returns content",
			setup: func(t *testing.T, deps *contractDeps) {
				t.Helper()
				deps.settingRepo.SetAll(map[string]string{
					service.SettingKeyCustomMenuItems: `[{"id":"page-admin","label":"Admin Guide","icon_svg":"","url":"","visibility":"admin","sort_order":0,"page_mode":"markdown","page_slug":"admin-guide","page_content":"# Admin Guide","page_published":true}]`,
				})
			},
			method: http.MethodGet,
			path:   "/api/v1/pages/admin-guide",
			headers: map[string]string{
				"Authorization": "Bearer admin-token",
			},
			wantStatus: http.StatusOK,
			wantJSON: `{
				"code": 0,
				"message": "success",
				"data": {
					"id": "page-admin",
					"slug": "admin-guide",
					"label": "Admin Guide",
					"visibility": "admin",
					"page_mode": "markdown",
					"content": "# Admin Guide"
				}
			}`,
		},
		{
			name: "GET /api/v1/pages/:slug draft admin page returns not found",
			setup: func(t *testing.T, deps *contractDeps) {
				t.Helper()
				deps.settingRepo.SetAll(map[string]string{
					service.SettingKeyCustomMenuItems: `[{"id":"page-draft","label":"Draft Guide","icon_svg":"","url":"","visibility":"admin","sort_order":0,"page_mode":"markdown","page_slug":"draft-guide","page_content":"# Draft","page_published":false}]`,
				})
			},
			method: http.MethodGet,
			path:   "/api/v1/pages/draft-guide",
			headers: map[string]string{
				"Authorization": "Bearer admin-token",
			},
			wantStatus: http.StatusNotFound,
			wantJSON: `{
				"code": 404,
				"message": "custom page not found",
				"reason": "CUSTOM_PAGE_NOT_FOUND"
			}`,
		},
		{
			name: "GET /api/v1/user/auth-identities",
			setup: func(t *testing.T, deps *contractDeps) {
				t.Helper()
				deps.authIdentityRepo.items = append(deps.authIdentityRepo.items, &service.AuthIdentity{
					ID:             11,
					Provider:       service.AuthProviderGitHub,
					ProviderUserID: "github-user-1",
					UserID:         1,
					Email:          "alice@example.com",
					EmailVerified:  true,
					DisplayName:    "alice-gh",
					AvatarURL:      "https://avatars.example.com/alice.png",
					CreatedAt:      deps.now,
					UpdatedAt:      deps.now,
				})
			},
			method:     http.MethodGet,
			path:       "/api/v1/user/auth-identities",
			wantStatus: http.StatusOK,
			wantJSON: `{
				"code": 0,
				"message": "success",
				"data": [
					{
						"id": 11,
						"provider": "github",
						"provider_user_id": "github-user-1",
						"email": "alice@example.com",
						"email_verified": true,
						"display_name": "alice-gh",
						"avatar_url": "https://avatars.example.com/alice.png",
						"created_at": "2025-01-02T03:04:05Z",
						"updated_at": "2025-01-02T03:04:05Z"
					}
				]
			}`,
		},
		{
			name: "DELETE /api/v1/user/auth-identities/:provider",
			setup: func(t *testing.T, deps *contractDeps) {
				t.Helper()
				deps.authIdentityRepo.items = append(deps.authIdentityRepo.items, &service.AuthIdentity{
					ID:             12,
					Provider:       service.AuthProviderGoogle,
					ProviderUserID: "google-user-1",
					UserID:         1,
					Email:          "alice@example.com",
					EmailVerified:  true,
					DisplayName:    "alice-google",
					CreatedAt:      deps.now,
					UpdatedAt:      deps.now,
				})
			},
			method:     http.MethodDelete,
			path:       "/api/v1/user/auth-identities/google",
			wantStatus: http.StatusOK,
			wantJSON: `{
				"code": 0,
				"message": "success",
				"data": {
					"message": "Auth identity removed"
				}
			}`,
		},
		{
			name:       "GET /api/v1/user/profile",
			method:     http.MethodGet,
			path:       "/api/v1/user/profile",
			wantStatus: http.StatusOK,
			wantJSON: `{
				"code": 0,
				"message": "success",
				"data": {
					"id": 1,
					"email": "alice@example.com",
					"username": "alice",
					"role": "user",
					"balance": 12.5,
					"concurrency": 5,
					"status": "active",
					"usage_model_display_mode": "model_only",
					"global_realtime_countdown_enabled": false,
					"account_realtime_countdown_enabled": true,
					"visual_preset_preference": "inherit",
					"account_visual_preset_override": "inherit",
					"account_today_stats_windows": ["today", "weekly", "total"],
					"account_group_display_mode": "full",
					"api_key_model_binding_mode": "model_required",
					"api_key_access_time_policy": null,
					"allowed_groups": null,
					"created_at": "2025-01-02T03:04:05Z",
					"updated_at": "2025-01-02T03:04:05Z",
					"admin_free_billing": false,
					"request_details_review": false
				}
			}`,
		},
		{
			name:       "GET /api/v1/auth/oauth/:provider/start unsupported provider",
			method:     http.MethodGet,
			path:       "/api/v1/auth/oauth/unknown/start",
			wantStatus: http.StatusBadRequest,
			wantJSON: `{
				"code": 400,
				"message": "oauth provider is unsupported",
				"reason": "OAUTH_PROVIDER_UNSUPPORTED"
			}`,
		},
		{
			name: "GET /api/v1/auth/oauth/:provider/start bind requires auth",
			setup: func(t *testing.T, deps *contractDeps) {
				t.Helper()
				deps.settingRepo.SetAll(map[string]string{
					service.SettingKeyGitHubOAuthEnabled:      "true",
					service.SettingKeyGitHubOAuthClientID:     "github-client-id",
					service.SettingKeyGitHubOAuthClientSecret: "github-secret",
					service.SettingKeyGitHubOAuthRedirectURL:  "https://example.com/api/v1/auth/oauth/github/callback",
				})
			},
			method:     http.MethodGet,
			path:       "/api/v1/auth/oauth/github/start?mode=bind&redirect=%2Fprofile",
			wantStatus: http.StatusBadRequest,
			wantJSON: `{
				"code": 400,
				"message": "bind mode requires authenticated user",
				"reason": "AUTH_IDENTITY_BIND_REQUIRED"
			}`,
		},
		{
			name:   "PUT /api/v1/user",
			method: http.MethodPut,
			path:   "/api/v1/user",
			body:   `{"username":"alice-2","usage_model_display_mode":"display_and_model","global_realtime_countdown_enabled":true,"account_realtime_countdown_enabled":false,"visual_preset_preference":"airy","account_visual_preset_override":"classic","account_today_stats_windows":["today","total"],"account_group_display_mode":"icon"}`,
			headers: map[string]string{
				"Content-Type": "application/json",
			},
			wantStatus: http.StatusOK,
			wantJSON: `{
				"code": 0,
				"message": "success",
				"data": {
					"id": 1,
					"email": "alice@example.com",
					"username": "alice-2",
					"role": "user",
					"balance": 12.5,
					"concurrency": 5,
					"status": "active",
					"usage_model_display_mode": "display_and_model",
					"global_realtime_countdown_enabled": true,
					"account_realtime_countdown_enabled": false,
					"visual_preset_preference": "airy",
					"account_visual_preset_override": "classic",
					"account_today_stats_windows": ["today", "total"],
					"account_group_display_mode": "icon",
					"api_key_model_binding_mode": "model_required",
					"api_key_access_time_policy": null,
					"allowed_groups": null,
					"created_at": "2025-01-02T03:04:05Z",
					"updated_at": "2025-01-02T03:04:05Z",
					"admin_free_billing": false,
					"request_details_review": false
				}
			}`,
		},
		{
			name:   "PUT /api/v1/user invalid usage_model_display_mode",
			method: http.MethodPut,
			path:   "/api/v1/user",
			body:   `{"usage_model_display_mode":"bad-mode"}`,
			headers: map[string]string{
				"Content-Type": "application/json",
			},
			wantStatus: http.StatusBadRequest,
			wantJSON: `{
				"code": 400,
				"message": "usage_model_display_mode must be one of model_only, display_only, display_and_model",
				"reason": "USER_USAGE_MODEL_DISPLAY_MODE_INVALID"
			}`,
		},
		{
			name:   "PUT /api/v1/user invalid visual_preset_preference",
			method: http.MethodPut,
			path:   "/api/v1/user",
			body:   `{"visual_preset_preference":"retro-future"}`,
			headers: map[string]string{
				"Content-Type": "application/json",
			},
			wantStatus: http.StatusBadRequest,
			wantJSON: `{
				"code": 400,
				"message": "visual_preset_preference must be one of inherit, classic, airy",
				"reason": "VISUAL_PRESET_PREFERENCE_INVALID"
			}`,
		},
		{
			name:   "PUT /api/v1/user invalid account_visual_preset_override",
			method: http.MethodPut,
			path:   "/api/v1/user",
			body:   `{"account_visual_preset_override":"retro-future"}`,
			headers: map[string]string{
				"Content-Type": "application/json",
			},
			wantStatus: http.StatusBadRequest,
			wantJSON: `{
				"code": 400,
				"message": "account_visual_preset_override must be one of inherit, classic, airy",
				"reason": "VISUAL_PRESET_PREFERENCE_INVALID"
			}`,
		},
		{
			name:   "PUT /api/v1/user invalid account_today_stats_windows",
			method: http.MethodPut,
			path:   "/api/v1/user",
			body:   `{"account_today_stats_windows":["today","bad"]}`,
			headers: map[string]string{
				"Content-Type": "application/json",
			},
			wantStatus: http.StatusBadRequest,
			wantJSON: `{
				"code": 400,
				"message": "account_today_stats_windows must only contain today, weekly, total",
				"reason": "ACCOUNT_TODAY_STATS_WINDOWS_INVALID"
			}`,
		},
		{
			name:   "PUT /api/v1/user empty account_today_stats_windows",
			method: http.MethodPut,
			path:   "/api/v1/user",
			body:   `{"account_today_stats_windows":[]}`,
			headers: map[string]string{
				"Content-Type": "application/json",
			},
			wantStatus: http.StatusBadRequest,
			wantJSON: `{
				"code": 400,
				"message": "account_today_stats_windows must contain at least one of today, weekly, total",
				"reason": "ACCOUNT_TODAY_STATS_WINDOWS_INVALID"
			}`,
		},
		{
			name:   "PUT /api/v1/user invalid account_group_display_mode",
			method: http.MethodPut,
			path:   "/api/v1/user",
			body:   `{"account_group_display_mode":"bad"}`,
			headers: map[string]string{
				"Content-Type": "application/json",
			},
			wantStatus: http.StatusBadRequest,
			wantJSON: `{
				"code": 400,
				"message": "account_group_display_mode must be one of full, icon",
				"reason": "ACCOUNT_GROUP_DISPLAY_MODE_INVALID"
			}`,
		},
		{
			name:       "GET /api/v1/user/aff",
			method:     http.MethodGet,
			path:       "/api/v1/user/aff",
			wantStatus: http.StatusOK,
			wantJSON: `{
				"code": 0,
				"message": "success",
				"data": {
					"enabled": false,
					"transfer_enabled": true,
					"aff_code": "AFFCODE0001",
					"invitee_count": 0,
					"rebate_balance": 3.5,
					"rebate_frozen_balance": 0,
					"lifetime_rebate": 3.5,
					"effective_rate_percent": 20,
					"rebate_on_usage_enabled": true,
					"rebate_on_topup_enabled": true,
					"rebate_freeze_hours": 0,
					"rebate_duration_days": 0,
					"rebate_per_invitee_cap": 0
				}
			}`,
		},
		{
			name:       "POST /api/v1/user/aff/transfer",
			method:     http.MethodPost,
			path:       "/api/v1/user/aff/transfer",
			wantStatus: http.StatusOK,
			wantJSON: `{
				"code": 0,
				"message": "success",
				"data": {
					"transferred_amount": 3.5,
					"new_balance": 16
				}
			}`,
		},
		{
			name:   "POST /api/v1/keys",
			method: http.MethodPost,
			path:   "/api/v1/keys",
			body:   `{"name":"Key One","custom_key":"sk_custom_1234567890"}`,
			headers: map[string]string{
				"Content-Type": "application/json",
			},
			wantStatus: http.StatusOK,
			wantJSON: `{
				"code": 0,
				"message": "success",
				"data": {
					"id": 100,
					"user_id": 1,
					"key": "sk_custom_1234567890",
					"name": "Key One",
					"deleted": false,
					"group_id": null,
					"status": "active",
					"model_display_mode": "alias_only",
					"ip_whitelist": null,
					"ip_blacklist": null,
					"last_used_at": null,
					"image_only_enabled": false,
					"image_count_billing_enabled": false,
					"image_max_count": 0,
					"image_count_used": 0,
					"image_count_weights": {"1K": 1, "2K": 1, "4K": 2},
					"quota": 0,
					"quota_used": 0,
					"rate_limit_5h": 0,
					"rate_limit_1d": 0,
					"rate_limit_7d": 0,
					"usage_5h": 0,
					"usage_1d": 0,
					"usage_7d": 0,
					"window_5h_start": null,
					"window_1d_start": null,
					"window_7d_start": null,
					"starts_at": null,
					"expires_at": null,
					"access_time_policy": null,
					"created_at": "2025-01-02T03:04:05Z",
					"updated_at": "2025-01-02T03:04:05Z"
				}
			}`,
		},
		{
			name: "GET /api/v1/keys (paginated)",
			setup: func(t *testing.T, deps *contractDeps) {
				t.Helper()
				deps.apiKeyRepo.MustSeed(&service.APIKey{
					ID:        100,
					UserID:    1,
					Key:       "sk_custom_1234567890",
					Name:      "Key One",
					Status:    service.StatusActive,
					CreatedAt: deps.now,
					UpdatedAt: deps.now,
				})
			},
			method:     http.MethodGet,
			path:       "/api/v1/keys?page=1&page_size=10",
			wantStatus: http.StatusOK,
			wantJSON: `{
				"code": 0,
				"message": "success",
				"data": {
					"items": [
						{
							"id": 100,
							"user_id": 1,
							"key": "sk_custom_1234567890",
							"name": "Key One",
							"deleted": false,
							"group_id": null,
							"status": "active",
							"model_display_mode": "alias_only",
							"ip_whitelist": null,
							"ip_blacklist": null,
							"last_used_at": null,
							"image_only_enabled": false,
							"image_count_billing_enabled": false,
							"image_max_count": 0,
							"image_count_used": 0,
							"image_count_weights": {"1K": 1, "2K": 1, "4K": 2},
							"quota": 0,
							"quota_used": 0,
							"rate_limit_5h": 0,
							"rate_limit_1d": 0,
							"rate_limit_7d": 0,
							"usage_5h": 0,
							"usage_1d": 0,
							"usage_7d": 0,
							"window_5h_start": null,
							"window_1d_start": null,
							"window_7d_start": null,
							"starts_at": null,
							"expires_at": null,
							"access_time_policy": null,
							"created_at": "2025-01-02T03:04:05Z",
							"updated_at": "2025-01-02T03:04:05Z"
						}
					],
					"total": 1,
					"page": 1,
					"page_size": 10,
					"pages": 1
				}
			}`,
		},
		{
			name: "GET /api/v1/groups/available",
			setup: func(t *testing.T, deps *contractDeps) {
				t.Helper()
				// 普通用户可见的分组列表不应包含内部字段（如 model_routing/account_count）。
				deps.groupRepo.SetActive([]service.Group{
					{
						ID:                  10,
						Name:                "Group One",
						Description:         "desc",
						Platform:            service.PlatformAnthropic,
						RateMultiplier:      1.5,
						IsExclusive:         false,
						Status:              service.StatusActive,
						SubscriptionType:    service.SubscriptionTypeStandard,
						ModelRoutingEnabled: true,
						ModelRouting: map[string][]int64{
							"claude-3-*": []int64{101, 102},
						},
						AccountCount: 2,
						CreatedAt:    deps.now,
						UpdatedAt:    deps.now,
					},
				})
				deps.userSubRepo.SetActiveByUserID(1, nil)
			},
			method:     http.MethodGet,
			path:       "/api/v1/groups/available",
			wantStatus: http.StatusOK,
			wantJSON: `{
				"code": 0,
				"message": "success",
				"data": [
					{
						"id": 10,
						"name": "Group One",
						"description": "desc",
						"platform": "anthropic",
						"priority": 0,
						"rate_multiplier": 1.5,
						"is_exclusive": false,
						"status": "active",
						"subscription_type": "standard",
						"gemini_mixed_protocol_enabled": false,
						"daily_limit_usd": null,
						"weekly_limit_usd": null,
						"monthly_limit_usd": null,
						"image_price_1k": null,
						"image_price_2k": null,
						"image_price_4k": null,
							"claude_code_only": false,
						"allow_messages_dispatch": false,
						"fallback_group_id": null,
						"fallback_group_id_on_invalid_request": null,
						"allow_messages_dispatch": false,
						"created_at": "2025-01-02T03:04:05Z",
						"updated_at": "2025-01-02T03:04:05Z"
					}
				]
			}`,
		},
		{
			name: "GET /api/v1/subscriptions",
			setup: func(t *testing.T, deps *contractDeps) {
				t.Helper()
				// 普通用户订阅接口不应包含 assigned_* / notes 等管理员字段。
				deps.userSubRepo.SetByUserID(1, []service.UserSubscription{
					{
						ID:              501,
						UserID:          1,
						GroupID:         10,
						StartsAt:        deps.now,
						ExpiresAt:       time.Date(2099, 1, 2, 3, 4, 5, 0, time.UTC), // 使用未来日期避免 normalizeSubscriptionStatus 标记为过期
						Status:          service.SubscriptionStatusActive,
						DailyUsageUSD:   1.23,
						WeeklyUsageUSD:  2.34,
						MonthlyUsageUSD: 3.45,
						AssignedBy:      ptr(int64(999)),
						AssignedAt:      deps.now,
						Notes:           "admin-note",
						CreatedAt:       deps.now,
						UpdatedAt:       deps.now,
					},
				})
			},
			method:     http.MethodGet,
			path:       "/api/v1/subscriptions",
			wantStatus: http.StatusOK,
			wantJSON: `{
				"code": 0,
				"message": "success",
				"data": [
					{
						"id": 501,
						"user_id": 1,
						"group_id": 10,
						"starts_at": "2025-01-02T03:04:05Z",
						"expires_at": "2099-01-02T03:04:05Z",
						"status": "active",
						"daily_window_start": null,
						"weekly_window_start": null,
						"monthly_window_start": null,
						"daily_usage_usd": 1.23,
						"weekly_usage_usd": 2.34,
						"monthly_usage_usd": 3.45,
						"created_at": "2025-01-02T03:04:05Z",
						"updated_at": "2025-01-02T03:04:05Z"
					}
				]
			}`,
		},
		{
			name: "GET /api/v1/redeem/history",
			setup: func(t *testing.T, deps *contractDeps) {
				t.Helper()
				// 普通用户兑换历史不应包含 notes 等内部字段。
				deps.redeemRepo.SetByUser(1, []service.RedeemCode{
					{
						ID:        900,
						Code:      "CODE-123",
						Type:      service.RedeemTypeBalance,
						Value:     1.25,
						Status:    service.StatusUsed,
						UsedBy:    ptr(int64(1)),
						UsedAt:    ptr(deps.now),
						Notes:     "internal-note",
						CreatedAt: deps.now,
					},
				})
			},
			method:     http.MethodGet,
			path:       "/api/v1/redeem/history",
			wantStatus: http.StatusOK,
			wantJSON: `{
				"code": 0,
				"message": "success",
				"data": [
					{
						"id": 900,
						"code": "CODE-123",
						"type": "balance",
						"value": 1.25,
						"status": "used",
						"used_by": 1,
						"used_at": "2025-01-02T03:04:05Z",
						"created_at": "2025-01-02T03:04:05Z",
						"expires_at": null,
						"group_id": null,
						"validity_days": 0
					}
				]
			}`,
		},
		{
			name: "GET /api/v1/usage/stats",
			setup: func(t *testing.T, deps *contractDeps) {
				t.Helper()
				deps.usageRepo.SetUserLogs(1, []service.UsageLog{
					{
						ID:                         1,
						UserID:                     1,
						APIKeyID:                   100,
						AccountID:                  200,
						Model:                      "claude-3",
						RequestContextLengthTokens: ptr(200000),
						InputTokens:                10,
						OutputTokens:               20,
						CacheCreationTokens:        1,
						CacheReadTokens:            2,
						TotalCost:                  0.5,
						ActualCost:                 0.5,
						DurationMs:                 ptr(100),
						CreatedAt:                  deps.now.Add(-2 * time.Minute),
					},
					{
						ID:           2,
						UserID:       1,
						APIKeyID:     100,
						AccountID:    200,
						Model:        "claude-3",
						InputTokens:  5,
						OutputTokens: 15,
						TotalCost:    0.25,
						ActualCost:   0.25,
						DurationMs:   ptr(300),
						CreatedAt:    deps.now.Add(-1 * time.Minute),
					},
				})
			},
			method:     http.MethodGet,
			path:       "/api/v1/usage/stats?start_date=2025-01-01&end_date=2025-01-02",
			wantStatus: http.StatusOK,
			wantJSON: `{
				"code": 0,
				"message": "success",
				"data": {
					"total_requests": 2,
					"total_input_tokens": 15,
					"total_output_tokens": 35,
					"total_cache_tokens": 3,
					"total_cache_creation_tokens": 0,
					"total_cache_read_tokens": 0,
					"total_tokens": 53,
					"total_cost": 0.75,
					"total_actual_cost": 0.75,
					"cache_hit_rate": 0,
					"today_requests": 2,
					"today_input_tokens": 15,
					"today_output_tokens": 35,
					"today_cache_tokens": 3,
					"today_cache_creation_tokens": 0,
					"today_cache_read_tokens": 0,
					"today_tokens": 53,
					"today_cost": 0.75,
					"today_actual_cost": 0.75,
					"today_cache_hit_rate": 0,
					"average_duration_ms": 200,
					"today_average_duration_ms": 200,
					"admin_free_requests": 0,
					"admin_free_standard_cost": 0
				}
			}`,
		},
		{
			name: "GET /api/v1/usage (paginated)",
			setup: func(t *testing.T, deps *contractDeps) {
				t.Helper()
				deps.usageRepo.SetUserLogs(1, []service.UsageLog{
					{
						ID:                         1,
						UserID:                     1,
						APIKeyID:                   100,
						AccountID:                  200,
						AccountRateMultiplier:      ptr(0.5),
						RequestID:                  "req_123",
						Model:                      "claude-3",
						RequestContextLengthTokens: ptr(1000000),
						InputTokens:                10,
						OutputTokens:               20,
						CacheCreationTokens:        1,
						CacheReadTokens:            2,
						TotalCost:                  0.5,
						ActualCost:                 0.5,
						BillingCurrency:            "USD",
						TotalCostUSDEquivalent:     0.5,
						ActualCostUSDEquivalent:    0.5,
						RateMultiplier:             1,
						BillingType:                service.BillingTypeBalance,
						Stream:                     true,
						DurationMs:                 ptr(100),
						FirstTokenMs:               ptr(50),
						CreatedAt:                  deps.now,
					},
				})
			},
			method:     http.MethodGet,
			path:       "/api/v1/usage?page=1&page_size=10",
			wantStatus: http.StatusOK,
			wantJSON: `{
				"code": 0,
				"message": "success",
				"data": {
					"items": [
						{
							"id": 1,
							"user_id": 1,
							"api_key_id": 100,
							"account_id": 200,
							"request_id": "req_123",
							"model": "claude-3",
							"request_context_length_tokens": 1000000,
							"request_type": "stream",
								"status": "succeeded",
								"openai_ws_mode": false,
								"group_id": null,
								"subscription_id": null,
							"input_tokens": 10,
							"output_tokens": 20,
							"cache_creation_tokens": 1,
							"cache_read_tokens": 2,
							"cache_creation_5m_tokens": 0,
							"cache_creation_1h_tokens": 0,
							"input_cost": 0,
							"output_cost": 0,
							"cache_creation_cost": 0,
							"cache_read_cost": 0,
						"total_cost": 0.5,
						"actual_cost": 0.5,
						"billing_currency": "USD",
						"total_cost_usd_equivalent": 0.5,
						"actual_cost_usd_equivalent": 0.5,
						"rate_multiplier": 1,
						"billing_type": 0,
							"stream": true,
							"duration_ms": 100,
							"first_token_ms": 50,
							"image_count": 0,
							"image_size": null,
							"cache_ttl_overridden": false,
							"created_at": "2025-01-02T03:04:05Z",
							"user_agent": null
						}
					],
					"total": 1,
					"page": 1,
					"page_size": 10,
					"pages": 1
				}
			}`,
		},
		{
			name: "GET /api/v1/admin/settings",
			setup: func(t *testing.T, deps *contractDeps) {
				t.Helper()
				deps.settingRepo.SetAll(map[string]string{
					service.SettingKeyRegistrationEnabled:              "true",
					service.SettingKeyEmailVerifyEnabled:               "false",
					service.SettingKeyRegistrationEmailSuffixWhitelist: "[]",
					service.SettingKeyPromoCodeEnabled:                 "true",

					service.SettingKeySMTPHost:     "smtp.example.com",
					service.SettingKeySMTPPort:     "587",
					service.SettingKeySMTPUsername: "user",
					service.SettingKeySMTPPassword: "secret",
					service.SettingKeySMTPFrom:     "no-reply@example.com",
					service.SettingKeySMTPFromName: "Sub2API",
					service.SettingKeySMTPUseTLS:   "true",

					service.SettingKeyTurnstileEnabled:   "true",
					service.SettingKeyTurnstileSiteKey:   "site-key",
					service.SettingKeyTurnstileSecretKey: "secret-key",

					service.SettingKeySiteName:     "Sub2API",
					service.SettingKeySiteLogo:     "",
					service.SettingKeySiteSubtitle: "Subtitle",
					service.SettingKeyAPIBaseURL:   "https://api.example.com",
					service.SettingKeyContactInfo:  "support",
					service.SettingKeyDocURL:       "https://docs.example.com",

					service.SettingKeyDefaultConcurrency: "5",
					service.SettingKeyDefaultBalance:     "1.25",

					service.SettingKeyOpsMonitoringEnabled:         "false",
					service.SettingKeyOpsRealtimeMonitoringEnabled: "true",
					service.SettingKeyOpsQueryModeDefault:          "auto",
					service.SettingKeyOpsMetricsIntervalSeconds:    "60",
				})
			},
			method:     http.MethodGet,
			path:       "/api/v1/admin/settings",
			wantStatus: http.StatusOK,
			wantJSON: `{
				"code": 0,
				"message": "success",
				"data": {
					"registration_enabled": true,
					"email_verify_enabled": false,
					"enable_anthropic_cache_ttl_1h_injection": false,
					"registration_email_suffix_whitelist": [],
					"promo_code_enabled": true,
					"password_reset_enabled": false,
					"frontend_url": "",
					"totp_enabled": false,
					"totp_encryption_key_configured": false,
					"smtp_host": "smtp.example.com",
					"smtp_port": 587,
					"smtp_username": "user",
					"smtp_password_configured": true,
					"smtp_from_email": "no-reply@example.com",
					"smtp_from_name": "Sub2API",
					"smtp_use_tls": true,
					"telegram_chat_id": "",
					"telegram_bot_token_configured": false,
					"telegram_bot_token_masked": "",
					"turnstile_enabled": true,
					"turnstile_site_key": "site-key",
					"turnstile_secret_key_configured": true,
					"linuxdo_connect_enabled": false,
						"linuxdo_connect_client_id": "",
						"linuxdo_connect_client_secret_configured": false,
						"linuxdo_connect_redirect_url": "",
						"github_oauth_enabled": false,
						"github_oauth_client_id": "",
						"github_oauth_client_secret_configured": false,
						"github_oauth_redirect_url": "",
						"google_oauth_enabled": false,
						"google_oauth_client_id": "",
						"google_oauth_client_secret_configured": false,
						"google_oauth_redirect_url": "",
						"dingtalk_oauth_enabled": false,
						"dingtalk_oauth_client_id": "",
						"dingtalk_oauth_client_secret_configured": false,
						"dingtalk_oauth_redirect_url": "",
						"content_moderation_enabled": false,
						"content_moderation_provider": "openai",
						"content_moderation_base_url": "",
						"content_moderation_api_key_configured": false,
						"content_moderation_api_key_statuses": [],
						"content_moderation_model": "",
						"content_moderation_keyword_block_enabled": false,
						"content_moderation_keywords": null,
						"content_moderation_model_filter": {
							"type": "all",
							"models": null
						},
						"content_moderation_timeout_ms": 1500,
						"content_moderation_dedupe_window_seconds": 300,
						"content_moderation_fail_open": true,
						"content_moderation_category_thresholds": {
							"harassment": 1,
							"harassment/threatening": 1,
							"hate": 1,
							"hate/threatening": 1,
							"illicit": 1,
							"illicit/violent": 1,
							"self-harm": 1,
							"self-harm/instructions": 1,
							"self-harm/intent": 1,
							"sexual": 1,
							"sexual/minors": 1,
							"violence": 1,
							"violence/graphic": 1
						},
						"ops_monitoring_enabled": false,
						"ops_realtime_monitoring_enabled": true,
						"ops_query_mode_default": "auto",
					"ops_metrics_interval_seconds": 60,
					"site_name": "Sub2API",
					"site_logo": "",
					"site_subtitle": "Subtitle",
					"visual_preset_default": "classic",
					"account_airy_white_surface_enabled": false,
					"api_base_url": "https://api.example.com",
					"contact_info": "support",
					"doc_url": "https://docs.example.com",
					"default_concurrency": 5,
					"default_balance": 1.25,
					"default_subscriptions": [],
					"enable_model_fallback": false,
					"fallback_model_anthropic": "claude-3-5-sonnet-20241022",
					"fallback_model_antigravity": "gemini-2.5-pro",
					"fallback_model_gemini": "gemini-2.5-pro",
						"fallback_model_openai": "gpt-4o",
						"enable_identity_patch": true,
						"identity_patch_prompt": "",
						"invitation_code_enabled": false,
					"home_content": "",
					"hide_ccs_import_button": false,
					"available_channels_enabled": false,
					"channel_monitor_enabled": false,
					"channel_monitor_default_interval_seconds": 60,
					"billing_currency_conversion_enabled": false,
					"billing_currency_usd_to_cny_rate": 7,
					"billing_currency_cny_to_usd_rate": 0.6,
					"codex_oauth_user_agent_mode": "default",
					"codex_oauth_user_agent_override": "",
					"public_model_catalog_enabled": true,
					"purchase_subscription_enabled": false,
					"purchase_subscription_url": "",
					"payment_provider_airwallex_enabled": false,
					"payment_provider_airwallex_effective": false,
					"airwallex_env": "demo",
					"airwallex_client_id": "",
					"airwallex_api_key_configured": false,
					"airwallex_webhook_secret_configured": false,
					"payment_allowed_currencies": ["USD", "CNY", "HKD"],
					"payment_default_currency": "USD",
					"payment_min_topup_amount": 1,
					"payment_max_topup_amount": 5000,
					"payment_mobile_force_qrcode_enabled": false,
					"payment_subscription_plans": [],
					"antigravity_user_agent_version": "",
					"login_agreement_enabled": false,
					"login_agreement_mode": "checkbox",
					"login_agreement_updated_at": "",
					"login_agreement_documents": [],
					"maintenance_mode_enabled": false,
					"min_claude_code_version": "",
					"max_claude_code_version": "",
					"openai_allow_claude_code_codex_plugin": false,
					"openai_allowed_codex_clients": [],
					"openai_fast_policy_settings": {
						"rules": [
							{ "service_tier": "priority", "action": "filter", "scope": "all" },
							{ "service_tier": "fast", "action": "filter", "scope": "all" },
							{ "service_tier": "flex", "action": "pass", "scope": "all" }
						]
					},
					"allow_ungrouped_key_scheduling": false,
					"backend_mode_enabled": false,
					"affiliate_enabled": false,
					"affiliate_transfer_enabled": true,
					"affiliate_rebate_on_usage_enabled": true,
					"affiliate_rebate_on_topup_enabled": true,
					"affiliate_rebate_rate": 20,
					"affiliate_rebate_freeze_hours": 0,
					"affiliate_rebate_duration_days": 0,
					"affiliate_rebate_per_invitee_cap": 0,
					"affiliate_aff_code_length": 10,
					"custom_menu_items": []
				}
			}`,
		},
		{
			name:       "GET /api/v1/admin/affiliates/users",
			method:     http.MethodGet,
			path:       "/api/v1/admin/affiliates/users?page=1&page_size=20",
			wantStatus: http.StatusOK,
			wantJSON: `{
				"code": 0,
				"message": "success",
				"data": {
					"items": [
						{
							"user_id": 1,
							"email": "alice@example.com",
							"aff_code": "AFFCODE0001",
							"custom_aff_code": false,
							"invitee_count": 0,
							"rebate_balance": 3.5,
							"rebate_frozen_balance": 0,
							"lifetime_rebate": 3.5,
							"updated_at": "2025-01-02T03:04:05Z"
						}
					],
					"total": 1,
					"page": 1,
					"page_size": 20,
					"pages": 1
				}
			}`,
		},
		{
			name:       "GET /api/v1/admin/affiliates/users/lookup",
			method:     http.MethodGet,
			path:       "/api/v1/admin/affiliates/users/lookup?q=alice",
			wantStatus: http.StatusOK,
			wantJSON: `{
				"code": 0,
				"message": "success",
				"data": [
					{
						"user_id": 1,
						"email": "alice@example.com",
						"aff_code": "AFFCODE0001",
						"custom_aff_code": false,
						"invitee_count": 0,
						"rebate_balance": 3.5,
						"rebate_frozen_balance": 0,
						"lifetime_rebate": 3.5,
						"updated_at": "2025-01-02T03:04:05Z"
					}
				]
			}`,
		},
		{
			name:   "PUT /api/v1/admin/affiliates/users/:user_id",
			method: http.MethodPut,
			path:   "/api/v1/admin/affiliates/users/1",
			body:   `{"aff_code":"CUSTOM00001","custom_rebate_rate_percent":30}`,
			headers: map[string]string{
				"Content-Type": "application/json",
			},
			wantStatus: http.StatusOK,
			wantJSON: `{
				"code": 0,
				"message": "success",
				"data": {
					"user_id": 1,
					"aff_code": "CUSTOM00001",
					"invitee_count": 0,
					"rebate_balance": 3.5,
					"rebate_frozen_balance": 0,
					"lifetime_rebate": 3.5,
					"custom_rebate_rate_percent": 30,
					"custom_aff_code": true,
					"created_at": "2025-01-02T03:04:05Z",
					"updated_at": "2025-01-02T03:04:05Z"
				}
			}`,
		},
		{
			name:       "DELETE /api/v1/admin/affiliates/users/:user_id",
			method:     http.MethodDelete,
			path:       "/api/v1/admin/affiliates/users/1",
			wantStatus: http.StatusOK,
			wantJSON: `{
				"code": 0,
				"message": "success",
				"data": {
					"user_id": 1,
					"aff_code": "RESETCODE01",
					"invitee_count": 0,
					"rebate_balance": 3.5,
					"rebate_frozen_balance": 0,
					"lifetime_rebate": 3.5,
					"custom_aff_code": false,
					"created_at": "2025-01-02T03:04:05Z",
					"updated_at": "2025-01-02T03:04:05Z"
				}
			}`,
		},
		{
			name:   "POST /api/v1/admin/affiliates/users/batch-rate",
			method: http.MethodPost,
			path:   "/api/v1/admin/affiliates/users/batch-rate",
			body:   `{"user_ids":[101,102],"custom_rebate_rate_percent":25}`,
			headers: map[string]string{
				"Content-Type": "application/json",
			},
			wantStatus: http.StatusOK,
			wantJSON: `{
				"code": 0,
				"message": "success",
				"data": {
					"updated": 2
				}
			}`,
		},
		{
			name:   "POST /api/v1/admin/accounts/bulk-update",
			method: http.MethodPost,
			path:   "/api/v1/admin/accounts/bulk-update",
			body:   `{"account_ids":[101,102],"schedulable":false}`,
			headers: map[string]string{
				"Content-Type": "application/json",
			},
			wantStatus: http.StatusOK,
			wantJSON: `{
				"code": 0,
				"message": "success",
				"data": {
					"success": 2,
					"failed": 0,
					"success_ids": [101, 102],
					"failed_ids": [],
					"results": [
						{"account_id": 101, "success": true},
						{"account_id": 102, "success": true}
					]
				}
			}`,
		},
		{
			name: "GET /api/v1/admin/accounts/daily-5h-trigger-settings",
			setup: func(t *testing.T, deps *contractDeps) {
				t.Helper()
				deps.settingRepo.SetAll(map[string]string{
					service.SettingKeyAccountDaily5HTriggerSettings: `{
						"enabled": true,
						"selected_account_types": ["chatgpt_oauth", "google_oauth", "unknown"],
						"include_paused_accounts": true,
						"ignore_free_accounts": true,
						"openai_model_mode": {"mode": "fixed", "fixed_model_id": "gpt-5.4-mini"},
						"anthropic_model_mode": {"mode": "invalid", "fixed_model_id": "claude-3.5-haiku"},
						"gemini_model_mode": {"mode": "auto", "fixed_model_id": ""}
					}`,
				})
			},
			method:     http.MethodGet,
			path:       "/api/v1/admin/accounts/daily-5h-trigger-settings",
			headers:    map[string]string{"Authorization": "Bearer admin-token"},
			wantStatus: http.StatusOK,
			wantJSON: `{
				"code": 0,
				"message": "success",
				"data": {
					"settings": {
						"enabled": true,
						"selected_account_types": ["chatgpt_oauth", "google_oauth"],
						"include_paused_accounts": true,
						"ignore_free_accounts": true,
						"openai_model_mode": {
							"mode": "fixed",
							"fixed_model_id": "gpt-5.4-mini"
						},
						"anthropic_model_mode": {
							"mode": "auto",
							"fixed_model_id": "claude-3.5-haiku"
						},
						"gemini_model_mode": {
							"mode": "auto"
						}
					},
					"candidates": []
				}
			}`,
		},
		{
			name:   "PUT /api/v1/admin/accounts/daily-5h-trigger-settings",
			method: http.MethodPut,
			path:   "/api/v1/admin/accounts/daily-5h-trigger-settings",
			body: `{
				"enabled": true,
				"selected_account_types": ["chatgpt_oauth", "google_oauth", "unknown", "chatgpt_oauth"],
				"include_paused_accounts": false,
				"ignore_free_accounts": true,
				"openai_model_mode": {"mode": "fixed", "fixed_model_id": "gpt-5.4-mini"},
				"anthropic_model_mode": {"mode": "invalid", "fixed_model_id": "claude-3.5-haiku"},
				"gemini_model_mode": {"mode": "fixed", "fixed_model_id": "gemini-2.5-flash"}
			}`,
			headers: map[string]string{
				"Authorization": "Bearer admin-token",
				"Content-Type":  "application/json",
			},
			wantStatus: http.StatusOK,
			wantJSON: `{
				"code": 0,
				"message": "success",
				"data": {
					"settings": {
						"enabled": true,
						"selected_account_types": ["chatgpt_oauth", "google_oauth"],
						"include_paused_accounts": false,
						"ignore_free_accounts": true,
						"openai_model_mode": {
							"mode": "fixed",
							"fixed_model_id": "gpt-5.4-mini"
						},
						"anthropic_model_mode": {
							"mode": "auto",
							"fixed_model_id": "claude-3.5-haiku"
						},
						"gemini_model_mode": {
							"mode": "fixed",
							"fixed_model_id": "gemini-2.5-flash"
						}
					},
					"candidates": []
				}
			}`,
		},
		{
			name: "POST /api/v1/admin/users/batch-concurrency",
			setup: func(t *testing.T, deps *contractDeps) {
				t.Helper()
				deps.userRepo.users[2] = &service.User{
					ID:                    2,
					Email:                 "bob@example.com",
					Username:              "bob",
					Role:                  service.RoleUser,
					Balance:               3,
					Concurrency:           2,
					Status:                service.StatusActive,
					UsageModelDisplayMode: service.UsageModelDisplayModeModelOnly,
					CreatedAt:             deps.now,
					UpdatedAt:             deps.now,
				}
			},
			method: http.MethodPost,
			path:   "/api/v1/admin/users/batch-concurrency",
			body:   `{"concurrency":7,"search":"example.com"}`,
			headers: map[string]string{
				"Content-Type":    "application/json",
				"Idempotency-Key": "contract-batch-concurrency-1",
			},
			wantStatus: http.StatusOK,
			wantJSON: `{
				"code": 0,
				"message": "success",
				"data": {
					"matched": 2,
					"success_count": 2,
					"failed_count": 0,
					"concurrency": 7,
					"results": [
						{"user_id": 1, "email": "alice@example.com", "success": true},
						{"user_id": 2, "email": "bob@example.com", "success": true}
					]
				}
			}`,
		},
		{
			name: "GET /api/v1/admin/moderation/audits",
			setup: func(t *testing.T, deps *contractDeps) {
				t.Helper()
				deps.moderationRepo.items = append(deps.moderationRepo.items, &service.ContentModerationAudit{
					ID:              31,
					RequestID:       "req-mod-1",
					ClientRequestID: "creq-mod-1",
					UserID:          ptr(int64(1)),
					APIKeyID:        ptr(int64(100)),
					Provider:        "openai",
					Model:           "gpt-4o-mini",
					SourceEndpoint:  service.ContentModerationSourceOpenAIChat,
					ContentHash:     "hash-1",
					ContentSummary:  "hello world",
					Categories:      []string{"moderation_unavailable"},
					Hit:             false,
					DedupeHit:       true,
					ErrorReason:     "moderation_not_configured",
					LatencyMs:       2,
					CreatedAt:       deps.now,
				})
			},
			method:     http.MethodGet,
			path:       "/api/v1/admin/moderation/audits?page=1&page_size=20",
			wantStatus: http.StatusOK,
			wantJSON: `{
				"code": 0,
				"message": "success",
				"data": {
					"items": [
						{
							"id": 31,
							"request_id": "req-mod-1",
							"client_request_id": "creq-mod-1",
							"user_id": 1,
							"api_key_id": 100,
							"provider": "openai",
							"model": "gpt-4o-mini",
							"source_endpoint": "openai_chat_completions",
							"content_hash": "hash-1",
							"content_summary": "hello world",
							"categories": ["moderation_unavailable"],
							"hit": false,
							"dedupe_hit": true,
							"error_reason": "moderation_not_configured",
							"latency_ms": 2,
							"created_at": "2025-01-02T03:04:05Z"
						}
					],
					"total": 1,
					"page": 1,
					"page_size": 20,
					"pages": 1
				}
			}`,
		},
		{
			name: "GET /api/v1/admin/moderation/audits/:id",
			setup: func(t *testing.T, deps *contractDeps) {
				t.Helper()
				deps.moderationRepo.items = append(deps.moderationRepo.items, &service.ContentModerationAudit{
					ID:              32,
					RequestID:       "req-mod-2",
					ClientRequestID: "creq-mod-2",
					Provider:        "gemini",
					Model:           "gemini-2.5-pro",
					SourceEndpoint:  service.ContentModerationSourceGeminiGenerate,
					ContentHash:     "hash-2",
					ContentSummary:  "summary",
					Categories:      []string{"moderation_flagged"},
					Hit:             true,
					DedupeHit:       false,
					ErrorReason:     "",
					LatencyMs:       18,
					CreatedAt:       deps.now,
				})
			},
			method:     http.MethodGet,
			path:       "/api/v1/admin/moderation/audits/32",
			wantStatus: http.StatusOK,
			wantJSON: `{
				"code": 0,
				"message": "success",
				"data": {
					"id": 32,
					"request_id": "req-mod-2",
					"client_request_id": "creq-mod-2",
					"user_id": null,
					"api_key_id": null,
					"provider": "gemini",
					"model": "gemini-2.5-pro",
					"source_endpoint": "gemini_generate_content",
					"content_hash": "hash-2",
					"content_summary": "summary",
					"categories": ["moderation_flagged"],
					"hit": true,
					"dedupe_hit": false,
					"error_reason": "",
					"latency_ms": 18,
					"created_at": "2025-01-02T03:04:05Z"
				}
			}`,
		},
		{
			name:       "GET /api/v1/admin/ops/runtime/payment",
			method:     http.MethodGet,
			path:       "/api/v1/admin/ops/runtime/payment",
			wantStatus: http.StatusOK,
			wantJSON: `{
				"code": 0,
				"message": "success",
				"data": {
					"create_success": 0,
					"create_failure": 0,
					"provider_latency_count": 0,
					"provider_latency_ms_total": 0,
					"webhook_success": 0,
					"webhook_failure": 0,
					"resume_success": 0,
					"resume_failure": 0,
					"refund_success": 0,
					"refund_failure": 0
				}
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := newContractDeps(t)
			if tt.setup != nil {
				tt.setup(t, deps)
			}

			status, body := doRequest(t, deps.router, tt.method, tt.path, tt.body, tt.headers)
			require.Equal(t, tt.wantStatus, status)
			require.JSONEq(t, tt.wantJSON, body)
			if tt.name == "GET /api/v1/settings/public exposes effective Airwallex flag without webhook secret" {
				require.Contains(t, deps.settingRepo.lastKeys, service.SettingKeyAirwallexClientID)
				require.Contains(t, deps.settingRepo.lastKeys, service.SettingKeyAirwallexAPIKey)
				require.NotContains(t, deps.settingRepo.lastKeys, service.SettingKeyAirwallexWebhookSecret)
			}
		})
	}
}

type contractDeps struct {
	now              time.Time
	router           http.Handler
	userRepo         *stubUserRepo
	apiKeyRepo       *stubApiKeyRepo
	groupRepo        *stubGroupRepo
	userSubRepo      *stubUserSubscriptionRepo
	usageRepo        *stubUsageLogRepo
	settingRepo      *stubSettingRepo
	redeemRepo       *stubRedeemCodeRepo
	affRepo          *stubAffiliateRepo
	authIdentityRepo *stubAuthIdentityRepo
	moderationRepo   *stubContentModerationAuditRepo
	authService      *service.AuthService
}

func newContractDeps(t *testing.T) *contractDeps {
	t.Helper()

	now := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	t.Cleanup(timezone.SetNowForTesting(func() time.Time {
		return now
	}))

	userRepo := &stubUserRepo{
		users: map[int64]*service.User{
			1: {
				ID:                              1,
				Email:                           "alice@example.com",
				Username:                        "alice",
				Notes:                           "hello",
				Role:                            service.RoleUser,
				Balance:                         12.5,
				Concurrency:                     5,
				Status:                          service.StatusActive,
				UsageModelDisplayMode:           service.UsageModelDisplayModeModelOnly,
				GlobalRealtimeCountdownEnabled:  false,
				AccountRealtimeCountdownEnabled: true,
				VisualPresetPreference:          service.VisualPresetPreferenceInherit,
				AccountVisualPresetOverride:     service.VisualPresetPreferenceInherit,
				AccountTodayStatsWindows:        service.DefaultAccountTodayStatsWindows(),
				AccountGroupDisplayMode:         service.AccountGroupDisplayModeFull,
				AllowedGroups:                   nil,
				CreatedAt:                       now,
				UpdatedAt:                       now,
			},
		},
	}

	apiKeyRepo := newStubApiKeyRepo(now)
	apiKeyCache := stubApiKeyCache{}
	groupRepo := &stubGroupRepo{}
	userSubRepo := &stubUserSubscriptionRepo{}
	accountRepo := stubAccountRepo{}
	proxyRepo := stubProxyRepo{}
	redeemRepo := &stubRedeemCodeRepo{}

	cfg := &config.Config{
		Default: config.DefaultConfig{
			APIKeyPrefix: "sk-",
		},
		RunMode: config.RunModeStandard,
	}

	userService := service.NewUserService(userRepo, nil, nil)
	apiKeyService := service.NewAPIKeyService(apiKeyRepo, userRepo, groupRepo, userSubRepo, nil, apiKeyCache, cfg)

	usageRepo := newStubUsageLogRepo()
	usageService := service.NewUsageService(usageRepo, userRepo, nil, nil)

	subscriptionService := service.NewSubscriptionService(groupRepo, userSubRepo, nil, nil, cfg)
	subscriptionHandler := handler.NewSubscriptionHandler(subscriptionService)

	redeemService := service.NewRedeemService(redeemRepo, userRepo, subscriptionService, nil, nil, nil, nil, nil)
	redeemHandler := handler.NewRedeemHandler(redeemService)

	settingRepo := newStubSettingRepo()
	settingService := service.NewSettingService(settingRepo, cfg)
	settingHandler := handler.NewSettingHandler(settingService, "0.0.0-test")
	authIdentityRepo := newStubAuthIdentityRepo()
	moderationRepo := newStubContentModerationAuditRepo()

	affiliateRepo := newStubAffiliateRepo(now)
	affiliateService := service.NewAffiliateService(settingService, affiliateRepo)
	userHandler := handler.NewUserHandler(userService, affiliateService)
	authService := service.NewAuthService(nil, userRepo, redeemRepo, &stubRefreshTokenCache{}, cfg, settingService, nil, nil, nil, nil, affiliateService, subscriptionService)
	authIdentityService := service.NewAuthIdentityService(authIdentityRepo, userRepo, authService)
	userHandler.SetAuthIdentityService(authIdentityService)
	adminAffiliateHandler := adminhandler.NewAffiliateHandler(affiliateService)
	contentModerationService := service.NewContentModerationService(moderationRepo, settingRepo)
	adminModerationHandler := adminhandler.NewContentModerationAuditHandler(contentModerationService)

	adminService := service.NewAdminService(userRepo, groupRepo, &accountRepo, proxyRepo, apiKeyRepo, redeemRepo, nil, nil, nil, nil, nil, nil, nil, settingService, nil, userSubRepo, affiliateService, cfg)
	adminUserHandler := adminhandler.NewUserHandler(adminService, nil)
	authHandler := handler.NewAuthHandler(cfg, authService, userService, settingService, nil, redeemService, nil)
	authHandler.SetAuthIdentityService(authIdentityService)
	apiKeyHandler := handler.NewAPIKeyHandler(apiKeyService)
	usageHandler := handler.NewUsageHandler(usageService, apiKeyService)
	adminSettingHandler := adminhandler.NewSettingHandler(settingService, nil, nil, nil, nil)
	adminOpsHandler := adminhandler.NewOpsHandler(nil)
	adminAccountHandler := adminhandler.NewAccountHandler(adminService, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	adminAccountHandler.SetSettingService(settingService)

	jwtAuth := func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{
			UserID:      1,
			Concurrency: 5,
		})
		c.Set(string(middleware.ContextKeyUserRole), service.RoleUser)
		c.Next()
	}
	adminAuth := func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{
			UserID:      1,
			Concurrency: 5,
		})
		c.Set(string(middleware.ContextKeyUserRole), service.RoleAdmin)
		c.Next()
	}

	r := gin.New()

	v1 := r.Group("/api/v1")
	v1.GET("/settings/public", settingHandler.GetPublicSettings)
	v1Pages := v1.Group("/pages")
	v1Pages.Use(func(c *gin.Context) {
		if c.GetHeader("Authorization") == "Bearer admin-token" {
			c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{
				UserID:      1,
				Concurrency: 5,
			})
			c.Set(string(middleware.ContextKeyUserRole), service.RoleAdmin)
		}
		c.Next()
	})
	v1Pages.GET("/:slug", settingHandler.GetCustomPage)
	v1.GET("/auth/oauth/:provider/start", authHandler.SocialOAuthStart)
	v1.POST("/auth/oauth/:provider/complete", authHandler.CompleteSocialOAuthRegistration)

	v1Auth := v1.Group("")
	v1Auth.Use(jwtAuth)
	v1Auth.GET("/auth/me", authHandler.GetCurrentUser)

	v1Keys := v1.Group("")
	v1Keys.Use(jwtAuth)
	v1Keys.GET("/keys", apiKeyHandler.List)
	v1Keys.POST("/keys", apiKeyHandler.Create)
	v1Keys.GET("/groups/available", apiKeyHandler.GetAvailableGroups)

	v1Usage := v1.Group("")
	v1Usage.Use(jwtAuth)
	v1Usage.GET("/usage", usageHandler.List)
	v1Usage.GET("/usage/stats", usageHandler.Stats)

	v1Subs := v1.Group("")
	v1Subs.Use(jwtAuth)
	v1Subs.GET("/subscriptions", subscriptionHandler.List)

	v1Redeem := v1.Group("")
	v1Redeem.Use(jwtAuth)
	v1Redeem.GET("/redeem/history", redeemHandler.GetHistory)

	v1User := v1.Group("/user")
	v1User.Use(jwtAuth)
	v1User.GET("/profile", userHandler.GetProfile)
	v1User.PUT("", userHandler.UpdateProfile)
	v1User.GET("/auth-identities", userHandler.ListAuthIdentities)
	v1User.DELETE("/auth-identities/:provider", userHandler.DeleteAuthIdentity)
	v1User.GET("/aff", userHandler.GetAffiliate)
	v1User.POST("/aff/transfer", userHandler.TransferAffiliate)

	v1Admin := v1.Group("/admin")
	v1Admin.Use(adminAuth)
	v1Admin.GET("/settings", adminSettingHandler.GetSettings)
	v1Admin.GET("/accounts/daily-5h-trigger-settings", adminAccountHandler.GetDaily5HTriggerSettings)
	v1Admin.PUT("/accounts/daily-5h-trigger-settings", adminAccountHandler.UpdateDaily5HTriggerSettings)
	v1Admin.POST("/accounts/data/import-jobs", adminAccountHandler.CreateImportJob)
	v1Admin.GET("/accounts/data/import-jobs/:job_id", adminAccountHandler.GetImportJob)
	v1Admin.POST("/accounts/data/import-jobs/:job_id/cancel", adminAccountHandler.CancelImportJob)
	v1Admin.POST("/accounts/data/import-jobs/:job_id/group-bindings", adminAccountHandler.BindImportJobGroups)
	v1Admin.POST("/accounts/bulk-update", adminAccountHandler.BulkUpdate)
	v1Admin.POST("/users/batch-concurrency", adminUserHandler.BatchUpdateConcurrency)
	v1Admin.GET("/moderation/audits", adminModerationHandler.List)
	v1Admin.GET("/moderation/audits/:id", adminModerationHandler.Detail)
	v1Admin.GET("/ops/runtime/payment", adminOpsHandler.GetPaymentRuntimeMetrics)
	v1AdminAff := v1Admin.Group("/affiliates")
	v1AdminAff.GET("/users", adminAffiliateHandler.ListUsers)
	v1AdminAff.GET("/users/lookup", adminAffiliateHandler.LookupUsers)
	v1AdminAff.PUT("/users/:user_id", adminAffiliateHandler.UpdateUser)
	v1AdminAff.DELETE("/users/:user_id", adminAffiliateHandler.DeleteUserCustom)
	v1AdminAff.POST("/users/batch-rate", adminAffiliateHandler.BatchRate)

	return &contractDeps{
		now:              now,
		router:           r,
		userRepo:         userRepo,
		apiKeyRepo:       apiKeyRepo,
		groupRepo:        groupRepo,
		userSubRepo:      userSubRepo,
		usageRepo:        usageRepo,
		settingRepo:      settingRepo,
		redeemRepo:       redeemRepo,
		affRepo:          affiliateRepo,
		authIdentityRepo: authIdentityRepo,
		moderationRepo:   moderationRepo,
		authService:      authService,
	}
}

func doRequest(t *testing.T, router http.Handler, method, path, body string, headers map[string]string) (int, string) {
	t.Helper()

	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	respBody, err := io.ReadAll(w.Result().Body)
	require.NoError(t, err)

	return w.Result().StatusCode, string(respBody)
}

func ptr[T any](v T) *T { return &v }

type stubUserRepo struct {
	users map[int64]*service.User
}

func (r *stubUserRepo) Create(ctx context.Context, user *service.User) error {
	return errors.New("not implemented")
}

func (r *stubUserRepo) GetByID(ctx context.Context, id int64) (*service.User, error) {
	user, ok := r.users[id]
	if !ok {
		return nil, service.ErrUserNotFound
	}
	clone := *user
	return &clone, nil
}

func (r *stubUserRepo) GetByEmail(ctx context.Context, email string) (*service.User, error) {
	for _, user := range r.users {
		if user.Email == email {
			clone := *user
			return &clone, nil
		}
	}
	return nil, service.ErrUserNotFound
}

func (r *stubUserRepo) GetFirstAdmin(ctx context.Context) (*service.User, error) {
	for _, user := range r.users {
		if user.Role == service.RoleAdmin && user.Status == service.StatusActive {
			clone := *user
			return &clone, nil
		}
	}
	return nil, service.ErrUserNotFound
}

func (r *stubUserRepo) Update(ctx context.Context, user *service.User) error {
	if user == nil {
		return errors.New("user is nil")
	}
	if _, ok := r.users[user.ID]; !ok {
		return service.ErrUserNotFound
	}
	clone := *user
	r.users[user.ID] = &clone
	return nil
}

func (r *stubUserRepo) Delete(ctx context.Context, id int64) error {
	return errors.New("not implemented")
}

func (r *stubUserRepo) List(ctx context.Context, params pagination.PaginationParams) ([]service.User, *pagination.PaginationResult, error) {
	return nil, nil, errors.New("not implemented")
}

func (r *stubUserRepo) ListWithFilters(ctx context.Context, params pagination.PaginationParams, filters service.UserListFilters) ([]service.User, *pagination.PaginationResult, error) {
	filtered := make([]service.User, 0, len(r.users))
	for _, user := range r.users {
		if filters.Status != "" && user.Status != filters.Status {
			continue
		}
		if filters.Role != "" && user.Role != filters.Role {
			continue
		}
		if filters.Search != "" {
			search := strings.ToLower(strings.TrimSpace(filters.Search))
			email := strings.ToLower(user.Email)
			username := strings.ToLower(user.Username)
			if !strings.Contains(email, search) && !strings.Contains(username, search) {
				continue
			}
		}
		clone := *user
		filtered = append(filtered, clone)
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].ID < filtered[j].ID
	})
	total := int64(len(filtered))
	start := params.Offset()
	if start > len(filtered) {
		start = len(filtered)
	}
	end := start + params.Limit()
	if end > len(filtered) {
		end = len(filtered)
	}
	pageItems := append([]service.User(nil), filtered[start:end]...)
	return pageItems, paginationResult(total, params), nil
}

func (r *stubUserRepo) UpdateBalance(ctx context.Context, id int64, amount float64) error {
	return errors.New("not implemented")
}

func (r *stubUserRepo) DeductBalance(ctx context.Context, id int64, amount float64) error {
	return errors.New("not implemented")
}

func (r *stubUserRepo) UpdateConcurrency(ctx context.Context, id int64, amount int) error {
	return errors.New("not implemented")
}

func (r *stubUserRepo) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	for _, user := range r.users {
		if user.Email == email {
			return true, nil
		}
	}
	return false, nil
}

func (r *stubUserRepo) RemoveGroupFromAllowedGroups(ctx context.Context, groupID int64) (int64, error) {
	return 0, errors.New("not implemented")
}

func (r *stubUserRepo) RemoveGroupFromUserAllowedGroups(ctx context.Context, userID int64, groupID int64) error {
	return errors.New("not implemented")
}

func (r *stubUserRepo) AddGroupToAllowedGroups(ctx context.Context, userID int64, groupID int64) error {
	return errors.New("not implemented")
}

func (r *stubUserRepo) UpdateTotpSecret(ctx context.Context, userID int64, encryptedSecret *string) error {
	return errors.New("not implemented")
}

func (r *stubUserRepo) EnableTotp(ctx context.Context, userID int64) error {
	return errors.New("not implemented")
}

func (r *stubUserRepo) DisableTotp(ctx context.Context, userID int64) error {
	return errors.New("not implemented")
}

type stubApiKeyCache struct{}

func (stubApiKeyCache) GetCreateAttemptCount(ctx context.Context, userID int64) (int, error) {
	return 0, nil
}

func (stubApiKeyCache) IncrementCreateAttemptCount(ctx context.Context, userID int64) error {
	return nil
}

func (stubApiKeyCache) DeleteCreateAttemptCount(ctx context.Context, userID int64) error {
	return nil
}

func (stubApiKeyCache) IncrementDailyUsage(ctx context.Context, apiKey string) error {
	return nil
}

type stubRefreshTokenCache struct{}

func (stubRefreshTokenCache) StoreRefreshToken(ctx context.Context, tokenHash string, data *service.RefreshTokenData, ttl time.Duration) error {
	return nil
}

func (stubRefreshTokenCache) GetRefreshToken(ctx context.Context, tokenHash string) (*service.RefreshTokenData, error) {
	return nil, service.ErrRefreshTokenNotFound
}

func (stubRefreshTokenCache) DeleteRefreshToken(ctx context.Context, tokenHash string) error {
	return nil
}

func (stubRefreshTokenCache) DeleteUserRefreshTokens(ctx context.Context, userID int64) error {
	return nil
}

func (stubRefreshTokenCache) DeleteTokenFamily(ctx context.Context, familyID string) error {
	return nil
}

func (stubRefreshTokenCache) AddToUserTokenSet(ctx context.Context, userID int64, tokenHash string, ttl time.Duration) error {
	return nil
}

func (stubRefreshTokenCache) AddToFamilyTokenSet(ctx context.Context, familyID string, tokenHash string, ttl time.Duration) error {
	return nil
}

func (stubRefreshTokenCache) GetUserTokenHashes(ctx context.Context, userID int64) ([]string, error) {
	return nil, nil
}

func (stubRefreshTokenCache) GetFamilyTokenHashes(ctx context.Context, familyID string) ([]string, error) {
	return nil, nil
}

func (stubRefreshTokenCache) IsTokenInFamily(ctx context.Context, familyID string, tokenHash string) (bool, error) {
	return false, nil
}

func (stubApiKeyCache) SetDailyUsageExpiry(ctx context.Context, apiKey string, ttl time.Duration) error {
	return nil
}

func (stubApiKeyCache) GetAuthCache(ctx context.Context, key string) (*service.APIKeyAuthCacheEntry, error) {
	return nil, nil
}

func (stubApiKeyCache) SetAuthCache(ctx context.Context, key string, entry *service.APIKeyAuthCacheEntry, ttl time.Duration) error {
	return nil
}

func (stubApiKeyCache) DeleteAuthCache(ctx context.Context, key string) error {
	return nil
}

func (stubApiKeyCache) PublishAuthCacheInvalidation(ctx context.Context, cacheKey string) error {
	return nil
}

func (stubApiKeyCache) SubscribeAuthCacheInvalidation(ctx context.Context, handler func(cacheKey string)) error {
	return nil
}

type stubGroupRepo struct {
	active []service.Group
}

func (r *stubGroupRepo) SetActive(groups []service.Group) {
	r.active = append([]service.Group(nil), groups...)
}

func (stubGroupRepo) Create(ctx context.Context, group *service.Group) error {
	return errors.New("not implemented")
}

func (stubGroupRepo) GetByID(ctx context.Context, id int64) (*service.Group, error) {
	return nil, service.ErrGroupNotFound
}

func (stubGroupRepo) GetByIDLite(ctx context.Context, id int64) (*service.Group, error) {
	return nil, service.ErrGroupNotFound
}
func (r *stubGroupRepo) GetByName(ctx context.Context, name string) (*service.Group, error) {
	for i := range r.active {
		if r.active[i].Name == name {
			clone := r.active[i]
			return &clone, nil
		}
	}
	return nil, service.ErrGroupNotFound
}

func (stubGroupRepo) Update(ctx context.Context, group *service.Group) error {
	return errors.New("not implemented")
}

func (stubGroupRepo) Delete(ctx context.Context, id int64) error {
	return errors.New("not implemented")
}

func (stubGroupRepo) DeleteCascade(ctx context.Context, id int64) ([]int64, error) {
	return nil, errors.New("not implemented")
}

func (stubGroupRepo) List(ctx context.Context, params pagination.PaginationParams) ([]service.Group, *pagination.PaginationResult, error) {
	return nil, nil, errors.New("not implemented")
}

func (stubGroupRepo) ListWithFilters(ctx context.Context, params pagination.PaginationParams, platform, status, search string, isExclusive *bool) ([]service.Group, *pagination.PaginationResult, error) {
	return nil, nil, errors.New("not implemented")
}

func (r *stubGroupRepo) ListActive(ctx context.Context) ([]service.Group, error) {
	return append([]service.Group(nil), r.active...), nil
}

func (r *stubGroupRepo) ListActiveByPlatform(ctx context.Context, platform string) ([]service.Group, error) {
	out := make([]service.Group, 0, len(r.active))
	for i := range r.active {
		g := r.active[i]
		if g.Platform == platform {
			out = append(out, g)
		}
	}
	return out, nil
}

func (stubGroupRepo) ExistsByName(ctx context.Context, name string) (bool, error) {
	return false, errors.New("not implemented")
}

func (stubGroupRepo) GetAccountCount(ctx context.Context, groupID int64) (int64, int64, error) {
	return 0, 0, errors.New("not implemented")
}

func (stubGroupRepo) DeleteAccountGroupsByGroupID(ctx context.Context, groupID int64) (int64, error) {
	return 0, errors.New("not implemented")
}

func (stubGroupRepo) BindAccountsToGroup(ctx context.Context, groupID int64, accountIDs []int64) error {
	return errors.New("not implemented")
}

func (stubGroupRepo) GetAccountIDsByGroupIDs(ctx context.Context, groupIDs []int64) ([]int64, error) {
	return nil, errors.New("not implemented")
}

func (stubGroupRepo) UpdateSortOrders(ctx context.Context, updates []service.GroupSortOrderUpdate) error {
	return nil
}

type stubAccountRepo struct {
	bulkUpdateIDs []int64
}

func (s *stubAccountRepo) Create(ctx context.Context, account *service.Account) error {
	return errors.New("not implemented")
}

func (s *stubAccountRepo) GetByID(ctx context.Context, id int64) (*service.Account, error) {
	return &service.Account{ID: id, Status: service.StatusActive, Schedulable: true}, nil
}

func (s *stubAccountRepo) GetByIDs(ctx context.Context, ids []int64) ([]*service.Account, error) {
	out := make([]*service.Account, 0, len(ids))
	for _, id := range ids {
		account := &service.Account{ID: id, Status: service.StatusActive, Schedulable: true}
		out = append(out, account)
	}
	return out, nil
}

func (s *stubAccountRepo) ExistsByID(ctx context.Context, id int64) (bool, error) {
	return false, errors.New("not implemented")
}

func (s *stubAccountRepo) GetByCRSAccountID(ctx context.Context, crsAccountID string) (*service.Account, error) {
	return nil, errors.New("not implemented")
}

func (s *stubAccountRepo) FindByExtraField(ctx context.Context, key string, value any) ([]service.Account, error) {
	return nil, errors.New("not implemented")
}

func (s *stubAccountRepo) Update(ctx context.Context, account *service.Account) error {
	return errors.New("not implemented")
}

func (s *stubAccountRepo) Delete(ctx context.Context, id int64) error {
	return errors.New("not implemented")
}

func (s *stubAccountRepo) List(ctx context.Context, params pagination.PaginationParams) ([]service.Account, *pagination.PaginationResult, error) {
	return nil, nil, errors.New("not implemented")
}

func (s *stubAccountRepo) ListWithFilters(ctx context.Context, params pagination.PaginationParams, platform, accountType, status, search string, groupID int64, lifecycle string, privacyMode string) ([]service.Account, *pagination.PaginationResult, error) {
	_ = platform
	_ = accountType
	_ = status
	_ = search
	_ = groupID
	_ = lifecycle
	_ = privacyMode
	return nil, nil, errors.New("not implemented")
}
func (s *stubAccountRepo) GetStatusSummary(ctx context.Context, filters service.AccountStatusSummaryFilters) (*service.AccountStatusSummary, error) {
	return nil, errors.New("not implemented")
}

func (s *stubAccountRepo) ListByGroup(ctx context.Context, groupID int64) ([]service.Account, error) {
	return nil, errors.New("not implemented")
}

func (s *stubAccountRepo) ListActive(ctx context.Context) ([]service.Account, error) {
	return nil, errors.New("not implemented")
}

func (s *stubAccountRepo) ListByPlatform(ctx context.Context, platform string) ([]service.Account, error) {
	return nil, errors.New("not implemented")
}

func (s *stubAccountRepo) UpdateLastUsed(ctx context.Context, id int64) error {
	return errors.New("not implemented")
}

func (s *stubAccountRepo) BatchUpdateLastUsed(ctx context.Context, updates map[int64]time.Time) error {
	return errors.New("not implemented")
}

func (s *stubAccountRepo) SetError(ctx context.Context, id int64, errorMsg string) error {
	return errors.New("not implemented")
}

func (s *stubAccountRepo) ClearError(ctx context.Context, id int64) error {
	return errors.New("not implemented")
}

func (s *stubAccountRepo) SetSchedulable(ctx context.Context, id int64, schedulable bool) error {
	return errors.New("not implemented")
}

func (s *stubAccountRepo) BindGroups(ctx context.Context, accountID int64, groupIDs []int64) error {
	return errors.New("not implemented")
}

func (s *stubAccountRepo) ListSchedulable(ctx context.Context) ([]service.Account, error) {
	return nil, errors.New("not implemented")
}

func (s *stubAccountRepo) ListSchedulableByGroupID(ctx context.Context, groupID int64) ([]service.Account, error) {
	return nil, errors.New("not implemented")
}

func (s *stubAccountRepo) ListSchedulableByPlatform(ctx context.Context, platform string) ([]service.Account, error) {
	return nil, errors.New("not implemented")
}

func (s *stubAccountRepo) ListSchedulableByGroupIDAndPlatform(ctx context.Context, groupID int64, platform string) ([]service.Account, error) {
	return nil, errors.New("not implemented")
}

func (s *stubAccountRepo) ListSchedulableByPlatforms(ctx context.Context, platforms []string) ([]service.Account, error) {
	return nil, errors.New("not implemented")
}

func (s *stubAccountRepo) ListSchedulableByGroupIDAndPlatforms(ctx context.Context, groupID int64, platforms []string) ([]service.Account, error) {
	return nil, errors.New("not implemented")
}

func (s *stubAccountRepo) ListSchedulableUngroupedByPlatform(ctx context.Context, platform string) ([]service.Account, error) {
	return nil, errors.New("not implemented")
}

func (s *stubAccountRepo) ListSchedulableUngroupedByPlatforms(ctx context.Context, platforms []string) ([]service.Account, error) {
	return nil, errors.New("not implemented")
}

func (s *stubAccountRepo) SetRateLimited(ctx context.Context, id int64, resetAt time.Time) error {
	return errors.New("not implemented")
}

func (s *stubAccountRepo) SetModelRateLimit(ctx context.Context, id int64, scope string, resetAt time.Time) error {
	return errors.New("not implemented")
}

func (s *stubAccountRepo) SetOverloaded(ctx context.Context, id int64, until time.Time) error {
	return errors.New("not implemented")
}

func (s *stubAccountRepo) SetTempUnschedulable(ctx context.Context, id int64, until time.Time, reason string) error {
	return errors.New("not implemented")
}

func (s *stubAccountRepo) ClearTempUnschedulable(ctx context.Context, id int64) error {
	return errors.New("not implemented")
}

func (s *stubAccountRepo) ClearRateLimit(ctx context.Context, id int64) error {
	return errors.New("not implemented")
}

func (s *stubAccountRepo) ClearAntigravityQuotaScopes(ctx context.Context, id int64) error {
	return errors.New("not implemented")
}

func (s *stubAccountRepo) ClearModelRateLimits(ctx context.Context, id int64) error {
	return errors.New("not implemented")
}

func (s *stubAccountRepo) UpdateSessionWindow(ctx context.Context, id int64, start, end *time.Time, status string) error {
	return errors.New("not implemented")
}

func (s *stubAccountRepo) UpdateExtra(ctx context.Context, id int64, updates map[string]any) error {
	return errors.New("not implemented")
}

func (s *stubAccountRepo) IncrementQuotaUsed(ctx context.Context, id int64, amount float64) error {
	return errors.New("not implemented")
}

func (s *stubAccountRepo) ResetQuotaUsed(ctx context.Context, id int64) error {
	return errors.New("not implemented")
}
func (s *stubAccountRepo) MarkBlacklisted(ctx context.Context, id int64, reasonCode, reasonMessage string, blacklistedAt, purgeAt time.Time) error {
	return errors.New("not implemented")
}
func (s *stubAccountRepo) RestoreBlacklisted(ctx context.Context, id int64) error {
	return errors.New("not implemented")
}
func (s *stubAccountRepo) ListBlacklistedIDs(ctx context.Context) ([]int64, error) {
	return nil, errors.New("not implemented")
}
func (s *stubAccountRepo) ListBlacklistedForPurge(ctx context.Context, now time.Time, limit int) ([]service.Account, error) {
	return nil, errors.New("not implemented")
}

func (s *stubAccountRepo) BulkUpdate(ctx context.Context, ids []int64, updates service.AccountBulkUpdate) (int64, error) {
	s.bulkUpdateIDs = append([]int64{}, ids...)
	return int64(len(ids)), nil
}

func (s *stubAccountRepo) ListCRSAccountIDs(ctx context.Context) (map[string]int64, error) {
	return nil, errors.New("not implemented")
}

type stubProxyRepo struct{}

func (stubProxyRepo) Create(ctx context.Context, proxy *service.Proxy) error {
	return errors.New("not implemented")
}

func (stubProxyRepo) GetByID(ctx context.Context, id int64) (*service.Proxy, error) {
	return nil, service.ErrProxyNotFound
}

func (stubProxyRepo) ListByIDs(ctx context.Context, ids []int64) ([]service.Proxy, error) {
	return nil, errors.New("not implemented")
}

func (stubProxyRepo) Update(ctx context.Context, proxy *service.Proxy) error {
	return errors.New("not implemented")
}

func (stubProxyRepo) Delete(ctx context.Context, id int64) error {
	return errors.New("not implemented")
}

func (stubProxyRepo) List(ctx context.Context, params pagination.PaginationParams) ([]service.Proxy, *pagination.PaginationResult, error) {
	return nil, nil, errors.New("not implemented")
}

func (stubProxyRepo) ListWithFilters(ctx context.Context, params pagination.PaginationParams, protocol, status, search string) ([]service.Proxy, *pagination.PaginationResult, error) {
	return nil, nil, errors.New("not implemented")
}

func (stubProxyRepo) ListWithFiltersAndAccountCount(ctx context.Context, params pagination.PaginationParams, protocol, status, search string) ([]service.ProxyWithAccountCount, *pagination.PaginationResult, error) {
	return nil, nil, errors.New("not implemented")
}

func (stubProxyRepo) ListActive(ctx context.Context) ([]service.Proxy, error) {
	return nil, errors.New("not implemented")
}

func (stubProxyRepo) ListActiveWithAccountCount(ctx context.Context) ([]service.ProxyWithAccountCount, error) {
	return nil, errors.New("not implemented")
}

func (stubProxyRepo) ExistsByHostPortAuth(ctx context.Context, host string, port int, username, password string) (bool, error) {
	return false, errors.New("not implemented")
}

func (stubProxyRepo) CountAccountsByProxyID(ctx context.Context, proxyID int64) (int64, error) {
	return 0, errors.New("not implemented")
}

func (stubProxyRepo) ListAccountSummariesByProxyID(ctx context.Context, proxyID int64) ([]service.ProxyAccountSummary, error) {
	return nil, errors.New("not implemented")
}

type stubRedeemCodeRepo struct {
	byUser map[int64][]service.RedeemCode
}

func (r *stubRedeemCodeRepo) SetByUser(userID int64, codes []service.RedeemCode) {
	if r.byUser == nil {
		r.byUser = make(map[int64][]service.RedeemCode)
	}
	r.byUser[userID] = append([]service.RedeemCode(nil), codes...)
}

func (stubRedeemCodeRepo) Create(ctx context.Context, code *service.RedeemCode) error {
	return errors.New("not implemented")
}

func (stubRedeemCodeRepo) CreateBatch(ctx context.Context, codes []service.RedeemCode) error {
	return errors.New("not implemented")
}

func (stubRedeemCodeRepo) GetByID(ctx context.Context, id int64) (*service.RedeemCode, error) {
	return nil, service.ErrRedeemCodeNotFound
}

func (stubRedeemCodeRepo) GetByCode(ctx context.Context, code string) (*service.RedeemCode, error) {
	return nil, service.ErrRedeemCodeNotFound
}

func (stubRedeemCodeRepo) Update(ctx context.Context, code *service.RedeemCode) error {
	return errors.New("not implemented")
}

func (stubRedeemCodeRepo) Delete(ctx context.Context, id int64) error {
	return errors.New("not implemented")
}

func (stubRedeemCodeRepo) Use(ctx context.Context, id, userID int64) error {
	return errors.New("not implemented")
}

func (stubRedeemCodeRepo) List(ctx context.Context, params pagination.PaginationParams) ([]service.RedeemCode, *pagination.PaginationResult, error) {
	return nil, nil, errors.New("not implemented")
}

func (stubRedeemCodeRepo) ListWithFilters(ctx context.Context, params pagination.PaginationParams, codeType, status, search string) ([]service.RedeemCode, *pagination.PaginationResult, error) {
	return nil, nil, errors.New("not implemented")
}

func (r *stubRedeemCodeRepo) ListByUser(ctx context.Context, userID int64, limit int) ([]service.RedeemCode, error) {
	if r.byUser == nil {
		return nil, nil
	}
	codes := r.byUser[userID]
	if limit > 0 && len(codes) > limit {
		codes = codes[:limit]
	}
	return append([]service.RedeemCode(nil), codes...), nil
}

func (stubRedeemCodeRepo) ListByUserPaginated(ctx context.Context, userID int64, params pagination.PaginationParams, codeType string) ([]service.RedeemCode, *pagination.PaginationResult, error) {
	return nil, nil, errors.New("not implemented")
}

func (stubRedeemCodeRepo) SumPositiveBalanceByUser(ctx context.Context, userID int64) (float64, error) {
	return 0, errors.New("not implemented")
}

type stubUserSubscriptionRepo struct {
	byUser       map[int64][]service.UserSubscription
	activeByUser map[int64][]service.UserSubscription
}

func (r *stubUserSubscriptionRepo) SetByUserID(userID int64, subs []service.UserSubscription) {
	if r.byUser == nil {
		r.byUser = make(map[int64][]service.UserSubscription)
	}
	r.byUser[userID] = append([]service.UserSubscription(nil), subs...)
}

func (r *stubUserSubscriptionRepo) SetActiveByUserID(userID int64, subs []service.UserSubscription) {
	if r.activeByUser == nil {
		r.activeByUser = make(map[int64][]service.UserSubscription)
	}
	r.activeByUser[userID] = append([]service.UserSubscription(nil), subs...)
}

func (stubUserSubscriptionRepo) Create(ctx context.Context, sub *service.UserSubscription) error {
	return errors.New("not implemented")
}
func (stubUserSubscriptionRepo) GetByID(ctx context.Context, id int64) (*service.UserSubscription, error) {
	return nil, errors.New("not implemented")
}
func (stubUserSubscriptionRepo) GetByUserIDAndGroupID(ctx context.Context, userID, groupID int64) (*service.UserSubscription, error) {
	return nil, errors.New("not implemented")
}
func (stubUserSubscriptionRepo) GetActiveByUserIDAndGroupID(ctx context.Context, userID, groupID int64) (*service.UserSubscription, error) {
	return nil, errors.New("not implemented")
}
func (stubUserSubscriptionRepo) Update(ctx context.Context, sub *service.UserSubscription) error {
	return errors.New("not implemented")
}
func (stubUserSubscriptionRepo) Delete(ctx context.Context, id int64) error {
	return errors.New("not implemented")
}
func (r *stubUserSubscriptionRepo) ListByUserID(ctx context.Context, userID int64) ([]service.UserSubscription, error) {
	if r.byUser == nil {
		return nil, nil
	}
	return append([]service.UserSubscription(nil), r.byUser[userID]...), nil
}
func (r *stubUserSubscriptionRepo) ListActiveByUserID(ctx context.Context, userID int64) ([]service.UserSubscription, error) {
	if r.activeByUser == nil {
		return nil, nil
	}
	return append([]service.UserSubscription(nil), r.activeByUser[userID]...), nil
}
func (stubUserSubscriptionRepo) ListByGroupID(ctx context.Context, groupID int64, params pagination.PaginationParams) ([]service.UserSubscription, *pagination.PaginationResult, error) {
	return nil, nil, errors.New("not implemented")
}
func (stubUserSubscriptionRepo) List(ctx context.Context, params pagination.PaginationParams, userID, groupID *int64, status, platform, sortBy, sortOrder string) ([]service.UserSubscription, *pagination.PaginationResult, error) {
	return nil, nil, errors.New("not implemented")
}
func (stubUserSubscriptionRepo) ExistsByUserIDAndGroupID(ctx context.Context, userID, groupID int64) (bool, error) {
	return false, errors.New("not implemented")
}
func (stubUserSubscriptionRepo) ExtendExpiry(ctx context.Context, subscriptionID int64, newExpiresAt time.Time) error {
	return errors.New("not implemented")
}
func (stubUserSubscriptionRepo) UpdateStatus(ctx context.Context, subscriptionID int64, status string) error {
	return errors.New("not implemented")
}
func (stubUserSubscriptionRepo) UpdateNotes(ctx context.Context, subscriptionID int64, notes string) error {
	return errors.New("not implemented")
}
func (stubUserSubscriptionRepo) ActivateWindows(ctx context.Context, id int64, start time.Time) error {
	return errors.New("not implemented")
}
func (stubUserSubscriptionRepo) ResetDailyUsage(ctx context.Context, id int64, newWindowStart time.Time) error {
	return errors.New("not implemented")
}
func (stubUserSubscriptionRepo) ResetWeeklyUsage(ctx context.Context, id int64, newWindowStart time.Time) error {
	return errors.New("not implemented")
}
func (stubUserSubscriptionRepo) ResetMonthlyUsage(ctx context.Context, id int64, newWindowStart time.Time) error {
	return errors.New("not implemented")
}
func (stubUserSubscriptionRepo) IncrementUsage(ctx context.Context, id int64, costUSD float64) error {
	return errors.New("not implemented")
}
func (stubUserSubscriptionRepo) BatchUpdateExpiredStatus(ctx context.Context) (int64, error) {
	return 0, errors.New("not implemented")
}

type stubApiKeyRepo struct {
	now time.Time

	nextID int64
	byID   map[int64]*service.APIKey
	byKey  map[string]*service.APIKey
}

func newStubApiKeyRepo(now time.Time) *stubApiKeyRepo {
	return &stubApiKeyRepo{
		now:    now,
		nextID: 100,
		byID:   make(map[int64]*service.APIKey),
		byKey:  make(map[string]*service.APIKey),
	}
}

func (r *stubApiKeyRepo) MustSeed(key *service.APIKey) {
	if key == nil {
		return
	}
	clone := *key
	r.byID[clone.ID] = &clone
	r.byKey[clone.Key] = &clone
}

func (r *stubApiKeyRepo) Create(ctx context.Context, key *service.APIKey) error {
	if key == nil {
		return errors.New("nil key")
	}
	if key.ID == 0 {
		key.ID = r.nextID
		r.nextID++
	}
	if key.CreatedAt.IsZero() {
		key.CreatedAt = r.now
	}
	if key.UpdatedAt.IsZero() {
		key.UpdatedAt = r.now
	}
	clone := *key
	r.byID[clone.ID] = &clone
	r.byKey[clone.Key] = &clone
	return nil
}

func (r *stubApiKeyRepo) GetByID(ctx context.Context, id int64) (*service.APIKey, error) {
	key, ok := r.byID[id]
	if !ok {
		return nil, service.ErrAPIKeyNotFound
	}
	clone := *key
	return &clone, nil
}

func (r *stubApiKeyRepo) GetKeyAndOwnerID(ctx context.Context, id int64) (string, int64, error) {
	key, ok := r.byID[id]
	if !ok {
		return "", 0, service.ErrAPIKeyNotFound
	}
	return key.Key, key.UserID, nil
}

func (r *stubApiKeyRepo) GetByKey(ctx context.Context, key string) (*service.APIKey, error) {
	found, ok := r.byKey[key]
	if !ok {
		return nil, service.ErrAPIKeyNotFound
	}
	clone := *found
	return &clone, nil
}

func (r *stubApiKeyRepo) GetByKeyForAuth(ctx context.Context, key string) (*service.APIKey, error) {
	return r.GetByKey(ctx, key)
}

func (r *stubApiKeyRepo) Update(ctx context.Context, key *service.APIKey) error {
	if key == nil {
		return errors.New("nil key")
	}
	if _, ok := r.byID[key.ID]; !ok {
		return service.ErrAPIKeyNotFound
	}
	if key.UpdatedAt.IsZero() {
		key.UpdatedAt = r.now
	}
	clone := *key
	r.byID[clone.ID] = &clone
	r.byKey[clone.Key] = &clone
	return nil
}

func (r *stubApiKeyRepo) Delete(ctx context.Context, id int64) error {
	key, ok := r.byID[id]
	if !ok {
		return service.ErrAPIKeyNotFound
	}
	delete(r.byID, id)
	delete(r.byKey, key.Key)
	return nil
}

func (r *stubApiKeyRepo) ListByUserID(ctx context.Context, userID int64, params pagination.PaginationParams, _ service.APIKeyListFilters) ([]service.APIKey, *pagination.PaginationResult, error) {
	ids := make([]int64, 0, len(r.byID))
	for id := range r.byID {
		if r.byID[id].UserID == userID {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] > ids[j] })

	start := params.Offset()
	if start > len(ids) {
		start = len(ids)
	}
	end := start + params.Limit()
	if end > len(ids) {
		end = len(ids)
	}

	out := make([]service.APIKey, 0, end-start)
	for _, id := range ids[start:end] {
		clone := *r.byID[id]
		out = append(out, clone)
	}

	total := int64(len(ids))
	pageSize := params.Limit()
	pages := int(math.Ceil(float64(total) / float64(pageSize)))
	if pages < 1 {
		pages = 1
	}
	return out, &pagination.PaginationResult{
		Total:    total,
		Page:     params.Page,
		PageSize: pageSize,
		Pages:    pages,
	}, nil
}

func (r *stubApiKeyRepo) VerifyOwnership(ctx context.Context, userID int64, apiKeyIDs []int64) ([]int64, error) {
	if len(apiKeyIDs) == 0 {
		return []int64{}, nil
	}
	seen := make(map[int64]struct{}, len(apiKeyIDs))
	out := make([]int64, 0, len(apiKeyIDs))
	for _, id := range apiKeyIDs {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		key, ok := r.byID[id]
		if ok && key.UserID == userID {
			out = append(out, id)
		}
	}
	return out, nil
}

func (r *stubApiKeyRepo) CountByUserID(ctx context.Context, userID int64) (int64, error) {
	var count int64
	for _, key := range r.byID {
		if key.UserID == userID {
			count++
		}
	}
	return count, nil
}

func (r *stubApiKeyRepo) ExistsByKey(ctx context.Context, key string) (bool, error) {
	_, ok := r.byKey[key]
	return ok, nil
}

func (r *stubApiKeyRepo) ListByGroupID(ctx context.Context, groupID int64, params pagination.PaginationParams) ([]service.APIKey, *pagination.PaginationResult, error) {
	return nil, nil, errors.New("not implemented")
}

func (r *stubApiKeyRepo) SearchAPIKeys(ctx context.Context, userID int64, keyword string, limit int) ([]service.APIKey, error) {
	return nil, errors.New("not implemented")
}

func (r *stubApiKeyRepo) ClearGroupIDByGroupID(ctx context.Context, groupID int64) (int64, error) {
	return 0, errors.New("not implemented")
}

func (r *stubApiKeyRepo) UpdateGroupIDByUserAndGroup(ctx context.Context, userID, oldGroupID, newGroupID int64) (int64, error) {
	var updated int64
	for id, key := range r.byID {
		if key.UserID != userID || key.GroupID == nil || *key.GroupID != oldGroupID {
			continue
		}
		clone := *key
		gid := newGroupID
		clone.GroupID = &gid
		r.byID[id] = &clone
		r.byKey[clone.Key] = &clone
		updated++
	}
	return updated, nil
}

func (r *stubApiKeyRepo) CountByGroupID(ctx context.Context, groupID int64) (int64, error) {
	return 0, errors.New("not implemented")
}

func (r *stubApiKeyRepo) ListKeysByUserID(ctx context.Context, userID int64) ([]string, error) {
	return nil, errors.New("not implemented")
}

func (r *stubApiKeyRepo) ListKeysByGroupID(ctx context.Context, groupID int64) ([]string, error) {
	return nil, errors.New("not implemented")
}
func (r *stubApiKeyRepo) GetAPIKeyGroups(ctx context.Context, keyID int64) ([]service.APIKeyGroupBinding, error) {
	return nil, errors.New("not implemented")
}
func (r *stubApiKeyRepo) SetAPIKeyGroups(ctx context.Context, keyID int64, bindings []service.APIKeyGroupBinding) error {
	return errors.New("not implemented")
}
func (r *stubApiKeyRepo) IncrementAPIKeyGroupQuotaUsed(ctx context.Context, keyID, groupID int64, amount float64) error {
	return errors.New("not implemented")
}

func (r *stubApiKeyRepo) IncrementQuotaUsed(ctx context.Context, id int64, amount float64) (float64, error) {
	return 0, errors.New("not implemented")
}

func (r *stubApiKeyRepo) TryReserveImageCount(ctx context.Context, id int64, count int) (bool, error) {
	if count <= 0 {
		return true, nil
	}
	key, ok := r.byID[id]
	if !ok {
		return false, service.ErrAPIKeyNotFound
	}
	if key.ImageMaxCount > 0 && key.ImageCountUsed+count > key.ImageMaxCount {
		return false, nil
	}
	key.ImageCountUsed += count
	clone := *key
	r.byID[id] = &clone
	r.byKey[clone.Key] = &clone
	return true, nil
}

func (r *stubApiKeyRepo) RollbackImageCount(ctx context.Context, id int64, count int) error {
	if count <= 0 {
		return nil
	}
	key, ok := r.byID[id]
	if !ok {
		return service.ErrAPIKeyNotFound
	}
	key.ImageCountUsed -= count
	if key.ImageCountUsed < 0 {
		key.ImageCountUsed = 0
	}
	clone := *key
	r.byID[id] = &clone
	r.byKey[clone.Key] = &clone
	return nil
}

func (r *stubApiKeyRepo) UpdateLastUsed(ctx context.Context, id int64, usedAt time.Time) error {
	key, ok := r.byID[id]
	if !ok {
		return service.ErrAPIKeyNotFound
	}
	ts := usedAt
	key.LastUsedAt = &ts
	key.UpdatedAt = usedAt
	clone := *key
	r.byID[id] = &clone
	r.byKey[clone.Key] = &clone
	return nil
}

func (r *stubApiKeyRepo) IncrementRateLimitUsage(ctx context.Context, id int64, cost float64) error {
	return nil
}
func (r *stubApiKeyRepo) ResetRateLimitWindows(ctx context.Context, id int64) error {
	return nil
}
func (r *stubApiKeyRepo) GetRateLimitData(ctx context.Context, id int64) (*service.APIKeyRateLimitData, error) {
	return nil, nil
}

type stubUsageLogRepo struct {
	userLogs map[int64][]service.UsageLog
}

func newStubUsageLogRepo() *stubUsageLogRepo {
	return &stubUsageLogRepo{userLogs: make(map[int64][]service.UsageLog)}
}

func (r *stubUsageLogRepo) SetUserLogs(userID int64, logs []service.UsageLog) {
	r.userLogs[userID] = logs
}

func (r *stubUsageLogRepo) Create(ctx context.Context, log *service.UsageLog) (bool, error) {
	return false, errors.New("not implemented")
}

func (r *stubUsageLogRepo) GetByID(ctx context.Context, id int64) (*service.UsageLog, error) {
	return nil, errors.New("not implemented")
}

func (r *stubUsageLogRepo) Delete(ctx context.Context, id int64) error {
	return errors.New("not implemented")
}

func (r *stubUsageLogRepo) ListByUser(ctx context.Context, userID int64, params pagination.PaginationParams) ([]service.UsageLog, *pagination.PaginationResult, error) {
	logs := r.userLogs[userID]
	total := int64(len(logs))
	out := paginateLogs(logs, params)
	return out, paginationResult(total, params), nil
}

func (r *stubUsageLogRepo) ListByAPIKey(ctx context.Context, apiKeyID int64, params pagination.PaginationParams) ([]service.UsageLog, *pagination.PaginationResult, error) {
	return nil, nil, errors.New("not implemented")
}

func (r *stubUsageLogRepo) ListByAccount(ctx context.Context, accountID int64, params pagination.PaginationParams) ([]service.UsageLog, *pagination.PaginationResult, error) {
	return nil, nil, errors.New("not implemented")
}

func (r *stubUsageLogRepo) ListByUserAndTimeRange(ctx context.Context, userID int64, startTime, endTime time.Time) ([]service.UsageLog, *pagination.PaginationResult, error) {
	logs := r.userLogs[userID]
	return logs, paginationResult(int64(len(logs)), pagination.PaginationParams{Page: 1, PageSize: 100}), nil
}

func (r *stubUsageLogRepo) ListByAPIKeyAndTimeRange(ctx context.Context, apiKeyID int64, startTime, endTime time.Time) ([]service.UsageLog, *pagination.PaginationResult, error) {
	return nil, nil, errors.New("not implemented")
}

func (r *stubUsageLogRepo) ListByAccountAndTimeRange(ctx context.Context, accountID int64, startTime, endTime time.Time) ([]service.UsageLog, *pagination.PaginationResult, error) {
	return nil, nil, errors.New("not implemented")
}

func (r *stubUsageLogRepo) ListByModelAndTimeRange(ctx context.Context, modelName string, startTime, endTime time.Time) ([]service.UsageLog, *pagination.PaginationResult, error) {
	return nil, nil, errors.New("not implemented")
}

func (r *stubUsageLogRepo) GetAccountWindowStats(ctx context.Context, accountID int64, startTime time.Time) (*usagestats.AccountStats, error) {
	return nil, errors.New("not implemented")
}

func (r *stubUsageLogRepo) GetAccountTodayStats(ctx context.Context, accountID int64) (*usagestats.AccountStats, error) {
	return nil, errors.New("not implemented")
}

func (r *stubUsageLogRepo) GetDashboardStats(ctx context.Context) (*usagestats.DashboardStats, error) {
	return nil, errors.New("not implemented")
}

func (r *stubUsageLogRepo) GetUsageTrendWithFilters(ctx context.Context, startTime, endTime time.Time, granularity string, userID, apiKeyID, accountID, groupID, channelID int64, model string, requestType *int16, stream *bool, billingType *int8) ([]usagestats.TrendDataPoint, error) {
	return nil, errors.New("not implemented")
}

func (r *stubUsageLogRepo) GetModelStatsWithFilters(ctx context.Context, startTime, endTime time.Time, userID, apiKeyID, accountID, groupID, channelID int64, requestType *int16, stream *bool, billingType *int8) ([]usagestats.ModelStat, error) {
	return nil, errors.New("not implemented")
}

func (r *stubUsageLogRepo) GetEndpointStatsWithFilters(ctx context.Context, startTime, endTime time.Time, userID, apiKeyID, accountID, groupID int64, model string, requestType *int16, stream *bool, billingType *int8) ([]usagestats.EndpointStat, error) {
	return nil, errors.New("not implemented")
}

func (r *stubUsageLogRepo) GetUpstreamEndpointStatsWithFilters(ctx context.Context, startTime, endTime time.Time, userID, apiKeyID, accountID, groupID int64, model string, requestType *int16, stream *bool, billingType *int8) ([]usagestats.EndpointStat, error) {
	return nil, errors.New("not implemented")
}

func (r *stubUsageLogRepo) GetGroupStatsWithFilters(ctx context.Context, startTime, endTime time.Time, userID, apiKeyID, accountID, groupID, channelID int64, requestType *int16, stream *bool, billingType *int8) ([]usagestats.GroupStat, error) {
	return nil, errors.New("not implemented")
}

func (r *stubUsageLogRepo) GetUserBreakdownStats(ctx context.Context, startTime, endTime time.Time, dim usagestats.UserBreakdownDimension, limit int) ([]usagestats.UserBreakdownItem, error) {
	return nil, errors.New("not implemented")
}

func (r *stubUsageLogRepo) GetAPIKeyUsageTrend(ctx context.Context, startTime, endTime time.Time, granularity string, limit int) ([]usagestats.APIKeyUsageTrendPoint, error) {
	return nil, errors.New("not implemented")
}

func (r *stubUsageLogRepo) GetUserUsageTrend(ctx context.Context, startTime, endTime time.Time, granularity string, limit int) ([]usagestats.UserUsageTrendPoint, error) {
	return nil, errors.New("not implemented")
}

func (r *stubUsageLogRepo) GetUserSpendingRanking(ctx context.Context, startTime, endTime time.Time, limit int) (*usagestats.UserSpendingRankingResponse, error) {
	return nil, errors.New("not implemented")
}

func (r *stubUsageLogRepo) GetUserStatsAggregated(ctx context.Context, userID int64, startTime, endTime time.Time) (*usagestats.UsageStats, error) {
	logs := r.userLogs[userID]
	if len(logs) == 0 {
		return &usagestats.UsageStats{}, nil
	}

	var totalRequests int64
	var totalInputTokens int64
	var totalOutputTokens int64
	var totalCacheTokens int64
	var totalCost float64
	var totalActualCost float64
	var totalDuration int64
	var durationCount int64

	for _, log := range logs {
		totalRequests++
		totalInputTokens += int64(log.InputTokens)
		totalOutputTokens += int64(log.OutputTokens)
		totalCacheTokens += int64(log.CacheCreationTokens + log.CacheReadTokens)
		totalCost += log.TotalCost
		totalActualCost += log.ActualCost
		if log.DurationMs != nil {
			totalDuration += int64(*log.DurationMs)
			durationCount++
		}
	}

	var avgDuration float64
	if durationCount > 0 {
		avgDuration = float64(totalDuration) / float64(durationCount)
	}

	return &usagestats.UsageStats{
		TotalRequests:     totalRequests,
		TotalInputTokens:  totalInputTokens,
		TotalOutputTokens: totalOutputTokens,
		TotalCacheTokens:  totalCacheTokens,
		TotalTokens:       totalInputTokens + totalOutputTokens + totalCacheTokens,
		TotalCost:         totalCost,
		TotalActualCost:   totalActualCost,
		AverageDurationMs: avgDuration,
	}, nil
}

func (r *stubUsageLogRepo) GetAPIKeyStatsAggregated(ctx context.Context, apiKeyID int64, startTime, endTime time.Time) (*usagestats.UsageStats, error) {
	return nil, errors.New("not implemented")
}

func (r *stubUsageLogRepo) GetAccountStatsAggregated(ctx context.Context, accountID int64, startTime, endTime time.Time) (*usagestats.UsageStats, error) {
	return nil, errors.New("not implemented")
}

func (r *stubUsageLogRepo) GetModelStatsAggregated(ctx context.Context, modelName string, startTime, endTime time.Time) (*usagestats.UsageStats, error) {
	return nil, errors.New("not implemented")
}

func (r *stubUsageLogRepo) GetDailyStatsAggregated(ctx context.Context, userID int64, startTime, endTime time.Time) ([]map[string]any, error) {
	return nil, errors.New("not implemented")
}

func (r *stubUsageLogRepo) GetBatchUserUsageStats(ctx context.Context, userIDs []int64, startTime, endTime time.Time) (map[int64]*usagestats.BatchUserUsageStats, error) {
	return nil, errors.New("not implemented")
}

func (r *stubUsageLogRepo) GetBatchAPIKeyUsageStats(ctx context.Context, apiKeyIDs []int64, startTime, endTime time.Time) (map[int64]*usagestats.BatchAPIKeyUsageStats, error) {
	return nil, errors.New("not implemented")
}

func (r *stubUsageLogRepo) GetUserDashboardStats(ctx context.Context, userID int64) (*usagestats.UserDashboardStats, error) {
	return nil, errors.New("not implemented")
}

func (r *stubUsageLogRepo) GetAPIKeyDashboardStats(ctx context.Context, apiKeyID int64) (*usagestats.UserDashboardStats, error) {
	return nil, errors.New("not implemented")
}

func (r *stubUsageLogRepo) GetUserUsageTrendByUserID(ctx context.Context, userID int64, startTime, endTime time.Time, granularity string) ([]usagestats.TrendDataPoint, error) {
	return nil, errors.New("not implemented")
}

func (r *stubUsageLogRepo) GetUserModelStats(ctx context.Context, userID int64, startTime, endTime time.Time) ([]usagestats.ModelStat, error) {
	return nil, errors.New("not implemented")
}

func (r *stubUsageLogRepo) ListWithFilters(ctx context.Context, params pagination.PaginationParams, filters usagestats.UsageLogFilters) ([]service.UsageLog, *pagination.PaginationResult, error) {
	filtered := filterUsageLogs(r.userLogs[filters.UserID], filters)
	total := int64(len(filtered))
	out := paginateLogs(filtered, params)
	return out, paginationResult(total, params), nil
}

func (r *stubUsageLogRepo) GetGlobalStats(ctx context.Context, startTime, endTime time.Time) (*usagestats.UsageStats, error) {
	return nil, errors.New("not implemented")
}

func (r *stubUsageLogRepo) GetAccountUsageStats(ctx context.Context, accountID int64, startTime, endTime time.Time) (*usagestats.AccountUsageStatsResponse, error) {
	return nil, errors.New("not implemented")
}

func (r *stubUsageLogRepo) GetStatsWithFilters(ctx context.Context, filters usagestats.UsageLogFilters) (*usagestats.UsageStats, error) {
	filtered := filterUsageLogs(r.userLogs[filters.UserID], filters)
	stats := buildUsageStatsFromLogs(filtered)
	if filters.TodayStart != nil || filters.TodayEnd != nil {
		todayFilters := filters
		todayFilters.StartTime = filters.TodayStart
		todayFilters.EndTime = filters.TodayEnd
		todayLogs := filterUsageLogs(r.userLogs[filters.UserID], todayFilters)
		todayStats := buildUsageStatsFromLogs(todayLogs)
		stats.TodayRequests = todayStats.TotalRequests
		stats.TodayInputTokens = todayStats.TotalInputTokens
		stats.TodayOutputTokens = todayStats.TotalOutputTokens
		stats.TodayCacheTokens = todayStats.TotalCacheTokens
		stats.TodayTokens = todayStats.TotalTokens
		stats.TodayCost = todayStats.TotalCost
		stats.TodayActualCost = todayStats.TotalActualCost
		stats.TodayAverageDurationMs = todayStats.AverageDurationMs
	}
	return stats, nil
}

func buildUsageStatsFromLogs(logs []service.UsageLog) *usagestats.UsageStats {
	stats := &usagestats.UsageStats{}
	totalDuration := 0
	durationCount := 0
	for _, log := range logs {
		stats.TotalRequests++
		stats.TotalInputTokens += int64(log.InputTokens)
		stats.TotalOutputTokens += int64(log.OutputTokens)
		stats.TotalCacheTokens += int64(log.CacheCreationTokens + log.CacheReadTokens + log.CacheCreation5mTokens + log.CacheCreation1hTokens)
		stats.TotalTokens += int64(log.InputTokens + log.OutputTokens + log.CacheCreationTokens + log.CacheReadTokens + log.CacheCreation5mTokens + log.CacheCreation1hTokens)
		stats.TotalCost += log.TotalCost
		stats.TotalActualCost += log.ActualCost
		if log.DurationMs != nil {
			totalDuration += *log.DurationMs
			durationCount++
		}
	}
	if durationCount > 0 {
		stats.AverageDurationMs = float64(totalDuration) / float64(durationCount)
	}
	return stats
}

func filterUsageLogs(logs []service.UsageLog, filters usagestats.UsageLogFilters) []service.UsageLog {
	filtered := make([]service.UsageLog, 0, len(logs))
	for _, log := range logs {
		if filters.APIKeyID > 0 && log.APIKeyID != filters.APIKeyID {
			continue
		}
		if filters.Model != "" && log.Model != filters.Model {
			continue
		}
		if filters.RequestType != nil && int16(log.RequestType) != *filters.RequestType {
			continue
		}
		if filters.Stream != nil && log.Stream != *filters.Stream {
			continue
		}
		if filters.BillingType != nil && log.BillingType != *filters.BillingType {
			continue
		}
		if filters.StartTime != nil && log.CreatedAt.Before(*filters.StartTime) {
			continue
		}
		if filters.EndTime != nil && !log.CreatedAt.Before(*filters.EndTime) {
			continue
		}
		filtered = append(filtered, log)
	}
	return filtered
}
func (r *stubUsageLogRepo) GetAllGroupUsageSummary(ctx context.Context, todayStart time.Time) ([]usagestats.GroupUsageSummary, error) {
	return nil, errors.New("not implemented")
}

type stubAffiliateRepo struct {
	now   time.Time
	users map[int64]*stubAffiliateUser
}

type stubAffiliateUser struct {
	email   string
	row     *service.UserAffiliate
	balance float64
}

func newStubAffiliateRepo(now time.Time) *stubAffiliateRepo {
	return &stubAffiliateRepo{
		now: now,
		users: map[int64]*stubAffiliateUser{
			1: {
				email: "alice@example.com",
				row: &service.UserAffiliate{
					UserID:              1,
					AffCode:             "AFFCODE0001",
					InviterUserID:       nil,
					InviterBoundAt:      nil,
					InviteeCount:        0,
					RebateBalance:       3.5,
					RebateFrozenBalance: 0,
					LifetimeRebate:      3.5,
					CustomAffCode:       false,
					CreatedAt:           now,
					UpdatedAt:           now,
				},
				balance: 12.5,
			},
		},
	}
}

func (r *stubAffiliateRepo) GetUserAffiliate(ctx context.Context, userID int64) (*service.UserAffiliate, error) {
	u, ok := r.users[userID]
	if !ok || u == nil || u.row == nil {
		return nil, errors.New("affiliate row not found")
	}
	clone := *u.row
	return &clone, nil
}

func (r *stubAffiliateRepo) EnsureAffiliateRow(ctx context.Context, userID int64, affCode string) (bool, error) {
	if userID <= 0 {
		return false, errors.New("user_id must be positive")
	}
	if _, ok := r.users[userID]; ok {
		return false, nil
	}
	if affCode == "" {
		affCode = "AFFCODE0000"
	}
	r.users[userID] = &stubAffiliateUser{
		email: "",
		row: &service.UserAffiliate{
			UserID:              userID,
			AffCode:             affCode,
			InviterUserID:       nil,
			InviterBoundAt:      nil,
			InviteeCount:        0,
			RebateBalance:       0,
			RebateFrozenBalance: 0,
			LifetimeRebate:      0,
			CustomAffCode:       false,
			CreatedAt:           r.now,
			UpdatedAt:           r.now,
		},
		balance: 0,
	}
	return true, nil
}

func (r *stubAffiliateRepo) BindInviterByCode(ctx context.Context, inviteeUserID int64, affCode string) (int64, bool, error) {
	return 0, false, nil
}

func (r *stubAffiliateRepo) AccrueTopupRebate(ctx context.Context, redeemCodeID int64, inviteeUserID int64, creditedAmount float64, policy service.AffiliateRebatePolicy) (float64, error) {
	return 0, nil
}

func (r *stubAffiliateRepo) ThawFrozenIfNeeded(ctx context.Context, inviterUserID int64) (float64, error) {
	return 0, nil
}

func (r *stubAffiliateRepo) TransferToBalance(ctx context.Context, userID int64) (*service.AffiliateTransferResult, error) {
	u, ok := r.users[userID]
	if !ok || u == nil || u.row == nil {
		return nil, errors.New("affiliate row not found")
	}
	transferred := u.row.RebateBalance
	u.row.RebateBalance = 0
	u.row.UpdatedAt = r.now
	u.balance += transferred
	return &service.AffiliateTransferResult{TransferredAmount: transferred, NewBalance: u.balance}, nil
}

func (r *stubAffiliateRepo) ListAffiliateUsers(ctx context.Context, params pagination.PaginationParams, filters service.AffiliateAdminUserListFilters) ([]service.AffiliateAdminUser, *pagination.PaginationResult, error) {
	u := r.users[1]
	if u == nil || u.row == nil {
		return []service.AffiliateAdminUser{}, paginationResult(0, params), nil
	}

	if filters.HasCustomCode != nil && *filters.HasCustomCode && !u.row.CustomAffCode {
		return []service.AffiliateAdminUser{}, paginationResult(0, params), nil
	}
	if filters.HasCustomRate != nil && *filters.HasCustomRate && u.row.CustomRebateRatePercent == nil {
		return []service.AffiliateAdminUser{}, paginationResult(0, params), nil
	}
	if filters.HasInviter != nil && *filters.HasInviter && u.row.InviterUserID == nil {
		return []service.AffiliateAdminUser{}, paginationResult(0, params), nil
	}

	item := service.AffiliateAdminUser{
		UserID:                  u.row.UserID,
		Email:                   u.email,
		AffCode:                 u.row.AffCode,
		CustomAffCode:           u.row.CustomAffCode,
		CustomRebateRatePercent: u.row.CustomRebateRatePercent,
		InviterUserID:           u.row.InviterUserID,
		InviteeCount:            u.row.InviteeCount,
		RebateBalance:           u.row.RebateBalance,
		RebateFrozenBalance:     u.row.RebateFrozenBalance,
		LifetimeRebate:          u.row.LifetimeRebate,
		UpdatedAt:               r.now,
	}

	return []service.AffiliateAdminUser{item}, paginationResult(1, params), nil
}

func (r *stubAffiliateRepo) LookupAffiliateUsers(ctx context.Context, q string, limit int) ([]service.AffiliateAdminUser, error) {
	if strings.TrimSpace(q) == "" {
		return []service.AffiliateAdminUser{}, nil
	}
	items, _, _ := r.ListAffiliateUsers(ctx, pagination.PaginationParams{Page: 1, PageSize: limit}, service.AffiliateAdminUserListFilters{})
	return items, nil
}

func (r *stubAffiliateRepo) UpdateAffiliateUserCustom(ctx context.Context, userID int64, update service.AffiliateAdminUserCustomUpdate, newAffCodeForClear string) (*service.UserAffiliate, error) {
	if _, err := r.EnsureAffiliateRow(ctx, userID, ""); err != nil {
		return nil, err
	}
	u := r.users[userID]
	if u == nil || u.row == nil {
		return nil, errors.New("affiliate row not found")
	}

	if update.AffCodeSet {
		if update.AffCode == nil {
			u.row.CustomAffCode = false
			if newAffCodeForClear != "" {
				u.row.AffCode = newAffCodeForClear
			}
		} else {
			u.row.CustomAffCode = true
			u.row.AffCode = *update.AffCode
		}
	}
	if update.CustomRateSet {
		u.row.CustomRebateRatePercent = update.CustomRate
	}
	u.row.UpdatedAt = r.now
	clone := *u.row
	return &clone, nil
}

func (r *stubAffiliateRepo) ResetAffiliateUserCustom(ctx context.Context, userID int64, newAffCode string) (*service.UserAffiliate, error) {
	if _, err := r.EnsureAffiliateRow(ctx, userID, ""); err != nil {
		return nil, err
	}
	u := r.users[userID]
	if u == nil || u.row == nil {
		return nil, errors.New("affiliate row not found")
	}
	u.row.CustomAffCode = false
	u.row.CustomRebateRatePercent = nil
	u.row.AffCode = "RESETCODE01"
	u.row.UpdatedAt = r.now
	clone := *u.row
	return &clone, nil
}

func (r *stubAffiliateRepo) BatchUpdateAffiliateUserCustomRates(ctx context.Context, userIDs []int64, customRatePercent float64) (int, error) {
	for _, id := range userIDs {
		if _, err := r.EnsureAffiliateRow(ctx, id, ""); err != nil {
			return 0, err
		}
		u := r.users[id]
		if u == nil || u.row == nil {
			continue
		}
		v := customRatePercent
		u.row.CustomRebateRatePercent = &v
		u.row.UpdatedAt = r.now
	}
	return len(userIDs), nil
}

type stubSettingRepo struct {
	all      map[string]string
	lastKeys []string
}

type stubAuthIdentityRepo struct {
	items []*service.AuthIdentity
}

type stubContentModerationAuditRepo struct {
	items []*service.ContentModerationAudit
}

func newStubContentModerationAuditRepo() *stubContentModerationAuditRepo {
	return &stubContentModerationAuditRepo{items: make([]*service.ContentModerationAudit, 0)}
}

func (r *stubContentModerationAuditRepo) CreateContentModerationAudit(ctx context.Context, audit *service.ContentModerationAudit) error {
	if audit == nil {
		return errors.New("audit required")
	}
	if audit.ID <= 0 {
		audit.ID = int64(len(r.items) + 1)
	}
	if audit.CreatedAt.IsZero() {
		audit.CreatedAt = time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	}
	clone := *audit
	r.items = append(r.items, &clone)
	return nil
}

func (r *stubContentModerationAuditRepo) FindRecentContentModerationAuditByHash(ctx context.Context, contentHash string, since time.Time) (*service.ContentModerationAudit, error) {
	for i := len(r.items) - 1; i >= 0; i-- {
		item := r.items[i]
		if item == nil || item.ContentHash != contentHash || item.CreatedAt.Before(since) {
			continue
		}
		clone := *item
		return &clone, nil
	}
	return nil, service.ErrContentModerationAuditNotFound
}

func (r *stubContentModerationAuditRepo) ListContentModerationAudits(ctx context.Context, filter *service.ContentModerationAuditFilter) (*service.ContentModerationAuditList, error) {
	page, pageSize := filter.Normalize()
	items := make([]*service.ContentModerationAudit, 0, len(r.items))
	for i := len(r.items) - 1; i >= 0; i-- {
		item := r.items[i]
		if item == nil {
			continue
		}
		if filter != nil {
			if filter.RequestID != "" && !strings.Contains(item.RequestID, filter.RequestID) {
				continue
			}
			if filter.ClientRequestID != "" && !strings.Contains(item.ClientRequestID, filter.ClientRequestID) {
				continue
			}
			if filter.Provider != "" && item.Provider != filter.Provider {
				continue
			}
			if filter.Model != "" && item.Model != filter.Model {
				continue
			}
			if filter.SourceEndpoint != "" && item.SourceEndpoint != filter.SourceEndpoint {
				continue
			}
			if filter.ContentHash != "" && item.ContentHash != filter.ContentHash {
				continue
			}
			if filter.UserID != nil {
				if item.UserID == nil || *item.UserID != *filter.UserID {
					continue
				}
			}
			if filter.Hit != nil && item.Hit != *filter.Hit {
				continue
			}
		}
		clone := *item
		items = append(items, &clone)
	}

	start := (page - 1) * pageSize
	if start > len(items) {
		start = len(items)
	}
	end := start + pageSize
	if end > len(items) {
		end = len(items)
	}
	return &service.ContentModerationAuditList{
		Items:    items[start:end],
		Total:    int64(len(items)),
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (r *stubContentModerationAuditRepo) GetContentModerationAuditByID(ctx context.Context, id int64) (*service.ContentModerationAudit, error) {
	for _, item := range r.items {
		if item != nil && item.ID == id {
			clone := *item
			return &clone, nil
		}
	}
	return nil, service.ErrContentModerationAuditNotFound
}

func newStubAuthIdentityRepo() *stubAuthIdentityRepo {
	return &stubAuthIdentityRepo{items: make([]*service.AuthIdentity, 0)}
}

func (r *stubAuthIdentityRepo) Create(ctx context.Context, identity *service.AuthIdentity) error {
	if identity == nil {
		return errors.New("identity required")
	}
	if identity.ID <= 0 {
		identity.ID = int64(len(r.items) + 1)
	}
	if identity.CreatedAt.IsZero() {
		identity.CreatedAt = time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	}
	if identity.UpdatedAt.IsZero() {
		identity.UpdatedAt = identity.CreatedAt
	}
	r.items = append(r.items, identity)
	return nil
}

func (r *stubAuthIdentityRepo) GetByProviderUserID(ctx context.Context, provider, providerUserID string) (*service.AuthIdentity, error) {
	for _, item := range r.items {
		if item.Provider == provider && item.ProviderUserID == providerUserID {
			clone := *item
			return &clone, nil
		}
	}
	return nil, service.ErrAuthIdentityNotFound
}

func (r *stubAuthIdentityRepo) GetByUserIDAndProvider(ctx context.Context, userID int64, provider string) (*service.AuthIdentity, error) {
	for _, item := range r.items {
		if item.UserID == userID && item.Provider == provider {
			clone := *item
			return &clone, nil
		}
	}
	return nil, service.ErrAuthIdentityNotFound
}

func (r *stubAuthIdentityRepo) ListByUserID(ctx context.Context, userID int64) ([]*service.AuthIdentity, error) {
	result := make([]*service.AuthIdentity, 0)
	for _, item := range r.items {
		if item.UserID != userID {
			continue
		}
		clone := *item
		result = append(result, &clone)
	}
	sort.Slice(result, func(i, j int) bool {
		return strconv.FormatInt(result[i].ID, 10) < strconv.FormatInt(result[j].ID, 10)
	})
	return result, nil
}

func (r *stubAuthIdentityRepo) DeleteByUserIDAndProvider(ctx context.Context, userID int64, provider string) error {
	next := make([]*service.AuthIdentity, 0, len(r.items))
	deleted := false
	for _, item := range r.items {
		if item.UserID == userID && item.Provider == provider {
			deleted = true
			continue
		}
		next = append(next, item)
	}
	if !deleted {
		return service.ErrAuthIdentityNotFound
	}
	r.items = next
	return nil
}

func newStubSettingRepo() *stubSettingRepo {
	return &stubSettingRepo{all: make(map[string]string)}
}

func (r *stubSettingRepo) SetAll(values map[string]string) {
	r.all = make(map[string]string, len(values))
	for k, v := range values {
		r.all[k] = v
	}
}

func (r *stubSettingRepo) Get(ctx context.Context, key string) (*service.Setting, error) {
	value, ok := r.all[key]
	if !ok {
		return nil, service.ErrSettingNotFound
	}
	return &service.Setting{Key: key, Value: value}, nil
}

func (r *stubSettingRepo) GetValue(ctx context.Context, key string) (string, error) {
	value, ok := r.all[key]
	if !ok {
		return "", service.ErrSettingNotFound
	}
	return value, nil
}

func (r *stubSettingRepo) Set(ctx context.Context, key, value string) error {
	r.all[key] = value
	return nil
}

func (r *stubSettingRepo) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	r.lastKeys = append([]string(nil), keys...)
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		out[key] = r.all[key]
	}
	return out, nil
}

func (r *stubSettingRepo) SetMultiple(ctx context.Context, settings map[string]string) error {
	for k, v := range settings {
		r.all[k] = v
	}
	return nil
}

func (r *stubSettingRepo) GetAll(ctx context.Context) (map[string]string, error) {
	out := make(map[string]string, len(r.all))
	for k, v := range r.all {
		out[k] = v
	}
	return out, nil
}

func (r *stubSettingRepo) Delete(ctx context.Context, key string) error {
	delete(r.all, key)
	return nil
}

func paginateLogs(logs []service.UsageLog, params pagination.PaginationParams) []service.UsageLog {
	start := params.Offset()
	if start > len(logs) {
		start = len(logs)
	}
	end := start + params.Limit()
	if end > len(logs) {
		end = len(logs)
	}
	out := make([]service.UsageLog, 0, end-start)
	out = append(out, logs[start:end]...)
	return out
}

func paginationResult(total int64, params pagination.PaginationParams) *pagination.PaginationResult {
	pageSize := params.Limit()
	pages := int(math.Ceil(float64(total) / float64(pageSize)))
	if pages < 1 {
		pages = 1
	}
	return &pagination.PaginationResult{
		Total:    total,
		Page:     params.Page,
		PageSize: pageSize,
		Pages:    pages,
	}
}

// Ensure compile-time interface compliance.
var (
	_ service.UserRepository                   = (*stubUserRepo)(nil)
	_ service.AuthIdentityRepository           = (*stubAuthIdentityRepo)(nil)
	_ service.ContentModerationAuditRepository = (*stubContentModerationAuditRepo)(nil)
	_ service.APIKeyRepository                 = (*stubApiKeyRepo)(nil)
	_ service.APIKeyCache                      = (*stubApiKeyCache)(nil)
	_ service.GroupRepository                  = (*stubGroupRepo)(nil)
	_ service.UserSubscriptionRepository       = (*stubUserSubscriptionRepo)(nil)
	_ service.UsageLogRepository               = (*stubUsageLogRepo)(nil)
	_ service.SettingRepository                = (*stubSettingRepo)(nil)
	_ service.AffiliateRepository              = (*stubAffiliateRepo)(nil)
)
