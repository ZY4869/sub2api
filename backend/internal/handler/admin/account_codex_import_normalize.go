package admin

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

const (
	codexImportClockSkewSeconds int64  = 120
	codexImportTimeLayout       string = time.RFC3339
)

type codexImportAccount struct {
	Name            string
	AccessToken     string
	RefreshToken    string
	IDToken         string
	Email           string
	AccountID       string
	UserID          string
	PlanType        string
	Organization    string
	Credentials     map[string]any
	Extra           map[string]any
	TokenExpiresAt  *time.Time
	IdentityKeys    []string
	WarningTexts    []string
	IsAgentIdentity bool
}

type codexJWTClaims struct {
	Sub        string                `json:"sub"`
	Email      string                `json:"email"`
	Exp        int64                 `json:"exp"`
	OpenAIAuth *codexJWTOpenAIClaims `json:"https://api.openai.com/auth,omitempty"`
}

type codexJWTOpenAIClaims struct {
	ChatGPTAccountID string                     `json:"chatgpt_account_id"`
	ChatGPTUserID    string                     `json:"chatgpt_user_id"`
	ChatGPTPlanType  string                     `json:"chatgpt_plan_type"`
	UserID           string                     `json:"user_id"`
	POID             string                     `json:"poid"`
	Organizations    []openai.OrganizationClaim `json:"organizations"`
}

func normalizeCodexImportEntry(entry codexImportEntry) (*codexImportAccount, error) {
	now := time.Now().UTC()
	item := &codexImportAccount{
		Credentials: map[string]any{},
		Extra: map[string]any{
			"import_source": "codex_session",
			"imported_at":   now.Format(codexImportTimeLayout),
		},
	}
	if err := extractCodexImportEntryFields(item, entry.Value, now); err != nil {
		return nil, err
	}
	if item.IsAgentIdentity {
		if err := validateCodexImportAgentIdentity(item); err != nil {
			return nil, err
		}
		agentRuntimeID := codexStringValue(item.Credentials["agent_runtime_id"])
		item.IdentityKeys = buildCodexImportIdentityKeys(item.AccountID, item.UserID, item.Email, agentRuntimeID, "")
		item.Name = buildCodexImportAccountName(item, entry.Index)
		return item, nil
	}
	if item.AccessToken == "" {
		return nil, errors.New("缺少 accessToken/access_token")
	}

	item.Credentials["access_token"] = item.AccessToken
	if item.RefreshToken != "" {
		item.Credentials["refresh_token"] = item.RefreshToken
		item.Credentials["client_id"] = openai.ClientID
	}
	if item.IDToken != "" {
		item.Credentials["id_token"] = item.IDToken
		_ = enrichCodexImportAccountFromJWT(item, item.IDToken, false, now)
	}
	if err := enrichCodexImportAccountFromJWT(item, item.AccessToken, true, now); err != nil {
		return nil, err
	}
	if _, ok := item.Credentials["expires_at"]; !ok {
		item.WarningTexts = append(item.WarningTexts, "无法从 accessToken 解析过期时间，导入后需自行确认令牌有效性")
	}
	if item.RefreshToken == "" {
		item.WarningTexts = append(item.WarningTexts, "未包含 refresh_token，accessToken 过期后无法自动续期")
	}

	setCodexCredentialIfNotEmpty(item.Credentials, "email", item.Email)
	setCodexCredentialIfNotEmpty(item.Credentials, "chatgpt_account_id", item.AccountID)
	setCodexCredentialIfNotEmpty(item.Credentials, "chatgpt_user_id", item.UserID)
	setCodexCredentialIfNotEmpty(item.Credentials, "organization_id", item.Organization)
	setCodexCredentialIfNotEmpty(item.Credentials, "plan_type", item.PlanType)
	item.Extra["access_token_sha256"] = codexTokenFingerprint(item.AccessToken)
	item.IdentityKeys = buildCodexImportIdentityKeys(item.AccountID, item.UserID, item.Email, item.AccessToken, item.RefreshToken)
	item.Name = buildCodexImportAccountName(item, entry.Index)
	return item, nil
}

func extractCodexImportEntryFields(item *codexImportAccount, value any, now time.Time) error {
	switch raw := value.(type) {
	case string:
		item.AccessToken = strings.TrimSpace(raw)
		return nil
	case map[string]any:
		extractCodexImportMapFields(item, raw)
		if expiresAt, ok := firstCodexTime(raw, []string{"tokens", "expires_at"}, []string{"tokens", "expiresAt"}, []string{"expires_at"}, []string{"expiresAt"}); ok {
			if expiresAt.Unix() <= now.Unix()-codexImportClockSkewSeconds {
				return fmt.Errorf("access_token 已过期: %s", expiresAt.Format(codexImportTimeLayout))
			}
			item.TokenExpiresAt = &expiresAt
			item.Credentials["expires_at"] = expiresAt.Format(codexImportTimeLayout)
		}
		return nil
	default:
		return errors.New("导入项格式不支持")
	}
}

