package repository

import (
	"context"
	"strings"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	dbaccount "github.com/Wei-Shaw/sub2api/ent/account"
	dbaccountgroup "github.com/Wei-Shaw/sub2api/ent/accountgroup"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (r *accountRepository) ListOpsAccountsForStats(ctx context.Context, platformFilter string, groupIDFilter *int64) ([]service.Account, error) {
	if r == nil || r.client == nil {
		return []service.Account{}, nil
	}

	q := r.client.Account.Query().
		Where(
			dbaccount.DeletedAtIsNil(),
			dbaccount.PlatformNotIn(service.UnsupportedPrimaryAccountPredicateValues()...),
		)
	if platform := strings.TrimSpace(platformFilter); platform != "" {
		values := platformFilterValues(platform)
		if len(values) == 1 {
			q = q.Where(dbaccount.PlatformEQ(values[0]))
		} else if len(values) > 1 {
			q = q.Where(dbaccount.PlatformIn(values...))
		}
	}
	if groupIDFilter != nil && *groupIDFilter > 0 {
		q = q.Where(dbaccount.HasAccountGroupsWith(dbaccountgroup.GroupIDEQ(*groupIDFilter)))
	}

	accounts, err := q.Select(
		dbaccount.FieldID,
		dbaccount.FieldName,
		dbaccount.FieldPlatform,
		dbaccount.FieldType,
		dbaccount.FieldCredentials,
		dbaccount.FieldExtra,
		dbaccount.FieldConcurrency,
		dbaccount.FieldLoadFactor,
		dbaccount.FieldStatus,
		dbaccount.FieldLifecycleState,
		dbaccount.FieldErrorMessage,
		dbaccount.FieldSchedulable,
		dbaccount.FieldRateLimitedAt,
		dbaccount.FieldRateLimitResetAt,
		dbaccount.FieldOverloadUntil,
		dbaccount.FieldTempUnschedulableUntil,
		dbaccount.FieldTempUnschedulableReason,
	).
		Order(dbent.Asc(dbaccount.FieldID)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	return r.accountsToService(ctx, accounts)
}
