package service

import (
	"context"
)

// refreshOAuthToken attempts to refresh OAuth token for a synced account
// Returns updated credentials or nil if refresh failed/not applicable
func (s *CRSSyncService) refreshOAuthToken(ctx context.Context, account *Account) map[string]any {
	if account.Type != AccountTypeOAuth {
		return nil
	}

	var newCredentials map[string]any
	var err error

	switch account.Platform {
	case PlatformAnthropic:
		if s.oauthService == nil {
			return nil
		}
		tokenInfo, refreshErr := s.oauthService.RefreshAccountToken(ctx, account)
		if refreshErr != nil {
			err = refreshErr
		} else {
			// Preserve existing credentials
			newCredentials = make(map[string]any)
			for k, v := range account.Credentials {
				newCredentials[k] = v
			}
			// Update token fields
			newCredentials["access_token"] = tokenInfo.AccessToken
			newCredentials["token_type"] = tokenInfo.TokenType
			newCredentials["expires_in"] = tokenInfo.ExpiresIn
			newCredentials["expires_at"] = tokenInfo.ExpiresAt
			if tokenInfo.RefreshToken != "" {
				newCredentials["refresh_token"] = tokenInfo.RefreshToken
			}
			if tokenInfo.Scope != "" {
				newCredentials["scope"] = tokenInfo.Scope
			}
		}
	case PlatformOpenAI:
		if s.openaiOAuthService == nil {
			return nil
		}
		tokenInfo, refreshErr := s.openaiOAuthService.RefreshAccountToken(ctx, account)
		if refreshErr != nil {
			err = refreshErr
		} else {
			newCredentials = s.openaiOAuthService.BuildAccountCredentials(tokenInfo)
			// Preserve non-token settings from existing credentials
			for k, v := range account.Credentials {
				if _, exists := newCredentials[k]; !exists {
					newCredentials[k] = v
				}
			}
		}
	case PlatformGemini:
		if s.geminiOAuthService == nil {
			return nil
		}
		tokenInfo, refreshErr := s.geminiOAuthService.RefreshAccountToken(ctx, account)
		if refreshErr != nil {
			err = refreshErr
		} else {
			newCredentials = s.geminiOAuthService.BuildAccountCredentials(tokenInfo)
			for k, v := range account.Credentials {
				if _, exists := newCredentials[k]; !exists {
					newCredentials[k] = v
				}
			}
		}
	default:
		return nil
	}

	if err != nil {
		// Log but don't fail the sync - token might still be valid or refreshable later
		return nil
	}

	return newCredentials
}