func extractCodexImportMapFields(item *codexImportAccount, raw map[string]any) {
	if isCodexAgentIdentityImport(raw) {
		extractCodexAgentIdentityFields(item, raw)
		return
	}
	item.AccessToken = firstCodexString(raw, []string{"tokens", "access_token"}, []string{"tokens", "accessToken"}, []string{"access_token"}, []string{"accessToken"}, []string{"token"})
	item.RefreshToken = firstCodexString(raw, []string{"tokens", "refresh_token"}, []string{"tokens", "refreshToken"}, []string{"refresh_token"}, []string{"refreshToken"})
	item.IDToken = firstCodexString(raw, []string{"tokens", "id_token"}, []string{"tokens", "idToken"}, []string{"id_token"}, []string{"idToken"})
	item.Email = firstCodexString(raw, []string{"email"}, []string{"user", "email"})
	item.AccountID = firstCodexString(raw, []string{"chatgpt_account_id"}, []string{"chatgptAccountId"}, []string{"account_id"}, []string{"accountId"}, []string{"account", "id"}, []string{"account", "account_id"}, []string{"account", "chatgpt_account_id"})
	item.UserID = firstCodexString(raw, []string{"chatgpt_user_id"}, []string{"chatgptUserId"}, []string{"user_id"}, []string{"userId"}, []string{"user", "id"})
	item.PlanType = firstCodexString(raw, []string{"plan_type"}, []string{"planType"}, []string{"account", "plan_type"}, []string{"account", "planType"})
	item.Organization = firstCodexString(raw, []string{"organization_id"}, []string{"organizationId"}, []string{"org_id"}, []string{"orgId"})
	item.Name = firstCodexString(raw, []string{"name"}, []string{"user", "name"})
	if authProvider := firstCodexString(raw, []string{"auth_provider"}, []string{"authProvider"}); authProvider != "" {
		item.Extra["auth_provider"] = authProvider
	}
	if sessionToken := firstCodexString(raw, []string{"session_token"}, []string{"sessionToken"}); sessionToken != "" {
		item.Extra["session_token_present"] = true
		item.WarningTexts = append(item.WarningTexts, "sessionToken 已忽略，不会作为 OAuth refresh_token 存储")
	}
	if sessionExpiresAt, ok := firstCodexTime(raw, []string{"expires"}); ok {
		item.Extra["session_expires_at"] = sessionExpiresAt.Format(codexImportTimeLayout)
	}
	copyCodexExtraString(raw, item.Extra, "user_image", []string{"user", "image"})
	copyCodexExtraString(raw, item.Extra, "user_picture", []string{"user", "picture"})
	copyCodexExtraString(raw, item.Extra, "account_structure", []string{"account", "structure"})
	copyCodexExtraString(raw, item.Extra, "account_residency_region", []string{"account", "residencyRegion"})
	copyCodexExtraString(raw, item.Extra, "compute_residency", []string{"account", "computeResidency"})
}

func isCodexAgentIdentityImport(raw map[string]any) bool {
	return strings.EqualFold(firstCodexString(raw, []string{"auth_mode"}, []string{"openai_auth_mode"}), service.OpenAIAuthModeAgentIdentity) ||
		firstCodexString(raw, []string{"agent_runtime_id"}, []string{"agent_identity", "agent_runtime_id"}) != "" ||
		firstCodexString(raw, []string{"agent_private_key"}, []string{"agent_identity", "agent_private_key"}) != ""
}

