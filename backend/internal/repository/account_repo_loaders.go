package repository

import (
	"context"
	entsql "entgo.io/ent/dialect/sql"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	dbaccount "github.com/Wei-Shaw/sub2api/ent/account"
	dbaccountgroup "github.com/Wei-Shaw/sub2api/ent/accountgroup"
	dbpredicate "github.com/Wei-Shaw/sub2api/ent/predicate"
	dbproxy "github.com/Wei-Shaw/sub2api/ent/proxy"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"strconv"
	"time"
)

func (r *accountRepository) accountsToService(ctx context.Context, accounts []*dbent.Account) ([]service.Account, error) {
	if len(accounts) == 0 {
		return []service.Account{}, nil
	}
	accountIDs := make([]int64, 0, len(accounts))
	proxyIDs := make([]int64, 0, len(accounts))
	for _, acc := range accounts {
		accountIDs = append(accountIDs, acc.ID)
		if acc.ProxyID != nil {
			proxyIDs = append(proxyIDs, *acc.ProxyID)
		}
	}
	proxyMap, err := r.loadProxies(ctx, proxyIDs)
	if err != nil {
		return nil, err
	}
	groupsByAccount, groupIDsByAccount, accountGroupsByAccount, err := r.loadAccountGroups(ctx, accountIDs)
	if err != nil {
		return nil, err
	}
	outAccounts := make([]service.Account, 0, len(accounts))
	for _, acc := range accounts {
		out := accountEntityToService(acc)
		if out == nil {
			continue
		}
		if acc.ProxyID != nil {
			if proxy, ok := proxyMap[*acc.ProxyID]; ok {
				out.Proxy = proxy
			}
		}
		if groups, ok := groupsByAccount[acc.ID]; ok {
			out.Groups = groups
		}
		if groupIDs, ok := groupIDsByAccount[acc.ID]; ok {
			out.GroupIDs = groupIDs
		}
		if ags, ok := accountGroupsByAccount[acc.ID]; ok {
			out.AccountGroups = ags
		}
		outAccounts = append(outAccounts, *out)
	}
	return outAccounts, nil
}
func tempUnschedulablePredicate() dbpredicate.Account {
	return dbpredicate.Account(func(s *entsql.Selector) {
		col := s.C("temp_unschedulable_until")
		s.Where(entsql.Or(entsql.IsNull(col), entsql.LTE(col, entsql.Expr("NOW()"))))
	})
}
func notExpiredPredicate(now time.Time) dbpredicate.Account {
	return dbaccount.Or(dbaccount.ExpiresAtIsNil(), dbaccount.ExpiresAtGT(now), dbaccount.AutoPauseOnExpiredEQ(false))
}
func lifecyclePredicate(lifecycle string) dbpredicate.Account {
	normalized := service.NormalizeAccountLifecycleInput(lifecycle)
	if normalized == service.AccountLifecycleAll {
		return nil
	}
	return dbaccount.LifecycleStateEQ(normalized)
}
func (r *accountRepository) loadProxies(ctx context.Context, proxyIDs []int64) (map[int64]*service.Proxy, error) {
	proxyMap := make(map[int64]*service.Proxy)
	if len(proxyIDs) == 0 {
		return proxyMap, nil
	}
	proxies, err := r.client.Proxy.Query().Where(dbproxy.IDIn(proxyIDs...)).All(ctx)
	if err != nil {
		return nil, err
	}
	for _, p := range proxies {
		proxyMap[p.ID] = proxyEntityToService(p)
	}
	return proxyMap, nil
}
func (r *accountRepository) loadAccountGroups(ctx context.Context, accountIDs []int64) (map[int64][]*service.Group, map[int64][]int64, map[int64][]service.AccountGroup, error) {
	groupsByAccount := make(map[int64][]*service.Group)
	groupIDsByAccount := make(map[int64][]int64)
	accountGroupsByAccount := make(map[int64][]service.AccountGroup)
	if len(accountIDs) == 0 {
		return groupsByAccount, groupIDsByAccount, accountGroupsByAccount, nil
	}
	entries, err := r.client.AccountGroup.Query().Where(dbaccountgroup.AccountIDIn(accountIDs...)).WithGroup().Order(dbaccountgroup.ByAccountID(), dbaccountgroup.ByPriority()).All(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	for _, ag := range entries {
		groupSvc := groupEntityToService(ag.Edges.Group)
		agSvc := service.AccountGroup{AccountID: ag.AccountID, GroupID: ag.GroupID, Priority: ag.Priority, CreatedAt: ag.CreatedAt, Group: groupSvc}
		accountGroupsByAccount[ag.AccountID] = append(accountGroupsByAccount[ag.AccountID], agSvc)
		groupIDsByAccount[ag.AccountID] = append(groupIDsByAccount[ag.AccountID], ag.GroupID)
		if groupSvc != nil {
			groupsByAccount[ag.AccountID] = append(groupsByAccount[ag.AccountID], groupSvc)
		}
	}
	return groupsByAccount, groupIDsByAccount, accountGroupsByAccount, nil
}
func (r *accountRepository) loadAccountGroupIDs(ctx context.Context, accountID int64) ([]int64, error) {
	entries, err := r.client.AccountGroup.Query().Where(dbaccountgroup.AccountIDEQ(accountID)).All(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(entries))
	for _, entry := range entries {
		ids = append(ids, entry.GroupID)
	}
	return ids, nil
}
func mergeGroupIDs(a []int64, b []int64) []int64 {
	seen := make(map[int64]struct{}, len(a)+len(b))
	out := make([]int64, 0, len(a)+len(b))
	for _, id := range a {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	for _, id := range b {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
func buildSchedulerGroupPayload(groupIDs []int64) map[string]any {
	if len(groupIDs) == 0 {
		return nil
	}
	return map[string]any{"group_ids": groupIDs}
}
func accountEntityToService(m *dbent.Account) *service.Account {
	if m == nil {
		return nil
	}
	rateMultiplier := m.RateMultiplier
	return &service.Account{
		ID:                     m.ID,
		Name:                   m.Name,
		Notes:                  m.Notes,
		Platform:               m.Platform,
		Type:                   m.Type,
		Credentials:            copyJSONMap(m.Credentials),
		Extra:                  copyJSONMap(m.Extra),
		ProxyID:                m.ProxyID,
		Concurrency:            m.Concurrency,
		Priority:               m.Priority,
		RateMultiplier:         &rateMultiplier,
		LoadFactor:             m.LoadFactor,
		Status:                 m.Status,
		LifecycleState:         m.LifecycleState,
		LifecycleReasonCode:    derefString(m.LifecycleReasonCode),
		LifecycleReasonMessage: derefString(m.LifecycleReasonMessage),
		ErrorMessage:           derefString(m.ErrorMessage),
		LastUsedAt:             m.LastUsedAt,
		ExpiresAt:              m.ExpiresAt,
		AutoPauseOnExpired:     m.AutoPauseOnExpired,
		CreatedAt:              m.CreatedAt,
		UpdatedAt:              m.UpdatedAt,
		BlacklistedAt:          m.BlacklistedAt,
		BlacklistPurgeAt:       m.BlacklistPurgeAt,
		Schedulable:            m.Schedulable,
		RateLimitedAt:          m.RateLimitedAt,
		RateLimitResetAt:       m.RateLimitResetAt,
		OverloadUntil:          m.OverloadUntil,
		TempUnschedulableUntil: m.TempUnschedulableUntil,
		TempUnschedulableReason: derefString(m.TempUnschedulableReason),
		SessionWindowStart:     m.SessionWindowStart,
		SessionWindowEnd:       m.SessionWindowEnd,
		SessionWindowStatus:    derefString(m.SessionWindowStatus),
	}
}
func normalizeJSONMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	return in
}
func copyJSONMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
func joinClauses(clauses []string, sep string) string {
	if len(clauses) == 0 {
		return ""
	}
	out := clauses[0]
	for i := 1; i < len(clauses); i++ {
		out += sep + clauses[i]
	}
	return out
}
func itoa(v int) string {
	return strconv.Itoa(v)
}
