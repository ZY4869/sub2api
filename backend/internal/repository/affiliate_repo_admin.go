package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

func (r *affiliateRepository) ListAffiliateUsers(ctx context.Context, params pagination.PaginationParams, filters service.AffiliateAdminUserListFilters) ([]service.AffiliateAdminUser, *pagination.PaginationResult, error) {
	if r == nil || r.db == nil {
		return nil, nil, errors.New("affiliate repository db is nil")
	}
	exec := affiliateSQLExecutorFromContext(ctx, r.db)
	if exec == nil {
		return nil, nil, errors.New("affiliate repository sql executor is nil")
	}

	whereSQL, args := buildAffiliateAdminListWhere(filters)
	countSQL := "SELECT COUNT(*) FROM user_affiliates ua JOIN users u ON u.id = ua.user_id " + whereSQL

	var total int64
	if err := exec.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, nil, err
	}

	queryArgs := append([]any{}, args...)
	queryArgs = append(queryArgs, params.Limit(), params.Offset())

	rows, err := exec.QueryContext(ctx, `
SELECT
	ua.user_id,
	u.email,
	ua.aff_code,
	ua.custom_aff_code,
	ua.custom_rebate_rate_percent,
	ua.inviter_user_id,
	ua.invitee_count,
	ua.rebate_balance,
	ua.rebate_frozen_balance,
	ua.lifetime_rebate,
	ua.updated_at
FROM user_affiliates ua
JOIN users u ON u.id = ua.user_id
`+whereSQL+`
ORDER BY ua.updated_at DESC, ua.user_id DESC
LIMIT $`+fmt.Sprint(len(queryArgs)-1)+` OFFSET $`+fmt.Sprint(len(queryArgs)),
		queryArgs...,
	)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = rows.Close() }()

	items := make([]service.AffiliateAdminUser, 0, params.Limit())
	for rows.Next() {
		var inviterUserID sql.NullInt64
		var customRate sql.NullFloat64
		row := service.AffiliateAdminUser{}
		if err := rows.Scan(
			&row.UserID,
			&row.Email,
			&row.AffCode,
			&row.CustomAffCode,
			&customRate,
			&inviterUserID,
			&row.InviteeCount,
			&row.RebateBalance,
			&row.RebateFrozenBalance,
			&row.LifetimeRebate,
			&row.UpdatedAt,
		); err != nil {
			return nil, nil, err
		}
		if inviterUserID.Valid {
			row.InviterUserID = &inviterUserID.Int64
		}
		if customRate.Valid {
			row.CustomRebateRatePercent = &customRate.Float64
		}
		items = append(items, row)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	return items, paginationResultFromTotal(total, params), nil
}

