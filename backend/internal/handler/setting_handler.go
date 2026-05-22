package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// SettingHandler 公开设置处理器（无需认证）
type SettingHandler struct {
	settingService *service.SettingService
	version        string
}

// NewSettingHandler 创建公开设置处理器
func NewSettingHandler(settingService *service.SettingService, version string) *SettingHandler {
	return &SettingHandler{
		settingService: settingService,
		version:        version,
	}
}

// GetPublicSettings 获取公开设置
// GET /api/v1/settings/public
func (h *SettingHandler) GetPublicSettings(c *gin.Context) {
	settings, err := h.settingService.GetPublicSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.PublicSettings{
		RegistrationEnabled:              settings.RegistrationEnabled,
		EmailVerifyEnabled:               settings.EmailVerifyEnabled,
		RegistrationEmailSuffixWhitelist: settings.RegistrationEmailSuffixWhitelist,
		PromoCodeEnabled:                 settings.PromoCodeEnabled,
		PasswordResetEnabled:             settings.PasswordResetEnabled,
		InvitationCodeEnabled:            settings.InvitationCodeEnabled,
		TotpEnabled:                      settings.TotpEnabled,
		TurnstileEnabled:                 settings.TurnstileEnabled,
		TurnstileSiteKey:                 settings.TurnstileSiteKey,
		SiteName:                         settings.SiteName,
		SiteLogo:                         settings.SiteLogo,
		SiteSubtitle:                     settings.SiteSubtitle,
		VisualPresetDefault:              settings.VisualPresetDefault,
		AccountAiryWhiteSurfaceEnabled:   settings.AccountAiryWhiteSurfaceEnabled,
		APIBaseURL:                       settings.APIBaseURL,
		ContactInfo:                      settings.ContactInfo,
		DocURL:                           settings.DocURL,
		HomeContent:                      settings.HomeContent,
		HideCcsImportButton:              settings.HideCcsImportButton,
		AvailableChannelsEnabled:         settings.AvailableChannelsEnabled,
		ChannelMonitorEnabled:            settings.ChannelMonitorEnabled,
		PublicModelCatalogEnabled:        settings.PublicModelCatalogEnabled,
		AffiliateEnabled:                 settings.AffiliateEnabled,
		PurchaseSubscriptionEnabled:      settings.PurchaseSubscriptionEnabled,
		PurchaseSubscriptionURL:          settings.PurchaseSubscriptionURL,
		PaymentProviderAirwallexEnabled:  settings.PaymentProviderAirwallexEnabled,
		PaymentAllowedCurrencies:         settings.PaymentAllowedCurrencies,
		PaymentDefaultCurrency:           settings.PaymentDefaultCurrency,
		PaymentMinTopupAmount:            settings.PaymentMinTopupAmount,
		PaymentMaxTopupAmount:            settings.PaymentMaxTopupAmount,
		PaymentSubscriptionPlans:         buildPublicPaymentPlanDTOs(settings.PaymentSubscriptionPlans),
		CustomMenuItems:                  dto.ParseUserVisibleMenuItems(settings.CustomMenuItems),
		LoginAgreementEnabled:            settings.LoginAgreementEnabled,
		LoginAgreementMode:               settings.LoginAgreementMode,
		LoginAgreementUpdatedAt:          settings.LoginAgreementUpdatedAt,
		LoginAgreementDocuments:          buildPublicLoginAgreementDocumentDTOs(settings.LoginAgreementDocuments),
		LinuxDoOAuthEnabled:              settings.LinuxDoOAuthEnabled,
		GitHubOAuthEnabled:               settings.GitHubOAuthEnabled,
		GoogleOAuthEnabled:               settings.GoogleOAuthEnabled,
		BackendModeEnabled:               settings.BackendModeEnabled,
		MaintenanceModeEnabled:           settings.MaintenanceModeEnabled,
		Version:                          h.version,
	})
}

func buildPublicPaymentPlanDTOs(items []service.PaymentSubscriptionPlan) []dto.PaymentSubscriptionPlan {
	out := make([]dto.PaymentSubscriptionPlan, 0, len(items))
	for _, item := range items {
		if !item.Enabled {
			continue
		}
		out = append(out, dto.PaymentSubscriptionPlan{
			PlanID:           item.PlanID,
			Name:             item.Name,
			GroupID:          item.GroupID,
			ValidityDays:     item.ValidityDays,
			PricesByCurrency: item.PricesByCurrency,
			Enabled:          item.Enabled,
		})
	}
	return out
}

func buildPublicLoginAgreementDocumentDTOs(items []service.LoginAgreementDocument) []dto.LoginAgreementDocument {
	out := make([]dto.LoginAgreementDocument, 0, len(items))
	for _, item := range items {
		out = append(out, dto.LoginAgreementDocument{
			ID:       item.ID,
			Title:    item.Title,
			PageSlug: item.PageSlug,
		})
	}
	return out
}

// GetCustomPage returns markdown-backed custom page content.
// GET /api/v1/pages/:slug
func (h *SettingHandler) GetCustomPage(c *gin.Context) {
	page, err := h.settingService.GetCustomPageBySlug(c.Request.Context(), c.Param("slug"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	role, _ := middleware.GetUserRoleFromContext(c)
	if page.Visibility == "admin" && role != "admin" {
		response.ErrorFrom(c, infraerrors.NotFound("CUSTOM_PAGE_NOT_FOUND", "custom page not found"))
		return
	}

	response.Success(c, dto.PageContentResponse{
		ID:         page.ID,
		Slug:       page.Slug,
		Label:      page.Label,
		Visibility: page.Visibility,
		PageMode:   page.PageMode,
		Content:    page.Content,
	})
}
