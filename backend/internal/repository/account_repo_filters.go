package repository

import (
	"fmt"
	"strings"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqljson"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	dbaccount "github.com/Wei-Shaw/sub2api/ent/account"
	dbaccountgroup "github.com/Wei-Shaw/sub2api/ent/accountgroup"
	dbpredicate "github.com/Wei-Shaw/sub2api/ent/predicate"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type adminAccountListFilters struct {
	Platform            string
	AccountType         string
	Status              string
	Search              string
	GroupID             int64
	Lifecycle           string
	PrivacyMode         string
	LimitedView         string
	LimitedReason       string
	RuntimeView         string
	CandidateAccountIDs []int64
}

func normalizeAdminAccountListFilters(platform, accountType, status, search string, groupID int64, lifecycle string, privacyMode string) adminAccountListFilters {
	return adminAccountListFilters{
		Platform:    strings.TrimSpace(platform),
		AccountType: strings.TrimSpace(accountType),
		Status:      service.NormalizeAdminAccountStatusInput(status),
		Search:      strings.TrimSpace(search),
		GroupID:     groupID,
		Lifecycle:   service.NormalizeAccountLifecycleInput(lifecycle),
		PrivacyMode: strings.TrimSpace(privacyMode),
	}
}

func applyAdminAccountListFilters(q *dbent.AccountQuery, filters adminAccountListFilters) *dbent.AccountQuery {
	switch filters.RuntimeView {
	case service.AccountRuntimeViewInUseOnly:
		if len(filters.CandidateAccountIDs) == 0 {
			return q.Where(dbpredicate.Account(func(s *entsql.Selector) {
				s.Where(entsql.ExprP("1 = 0"))
			}))
		}
		q = q.Where(dbaccount.IDIn(filters.CandidateAccountIDs...))
	case service.AccountRuntimeViewAvailableOnly:
		q = q.Where(dispatchableAccountPredicate())
		if len(filters.CandidateAccountIDs) > 0 {
			q = q.Where(dbaccount.IDNotIn(filters.CandidateAccountIDs...))
		}
	}
	if filters.Platform != "" {
		q = q.Where(dbaccount.PlatformEQ(filters.Platform))
	}
	if filters.AccountType != "" {
		q = q.Where(dbaccount.TypeEQ(filters.AccountType))
	}
	if filters.Status != "" {
		switch filters.Status {
		case service.StatusActive:
			q = q.Where(
				dbaccount.StatusEQ(filters.Status),
				dbaccount.Or(
					dbaccount.RateLimitResetAtIsNil(),
					dbaccount.RateLimitResetAtLTE(time.Now()),
				),
			)
		case "rate_limited":
			q = q.Where(dbaccount.RateLimitResetAtGT(time.Now()))
		case "temp_unschedulable":
			q = q.Where(dbpredicate.Account(func(s *entsql.Selector) {
				col := s.C("temp_unschedulable_until")
				s.Where(entsql.And(entsql.Not(entsql.IsNull(col)), entsql.GT(col, entsql.Expr("NOW()"))))
			}))
		case "paused":
			q = q.Where(dbaccount.SchedulableEQ(false))
		default:
			q = q.Where(dbaccount.StatusEQ(filters.Status))
		}
	}
	if filters.Search != "" {
		q = q.Where(dbaccount.NameContainsFold(filters.Search))
	}
	if filters.GroupID == service.AccountListGroupUngrouped {
		q = q.Where(dbaccount.Not(dbaccount.HasAccountGroups()))
	} else if filters.GroupID > 0 {
		q = q.Where(dbaccount.HasAccountGroupsWith(dbaccountgroup.GroupIDEQ(filters.GroupID)))
	}
	if predicate := lifecyclePredicate(filters.Lifecycle); predicate != nil {
		q = q.Where(predicate)
	}
	if filters.PrivacyMode != "" {
		switch filters.PrivacyMode {
		case "unset":
			q = q.Where(dbpredicate.Account(func(s *entsql.Selector) {
				s.Where(entsql.Or(
					entsql.IsNull(s.C("extra")),
					entsql.Not(entsql.ExprP("COALESCE(extra, '{}'::jsonb) ? 'privacy_mode'")),
				))
			}))
		default:
			q = q.Where(dbpredicate.Account(func(s *entsql.Selector) {
				s.Where(sqljson.ValueEQ(dbaccount.FieldExtra, filters.PrivacyMode, sqljson.Path("privacy_mode"), sqljson.Unquote(true)))
			}))
		}
	}
	if predicate := limitedAccountPredicate(filters); predicate != nil {
		q = q.Where(predicate)
	}
	return q
}

func appendAdminAccountFilterWhereClauses(whereClauses []string, args []any, argIndex int, filters adminAccountListFilters, tableAlias string, includePlatform bool) ([]string, []any, int) {
	switch filters.RuntimeView {
	case service.AccountRuntimeViewInUseOnly:
		if len(filters.CandidateAccountIDs) == 0 {
			whereClauses = append(whereClauses, "1 = 0")
			return whereClauses, args, argIndex
		}
		whereClauses = append(whereClauses, fmt.Sprintf("%s.id = ANY($%d)", tableAlias, argIndex))
		args = append(args, pq.Array(filters.CandidateAccountIDs))
		argIndex++
	case service.AccountRuntimeViewAvailableOnly:
		whereClauses = appendDispatchableAccountWhereClauses(whereClauses, tableAlias)
		if len(filters.CandidateAccountIDs) > 0 {
			whereClauses = append(whereClauses, fmt.Sprintf("NOT (%s.id = ANY($%d))", tableAlias, argIndex))
			args = append(args, pq.Array(filters.CandidateAccountIDs))
			argIndex++
		}
	}
	if includePlatform && filters.Platform != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("%s.platform = $%d", tableAlias, argIndex))
		args = append(args, filters.Platform)
		argIndex++
	}
	if filters.AccountType != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("%s.type = $%d", tableAlias, argIndex))
		args = append(args, filters.AccountType)
		argIndex++
	}
	if filters.Status == service.StatusActive {
		whereClauses = append(whereClauses,
			fmt.Sprintf("%s.status = $%d", tableAlias, argIndex),
			fmt.Sprintf("(%s.rate_limit_reset_at IS NULL OR %s.rate_limit_reset_at <= NOW())", tableAlias, tableAlias),
		)
		args = append(args, filters.Status)
		argIndex++
	}
	if filters.Search != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("%s.name ILIKE $%d", tableAlias, argIndex))
		args = append(args, "%"+filters.Search+"%")
		argIndex++
	}
	if filters.PrivacyMode != "" {
		if filters.PrivacyMode == "unset" {
			whereClauses = append(whereClauses, fmt.Sprintf("NOT (COALESCE(%s.extra, '{}'::jsonb) ? 'privacy_mode')", tableAlias))
		} else {
			whereClauses = append(whereClauses, fmt.Sprintf("COALESCE(%s.extra, '{}'::jsonb) ->> 'privacy_mode' = $%d", tableAlias, argIndex))
			args = append(args, filters.PrivacyMode)
			argIndex++
		}
	}
	if filters.GroupID == service.AccountListGroupUngrouped {
		whereClauses = append(whereClauses, fmt.Sprintf("NOT EXISTS (SELECT 1 FROM account_groups agf WHERE agf.account_id = %s.id)", tableAlias))
	} else if filters.GroupID > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("EXISTS (SELECT 1 FROM account_groups agf WHERE agf.account_id = %s.id AND agf.group_id = $%d)", tableAlias, argIndex))
		args = append(args, filters.GroupID)
		argIndex++
	}
	if filters.Lifecycle != "" && filters.Lifecycle != service.AccountLifecycleAll {
		whereClauses = append(whereClauses, fmt.Sprintf("%s.lifecycle_state = $%d", tableAlias, argIndex))
		args = append(args, filters.Lifecycle)
		argIndex++
	}
	whereClauses, args, argIndex = appendAdminLimitedWhereClauses(whereClauses, args, argIndex, filters, tableAlias)
	return whereClauses, args, argIndex
}