func (r *affiliateRepository) LookupAffiliateUsers(ctx context.Context, q string, limit int) ([]service.AffiliateAdminUser, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("affiliate repository db is nil")
	}
	exec := affiliateSQLExecutorFromContext(ctx, r.db)
	if exec == nil {
		return nil, errors.New("affiliate repository sql executor is nil")
	}

	q = strings.TrimSpace(q)
	if q == "" {
		return nil, nil
	}
	if runes := []rune(q); len(runes) > 100 {
		q = string(runes[:100])
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	pattern := "%" + q + "%"
	conditions := []string{"u.deleted_at IS NULL"}
	args := []any{pattern}

	intID, intErr := strconv.ParseInt(q, 10, 64)
	if intErr == nil && intID > 0 {
		conditions = append(conditions, fmt.Sprintf("(u.email ILIKE $1 OR ua.aff_code ILIKE $1 OR ua.user_id = $%d)", len(args)+1))
		args = append(args, intID)
	} else {
		conditions = append(conditions, "(u.email ILIKE $1 OR ua.aff_code ILIKE $1)")
	}

	args = append(args, limit)

	rows, err := exec.QueryContext(ctx, `
SELECT
	ua.user_id,
	u.email,
	ua.aff_code,
	ua.custom_aff_code,
	ua.custom_rebate_rate_percent,
	ua.inviter_user_id,
	ua.invitee_count,
	ua.rebate_balance,
	ua.rebate_frozen_balance,
	ua.lifetime_rebate,
	ua.updated_at
FROM user_affiliates ua
JOIN users u ON u.id = ua.user_id
WHERE `+strings.Join(conditions, " AND ")+`
ORDER BY ua.updated_at DESC, ua.user_id DESC
LIMIT $`+fmt.Sprint(len(args)),
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	items := make([]service.AffiliateAdminUser, 0, limit)
	for rows.Next() {
		var inviterUserID sql.NullInt64
		var customRate sql.NullFloat64
		row := service.AffiliateAdminUser{}
		if err := rows.Scan(
			&row.UserID,
			&row.Email,
			&row.AffCode,
			&row.CustomAffCode,
			&customRate,
			&inviterUserID,
			&row.InviteeCount,
			&row.RebateBalance,
			&row.RebateFrozenBalance,
			&row.LifetimeRebate,
			&row.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if inviterUserID.Valid {
			row.InviterUserID = &inviterUserID.Int64
		}
		if customRate.Valid {
			row.CustomRebateRatePercent = &customRate.Float64
		}
		items = append(items, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *affiliateRepository) UpdateAffiliateUserCustom(ctx context.Context, userID int64, update service.AffiliateAdminUserCustomUpdate, newAffCodeForClear string) (*service.UserAffiliate, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("affiliate repository db is nil")
	}
	exec := affiliateSQLExecutorFromContext(ctx, r.db)
	if exec == nil {
		return nil, errors.New("affiliate repository sql executor is nil")
	}
	txExec, commit, rollback, err := beginAffiliateSQLTx(ctx, exec)
	if err != nil {
		return nil, err
	}
	defer rollback()

	var locked int
	if err := txExec.QueryRowContext(ctx, `
		SELECT 1
		FROM user_affiliates
		WHERE user_id = $1
		FOR UPDATE
	`, userID).Scan(&locked); err != nil {
		return nil, err
	}

	if update.AffCodeSet {
		if update.AffCode == nil || strings.TrimSpace(*update.AffCode) == "" {
			if strings.TrimSpace(newAffCodeForClear) == "" {
				return nil, errors.New("new affiliate code required for clear")
			}
			if _, err := txExec.ExecContext(ctx, `
				UPDATE user_affiliates
				SET aff_code = $2,
					custom_aff_code = false,
					updated_at = NOW()
				WHERE user_id = $1
			`, userID, strings.TrimSpace(strings.ToUpper(newAffCodeForClear))); err != nil {
				return nil, err
			}
		} else {
			code := strings.TrimSpace(strings.ToUpper(*update.AffCode))
			code = strings.ReplaceAll(code, "-", "")
			code = strings.ReplaceAll(code, " ", "")
			if _, err := txExec.ExecContext(ctx, `
				UPDATE user_affiliates
				SET aff_code = $2,
					custom_aff_code = true,
					updated_at = NOW()
				WHERE user_id = $1
			`, userID, code); err != nil {
				return nil, err
			}
		}
	}

	if update.CustomRateSet {
		if update.CustomRate == nil {
			if _, err := txExec.ExecContext(ctx, `
				UPDATE user_affiliates
				SET custom_rebate_rate_percent = NULL,
					updated_at = NOW()
				WHERE user_id = $1
			`, userID); err != nil {
				return nil, err
			}
		} else {
			if _, err := txExec.ExecContext(ctx, `
				UPDATE user_affiliates
				SET custom_rebate_rate_percent = $2,
					updated_at = NOW()
				WHERE user_id = $1
			`, userID, *update.CustomRate); err != nil {
				return nil, err
			}
		}
	}

	row, err := getUserAffiliateWithExec(ctx, txExec, userID)
	if err != nil {
		return nil, err
	}

	if err := commit(); err != nil {
		return nil, err
	}
	return row, nil
}

func (r *affiliateRepository) ResetAffiliateUserCustom(ctx context.Context, userID int64, newAffCode string) (*service.UserAffiliate, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("affiliate repository db is nil")
	}
	exec := affiliateSQLExecutorFromContext(ctx, r.db)
	if exec == nil {
		return nil, errors.New("affiliate repository sql executor is nil")
	}
	txExec, commit, rollback, err := beginAffiliateSQLTx(ctx, exec)
	if err != nil {
		return nil, err
	}
	defer rollback()

	if strings.TrimSpace(newAffCode) == "" {
		return nil, errors.New("new affiliate code is required")
	}
	newAffCode = strings.TrimSpace(strings.ToUpper(newAffCode))

	if _, err := txExec.ExecContext(ctx, `
		UPDATE user_affiliates
		SET aff_code = $2,
			custom_aff_code = false,
			custom_rebate_rate_percent = NULL,
			updated_at = NOW()
		WHERE user_id = $1
	`, userID, newAffCode); err != nil {
		return nil, err
	}

	row, err := getUserAffiliateWithExec(ctx, txExec, userID)
	if err != nil {
		return nil, err
	}
	if err := commit(); err != nil {
		return nil, err
	}
	return row, nil
}

func (r *affiliateRepository) BatchUpdateAffiliateUserCustomRates(ctx context.Context, userIDs []int64, customRatePercent float64) (updated int, err error) {
	if r == nil || r.db == nil {
		return 0, errors.New("affiliate repository db is nil")
	}
	if len(userIDs) == 0 {
		return 0, nil
	}
	exec := affiliateSQLExecutorFromContext(ctx, r.db)
	if exec == nil {
		return 0, errors.New("affiliate repository sql executor is nil")
	}

	res, err := exec.ExecContext(ctx, `
		UPDATE user_affiliates
		SET custom_rebate_rate_percent = $1,
			updated_at = NOW()
		WHERE user_id = ANY($2)
	`, customRatePercent, pq.Array(userIDs))
	if err != nil {
		return 0, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	return int(affected), nil
}

func buildAffiliateAdminListWhere(filters service.AffiliateAdminUserListFilters) (string, []any) {
	conditions := []string{"u.deleted_at IS NULL"}
	args := make([]any, 0, 3)

	if filters.HasCustomCode != nil {
		conditions = append(conditions, fmt.Sprintf("ua.custom_aff_code = $%d", len(args)+1))
		args = append(args, *filters.HasCustomCode)
	}
	if filters.HasCustomRate != nil {
		if *filters.HasCustomRate {
			conditions = append(conditions, "ua.custom_rebate_rate_percent IS NOT NULL")
		} else {
			conditions = append(conditions, "ua.custom_rebate_rate_percent IS NULL")
		}
	}
	if filters.HasInviter != nil {
		if *filters.HasInviter {
			conditions = append(conditions, "ua.inviter_user_id IS NOT NULL")
		} else {
			conditions = append(conditions, "ua.inviter_user_id IS NULL")
		}
	}

	if len(conditions) == 0 {
		return "", args
	}
	return "WHERE " + strings.Join(conditions, " AND "), args
}
