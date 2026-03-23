package repository

import (
	"context"
	entsql "entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqljson"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	dbaccount "github.com/Wei-Shaw/sub2api/ent/account"
	dbaccountgroup "github.com/Wei-Shaw/sub2api/ent/accountgroup"
	dbpredicate "github.com/Wei-Shaw/sub2api/ent/predicate"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
	"strconv"
	"time"
)

func (r *accountRepository) Create(ctx context.Context, account *service.Account) error {
	if account == nil {
		return service.ErrAccountNilInput
	}
	builder := r.client.Account.Create().SetName(account.Name).SetNillableNotes(account.Notes).SetPlatform(account.Platform).SetType(account.Type).SetCredentials(normalizeJSONMap(account.Credentials)).SetExtra(normalizeJSONMap(account.Extra)).SetConcurrency(account.Concurrency).SetPriority(account.Priority).SetStatus(account.Status).SetErrorMessage(account.ErrorMessage).SetSchedulable(account.Schedulable).SetAutoPauseOnExpired(account.AutoPauseOnExpired)
	lifecycleState := service.NormalizeAccountLifecycleInput(account.LifecycleState)
	if lifecycleState == service.AccountLifecycleAll {
		lifecycleState = service.AccountLifecycleNormal
	}
	builder.SetLifecycleState(lifecycleState)
	if trimmed := account.LifecycleReasonCode; trimmed != "" {
		builder.SetLifecycleReasonCode(trimmed)
	}
	if trimmed := account.LifecycleReasonMessage; trimmed != "" {
		builder.SetLifecycleReasonMessage(trimmed)
	}
	if account.BlacklistedAt != nil {
		builder.SetBlacklistedAt(*account.BlacklistedAt)
	}
	if account.BlacklistPurgeAt != nil {
		builder.SetBlacklistPurgeAt(*account.BlacklistPurgeAt)
	}
	if account.RateMultiplier != nil {
		builder.SetRateMultiplier(*account.RateMultiplier)
	}
	if account.LoadFactor != nil {
		builder.SetLoadFactor(*account.LoadFactor)
	}
	if account.ProxyID != nil {
		builder.SetProxyID(*account.ProxyID)
	}
	if account.LastUsedAt != nil {
		builder.SetLastUsedAt(*account.LastUsedAt)
	}
	if account.ExpiresAt != nil {
		builder.SetExpiresAt(*account.ExpiresAt)
	}
	if account.RateLimitedAt != nil {
		builder.SetRateLimitedAt(*account.RateLimitedAt)
	}
	if account.RateLimitResetAt != nil {
		builder.SetRateLimitResetAt(*account.RateLimitResetAt)
	}
	if account.OverloadUntil != nil {
		builder.SetOverloadUntil(*account.OverloadUntil)
	}
	if account.SessionWindowStart != nil {
		builder.SetSessionWindowStart(*account.SessionWindowStart)
	}
	if account.SessionWindowEnd != nil {
		builder.SetSessionWindowEnd(*account.SessionWindowEnd)
	}
	if account.SessionWindowStatus != "" {
		builder.SetSessionWindowStatus(account.SessionWindowStatus)
	}
	created, err := builder.Save(ctx)
	if err != nil {
		return translatePersistenceError(err, service.ErrAccountNotFound, nil)
	}
	account.ID = created.ID
	account.CreatedAt = created.CreatedAt
	account.UpdatedAt = created.UpdatedAt
	if err := enqueueSchedulerOutbox(ctx, r.sql, service.SchedulerOutboxEventAccountChanged, &account.ID, nil, buildSchedulerGroupPayload(account.GroupIDs)); err != nil {
		logger.LegacyPrintf("repository.account", "[SchedulerOutbox] enqueue account create failed: account=%d err=%v", account.ID, err)
	}
	return nil
}
func (r *accountRepository) GetByID(ctx context.Context, id int64) (*service.Account, error) {
	m, err := r.client.Account.Query().Where(dbaccount.IDEQ(id)).Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrAccountNotFound, nil)
	}
	accounts, err := r.accountsToService(ctx, []*dbent.Account{m})
	if err != nil {
		return nil, err
	}
	if len(accounts) == 0 {
		return nil, service.ErrAccountNotFound
	}
	return &accounts[0], nil
}
func (r *accountRepository) GetByIDs(ctx context.Context, ids []int64) ([]*service.Account, error) {
	if len(ids) == 0 {
		return []*service.Account{}, nil
	}
	uniqueIDs := make([]int64, 0, len(ids))
	seen := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		uniqueIDs = append(uniqueIDs, id)
	}
	if len(uniqueIDs) == 0 {
		return []*service.Account{}, nil
	}
	entAccounts, err := r.client.Account.Query().Where(dbaccount.IDIn(uniqueIDs...)).WithProxy().All(ctx)
	if err != nil {
		return nil, err
	}
	if len(entAccounts) == 0 {
		return []*service.Account{}, nil
	}
	accountIDs := make([]int64, 0, len(entAccounts))
	entByID := make(map[int64]*dbent.Account, len(entAccounts))
	for _, acc := range entAccounts {
		entByID[acc.ID] = acc
		accountIDs = append(accountIDs, acc.ID)
	}
	groupsByAccount, groupIDsByAccount, accountGroupsByAccount, err := r.loadAccountGroups(ctx, accountIDs)
	if err != nil {
		return nil, err
	}
	outByID := make(map[int64]*service.Account, len(entAccounts))
	for _, entAcc := range entAccounts {
		out := accountEntityToService(entAcc)
		if out == nil {
			continue
		}
		if entAcc.Edges.Proxy != nil {
			out.Proxy = proxyEntityToService(entAcc.Edges.Proxy)
		}
		if groups, ok := groupsByAccount[entAcc.ID]; ok {
			out.Groups = groups
		}
		if groupIDs, ok := groupIDsByAccount[entAcc.ID]; ok {
			out.GroupIDs = groupIDs
		}
		if ags, ok := accountGroupsByAccount[entAcc.ID]; ok {
			out.AccountGroups = ags
		}
		outByID[entAcc.ID] = out
	}
	out := make([]*service.Account, 0, len(uniqueIDs))
	for _, id := range uniqueIDs {
		if _, ok := entByID[id]; !ok {
			continue
		}
		if acc, ok := outByID[id]; ok && acc != nil {
			out = append(out, acc)
		}
	}
	return out, nil
}
func (r *accountRepository) ExistsByID(ctx context.Context, id int64) (bool, error) {
	exists, err := r.client.Account.Query().Where(dbaccount.IDEQ(id)).Exist(ctx)
	if err != nil {
		return false, err
	}
	return exists, nil
}
func (r *accountRepository) GetByCRSAccountID(ctx context.Context, crsAccountID string) (*service.Account, error) {
	if crsAccountID == "" {
		return nil, nil
	}
	m, err := r.client.Account.Query().Where(func(s *entsql.Selector) {
		s.Where(sqljson.ValueEQ(dbaccount.FieldExtra, crsAccountID, sqljson.Path("crs_account_id")))
	}).Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	accounts, err := r.accountsToService(ctx, []*dbent.Account{m})
	if err != nil {
		return nil, err
	}
	if len(accounts) == 0 {
		return nil, nil
	}
	return &accounts[0], nil
}
func (r *accountRepository) ListCRSAccountIDs(ctx context.Context) (map[string]int64, error) {
	rows, err := r.sql.QueryContext(ctx, `
		SELECT id, extra->>'crs_account_id'
		FROM accounts
		WHERE deleted_at IS NULL
			AND extra->>'crs_account_id' IS NOT NULL
			AND extra->>'crs_account_id' != ''
	`)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()
	result := make(map[string]int64)
	for rows.Next() {
		var id int64
		var crsID string
		if err := rows.Scan(&id, &crsID); err != nil {
			return nil, err
		}
		result[crsID] = id
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}
func (r *accountRepository) Update(ctx context.Context, account *service.Account) error {
	if account == nil {
		return nil
	}
	builder := r.client.Account.UpdateOneID(account.ID).SetName(account.Name).SetNillableNotes(account.Notes).SetPlatform(account.Platform).SetType(account.Type).SetCredentials(normalizeJSONMap(account.Credentials)).SetExtra(normalizeJSONMap(account.Extra)).SetConcurrency(account.Concurrency).SetPriority(account.Priority).SetStatus(account.Status).SetErrorMessage(account.ErrorMessage).SetSchedulable(account.Schedulable).SetAutoPauseOnExpired(account.AutoPauseOnExpired)
	lifecycleState := service.NormalizeAccountLifecycleInput(account.LifecycleState)
	if lifecycleState == service.AccountLifecycleAll {
		lifecycleState = service.AccountLifecycleNormal
	}
	builder.SetLifecycleState(lifecycleState)
	if trimmed := account.LifecycleReasonCode; trimmed != "" {
		builder.SetLifecycleReasonCode(trimmed)
	} else {
		builder.ClearLifecycleReasonCode()
	}
	if trimmed := account.LifecycleReasonMessage; trimmed != "" {
		builder.SetLifecycleReasonMessage(trimmed)
	} else {
		builder.ClearLifecycleReasonMessage()
	}
	if account.BlacklistedAt != nil {
		builder.SetBlacklistedAt(*account.BlacklistedAt)
	} else {
		builder.ClearBlacklistedAt()
	}
	if account.BlacklistPurgeAt != nil {
		builder.SetBlacklistPurgeAt(*account.BlacklistPurgeAt)
	} else {
		builder.ClearBlacklistPurgeAt()
	}
	if account.RateMultiplier != nil {
		builder.SetRateMultiplier(*account.RateMultiplier)
	}
	if account.LoadFactor != nil {
		builder.SetLoadFactor(*account.LoadFactor)
	} else {
		builder.ClearLoadFactor()
	}
	if account.ProxyID != nil {
		builder.SetProxyID(*account.ProxyID)
	} else {
		builder.ClearProxyID()
	}
	if account.LastUsedAt != nil {
		builder.SetLastUsedAt(*account.LastUsedAt)
	} else {
		builder.ClearLastUsedAt()
	}
	if account.ExpiresAt != nil {
		builder.SetExpiresAt(*account.ExpiresAt)
	} else {
		builder.ClearExpiresAt()
	}
	if account.RateLimitedAt != nil {
		builder.SetRateLimitedAt(*account.RateLimitedAt)
	} else {
		builder.ClearRateLimitedAt()
	}
	if account.RateLimitResetAt != nil {
		builder.SetRateLimitResetAt(*account.RateLimitResetAt)
	} else {
		builder.ClearRateLimitResetAt()
	}
	if account.OverloadUntil != nil {
		builder.SetOverloadUntil(*account.OverloadUntil)
	} else {
		builder.ClearOverloadUntil()
	}
	if account.SessionWindowStart != nil {
		builder.SetSessionWindowStart(*account.SessionWindowStart)
	} else {
		builder.ClearSessionWindowStart()
	}
	if account.SessionWindowEnd != nil {
		builder.SetSessionWindowEnd(*account.SessionWindowEnd)
	} else {
		builder.ClearSessionWindowEnd()
	}
	if account.SessionWindowStatus != "" {
		builder.SetSessionWindowStatus(account.SessionWindowStatus)
	} else {
		builder.ClearSessionWindowStatus()
	}
	if account.Notes == nil {
		builder.ClearNotes()
	}
	updated, err := builder.Save(ctx)
	if err != nil {
		return translatePersistenceError(err, service.ErrAccountNotFound, nil)
	}
	account.UpdatedAt = updated.UpdatedAt
	if err := enqueueSchedulerOutbox(ctx, r.sql, service.SchedulerOutboxEventAccountChanged, &account.ID, nil, buildSchedulerGroupPayload(account.GroupIDs)); err != nil {
		logger.LegacyPrintf("repository.account", "[SchedulerOutbox] enqueue account update failed: account=%d err=%v", account.ID, err)
	}
	// Keep scheduler cache in sync immediately (e.g. model_mapping / credentials changes),
	// even if the outbox worker is delayed or down.
	r.syncSchedulerAccountSnapshot(ctx, account.ID)
	return nil
}
func (r *accountRepository) List(ctx context.Context, params pagination.PaginationParams) ([]service.Account, *pagination.PaginationResult, error) {
	return r.ListWithFilters(ctx, params, "", "", "", "", 0, service.AccountLifecycleNormal)
}
func (r *accountRepository) ListWithFilters(ctx context.Context, params pagination.PaginationParams, platform, accountType, status, search string, groupID int64, lifecycle string) ([]service.Account, *pagination.PaginationResult, error) {
	q := r.client.Account.Query()
	if platform != "" {
		q = q.Where(dbaccount.PlatformEQ(platform))
	}
	if accountType != "" {
		q = q.Where(dbaccount.TypeEQ(accountType))
	}
	if status != "" {
		switch status {
		case "rate_limited":
			q = q.Where(dbaccount.RateLimitResetAtGT(time.Now()))
		case "temp_unschedulable":
			q = q.Where(dbpredicate.Account(func(s *entsql.Selector) {
				col := s.C("temp_unschedulable_until")
				s.Where(entsql.And(entsql.Not(entsql.IsNull(col)), entsql.GT(col, entsql.Expr("NOW()"))))
			}))
		default:
			q = q.Where(dbaccount.StatusEQ(status))
		}
	}
	if search != "" {
		q = q.Where(dbaccount.NameContainsFold(search))
	}
	if groupID == service.AccountListGroupUngrouped {
		q = q.Where(dbaccount.Not(dbaccount.HasAccountGroups()))
	} else if groupID > 0 {
		q = q.Where(dbaccount.HasAccountGroupsWith(dbaccountgroup.GroupIDEQ(groupID)))
	}
	if predicate := lifecyclePredicate(lifecycle); predicate != nil {
		q = q.Where(predicate)
	}
	total, err := q.Count(ctx)
	if err != nil {
		return nil, nil, err
	}
	accounts, err := q.Offset(params.Offset()).Limit(params.Limit()).Order(dbent.Desc(dbaccount.FieldID)).All(ctx)
	if err != nil {
		return nil, nil, err
	}
	outAccounts, err := r.accountsToService(ctx, accounts)
	if err != nil {
		return nil, nil, err
	}
	return outAccounts, paginationResultFromTotal(int64(total), params), nil
}
func (r *accountRepository) ListByGroup(ctx context.Context, groupID int64) ([]service.Account, error) {
	accounts, err := r.queryAccountsByGroup(ctx, groupID, accountGroupQueryOptions{status: service.StatusActive, lifecycle: service.AccountLifecycleNormal})
	if err != nil {
		return nil, err
	}
	return accounts, nil
}
func (r *accountRepository) ListActive(ctx context.Context) ([]service.Account, error) {
	accounts, err := r.client.Account.Query().Where(dbaccount.StatusEQ(service.StatusActive), dbaccount.LifecycleStateEQ(service.AccountLifecycleNormal)).Order(dbent.Asc(dbaccount.FieldPriority)).All(ctx)
	if err != nil {
		return nil, err
	}
	return r.accountsToService(ctx, accounts)
}
func (r *accountRepository) ListByPlatform(ctx context.Context, platform string) ([]service.Account, error) {
	accounts, err := r.client.Account.Query().Where(dbaccount.PlatformEQ(platform), dbaccount.StatusEQ(service.StatusActive), dbaccount.LifecycleStateEQ(service.AccountLifecycleNormal)).Order(dbent.Asc(dbaccount.FieldPriority)).All(ctx)
	if err != nil {
		return nil, err
	}
	return r.accountsToService(ctx, accounts)
}
func (r *accountRepository) UpdateLastUsed(ctx context.Context, id int64) error {
	now := time.Now()
	_, err := r.client.Account.Update().Where(dbaccount.IDEQ(id)).SetLastUsedAt(now).Save(ctx)
	if err != nil {
		return err
	}
	payload := map[string]any{"last_used": map[string]int64{strconv.FormatInt(id, 10): now.Unix()}}
	if err := enqueueSchedulerOutbox(ctx, r.sql, service.SchedulerOutboxEventAccountLastUsed, &id, nil, payload); err != nil {
		logger.LegacyPrintf("repository.account", "[SchedulerOutbox] enqueue last used failed: account=%d err=%v", id, err)
	}
	return nil
}
func (r *accountRepository) BatchUpdateLastUsed(ctx context.Context, updates map[int64]time.Time) error {
	if len(updates) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(updates))
	args := make([]any, 0, len(updates)*2+1)
	caseSQL := "UPDATE accounts SET last_used_at = CASE id"
	idx := 1
	for id, ts := range updates {
		caseSQL += " WHEN $" + itoa(idx) + " THEN $" + itoa(idx+1) + "::timestamptz"
		args = append(args, id, ts)
		ids = append(ids, id)
		idx += 2
	}
	caseSQL += " END, updated_at = NOW() WHERE id = ANY($" + itoa(idx) + ") AND deleted_at IS NULL"
	args = append(args, pq.Array(ids))
	_, err := r.sql.ExecContext(ctx, caseSQL, args...)
	if err != nil {
		return err
	}
	lastUsedPayload := make(map[string]int64, len(updates))
	for id, ts := range updates {
		lastUsedPayload[strconv.FormatInt(id, 10)] = ts.Unix()
	}
	payload := map[string]any{"last_used": lastUsedPayload}
	if err := enqueueSchedulerOutbox(ctx, r.sql, service.SchedulerOutboxEventAccountLastUsed, nil, nil, payload); err != nil {
		logger.LegacyPrintf("repository.account", "[SchedulerOutbox] enqueue batch last used failed: err=%v", err)
	}
	return nil
}
