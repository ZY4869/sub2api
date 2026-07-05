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
}

func buildCodexImportIdentityKeys(accountID, userID, email, accessToken, refreshToken string) []string {
	accessToken = strings.TrimSpace(accessToken)
	if strings.TrimSpace(refreshToken) == "" && accessToken != "" {
		return []string{"access:" + codexTokenFingerprint(accessToken)}
	}
	return buildCodexStoredIdentityKeys(accountID, userID, email, accessToken)
}

func buildCodexStoredIdentityKeys(accountID, userID, email, accessToken string) []string {
	keys := make([]string, 0, 4)
	accountID = strings.TrimSpace(accountID)
	userID = strings.TrimSpace(userID)
	accessToken = strings.TrimSpace(accessToken)
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
	if accountID != "" {
		keys = append(keys, "account:"+accountID)
	}
	return keys
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
	keys := buildCodexStoredIdentityKeys(
		codexCredentialString(account.Credentials, "chatgpt_account_id"),
		codexCredentialString(account.Credentials, "chatgpt_user_id"),
		codexCredentialString(account.Credentials, "email"),
		codexCredentialString(account.Credentials, "access_token"),
	)
	for _, key := range keys {
		i.accountsByKey[key] = upsertCodexAccount(i.accountsByKey[key], account)
	}
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
			if codexIdentityConflicts(key, userID, storedUserID) {
				continue
			}
			return &account, key
		}
	}
	return nil, ""
}

func codexIdentityConflicts(key, userID, storedUserID string) bool {
	if !strings.HasPrefix(key, "account:") {
		return false
	}
	userID = strings.TrimSpace(userID)
	storedUserID = strings.TrimSpace(storedUserID)
	return userID != "" && storedUserID != "" && userID != storedUserID
}

func firstSeenCodexIdentity(seen map[string]codexSeenIdentity, keys []string, userID string) (int, bool) {
	for _, key := range keys {
		entry, ok := seen[key]
		if !ok {
			continue
		}
		if codexIdentityConflicts(key, userID, entry.userID) {
			continue
		}
		return entry.index, true
	}
	return 0, false
}

func markCodexIdentitySeen(seen map[string]codexSeenIdentity, keys []string, index int, userID string) {
	for _, key := range keys {
		seen[key] = codexSeenIdentity{index: index, userID: userID}
	}
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
