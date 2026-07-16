package admin

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func fillCodexImportIdentityFromClaims(item *codexImportAccount, claims *codexJWTClaims) {
	if item.Email == "" {
		item.Email = strings.TrimSpace(claims.Email)
	}
	if claims.OpenAIAuth == nil {
		if item.UserID == "" {
			item.UserID = strings.TrimSpace(claims.Sub)
		}
		return
	}
	if item.AccountID == "" {
		item.AccountID = strings.TrimSpace(claims.OpenAIAuth.ChatGPTAccountID)
	}
	if item.UserID == "" {
		item.UserID = strings.TrimSpace(claims.OpenAIAuth.ChatGPTUserID)
	}
	if item.UserID == "" {
		item.UserID = strings.TrimSpace(claims.OpenAIAuth.UserID)
	}
	if item.PlanType == "" {
		item.PlanType = strings.TrimSpace(claims.OpenAIAuth.ChatGPTPlanType)
	}
	if item.Organization == "" {
		item.Organization = strings.TrimSpace(claims.OpenAIAuth.POID)
	}
	if item.Organization == "" {
		for _, org := range claims.OpenAIAuth.Organizations {
			if org.IsDefault {
				item.Organization = org.ID
				break
			}
		}
	}
	if item.Organization == "" && len(claims.OpenAIAuth.Organizations) > 0 {
		item.Organization = claims.OpenAIAuth.Organizations[0].ID
	}
	if item.UserID == "" {
		item.UserID = strings.TrimSpace(claims.Sub)
	}
}

func buildCodexImportAccountName(item *codexImportAccount, index int) string {
	for _, candidate := range []string{item.Name, item.Email, item.AccountID, item.UserID} {
		if text := strings.TrimSpace(candidate); text != "" {
			return text
		}
	}
	return fmt.Sprintf("Codex 导入账号 %d", index)
}

func buildCodexCreateAccountName(base string, item *codexImportAccount, index, total int) string {
	base = strings.TrimSpace(base)
	if base == "" {
		if item == nil {
			return fmt.Sprintf("Codex 导入账号 %d", index)
		}
		return item.Name
	}
	if total > 1 {
		return fmt.Sprintf("%s #%d", base, index)
	}
	return base
}

func resolveCodexImportExpiry(req CodexSessionImportRequest, item *codexImportAccount) (*int64, *time.Time, *bool, []string, error) {
	if item == nil {
		return nil, nil, nil, nil, errors.New("导入项为空")
	}
	if item.IsAgentIdentity {
		return nil, nil, req.AutoPauseOnExpired, nil, nil
	}
	var requestExpiresAt *time.Time
	if req.ExpiresAt != nil && *req.ExpiresAt > 0 {
		t := time.Unix(*req.ExpiresAt, 0).UTC()
		requestExpiresAt = &t
	}
	if item.RefreshToken != "" {
		return resolveRefreshableCodexExpiry(req, item, requestExpiresAt)
	}
	return resolveAccessOnlyCodexExpiry(req, item, requestExpiresAt)
}

func resolveAccessOnlyCodexExpiry(req CodexSessionImportRequest, item *codexImportAccount, requestExpiresAt *time.Time) (*int64, *time.Time, *bool, []string, error) {
	accountExpiresAt := item.TokenExpiresAt
	credentialExpiresAt := item.TokenExpiresAt
	if requestExpiresAt != nil {
		accountExpiresAt = earlierCodexTime(accountExpiresAt, requestExpiresAt)
		credentialExpiresAt = earlierCodexTime(credentialExpiresAt, requestExpiresAt)
	}
	if accountExpiresAt == nil {
		return nil, nil, nil, nil, errors.New("未包含 refresh_token，且无法解析 accessToken 过期时间；请在第一步设置过期时间后再导入")
	}
	if accountExpiresAt.Unix() <= time.Now().UTC().Unix()-codexImportClockSkewSeconds {
		return nil, nil, nil, nil, fmt.Errorf("过期时间已过期: %s", accountExpiresAt.Format(codexImportTimeLayout))
	}
	warnings := []string{"未包含 refresh_token，已按 accessToken/账号过期时间设置自动停止调度"}
	if req.AutoPauseOnExpired != nil && !*req.AutoPauseOnExpired {
		warnings = append(warnings, "未包含 refresh_token，已强制开启过期自动暂停")
	}
	autoPause := true
	expiresAtUnix := accountExpiresAt.Unix()
	return &expiresAtUnix, credentialExpiresAt, &autoPause, warnings, nil
}

func resolveRefreshableCodexExpiry(req CodexSessionImportRequest, item *codexImportAccount, requestExpiresAt *time.Time) (*int64, *time.Time, *bool, []string, error) {
	var accountExpiresAt *time.Time
	var credentialExpiresAt *time.Time
	if requestExpiresAt != nil {
		accountExpiresAt = requestExpiresAt
	}
	if item.TokenExpiresAt != nil {
		tokenExpiresAt := item.TokenExpiresAt.UTC()
		credentialExpiresAt = &tokenExpiresAt
	}
	var expiresAtUnix *int64
	if accountExpiresAt != nil {
		v := accountExpiresAt.Unix()
		expiresAtUnix = &v
	}
	return expiresAtUnix, credentialExpiresAt, req.AutoPauseOnExpired, nil, nil
}

func earlierCodexTime(current, candidate *time.Time) *time.Time {
	if candidate == nil {
		return current
	}
	if current == nil || candidate.Before(*current) {
		t := candidate.UTC()
		return &t
	}
	t := current.UTC()
	return &t
}

func copyCodexExtraString(obj map[string]any, extra map[string]any, key string, path []string) {
	if value := firstCodexString(obj, path); value != "" {
		extra[key] = value
	}
}

func setCodexCredentialIfNotEmpty(credentials map[string]any, key, value string) {
	if value = strings.TrimSpace(value); value != "" {
		credentials[key] = value
	}
}

func codexTokenFingerprint(token string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(sum[:])
}

func parseCodexTimeString(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false
	}
	if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return parsed.UTC(), true
	}
	if n, err := strconv.ParseInt(value, 10, 64); err == nil {
		return codexUnixTime(n), true
	}
	return time.Time{}, false
}

func codexUnixTime(value int64) time.Time {
	if value > 1_000_000_000_000 {
		return time.UnixMilli(value).UTC()
	}
	return time.Unix(value, 0).UTC()
}