func dispatchableAccountPredicate() dbpredicate.Account {
	return dbpredicate.Account(func(s *entsql.Selector) {
		rateLimitResetAt := s.C("rate_limit_reset_at")
		tempUnschedulableUntil := s.C("temp_unschedulable_until")
		overloadUntil := s.C("overload_until")
		s.Where(entsql.And(
			entsql.EQ(s.C("status"), service.StatusActive),
			entsql.EQ(s.C("schedulable"), true),
			entsql.Or(entsql.IsNull(rateLimitResetAt), entsql.LTE(rateLimitResetAt, entsql.Expr("NOW()"))),
			entsql.Or(entsql.IsNull(tempUnschedulableUntil), entsql.LTE(tempUnschedulableUntil, entsql.Expr("NOW()"))),
			entsql.Or(entsql.IsNull(overloadUntil), entsql.LTE(overloadUntil, entsql.Expr("NOW()"))),
		))
	})
}

func appendDispatchableAccountWhereClauses(whereClauses []string, tableAlias string) []string {
	whereClauses = append(whereClauses,
		fmt.Sprintf("%s.status = '%s'", tableAlias, service.StatusActive),
		fmt.Sprintf("%s.schedulable = TRUE", tableAlias),
		fmt.Sprintf("(%s.rate_limit_reset_at IS NULL OR %s.rate_limit_reset_at <= NOW())", tableAlias, tableAlias),
		fmt.Sprintf("(%s.temp_unschedulable_until IS NULL OR %s.temp_unschedulable_until <= NOW())", tableAlias, tableAlias),
		fmt.Sprintf("(%s.overload_until IS NULL OR %s.overload_until <= NOW())", tableAlias, tableAlias),
	)
	return whereClauses
}
