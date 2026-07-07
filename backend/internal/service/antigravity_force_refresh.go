package service

const (
	antigravityForceTokenRefreshExtraKey       = "antigravity_force_token_refresh"
	antigravityForceTokenRefreshReasonExtraKey = "antigravity_force_token_refresh_reason"
)

func antigravityForceTokenRefreshExtra(reason string) map[string]any {
	return map[string]any{
		antigravityForceTokenRefreshExtraKey:       true,
		antigravityForceTokenRefreshReasonExtraKey: reason,
	}
}

func clearAntigravityForceTokenRefreshExtra() map[string]any {
	return map[string]any{
		antigravityForceTokenRefreshExtraKey:       false,
		antigravityForceTokenRefreshReasonExtraKey: "",
	}
}

func accountNeedsAntigravityForceTokenRefresh(account *Account) bool {
	return account != nil &&
		account.Platform == PlatformAntigravity &&
		account.Type == AccountTypeOAuth &&
		account.GetExtraBool(antigravityForceTokenRefreshExtraKey)
}
