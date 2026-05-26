package service

import "context"

type PaymentService struct {
	repo                    PaymentRepository
	settings                *SettingService
	airwallex               AirwallexClient
	subscriptionSvc         *SubscriptionService
	affiliateService        *AffiliateService
	emailService            *EmailService
	emailTemplates          *EmailTemplateService
	userRepo                UserRepository
	billingCacheService     *BillingCacheService
	authCacheInvalidator    APIKeyAuthCacheInvalidator
	paymentSettingsOverride func(context.Context) PaymentSettings
}

func NewPaymentService(repo PaymentRepository, settings *SettingService, airwallex AirwallexClient, subscriptionSvc *SubscriptionService, affiliateService *AffiliateService) *PaymentService {
	return &PaymentService{repo: repo, settings: settings, airwallex: airwallex, subscriptionSvc: subscriptionSvc, affiliateService: affiliateService}
}

func (s *PaymentService) SetNotificationServices(emailService *EmailService, templates *EmailTemplateService, userRepo UserRepository) {
	if s == nil {
		return
	}
	s.emailService = emailService
	s.emailTemplates = templates
	s.userRepo = userRepo
}

func (s *PaymentService) SetBillingCacheInvalidators(billingCacheService *BillingCacheService, authCacheInvalidator APIKeyAuthCacheInvalidator) {
	if s == nil {
		return
	}
	s.billingCacheService = billingCacheService
	s.authCacheInvalidator = authCacheInvalidator
}
