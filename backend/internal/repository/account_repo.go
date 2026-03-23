package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	entsql "entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqljson"
	"errors"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	dbaccount "github.com/Wei-Shaw/sub2api/ent/account"
	dbaccountgroup "github.com/Wei-Shaw/sub2api/ent/accountgroup"
	dbpredicate "github.com/Wei-Shaw/sub2api/ent/predicate"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
	"strconv"
	"strings"
	"time"
)

type accountRepository struct {
	client         *dbent.Client
	sql            sqlExecutor
	schedulerCache service.SchedulerCache
}

func NewAccountRepository(client *dbent.Client, sqlDB *sql.DB, schedulerCache service.SchedulerCache) service.AccountRepository {
	return newAccountRepositoryWithSQL(client, sqlDB, schedulerCache)
}
func newAccountRepositoryWithSQL(client *dbent.Client, sqlq sqlExecutor, schedulerCache service.SchedulerCache) *accountRepository {
	return &accountRepository{client: client, sql: sqlq, schedulerCache: schedulerCache}
}
func (r *accountRepository) Delete(ctx context.Context, id int64) error {
	groupIDs, err := r.loadAccountGroupIDs(ctx, id)
	if err != nil {
		return err
	}
	tx, err := r.client.Tx(ctx)
	if err != nil && !errors.Is(err, dbent.ErrTxStarted) {
		return err
	}
	var txClient *dbent.Client
	if err == nil {
		defer func() {
			_ = tx.Rollback()
		}()
		txClient = tx.Client()
	} else {
		txClient = r.client
	}
	if _, err := txClient.AccountGroup.Delete().Where(dbaccountgroup.AccountIDEQ(id)).Exec(ctx); err != nil {
		return err
	}
	if _, err := txClient.Account.Delete().Where(dbaccount.IDEQ(id)).Exec(ctx); err != nil {
		return err
	}
	if tx != nil {
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	if err := enqueueSchedulerOutbox(ctx, r.sql, service.SchedulerOutboxEventAccountChanged, &id, nil, buildSchedulerGroupPayload(groupIDs)); err != nil {
		logger.LegacyPrintf("repository.account", "[SchedulerOutbox] enqueue account delete failed: account=%d err=%v", id, err)
	}
	return nil
}
func (r *accountRepository) SetError(ctx context.Context, id int64, errorMsg string) error {
	_, err := r.sql.ExecContext(ctx, `
		UPDATE accounts
		SET status = CASE
				WHEN lifecycle_state = $2 THEN status
				ELSE $3
			END,
			error_message = $4,
			updated_at = NOW()
		WHERE id = $1
			AND deleted_at IS NULL
	`, id, service.AccountLifecycleBlacklisted, service.StatusError, errorMsg)
	if err != nil {
		return err
	}
	if err := enqueueSchedulerOutbox(ctx, r.sql, service.SchedulerOutboxEventAccountChanged, &id, nil, nil); err != nil {
		logger.LegacyPrintf("repository.account", "[SchedulerOutbox] enqueue set error failed: account=%d err=%v", id, err)
	}
	r.syncSchedulerAccountSnapshot(ctx, id)
	return nil
}
func (r *accountRepository) syncSchedulerAccountSnapshot(ctx context.Context, accountID int64) {
	if r == nil || r.schedulerCache == nil || accountID <= 0 {
		return
	}
	account, err := r.GetByID(ctx, accountID)
	if err != nil {
		logger.LegacyPrintf("repository.account", "[Scheduler] sync account snapshot read failed: id=%d err=%v", accountID, err)
		return
	}
	if err := r.schedulerCache.SetAccount(ctx, account); err != nil {
		logger.LegacyPrintf("repository.account", "[Scheduler] sync account snapshot write failed: id=%d err=%v", accountID, err)
	}
}
func (r *accountRepository) syncSchedulerAccountSnapshots(ctx context.Context, accountIDs []int64) {
	if r == nil || r.schedulerCache == nil || len(accountIDs) == 0 {
		return
	}
	uniqueIDs := make([]int64, 0, len(accountIDs))
	seen := make(map[int64]struct{}, len(accountIDs))
	for _, id := range accountIDs {
		if id <= 0 {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		uniqueIDs = append(uniqueIDs, id)
	}
	if len(uniqueIDs) == 0 {
		return
	}
	accounts, err := r.GetByIDs(ctx, uniqueIDs)
	if err != nil {
		logger.LegacyPrintf("repository.account", "[Scheduler] batch sync account snapshot read failed: count=%d err=%v", len(uniqueIDs), err)
		return
	}
	for _, account := range accounts {
		if account == nil {
			continue
		}
		if err := r.schedulerCache.SetAccount(ctx, account); err != nil {
			logger.LegacyPrintf("repository.account", "[Scheduler] batch sync account snapshot write failed: id=%d err=%v", account.ID, err)
		}
	}
}
func (r *accountRepository) ClearError(ctx context.Context, id int64) error {
	_, err := r.sql.ExecContext(ctx, `
		UPDATE accounts
		SET status = CASE
				WHEN lifecycle_state = $2 THEN status
				ELSE $3
			END,
			error_message = '',
			updated_at = NOW()
		WHERE id = $1
			AND deleted_at IS NULL
	`, id, service.AccountLifecycleBlacklisted, service.StatusActive)
	if err != nil {
		return err
	}
	if err := enqueueSchedulerOutbox(ctx, r.sql, service.SchedulerOutboxEventAccountChanged, &id, nil, nil); err != nil {
		logger.LegacyPrintf("repository.account", "[SchedulerOutbox] enqueue clear error failed: account=%d err=%v", id, err)
	}
	r.syncSchedulerAccountSnapshot(ctx, id)
	return nil
}
func (r *accountRepository) BindGroups(ctx context.Context, accountID int64, groupIDs []int64) error {
	existingGroupIDs, err := r.loadAccountGroupIDs(ctx, accountID)
	if err != nil {
		return err
	}
	tx, err := r.client.Tx(ctx)
	if err != nil && !errors.Is(err, dbent.ErrTxStarted) {
		return err
	}
	var txClient *dbent.Client
	if err == nil {
		defer func() {
			_ = tx.Rollback()
		}()
		txClient = tx.Client()
	} else {
		txClient = r.client
	}
	if _, err := txClient.AccountGroup.Delete().Where(dbaccountgroup.AccountIDEQ(accountID)).Exec(ctx); err != nil {
		return err
	}
	if len(groupIDs) == 0 {
		if tx != nil {
			return tx.Commit()
		}
		return nil
	}
	builders := make([]*dbent.AccountGroupCreate, 0, len(groupIDs))
	for i, groupID := range groupIDs {
		builders = append(builders, txClient.AccountGroup.Create().SetAccountID(accountID).SetGroupID(groupID).SetPriority(i+1))
	}
	if _, err := txClient.AccountGroup.CreateBulk(builders...).Save(ctx); err != nil {
		return err
	}
	if tx != nil {
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	payload := buildSchedulerGroupPayload(mergeGroupIDs(existingGroupIDs, groupIDs))
	if err := enqueueSchedulerOutbox(ctx, r.sql, service.SchedulerOutboxEventAccountGroupsChanged, &accountID, nil, payload); err != nil {
		logger.LegacyPrintf("repository.account", "[SchedulerOutbox] enqueue bind groups failed: account=%d err=%v", accountID, err)
	}
	return nil
}
func (r *accountRepository) AutoPauseExpiredAccounts(ctx context.Context, now time.Time) (int64, error) {
	result, err := r.sql.ExecContext(ctx, `
		UPDATE accounts
		SET schedulable = FALSE,
			updated_at = NOW()
		WHERE deleted_at IS NULL
			AND schedulable = TRUE
			AND auto_pause_on_expired = TRUE
			AND expires_at IS NOT NULL
			AND expires_at <= $1
	`, now)
	if err != nil {
		return 0, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	if rows > 0 {
		if err := enqueueSchedulerOutbox(ctx, r.sql, service.SchedulerOutboxEventFullRebuild, nil, nil, nil); err != nil {
			logger.LegacyPrintf("repository.account", "[SchedulerOutbox] enqueue auto pause rebuild failed: err=%v", err)
		}
	}
	return rows, nil
}
func extraValueToFloat64(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int8:
		return float64(v), true
	case int16:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint8:
		return float64(v), true
	case uint16:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		return float64(v), true
	case json.Number:
		parsed, err := v.Float64()
		return parsed, err == nil
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}
func (r *accountRepository) BulkUpdate(ctx context.Context, ids []int64, updates service.AccountBulkUpdate) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	setClauses := make([]string, 0, 8)
	args := make([]any, 0, 8)
	idx := 1
	if updates.Name != nil {
		setClauses = append(setClauses, "name = $"+itoa(idx))
		args = append(args, *updates.Name)
		idx++
	}
	if updates.ProxyID != nil {
		if *updates.ProxyID == 0 {
			setClauses = append(setClauses, "proxy_id = NULL")
		} else {
			setClauses = append(setClauses, "proxy_id = $"+itoa(idx))
			args = append(args, *updates.ProxyID)
			idx++
		}
	}
	if updates.Concurrency != nil {
		setClauses = append(setClauses, "concurrency = $"+itoa(idx))
		args = append(args, *updates.Concurrency)
		idx++
	}
	if updates.Priority != nil {
		setClauses = append(setClauses, "priority = $"+itoa(idx))
		args = append(args, *updates.Priority)
		idx++
	}
	if updates.RateMultiplier != nil {
		setClauses = append(setClauses, "rate_multiplier = $"+itoa(idx))
		args = append(args, *updates.RateMultiplier)
		idx++
	}
	if updates.LoadFactor != nil {
		if *updates.LoadFactor <= 0 {
			setClauses = append(setClauses, "load_factor = NULL")
		} else {
			setClauses = append(setClauses, "load_factor = $"+itoa(idx))
			args = append(args, *updates.LoadFactor)
			idx++
		}
	}
	if updates.Status != nil {
		setClauses = append(setClauses, "status = $"+itoa(idx))
		args = append(args, *updates.Status)
		idx++
	}
	if updates.Schedulable != nil {
		setClauses = append(setClauses, "schedulable = $"+itoa(idx))
		args = append(args, *updates.Schedulable)
		idx++
	}
	if updates.LifecycleState != nil {
		setClauses = append(setClauses, "lifecycle_state = $"+itoa(idx))
		args = append(args, service.NormalizeAccountLifecycleInput(*updates.LifecycleState))
		idx++
	}
	if updates.LifecycleReasonCode != nil {
		setClauses = append(setClauses, "lifecycle_reason_code = $"+itoa(idx))
		args = append(args, *updates.LifecycleReasonCode)
		idx++
	}
	if updates.LifecycleReasonMessage != nil {
		setClauses = append(setClauses, "lifecycle_reason_message = $"+itoa(idx))
		args = append(args, *updates.LifecycleReasonMessage)
		idx++
	}
	if len(updates.Credentials) > 0 {
		payload, err := json.Marshal(updates.Credentials)
		if err != nil {
			return 0, err
		}
		setClauses = append(setClauses, "credentials = COALESCE(credentials, '{}'::jsonb) || $"+itoa(idx)+"::jsonb")
		args = append(args, payload)
		idx++
	}
	if len(updates.Extra) > 0 {
		payload, err := json.Marshal(updates.Extra)
		if err != nil {
			return 0, err
		}
		setClauses = append(setClauses, "extra = COALESCE(extra, '{}'::jsonb) || $"+itoa(idx)+"::jsonb")
		args = append(args, payload)
		idx++
	}
	if len(setClauses) == 0 {
		return 0, nil
	}
	setClauses = append(setClauses, "updated_at = NOW()")
	query := "UPDATE accounts SET " + joinClauses(setClauses, ", ") + " WHERE id = ANY($" + itoa(idx) + ") AND deleted_at IS NULL"
	args = append(args, pq.Array(ids))
	result, err := r.sql.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	if rows > 0 {
		payload := map[string]any{"account_ids": ids}
		if err := enqueueSchedulerOutbox(ctx, r.sql, service.SchedulerOutboxEventAccountBulkChanged, nil, nil, payload); err != nil {
			logger.LegacyPrintf("repository.account", "[SchedulerOutbox] enqueue bulk update failed: err=%v", err)
		}
		shouldSync := false
		if updates.Status != nil && (*updates.Status == service.StatusError || *updates.Status == service.StatusDisabled) {
			shouldSync = true
		}
		if updates.Schedulable != nil && !*updates.Schedulable {
			shouldSync = true
		}
		if updates.LifecycleState != nil {
			shouldSync = true
		}
		if shouldSync {
			r.syncSchedulerAccountSnapshots(ctx, ids)
		}
	}
	return rows, nil
}

func (r *accountRepository) MarkBlacklisted(ctx context.Context, id int64, reasonCode, reasonMessage string, blacklistedAt, purgeAt time.Time) error {
	_, err := r.sql.ExecContext(ctx, `
		UPDATE accounts
		SET lifecycle_state = $2,
			lifecycle_reason_code = $3,
			lifecycle_reason_message = $4,
			blacklisted_at = $5,
			blacklist_purge_at = $6,
			status = $7,
			schedulable = FALSE,
			error_message = $4,
			rate_limited_at = NULL,
			rate_limit_reset_at = NULL,
			overload_until = NULL,
			temp_unschedulable_until = NULL,
			temp_unschedulable_reason = NULL,
			extra = COALESCE(extra, '{}'::jsonb) - 'model_rate_limits' - 'antigravity_quota_scopes',
			updated_at = NOW()
		WHERE id = $1
			AND deleted_at IS NULL
	`, id, service.AccountLifecycleBlacklisted, strings.TrimSpace(reasonCode), strings.TrimSpace(reasonMessage), blacklistedAt, purgeAt, service.StatusDisabled)
	if err != nil {
		return err
	}
	if err := enqueueSchedulerOutbox(ctx, r.sql, service.SchedulerOutboxEventAccountChanged, &id, nil, nil); err != nil {
		logger.LegacyPrintf("repository.account", "[SchedulerOutbox] enqueue mark blacklisted failed: account=%d err=%v", id, err)
	}
	r.syncSchedulerAccountSnapshot(ctx, id)
	return nil
}

func (r *accountRepository) RestoreBlacklisted(ctx context.Context, id int64) error {
	_, err := r.sql.ExecContext(ctx, `
		UPDATE accounts
		SET lifecycle_state = $2,
			lifecycle_reason_code = NULL,
			lifecycle_reason_message = NULL,
			blacklisted_at = NULL,
			blacklist_purge_at = NULL,
			status = $3,
			schedulable = TRUE,
			error_message = '',
			rate_limited_at = NULL,
			rate_limit_reset_at = NULL,
			overload_until = NULL,
			temp_unschedulable_until = NULL,
			temp_unschedulable_reason = NULL,
			extra = COALESCE(extra, '{}'::jsonb) - 'model_rate_limits' - 'antigravity_quota_scopes',
			updated_at = NOW()
		WHERE id = $1
			AND deleted_at IS NULL
	`, id, service.AccountLifecycleNormal, service.StatusActive)
	if err != nil {
		return err
	}
	if err := enqueueSchedulerOutbox(ctx, r.sql, service.SchedulerOutboxEventAccountChanged, &id, nil, nil); err != nil {
		logger.LegacyPrintf("repository.account", "[SchedulerOutbox] enqueue restore blacklisted failed: account=%d err=%v", id, err)
	}
	r.syncSchedulerAccountSnapshot(ctx, id)
	return nil
}

func (r *accountRepository) ListBlacklistedForPurge(ctx context.Context, now time.Time, limit int) ([]service.Account, error) {
	if limit <= 0 {
		limit = 100
	}
	accounts, err := r.client.Account.Query().
		Where(
			dbaccount.LifecycleStateEQ(service.AccountLifecycleBlacklisted),
			dbaccount.BlacklistPurgeAtLTE(now),
		).
		Order(dbent.Asc(dbaccount.FieldBlacklistPurgeAt)).
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, err
	}
	return r.accountsToService(ctx, accounts)
}

type accountGroupQueryOptions struct {
	status      string
	schedulable bool
	platforms   []string
	lifecycle   string
}

func (r *accountRepository) queryAccountsByGroup(ctx context.Context, groupID int64, opts accountGroupQueryOptions) ([]service.Account, error) {
	q := r.client.AccountGroup.Query().Where(dbaccountgroup.GroupIDEQ(groupID))
	preds := make([]dbpredicate.Account, 0, 6)
	preds = append(preds, dbaccount.DeletedAtIsNil())
	if opts.status != "" {
		preds = append(preds, dbaccount.StatusEQ(opts.status))
	}
	if len(opts.platforms) > 0 {
		preds = append(preds, dbaccount.PlatformIn(opts.platforms...))
	}
	if predicate := lifecyclePredicate(opts.lifecycle); predicate != nil {
		preds = append(preds, predicate)
	}
	if opts.schedulable {
		now := time.Now()
		preds = append(preds, dbaccount.SchedulableEQ(true), tempUnschedulablePredicate(), notExpiredPredicate(now), dbaccount.Or(dbaccount.OverloadUntilIsNil(), dbaccount.OverloadUntilLTE(now)), dbaccount.Or(dbaccount.RateLimitResetAtIsNil(), dbaccount.RateLimitResetAtLTE(now)))
	}
	if len(preds) > 0 {
		q = q.Where(dbaccountgroup.HasAccountWith(preds...))
	}
	groups, err := q.Order(dbaccountgroup.ByPriority(), dbaccountgroup.ByAccountField(dbaccount.FieldPriority)).WithAccount().All(ctx)
	if err != nil {
		return nil, err
	}
	orderedIDs := make([]int64, 0, len(groups))
	accountMap := make(map[int64]*dbent.Account, len(groups))
	for _, ag := range groups {
		if ag.Edges.Account == nil {
			continue
		}
		if _, exists := accountMap[ag.AccountID]; exists {
			continue
		}
		accountMap[ag.AccountID] = ag.Edges.Account
		orderedIDs = append(orderedIDs, ag.AccountID)
	}
	accounts := make([]*dbent.Account, 0, len(orderedIDs))
	for _, id := range orderedIDs {
		if acc, ok := accountMap[id]; ok {
			accounts = append(accounts, acc)
		}
	}
	return r.accountsToService(ctx, accounts)
}
func (r *accountRepository) FindByExtraField(ctx context.Context, key string, value any) ([]service.Account, error) {
	accounts, err := r.client.Account.Query().Where(dbaccount.PlatformEQ("sora"), dbaccount.DeletedAtIsNil(), func(s *entsql.Selector) {
		path := sqljson.Path(key)
		switch v := value.(type) {
		case string:
			preds := []*entsql.Predicate{sqljson.ValueEQ(dbaccount.FieldExtra, v, path)}
			if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
				preds = append(preds, sqljson.ValueEQ(dbaccount.FieldExtra, parsed, path))
			}
			if len(preds) == 1 {
				s.Where(preds[0])
			} else {
				s.Where(entsql.Or(preds...))
			}
		case int:
			s.Where(entsql.Or(sqljson.ValueEQ(dbaccount.FieldExtra, v, path), sqljson.ValueEQ(dbaccount.FieldExtra, strconv.Itoa(v), path)))
		case int64:
			s.Where(entsql.Or(sqljson.ValueEQ(dbaccount.FieldExtra, v, path), sqljson.ValueEQ(dbaccount.FieldExtra, strconv.FormatInt(v, 10), path)))
		case json.Number:
			if parsed, err := v.Int64(); err == nil {
				s.Where(entsql.Or(sqljson.ValueEQ(dbaccount.FieldExtra, parsed, path), sqljson.ValueEQ(dbaccount.FieldExtra, v.String(), path)))
			} else {
				s.Where(sqljson.ValueEQ(dbaccount.FieldExtra, v.String(), path))
			}
		default:
			s.Where(sqljson.ValueEQ(dbaccount.FieldExtra, value, path))
		}
	}).All(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrAccountNotFound, nil)
	}
	return r.accountsToService(ctx, accounts)
}
func (r *accountRepository) ResetQuotaUsed(ctx context.Context, id int64) error {
	_, err := r.sql.ExecContext(ctx, `UPDATE accounts SET extra = (
			COALESCE(extra, '{}'::jsonb)
			|| '{"quota_used": 0, "quota_daily_used": 0, "quota_weekly_used": 0}'::jsonb
		) - 'quota_daily_start' - 'quota_weekly_start', updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	if err := enqueueSchedulerOutbox(ctx, r.sql, service.SchedulerOutboxEventAccountChanged, &id, nil, nil); err != nil {
		logger.LegacyPrintf("repository.account", "[SchedulerOutbox] enqueue quota reset failed: account=%d err=%v", id, err)
	}
	return nil
}
