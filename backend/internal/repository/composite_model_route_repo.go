package repository

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/compositemodelroute"
	"github.com/Wei-Shaw/sub2api/ent/group"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (r *groupRepository) ListCompositeRoutes(ctx context.Context, parentGroupID int64) ([]service.CompositeModelRoute, error) {
	if parentGroupID <= 0 {
		return nil, service.ErrGroupNotFound
	}
	routes, err := r.client.CompositeModelRoute.Query().
		Where(compositemodelroute.ParentGroupIDEQ(parentGroupID)).
		WithTargetGroup().
		Order(dbent.Asc(compositemodelroute.FieldPriority), dbent.Asc(compositemodelroute.FieldID)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]service.CompositeModelRoute, 0, len(routes))
	for _, route := range routes {
		out = append(out, *compositeModelRouteEntityToService(route))
	}
	return out, nil
}

func (r *groupRepository) ReplaceCompositeRoutes(ctx context.Context, parentGroupID int64, routes []service.CompositeModelRoute) ([]service.CompositeModelRoute, error) {
	if parentGroupID <= 0 {
		return nil, service.ErrGroupNotFound
	}
	parent, err := r.client.Group.Query().Where(group.IDEQ(parentGroupID)).Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrGroupNotFound, nil)
	}
	if service.CanonicalizePlatformValue(parent.Platform) != service.PlatformComposite {
		return nil, service.ErrCompositeGroupRequired
	}

	normalized, err := service.NormalizeCompositeModelRoutes(parentGroupID, routes)
	if err != nil {
		return nil, err
	}

	tx, err := r.client.Tx(ctx)
	if err != nil && !errors.Is(err, dbent.ErrTxStarted) {
		return nil, err
	}
	exec := r.client
	if err == nil {
		defer func() { _ = tx.Rollback() }()
		exec = tx.Client()
	}

	if _, err := exec.CompositeModelRoute.Delete().
		Where(compositemodelroute.ParentGroupIDEQ(parentGroupID)).
		Exec(ctx); err != nil {
		return nil, err
	}

	creates := make([]*dbent.CompositeModelRouteCreate, 0, len(normalized))
	for i := range normalized {
		route := normalized[i]
		target, err := exec.Group.Query().Where(group.IDEQ(route.TargetGroupID)).Only(ctx)
		if err != nil {
			return nil, fmt.Errorf("target group %d not found: %w", route.TargetGroupID, translatePersistenceError(err, service.ErrGroupNotFound, nil))
		}
		if service.CanonicalizePlatformValue(target.Platform) == service.PlatformComposite {
			return nil, service.ErrCompositeTargetGroupInvalid
		}
		create := exec.CompositeModelRoute.Create().
			SetParentGroupID(parentGroupID).
			SetDisplayModelID(route.DisplayModelID).
			SetTargetGroupID(route.TargetGroupID).
			SetTargetModelID(route.TargetModelID).
			SetPriority(route.Priority).
			SetEnabled(route.Enabled).
			SetNotes(route.Notes)
		creates = append(creates, create)
	}
	if len(creates) > 0 {
		if err := exec.CompositeModelRoute.CreateBulk(creates...).Exec(ctx); err != nil {
			return nil, translatePersistenceError(err, nil, nil)
		}
	}
	if tx != nil {
		if err := tx.Commit(); err != nil {
			return nil, err
		}
	}
	if err := enqueueSchedulerOutbox(ctx, r.sql, service.SchedulerOutboxEventGroupChanged, nil, &parentGroupID, nil); err != nil {
		logger.LegacyPrintf("repository.group", "[SchedulerOutbox] enqueue composite route replace failed: group=%d err=%v", parentGroupID, err)
	}
	return r.ListCompositeRoutes(ctx, parentGroupID)
}

func (r *groupRepository) FindCompositeRoute(ctx context.Context, parentGroupID int64, displayModelID string) (*service.CompositeModelRoute, error) {
	routes, err := r.ListCompositeRoutes(ctx, parentGroupID)
	if err != nil {
		return nil, err
	}
	return service.SelectCompositeModelRoute(routes, displayModelID), nil
}

func compositeModelRouteEntityToService(route *dbent.CompositeModelRoute) *service.CompositeModelRoute {
	if route == nil {
		return nil
	}
	out := &service.CompositeModelRoute{
		ID:             route.ID,
		ParentGroupID:  route.ParentGroupID,
		DisplayModelID: strings.TrimSpace(route.DisplayModelID),
		TargetGroupID:  route.TargetGroupID,
		TargetModelID:  strings.TrimSpace(route.TargetModelID),
		Priority:       route.Priority,
		Enabled:        route.Enabled,
		Notes:          route.Notes,
		CreatedAt:      route.CreatedAt,
		UpdatedAt:      route.UpdatedAt,
	}
	if route.Edges.TargetGroup != nil {
		out.TargetGroup = groupEntityToService(route.Edges.TargetGroup)
	}
	return out
}

func reasoningEffortMappingsToDB(values []service.ReasoningEffortMapping) []map[string]string {
	normalized := service.NormalizeReasoningEffortMappings(values)
	out := make([]map[string]string, 0, len(normalized))
	for _, item := range normalized {
		out = append(out, map[string]string{
			"model":            item.Model,
			"from":             item.From,
			"to":               item.To,
			"reasoning_effort": item.ReasoningEffort,
		})
	}
	return out
}

func reasoningEffortMappingsFromDB(values []map[string]string) []service.ReasoningEffortMapping {
	out := make([]service.ReasoningEffortMapping, 0, len(values))
	for _, item := range values {
		out = append(out, service.ReasoningEffortMapping{
			Model:           item["model"],
			From:            item["from"],
			To:              item["to"],
			ReasoningEffort: item["reasoning_effort"],
		})
	}
	return service.NormalizeReasoningEffortMappings(out)
}

func sortCompositeRoutesForPreview(routes []service.CompositeModelRoute) {
	sort.SliceStable(routes, func(i, j int) bool {
		if routes[i].Priority == routes[j].Priority {
			return routes[i].ID < routes[j].ID
		}
		return routes[i].Priority < routes[j].Priority
	})
}

var _ = sortCompositeRoutesForPreview
