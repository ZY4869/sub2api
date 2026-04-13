package service

import (
	"context"
	"log/slog"
)

func (s *adminServiceImpl) ensureOAuthPrivacyAsync(account *Account) {
	if s == nil || account == nil || account.Type != AccountTypeOAuth {
		return
	}

	switch account.Platform {
	case PlatformOpenAI:
		go func() {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("create_account_openai_privacy_panic", "account_id", account.ID, "recover", r)
				}
			}()
			s.EnsureOpenAIPrivacy(context.Background(), account)
		}()
	case PlatformAntigravity:
		go func() {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("create_account_antigravity_privacy_panic", "account_id", account.ID, "recover", r)
				}
			}()
			s.EnsureAntigravityPrivacy(context.Background(), account)
		}()
	}
}
