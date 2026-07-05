package repository

import (
	"context"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	dbgroup "github.com/Wei-Shaw/sub2api/ent/group"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

func (r *groupRepository) ListActiveIDs(ctx context.Context) ([]int64, error) {
	if r == nil {
		return []int64{}, nil
	}
	if r.sql != nil {
		rows, err := r.sql.QueryContext(ctx, `
			SELECT id
			FROM groups
			WHERE status = $1
				AND deleted_at IS NULL
				AND platform <> ALL($2)
			ORDER BY sort_order ASC, id ASC
		`, service.StatusActive, pq.Array(service.UnsupportedPrimaryAccountPredicateValues()))
		if err != nil {
			return nil, err
		}
		defer func() { _ = rows.Close() }()

		ids := make([]int64, 0)
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err != nil {
				return nil, err
			}
			ids = append(ids, id)
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return ids, nil
	}

	groups, err := r.client.Group.Query().
		Where(
			dbgroup.StatusEQ(service.StatusActive),
			dbgroup.PlatformNotIn(service.UnsupportedPrimaryAccountPredicateValues()...),
		).
		Select(dbgroup.FieldID).
		Order(dbent.Asc(dbgroup.FieldSortOrder), dbent.Asc(dbgroup.FieldID)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(groups))
	for i := range groups {
		ids = append(ids, groups[i].ID)
	}
	return ids, nil
}