func extractCodexAgentIdentityFields(item *codexImportAccount, raw map[string]any) {
	item.IsAgentIdentity = true
	item.Credentials["auth_mode"] = service.OpenAIAuthModeAgentIdentity
	setCodexCredentialIfNotEmpty(item.Credentials, "agent_runtime_id", firstCodexString(raw, []string{"agent_runtime_id"}, []string{"agentIdentity", "agentRuntimeId"}, []string{"agent_identity", "agent_runtime_id"}, []string{"agent_identity", "agentRuntimeId"}))
	setCodexCredentialIfNotEmpty(item.Credentials, "agent_private_key", firstCodexString(raw, []string{"agent_private_key"}, []string{"agentIdentity", "agentPrivateKey"}, []string{"agent_identity", "agent_private_key"}, []string{"agent_identity", "agentPrivateKey"}))
	setCodexCredentialIfNotEmpty(item.Credentials, "task_id", firstCodexString(raw, []string{"task_id"}, []string{"taskId"}, []string{"agent_identity", "task_id"}, []string{"agent_identity", "taskId"}))
	item.Email = firstCodexString(raw, []string{"email"}, []string{"user", "email"}, []string{"agent_identity", "email"})
	item.AccountID = firstCodexString(raw, []string{"chatgpt_account_id"}, []string{"chatgptAccountId"}, []string{"account_id"}, []string{"accountId"}, []string{"agent_identity", "chatgpt_account_id"}, []string{"agent_identity", "account_id"})
	item.UserID = firstCodexString(raw, []string{"chatgpt_user_id"}, []string{"chatgptUserId"}, []string{"user_id"}, []string{"userId"}, []string{"agent_identity", "chatgpt_user_id"}, []string{"agent_identity", "user_id"})
	item.PlanType = firstCodexString(raw, []string{"plan_type"}, []string{"planType"}, []string{"agent_identity", "plan_type"}, []string{"agent_identity", "planType"})
	item.Organization = firstCodexString(raw, []string{"organization_id"}, []string{"organizationId"}, []string{"agent_identity", "organization_id"}, []string{"agent_identity", "organizationId"})
	item.Name = firstCodexString(raw, []string{"name"}, []string{"user", "name"}, []string{"agent_identity", "name"})
	setCodexCredentialIfNotEmpty(item.Credentials, "email", item.Email)
	setCodexCredentialIfNotEmpty(item.Credentials, "chatgpt_account_id", item.AccountID)
	setCodexCredentialIfNotEmpty(item.Credentials, "chatgpt_user_id", item.UserID)
	setCodexCredentialIfNotEmpty(item.Credentials, "organization_id", item.Organization)
	setCodexCredentialIfNotEmpty(item.Credentials, "plan_type", item.PlanType)
	if fedramp, ok := firstCodexBool(raw, []string{"chatgpt_account_is_fedramp"}, []string{"agent_identity", "chatgpt_account_is_fedramp"}); ok {
		item.Credentials["chatgpt_account_is_fedramp"] = fedramp
	}
	item.WarningTexts = append(item.WarningTexts, "Agent Identity 已导入；首次调用时会自动注册或恢复 task_id")
}

func validateCodexImportAgentIdentity(item *codexImportAccount) error {
	runtimeID, _ := item.Credentials["agent_runtime_id"].(string)
	privateKey, _ := item.Credentials["agent_private_key"].(string)
	if strings.TrimSpace(runtimeID) == "" {
		return errors.New("缺少 agent_runtime_id")
	}
	if strings.TrimSpace(privateKey) == "" {
		return errors.New("缺少 agent_private_key")
	}
	if err := service.ValidateOpenAIAgentIdentityPrivateKey(privateKey); err != nil {
		return fmt.Errorf("agent_private_key 无效: %w", err)
	}
	return nil
}

func enrichCodexImportAccountFromJWT(item *codexImportAccount, token string, validateExpiry bool, now time.Time) error {
	claims, err := decodeCodexJWTClaims(token)
	if err != nil {
		if validateExpiry {
			item.WarningTexts = append(item.WarningTexts, "accessToken 不是可解析 JWT，无法校验过期时间和账号身份")
		}
		return nil
	}
	if validateExpiry && claims.Exp > 0 {
		if now.Unix() > claims.Exp+codexImportClockSkewSeconds {
			return fmt.Errorf("access_token 已过期: %s", time.Unix(claims.Exp, 0).UTC().Format(codexImportTimeLayout))
		}
		expiresAt := time.Unix(claims.Exp, 0).UTC()
		item.TokenExpiresAt = &expiresAt
		item.Credentials["expires_at"] = expiresAt.Format(codexImportTimeLayout)
	}
	fillCodexImportIdentityFromClaims(item, claims)
	return nil
}

func decodeCodexJWTClaims(token string) (*codexJWTClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid JWT format")
	}
	payload, err := decodeCodexJWTSegment(parts[1])
	if err != nil {
		return nil, err
	}
	var claims codexJWTClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, err
	}
	return &claims, nil
}

func decodeCodexJWTSegment(segment string) ([]byte, error) {
	if decoded, err := base64.RawURLEncoding.DecodeString(segment); err == nil {
		return decoded, nil
	}
	if decoded, err := base64.RawStdEncoding.DecodeString(segment); err == nil {
		return decoded, nil
	}
	padded := segment
	if rem := len(padded) % 4; rem > 0 {
		padded += strings.Repeat("=", 4-rem)
	}
	if decoded, err := base64.URLEncoding.DecodeString(padded); err == nil {
		return decoded, nil
	}
	return base64.StdEncoding.DecodeString(padded)
}

func firstCodexBool(obj map[string]any, paths ...[]string) (bool, bool) {
	for _, path := range paths {
		raw, ok := codexPathValue(obj, path)
		if !ok {
			continue
		}
		switch v := raw.(type) {
		case bool:
			return v, true
		case string:
			switch strings.ToLower(strings.TrimSpace(v)) {
			case "true", "1", "yes":
				return true, true
			case "false", "0", "no":
				return false, true
			}
		}
	}
	return false, false
}
