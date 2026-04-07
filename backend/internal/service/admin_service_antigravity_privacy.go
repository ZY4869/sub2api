package service

import "context"

func (s *adminServiceImpl) EnsureAntigravityPrivacy(ctx context.Context, account *Account) string {
	if account == nil || account.Platform != PlatformAntigravity || account.Type != AccountTypeOAuth {
		return ""
	}
	if account.Extra != nil {
		if existing, ok := account.Extra["privacy_mode"].(string); ok && existing == AntigravityPrivacySet {
			return existing
		}
	}
	return s.forceAntigravityPrivacy(ctx, account)
}

func (s *adminServiceImpl) forceAntigravityPrivacy(ctx context.Context, account *Account) string {
	if s == nil || account == nil {
		return ""
	}

	accessToken := account.GetCredential("access_token")
	projectID := account.GetCredential("project_id")
	if accessToken == "" {
		return ""
	}

	var proxyURL string
	if account.ProxyID != nil && s.proxyRepo != nil {
		if proxy, err := s.proxyRepo.GetByID(ctx, *account.ProxyID); err == nil && proxy != nil {
			proxyURL = proxy.URL()
		}
	}

	mode := setAntigravityPrivacy(ctx, accessToken, projectID, proxyURL)
	if mode == "" {
		return ""
	}
	if s.accountRepo != nil {
		if err := s.accountRepo.UpdateExtra(ctx, account.ID, map[string]any{"privacy_mode": mode}); err == nil {
			applyAntigravityPrivacyMode(account, mode)
			return mode
		}
	}
	applyAntigravityPrivacyMode(account, mode)
	return mode
}
