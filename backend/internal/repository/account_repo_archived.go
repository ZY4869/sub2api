package repository

import (
	"context"
	"errors"
	"fmt"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	dbaccountgroup "github.com/Wei-Shaw/sub2api/ent/accountgroup"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"strings"
)

func (r *accountRepository) ListArchivedGroups(ctx context.Context, filters service.ArchivedAccountGroupFilters) ([]service.ArchivedAccountGroupSummary, error) {
	if filters.GroupID == service.AccountListGroupUngrouped {
		return []service.ArchivedAccountGroupSummary{}, nil
	}

	whereClauses := []string{
		"a.deleted_at IS NULL",
		"g.deleted_at IS NULL",
		"a.lifecycle_state = $1",
	}
	args := []any{service.AccountLifecycleArchived}
	argIndex := 3

	if platform := strings.TrimSpace(filters.Platform); platform != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("a.platform = $%d", argIndex))
		args = append(args, platform)
		argIndex++
	}
	if accountType := strings.TrimSpace(filters.AccountType); accountType != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("a.type = $%d", argIndex))
		args = append(args, accountType)
		argIndex++
	}

	switch status := strings.TrimSpace(filters.Status); status {
	case "":
	case "rate_limited":
		whereClauses = append(whereClauses, "a.rate_limit_reset_at IS NOT NULL", "a.rate_limit_reset_at > NOW()")
	case "temp_unschedulable":
		whereClauses = append(whereClauses, "a.temp_unschedulable_until IS NOT NULL", "a.temp_unschedulable_until > NOW()")
	case "paused":
		whereClauses = append(whereClauses, "a.schedulable = FALSE")
	default:
		whereClauses = append(whereClauses, fmt.Sprintf("a.status = $%d", argIndex))
		args = append(args, status)
		argIndex++
	}

	if search := strings.TrimSpace(filters.Search); search != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("a.name ILIKE $%d", argIndex))
		args = append(args, "%"+search+"%")
		argIndex++
	}

	if filters.GroupID > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("ag.group_id = $%d", argIndex))
		args = append(args, filters.GroupID)
	}

	query := `
		SELECT
			ag.group_id,
			g.name,
			COUNT(DISTINCT a.id) AS total_count,
			COUNT(DISTINCT CASE WHEN a.status = $2 THEN a.id END) AS available_count,
			MAX(a.updated_at) AS latest_updated_at
		FROM accounts a
		INNER JOIN account_groups ag ON ag.account_id = a.id
		INNER JOIN groups g ON g.id = ag.group_id
		WHERE ` + strings.Join(whereClauses, " AND ") + `
		GROUP BY ag.group_id, g.name
		ORDER BY MAX(a.updated_at) DESC, g.name ASC
	`

	args = append([]any{args[0], service.StatusActive}, args[1:]...)

	rows, err := r.sql.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	summaries := make([]service.ArchivedAccountGroupSummary, 0)
	for rows.Next() {
		var summary service.ArchivedAccountGroupSummary
		if err := rows.Scan(
			&summary.GroupID,
			&summary.GroupName,
			&summary.TotalCount,
			&summary.AvailableCount,
			&summary.LatestUpdatedAt,
		); err != nil {
			return nil, err
		}
		summary.InvalidCount = summary.TotalCount - summary.AvailableCount
		summaries = append(summaries, summary)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return summaries, nil
}

func (r *accountRepository) RestoreArchived(ctx context.Context, id int64, restoreGroupIDs []int64, keepCurrentGroups bool) error {
	existingGroupIDs, err := r.loadAccountGroupIDs(ctx, id)
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
		txClient = clientFromContext(ctx, r.client)
	}

	finalGroupIDs := append([]int64(nil), existingGroupIDs...)
	if !keepCurrentGroups {
		finalGroupIDs = uniqueGroupIDs(restoreGroupIDs)
		if _, err := txClient.AccountGroup.Delete().Where(dbaccountgroup.AccountIDEQ(id)).Exec(ctx); err != nil {
			return err
		}
		if len(finalGroupIDs) > 0 {
			builders := make([]*dbent.AccountGroupCreate, 0, len(finalGroupIDs))
			for index, groupID := range finalGroupIDs {
				builders = append(builders, txClient.AccountGroup.Create().SetAccountID(id).SetGroupID(groupID).SetPriority(index+1))
			}
			if _, err := txClient.AccountGroup.CreateBulk(builders...).Save(ctx); err != nil {
				return err
			}
		}
	}

	result, err := txClient.ExecContext(ctx, `
		UPDATE accounts
		SET lifecycle_state = $2,
			lifecycle_reason_code = NULL,
			lifecycle_reason_message = NULL,
			extra = (COALESCE(extra, '{}'::jsonb) - $3) - $4,
			updated_at = NOW()
		WHERE id = $1
			AND deleted_at IS NULL
			AND lifecycle_state = $5
	`, id, service.AccountLifecycleNormal, service.AccountExtraKeyArchiveRestoreGroupIDs, service.AccountExtraKeyArchiveRestoreGroups, service.AccountLifecycleArchived)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrAccountNotFound
	}

	if tx != nil {
		if err := tx.Commit(); err != nil {
			return err
		}
	}

	affectedGroupIDs := mergeGroupIDs(existingGroupIDs, finalGroupIDs)
	if !keepCurrentGroups {
		if err := enqueueSchedulerOutbox(ctx, r.sql, service.SchedulerOutboxEventAccountGroupsChanged, &id, nil, buildSchedulerGroupPayload(affectedGroupIDs)); err != nil {
			logger.LegacyPrintf("repository.account", "[SchedulerOutbox] enqueue restore archived groups failed: account=%d err=%v", id, err)
		}
	}
	if err := enqueueSchedulerOutbox(ctx, r.sql, service.SchedulerOutboxEventAccountChanged, &id, nil, buildSchedulerGroupPayload(affectedGroupIDs)); err != nil {
		logger.LegacyPrintf("repository.account", "[SchedulerOutbox] enqueue restore archived failed: account=%d err=%v", id, err)
	}
	r.syncSchedulerAccountSnapshot(ctx, id)
	return nil
}

func uniqueGroupIDs(ids []int64) []int64 {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[int64]struct{}, len(ids))
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
