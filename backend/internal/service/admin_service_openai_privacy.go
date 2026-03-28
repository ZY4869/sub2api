package service

import "context"

func shouldSkipOpenAIPrivacyEnsure(extra map[string]any) bool {
	if len(extra) == 0 {
		return false
	}
	_, exists := extra["privacy_mode"]
	return exists
}

func applyOpenAIPrivacyMode(account *Account, mode string) {
	if account == nil || mode == "" {
		return
	}
	if account.Extra == nil {
		account.Extra = make(map[string]any)
	}
	account.Extra["privacy_mode"] = mode
}

func (s *adminServiceImpl) EnsureOpenAIPrivacy(ctx context.Context, account *Account) string {
	if account == nil || account.Platform != PlatformOpenAI || account.Type != AccountTypeOAuth {
		return ""
	}
	if shouldSkipOpenAIPrivacyEnsure(account.Extra) {
		return ""
	}
	return s.forceOpenAIPrivacy(ctx, account)
}

func (s *adminServiceImpl) ForceOpenAIPrivacy(ctx context.Context, account *Account) string {
	if account == nil || account.Platform != PlatformOpenAI || account.Type != AccountTypeOAuth {
		return ""
	}
	return s.forceOpenAIPrivacy(ctx, account)
}

func (s *adminServiceImpl) forceOpenAIPrivacy(ctx context.Context, account *Account) string {
	if s == nil || account == nil {
		return ""
	}
	token := account.GetCredential("access_token")
	if token == "" {
		return ""
	}

	var proxyURL string
	if account.ProxyID != nil && s.proxyRepo != nil {
		if proxy, err := s.proxyRepo.GetByID(ctx, *account.ProxyID); err == nil && proxy != nil {
			proxyURL = proxy.URL()
		}
	}

	mode := disableOpenAITraining(ctx, s.privacyClientFactory, token, proxyURL)
	if mode == "" {
		return ""
	}
	if s.accountRepo != nil {
		if err := s.accountRepo.UpdateExtra(ctx, account.ID, map[string]any{"privacy_mode": mode}); err == nil {
			applyOpenAIPrivacyMode(account, mode)
			return mode
		}
	}
	applyOpenAIPrivacyMode(account, mode)
	return mode
}
