package repository

import (
	"fmt"
	"strings"
	"time"

	entsql "entgo.io/ent/dialect/sql"
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
	LimitedView         string
	LimitedReason       string
	RuntimeView         string
	CandidateAccountIDs []int64
}

func normalizeAdminAccountListFilters(platform, accountType, status, search string, groupID int64, lifecycle string) adminAccountListFilters {
	return adminAccountListFilters{
		Platform:    strings.TrimSpace(platform),
		AccountType: strings.TrimSpace(accountType),
		Status:      service.NormalizeAdminAccountStatusInput(status),
		Search:      strings.TrimSpace(search),
		GroupID:     groupID,
		Lifecycle:   service.NormalizeAccountLifecycleInput(lifecycle),
	}
}

func applyAdminAccountListFilters(q *dbent.AccountQuery, filters adminAccountListFilters) *dbent.AccountQuery {
	if filters.RuntimeView == service.AccountRuntimeViewInUseOnly {
		if len(filters.CandidateAccountIDs) == 0 {
			return q.Where(dbpredicate.Account(func(s *entsql.Selector) {
				s.Where(entsql.ExprP("1 = 0"))
			}))
		}
		q = q.Where(dbaccount.IDIn(filters.CandidateAccountIDs...))
	}
	if filters.Platform != "" {
		q = q.Where(dbaccount.PlatformEQ(filters.Platform))
	}
	if filters.AccountType != "" {
		q = q.Where(dbaccount.TypeEQ(filters.AccountType))
	}
	if filters.Status != "" {
		switch filters.Status {
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
	if predicate := limitedAccountPredicate(filters); predicate != nil {
		q = q.Where(predicate)
	}
	return q
}

func appendAdminAccountFilterWhereClauses(whereClauses []string, args []any, argIndex int, filters adminAccountListFilters, tableAlias string, includePlatform bool) ([]string, []any, int) {
	if filters.RuntimeView == service.AccountRuntimeViewInUseOnly {
		if len(filters.CandidateAccountIDs) == 0 {
			whereClauses = append(whereClauses, "1 = 0")
			return whereClauses, args, argIndex
		}
		whereClauses = append(whereClauses, fmt.Sprintf("%s.id = ANY($%d)", tableAlias, argIndex))
		args = append(args, pq.Array(filters.CandidateAccountIDs))
		argIndex++
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
	if filters.Search != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("%s.name ILIKE $%d", tableAlias, argIndex))
		args = append(args, "%"+filters.Search+"%")
		argIndex++
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
