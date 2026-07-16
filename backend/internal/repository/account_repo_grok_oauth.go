package repository

import (
	"context"
	"encoding/json"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	dbaccount "github.com/Wei-Shaw/sub2api/ent/account"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (r *accountRepository) ListActiveGrokOAuthAfterID(ctx context.Context, afterID int64, limit int) ([]service.Account, error) {
	if limit <= 0 {
		return []service.Account{}, nil
	}
	accounts, err := r.client.Account.Query().
		Where(
			dbaccount.IDGT(afterID),
			dbaccount.PlatformEQ(service.PlatformGrok),
			dbaccount.TypeEQ(service.AccountTypeOAuth),
			dbaccount.StatusEQ(service.StatusActive),
			dbaccount.DeletedAtIsNil(),
		).
		Order(dbent.Asc(dbaccount.FieldID)).
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, err
	}
	return r.accountsToService(ctx, accounts)
}

func (r *accountRepository) SetGrokOAuthErrorIfCredentialsUnchanged(ctx context.Context, id int64, expectedCredentials map[string]any, errorMsg string) (bool, error) {
	payload, err := json.Marshal(normalizeJSONMap(expectedCredentials))
	if err != nil {
		return false, err
	}
	result, err := r.sql.ExecContext(ctx, `
		UPDATE accounts
		SET status = $4,
			schedulable = FALSE,
			error_message = $5,
			updated_at = NOW()
		WHERE id = $1
			AND platform = $2
			AND type = $3
			AND deleted_at IS NULL
			AND credentials = $6::jsonb
	`, id, service.PlatformGrok, service.AccountTypeOAuth, service.StatusError, errorMsg, string(payload))
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	if affected > 0 {
		if err := enqueueSchedulerOutbox(ctx, r.sql, service.SchedulerOutboxEventAccountChanged, &id, nil, nil); err != nil {
			logger.LegacyPrintf("repository.account", "[SchedulerOutbox] enqueue grok oauth CAS error failed: account=%d err=%v", id, err)
		}
		r.syncSchedulerAccountSnapshot(ctx, id)
	}
	return affected > 0, nil
}

func (r *accountRepository) UpdateGrokOAuthCredentialsIfCredentialsUnchanged(ctx context.Context, id int64, expectedCredentials map[string]any, credentials map[string]any) (bool, error) {
	expectedPayload, err := json.Marshal(normalizeJSONMap(expectedCredentials))
	if err != nil {
		return false, err
	}
	credentialsPayload, err := json.Marshal(normalizeJSONMap(credentials))
	if err != nil {
		return false, err
	}
	result, err := r.sql.ExecContext(ctx, `
		UPDATE accounts
		SET credentials = $4::jsonb,
			updated_at = NOW()
		WHERE id = $1
			AND platform = $2
			AND type = $3
			AND deleted_at IS NULL
			AND credentials = $5::jsonb
	`, id, service.PlatformGrok, service.AccountTypeOAuth, string(credentialsPayload), string(expectedPayload))
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	if affected > 0 {
		if err := enqueueSchedulerOutbox(ctx, r.sql, service.SchedulerOutboxEventAccountChanged, &id, nil, nil); err != nil {
			logger.LegacyPrintf("repository.account", "[SchedulerOutbox] enqueue grok oauth CAS credentials failed: account=%d err=%v", id, err)
		}
		r.syncSchedulerAccountSnapshot(ctx, id)
	}
	return affected > 0, nil
}

func (r *accountRepository) SetRateLimitedIfLater(ctx context.Context, id int64, resetAt time.Time) error {
	now := time.Now()
	result, err := r.sql.ExecContext(ctx, `
		UPDATE accounts
		SET rate_limited_at = COALESCE(rate_limited_at, $2),
			rate_limit_reset_at = $3,
			updated_at = NOW()
		WHERE id = $1
			AND deleted_at IS NULL
			AND (rate_limit_reset_at IS NULL OR rate_limit_reset_at < $3)
	`, id, now, resetAt)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected > 0 {
		if err := enqueueSchedulerOutbox(ctx, r.sql, service.SchedulerOutboxEventAccountChanged, &id, nil, nil); err != nil {
			logger.LegacyPrintf("repository.account", "[SchedulerOutbox] enqueue grok rate limit extension failed: account=%d err=%v", id, err)
		}
		r.syncSchedulerAccountSnapshot(ctx, id)
	}
	return nil
}

func (r *accountRepository) ClearRateLimitIfObserved(ctx context.Context, id int64, observedLimitedAt, observedResetAt time.Time) (bool, error) {
	result, err := r.sql.ExecContext(ctx, `
		UPDATE accounts
		SET rate_limited_at = NULL,
			rate_limit_reset_at = NULL,
			overload_until = NULL,
			updated_at = NOW()
		WHERE id = $1
			AND deleted_at IS NULL
			AND rate_limited_at = $2
			AND rate_limit_reset_at = $3
	`, id, observedLimitedAt, observedResetAt)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	if affected > 0 {
		if err := enqueueSchedulerOutbox(ctx, r.sql, service.SchedulerOutboxEventAccountChanged, &id, nil, nil); err != nil {
			logger.LegacyPrintf("repository.account", "[SchedulerOutbox] enqueue grok rate limit recovery failed: account=%d err=%v", id, err)
		}
		r.syncSchedulerAccountSnapshot(ctx, id)
	}
	return affected > 0, nil
}
