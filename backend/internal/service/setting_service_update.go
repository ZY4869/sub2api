package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"strconv"
	"strings"
	"time"
)

func (s *SettingService) UpdateSettings(ctx context.Context, settings *SystemSettings) error {
	if err := s.validateDefaultSubscriptionGroups(ctx, settings.DefaultSubscriptions); err != nil {
		return err
	}
	normalizedWhitelist, err := NormalizeRegistrationEmailSuffixWhitelist(settings.RegistrationEmailSuffixWhitelist)
	if err != nil {
		return infraerrors.BadRequest("INVALID_REGISTRATION_EMAIL_SUFFIX_WHITELIST", err.Error())
	}
	if normalizedWhitelist == nil {
		normalizedWhitelist = []string{}
	}
	settings.RegistrationEmailSuffixWhitelist = normalizedWhitelist
	updates := make(map[string]string)
	updates[SettingKeyRegistrationEnabled] = strconv.FormatBool(settings.RegistrationEnabled)
	updates[SettingKeyEmailVerifyEnabled] = strconv.FormatBool(settings.EmailVerifyEnabled)
	registrationEmailSuffixWhitelistJSON, err := json.Marshal(settings.RegistrationEmailSuffixWhitelist)
	if err != nil {
		return fmt.Errorf("marshal registration email suffix whitelist: %w", err)
	}
	updates[SettingKeyRegistrationEmailSuffixWhitelist] = string(registrationEmailSuffixWhitelistJSON)
	updates[SettingKeyPromoCodeEnabled] = strconv.FormatBool(settings.PromoCodeEnabled)
	updates[SettingKeyPasswordResetEnabled] = strconv.FormatBool(settings.PasswordResetEnabled)
	updates[SettingKeyFrontendURL] = strings.TrimSpace(settings.FrontendURL)
	updates[SettingKeyInvitationCodeEnabled] = strconv.FormatBool(settings.InvitationCodeEnabled)
	updates[SettingKeyTotpEnabled] = strconv.FormatBool(settings.TotpEnabled)
	updates[SettingKeySMTPHost] = settings.SMTPHost
	updates[SettingKeySMTPPort] = strconv.Itoa(settings.SMTPPort)
	updates[SettingKeySMTPUsername] = settings.SMTPUsername
	if settings.SMTPPassword != "" {
		updates[SettingKeySMTPPassword] = settings.SMTPPassword
	}
	updates[SettingKeySMTPFrom] = settings.SMTPFrom
	updates[SettingKeySMTPFromName] = settings.SMTPFromName
	updates[SettingKeySMTPUseTLS] = strconv.FormatBool(settings.SMTPUseTLS)
	updates[SettingKeyTelegramChatID] = strings.TrimSpace(settings.TelegramChatID)
	if strings.TrimSpace(settings.TelegramBotToken) != "" {
		updates[SettingKeyTelegramBotToken] = strings.TrimSpace(settings.TelegramBotToken)
	}
	updates[SettingKeyTurnstileEnabled] = strconv.FormatBool(settings.TurnstileEnabled)
	updates[SettingKeyTurnstileSiteKey] = settings.TurnstileSiteKey
	if settings.TurnstileSecretKey != "" {
		updates[SettingKeyTurnstileSecretKey] = settings.TurnstileSecretKey
	}
	updates[SettingKeyLinuxDoConnectEnabled] = strconv.FormatBool(settings.LinuxDoConnectEnabled)
	updates[SettingKeyLinuxDoConnectClientID] = settings.LinuxDoConnectClientID
	updates[SettingKeyLinuxDoConnectRedirectURL] = settings.LinuxDoConnectRedirectURL
	if settings.LinuxDoConnectClientSecret != "" {
		updates[SettingKeyLinuxDoConnectClientSecret] = settings.LinuxDoConnectClientSecret
	}
	updates[SettingKeySiteName] = settings.SiteName
	updates[SettingKeySiteLogo] = settings.SiteLogo
	updates[SettingKeySiteSubtitle] = settings.SiteSubtitle
	updates[SettingKeyAPIBaseURL] = settings.APIBaseURL
	updates[SettingKeyContactInfo] = settings.ContactInfo
	updates[SettingKeyDocURL] = settings.DocURL
	updates[SettingKeyHomeContent] = settings.HomeContent
	updates[SettingKeyHideCcsImportButton] = strconv.FormatBool(settings.HideCcsImportButton)
	updates[SettingKeyAvailableChannelsEnabled] = strconv.FormatBool(settings.AvailableChannelsEnabled)
	updates[SettingKeyChannelMonitorEnabled] = strconv.FormatBool(settings.ChannelMonitorEnabled)
	monitorInterval := settings.ChannelMonitorDefaultIntervalSeconds
	if monitorInterval <= 0 {
		monitorInterval = 60
	}
	if monitorInterval < 15 {
		monitorInterval = 15
	}
	if monitorInterval > 3600 {
		monitorInterval = 3600
	}
	updates[SettingKeyChannelMonitorDefaultIntervalSeconds] = strconv.Itoa(monitorInterval)
	updates[SettingKeyPublicModelCatalogEnabled] = strconv.FormatBool(settings.PublicModelCatalogEnabled)
	updates[SettingKeyPurchaseSubscriptionEnabled] = strconv.FormatBool(settings.PurchaseSubscriptionEnabled)
	updates[SettingKeyPurchaseSubscriptionURL] = strings.TrimSpace(settings.PurchaseSubscriptionURL)
	updates[SettingKeyCustomMenuItems] = settings.CustomMenuItems
	updates[SettingKeyDefaultConcurrency] = strconv.Itoa(settings.DefaultConcurrency)
	updates[SettingKeyDefaultBalance] = strconv.FormatFloat(settings.DefaultBalance, 'f', 8, 64)
	defaultSubsJSON, err := json.Marshal(settings.DefaultSubscriptions)
	if err != nil {
		return fmt.Errorf("marshal default subscriptions: %w", err)
	}
	updates[SettingKeyDefaultSubscriptions] = string(defaultSubsJSON)
	updates[SettingKeyEnableModelFallback] = strconv.FormatBool(settings.EnableModelFallback)
	updates[SettingKeyFallbackModelAnthropic] = settings.FallbackModelAnthropic
	updates[SettingKeyFallbackModelOpenAI] = settings.FallbackModelOpenAI
	updates[SettingKeyFallbackModelGemini] = settings.FallbackModelGemini
	updates[SettingKeyFallbackModelAntigravity] = settings.FallbackModelAntigravity
	updates[SettingKeyEnableIdentityPatch] = strconv.FormatBool(settings.EnableIdentityPatch)
	updates[SettingKeyIdentityPatchPrompt] = settings.IdentityPatchPrompt
	updates[SettingKeyOpsMonitoringEnabled] = strconv.FormatBool(settings.OpsMonitoringEnabled)
	updates[SettingKeyOpsRealtimeMonitoringEnabled] = strconv.FormatBool(settings.OpsRealtimeMonitoringEnabled)
	updates[SettingKeyOpsQueryModeDefault] = string(ParseOpsQueryMode(settings.OpsQueryModeDefault))
	if settings.OpsMetricsIntervalSeconds > 0 {
		updates[SettingKeyOpsMetricsIntervalSeconds] = strconv.Itoa(settings.OpsMetricsIntervalSeconds)
	}
	updates[SettingKeyMinClaudeCodeVersion] = settings.MinClaudeCodeVersion
	updates[SettingKeyMaxClaudeCodeVersion] = settings.MaxClaudeCodeVersion
	updates[SettingKeyAllowUngroupedKeyScheduling] = strconv.FormatBool(settings.AllowUngroupedKeyScheduling)
	updates[SettingKeyBackendModeEnabled] = strconv.FormatBool(settings.BackendModeEnabled)
	updates[SettingKeyMaintenanceModeEnabled] = strconv.FormatBool(settings.MaintenanceModeEnabled)

	updates[SettingKeyAffiliateEnabled] = strconv.FormatBool(settings.AffiliateEnabled)
	updates[SettingKeyAffiliateTransferEnabled] = strconv.FormatBool(settings.AffiliateTransferEnabled)
	updates[SettingKeyAffiliateRebateOnUsageEnabled] = strconv.FormatBool(settings.AffiliateRebateOnUsageEnabled)
	updates[SettingKeyAffiliateRebateOnTopupEnabled] = strconv.FormatBool(settings.AffiliateRebateOnTopupEnabled)
	updates[SettingKeyAffiliateRebateRate] = strconv.FormatFloat(settings.AffiliateRebateRate, 'f', 4, 64)
	updates[SettingKeyAffiliateRebateFreezeHours] = strconv.Itoa(settings.AffiliateRebateFreezeHours)
	updates[SettingKeyAffiliateRebateDurationDays] = strconv.Itoa(settings.AffiliateRebateDurationDays)
	updates[SettingKeyAffiliateRebatePerInviteeCap] = strconv.FormatFloat(settings.AffiliateRebatePerInviteeCap, 'f', 8, 64)
	updates[SettingKeyAffiliateAffCodeLength] = strconv.Itoa(settings.AffiliateAffCodeLength)
	err = s.settingRepo.SetMultiple(ctx, updates)
	if err == nil {
		versionBoundsSF.Forget("version_bounds")
		versionBoundsCache.Store(&cachedVersionBounds{
			min:       settings.MinClaudeCodeVersion,
			max:       settings.MaxClaudeCodeVersion,
			expiresAt: time.Now().Add(versionBoundsCacheTTL).UnixNano(),
		})
		backendModeSF.Forget("backend_mode_enabled")
		backendModeCache.Store(&cachedBackendMode{value: settings.BackendModeEnabled, expiresAt: time.Now().Add(backendModeCacheTTL).UnixNano()})
		maintenanceModeSF.Forget("maintenance_mode_enabled")
		maintenanceModeCache.Store(&cachedMaintenanceMode{value: settings.MaintenanceModeEnabled, expiresAt: time.Now().Add(maintenanceModeCacheTTL).UnixNano()})
		if s.onUpdate != nil {
			s.onUpdate()
		}
	}
	return err
}
func (s *SettingService) validateDefaultSubscriptionGroups(ctx context.Context, items []DefaultSubscriptionSetting) error {
	if len(items) == 0 {
		return nil
	}
	checked := make(map[int64]struct{}, len(items))
	for _, item := range items {
		if item.GroupID <= 0 {
			continue
		}
		if _, ok := checked[item.GroupID]; ok {
			return ErrDefaultSubGroupDuplicate.WithMetadata(map[string]string{"group_id": strconv.FormatInt(item.GroupID, 10)})
		}
		checked[item.GroupID] = struct{}{}
		if s.defaultSubGroupReader == nil {
			continue
		}
		group, err := s.defaultSubGroupReader.GetByID(ctx, item.GroupID)
		if err != nil {
			if errors.Is(err, ErrGroupNotFound) {
				return ErrDefaultSubGroupInvalid.WithMetadata(map[string]string{"group_id": strconv.FormatInt(item.GroupID, 10)})
			}
			return fmt.Errorf("get default subscription group %d: %w", item.GroupID, err)
		}
		if !group.IsSubscriptionType() {
			return ErrDefaultSubGroupInvalid.WithMetadata(map[string]string{"group_id": strconv.FormatInt(item.GroupID, 10)})
		}
	}
	return nil
}
