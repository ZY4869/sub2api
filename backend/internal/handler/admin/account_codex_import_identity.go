package admin

import (
	"context"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type codexAccountIndex struct {
	accountsByKey map[string][]service.Account
}

type codexSeenIdentity struct {
	index  int
	userID string
	scope  string
}

func buildCodexImportAgentIdentityKeysWithScope(accountID, userID, email, agentRuntimeID, organization, teamID, team string) []string {
	agentRuntimeID = strings.TrimSpace(agentRuntimeID)
	keys := buildCodexStoredIdentityKeys(accountID, userID, email, "")
	if agentRuntimeID != "" {
		keys = append([]string{"agent_runtime:" + agentRuntimeID}, keys...)
	}
	keys = appendScopedCodexIdentityKeys(keys, organization, teamID, team)
	return appendUniqueCodexIdentityKeys(nil, keys...)
}

func buildCodexImportIdentityKeysWithScope(accountID, userID, email, accessToken, refreshToken, organization, teamID, team string) []string {
	accessToken = strings.TrimSpace(accessToken)
	if strings.TrimSpace(refreshToken) == "" && accessToken != "" {
		return appendScopedCodexIdentityKeys([]string{"access:" + codexTokenFingerprint(accessToken)}, organization, teamID, team)
	}
	return buildCodexStoredIdentityKeysWithScope(accountID, userID, email, accessToken, organization, teamID, team)
}

func buildCodexStoredIdentityKeys(accountID, userID, email, accessToken string) []string {
	return buildCodexStoredIdentityKeysWithScope(accountID, userID, email, accessToken, "", "", "")
}

func buildCodexStoredIdentityKeysWithScope(accountID, userID, email, accessToken, organization, teamID, team string) []string {
	keys := make([]string, 0, 4)
	accountID = strings.TrimSpace(accountID)
	userID = strings.TrimSpace(userID)
	accessToken = strings.TrimSpace(accessToken)
	if accountID != "" {
		keys = append(keys, "account:"+accountID)
	}
	if userID != "" {
		keys = append(keys, "user:"+userID)
	}
	if accountID == "" && userID == "" {
		if email = strings.ToLower(strings.TrimSpace(email)); email != "" {
			keys = append(keys, "email:"+email)
		}
	}
	if accessToken != "" {
		keys = append(keys, "access:"+codexTokenFingerprint(accessToken))
	}
	return appendScopedCodexIdentityKeys(keys, organization, teamID, team)
}

func appendScopedCodexIdentityKeys(base []string, organization, teamID, team string) []string {
	scope := buildCodexIdentityScope(organization, teamID, team)
	if scope == "" || len(base) == 0 {
		return base
	}
	out := make([]string, 0, len(base)*2)
	for _, key := range base {
		out = append(out, key+"|"+scope)
	}
	out = append(out, base...)
	return out
}

func buildCodexIdentityScope(organization, teamID, team string) string {
	parts := make([]string, 0, 3)
	if organization = normalizeCodexIdentityScopeValue(organization); organization != "" {
		parts = append(parts, "org:"+organization)
	}
	if teamID = normalizeCodexIdentityScopeValue(teamID); teamID != "" {
		parts = append(parts, "team_id:"+teamID)
	}
	if team = normalizeCodexIdentityScopeValue(team); team != "" {
		parts = append(parts, "team:"+team)
	}
	return strings.Join(parts, "|")
}

func normalizeCodexIdentityScopeValue(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func buildCodexAccountIndex(accounts []service.Account) *codexAccountIndex {
	index := &codexAccountIndex{accountsByKey: map[string][]service.Account{}}
	for _, account := range accounts {
		index.Add(account)
	}
	return index
}

func (i *codexAccountIndex) Add(account service.Account) {
	if i == nil {
		return
	}
	if i.accountsByKey == nil {
		i.accountsByKey = map[string][]service.Account{}
	}
	i.remove(account.ID)
	accountID := codexCredentialString(account.Credentials, "chatgpt_account_id")
	userID := codexCredentialString(account.Credentials, "chatgpt_user_id")
	email := codexCredentialString(account.Credentials, "email")
	accessToken := codexCredentialString(account.Credentials, "access_token")
	organization := firstNonEmptyCodexIdentityScopeValue(
		codexCredentialString(account.Credentials, "organization"),
		codexCredentialString(account.Credentials, "organization_id"),
	)
	teamID := codexCredentialString(account.Credentials, "team_id")
	team := codexCredentialString(account.Credentials, "team")
	agentRuntimeID := strings.TrimSpace(codexCredentialString(account.Credentials, "agent_runtime_id"))

	keys := buildCodexStoredIdentityKeys(
		accountID,
		userID,
		email,
		accessToken,
	)
	if agentRuntimeID != "" {
		keys = append(keys, "agent_runtime:"+agentRuntimeID)
	}
	scopedKeys := buildCodexStoredIdentityKeysWithScope(
		accountID,
		userID,
		email,
		accessToken,
		organization,
		teamID,
		team,
	)
	if agentRuntimeID != "" {
		scopedKeys = appendUniqueCodexIdentityKeys(scopedKeys, appendScopedCodexIdentityKeys([]string{"agent_runtime:" + agentRuntimeID}, organization, teamID, team)...)
	}
	for _, key := range appendUniqueCodexIdentityKeys(keys, scopedKeys...) {
		i.accountsByKey[key] = upsertCodexAccount(i.accountsByKey[key], account)
	}
}

func appendUniqueCodexIdentityKeys(base []string, values ...string) []string {
	out := append([]string(nil), base...)
	seen := make(map[string]struct{}, len(out)+len(values))
	for _, key := range out {
		seen[key] = struct{}{}
	}
	for _, key := range values {
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	return out
}

func firstNonEmptyCodexIdentityScopeValue(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func (i *codexAccountIndex) remove(accountID int64) {
	for key, accounts := range i.accountsByKey {
		kept := accounts[:0]
		for _, account := range accounts {
			if account.ID != accountID {
				kept = append(kept, account)
			}
		}
		if len(kept) == 0 {
			delete(i.accountsByKey, key)
			continue
		}
		i.accountsByKey[key] = kept
	}
}

func upsertCodexAccount(accounts []service.Account, account service.Account) []service.Account {
	for idx := range accounts {
		if accounts[idx].ID == account.ID {
			accounts[idx] = account
			return accounts
		}
	}
	return append(accounts, account)
}

func (i *codexAccountIndex) Find(keys []string, userID string) (*service.Account, string) {
	if i == nil {
		return nil, ""
	}
	for _, key := range keys {
		for _, account := range i.accountsByKey[key] {
			storedUserID := codexCredentialString(account.Credentials, "chatgpt_user_id")
			if codexIdentityConflicts(key, userID, storedUserID) || codexIdentityScopeConflicts(key, account.Credentials) {
				continue
			}
			return &account, key
		}
	}
	return nil, ""
}

func codexIdentityConflicts(key, userID, storedUserID string) bool {
	baseKey := strings.Split(key, "|")[0]
	if !strings.HasPrefix(baseKey, "account:") {
		return false
	}
	userID = strings.TrimSpace(userID)
	storedUserID = strings.TrimSpace(storedUserID)
	return userID != "" && storedUserID != "" && userID != storedUserID
}

func codexIdentityScopeConflicts(key string, credentials map[string]any) bool {
	keyScope := codexIdentityScopeFromKey(key)
	if keyScope == "" {
		return false
	}
	storedScope := buildCodexIdentityScope(
		firstNonEmptyCodexIdentityScopeValue(
			codexCredentialString(credentials, "organization"),
			codexCredentialString(credentials, "organization_id"),
		),
		codexCredentialString(credentials, "team_id"),
		codexCredentialString(credentials, "team"),
	)
	return storedScope != "" && storedScope != keyScope
}

func codexIdentityScopeFromKey(key string) string {
	parts := strings.Split(key, "|")
	if len(parts) <= 1 {
		return ""
	}
	return strings.Join(parts[1:], "|")
}

func firstSeenCodexIdentity(seen map[string]codexSeenIdentity, keys []string, userID string) (int, bool) {
	for _, key := range keys {
		entry, ok := seen[key]
		if !ok {
			continue
		}
		if codexIdentityConflicts(key, userID, entry.userID) || codexSeenIdentityScopeConflicts(key, entry.scope) {
			continue
		}
		return entry.index, true
	}
	return 0, false
}

func markCodexIdentitySeen(seen map[string]codexSeenIdentity, keys []string, index int, userID string) {
	for _, key := range keys {
		seen[key] = codexSeenIdentity{index: index, userID: userID, scope: codexIdentityScopeFromKey(key)}
	}
}

func codexSeenIdentityScopeConflicts(key, storedScope string) bool {
	keyScope := codexIdentityScopeFromKey(key)
	if keyScope == "" || storedScope == "" {
		return false
	}
	return keyScope != storedScope
}

func codexCredentialString(credentials map[string]any, key string) string {
	if credentials == nil {
		return ""
	}
	return codexStringValue(credentials[key])
}

func (h *AccountHandler) invalidateCodexImportTokenCache(ctx context.Context, account *service.Account) {
	if h.tokenCacheInvalidator == nil || account == nil {
		return
	}
	_ = h.tokenCacheInvalidator.InvalidateToken(ctx, account)
}
