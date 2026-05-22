package service

import "context"

type PaymentService struct {
	repo                    PaymentRepository
	settings                *SettingService
	airwallex               AirwallexClient
	subscriptionSvc         *SubscriptionService
	affiliateService        *AffiliateService
	paymentSettingsOverride func(context.Context) PaymentSettings
}

func NewPaymentService(repo PaymentRepository, settings *SettingService, airwallex AirwallexClient, subscriptionSvc *SubscriptionService, affiliateService *AffiliateService) *PaymentService {
	return &PaymentService{repo: repo, settings: settings, airwallex: airwallex, subscriptionSvc: subscriptionSvc, affiliateService: affiliateService}
}
