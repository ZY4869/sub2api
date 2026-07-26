package service

import (
	"context"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

var (
	ErrCompositeGroupRequired       = infraerrors.BadRequest("COMPOSITE_GROUP_REQUIRED", "composite routes can only be configured on composite groups")
	ErrCompositeTargetGroupInvalid  = infraerrors.BadRequest("COMPOSITE_TARGET_GROUP_INVALID", "composite route target group must be a non-composite group")
	ErrCompositeRouteInvalid        = infraerrors.BadRequest("COMPOSITE_ROUTE_INVALID", "invalid composite route")
	ErrCompositeRouteNotFound       = infraerrors.NotFound("COMPOSITE_ROUTE_NOT_FOUND", "composite route not found")
	ErrCompositeRouteCycleForbidden = infraerrors.BadRequest("COMPOSITE_ROUTE_CYCLE_FORBIDDEN", "composite route cannot target itself")
)

var openAIReasoningEffortRank = map[string]int{
	"none":   0,
	"low":    1,
	"medium": 2,
	"high":   3,
	"xhigh":  4,
	"max":    5,
}

func NormalizeOpenAIReasoningEffortSetting(value string) string {
	normalized := normalizeOpenAIReasoningEffortAlias(value)
	if _, ok := openAIReasoningEffortRank[normalized]; ok {
		return normalized
	}
	return ""
}

func NormalizeReasoningEffortMappings(values []ReasoningEffortMapping) []ReasoningEffortMapping {
	out := make([]ReasoningEffortMapping, 0, len(values))
	for _, item := range values {
		model := strings.TrimSpace(item.Model)
		from := NormalizeOpenAIReasoningEffortSetting(item.From)
		to := NormalizeOpenAIReasoningEffortSetting(firstNonEmptyString(item.To, item.ReasoningEffort))
		if model == "" || to == "" {
			continue
		}
		out = append(out, ReasoningEffortMapping{
			Model:           model,
			From:            from,
			To:              to,
			ReasoningEffort: to,
		})
	}
	return out
}

func ApplyGroupOpenAIReasoningPolicy(group *Group, resolution GatewayEffortResolution, modelCandidates ...string) GatewayEffortResolution {
	if group == nil {
		return resolution
	}
	policyApplies := group.Platform == PlatformOpenAI || group.Platform == PlatformComposite || group.Platform == ""
	if !policyApplies {
		return resolution
	}
	candidates := make([]string, 0, len(modelCandidates))
	for _, model := range modelCandidates {
		if strings.TrimSpace(model) != "" {
			candidates = append(candidates, strings.TrimSpace(model))
		}
	}
	effective := ""
	if resolution.Effective != nil {
		effective = NormalizeOpenAIReasoningEffortSetting(*resolution.Effective)
	}
	if effective == "" && resolution.Raw != nil {
		effective = NormalizeOpenAIReasoningEffortSetting(*resolution.Raw)
	}
	for _, mapping := range NormalizeReasoningEffortMappings(group.ReasoningEffortMappings) {
		if !anyModelMatches(mapping.Model, candidates...) {
			continue
		}
		if mapping.From != "" && effective != "" && mapping.From != effective {
			continue
		}
		next := mapping.To
		if next == "" {
			continue
		}
		effective = next
		resolution.Effective = reasoningStringPtr(next)
		if resolution.Raw == nil {
			resolution.Raw = reasoningStringPtr(next)
		}
		resolution.Source = firstNonEmptyString(resolution.Source, "group_policy")
		break
	}
	maxEffort := NormalizeOpenAIReasoningEffortSetting(group.MaxReasoningEffort)
	if maxEffort != "" && effective != "" && compareOpenAIReasoningEffort(effective, maxEffort) > 0 {
		effective = maxEffort
		resolution.Effective = reasoningStringPtr(maxEffort)
		if resolution.Raw == nil {
			resolution.Raw = reasoningStringPtr(maxEffort)
		}
		resolution.Source = firstNonEmptyString(resolution.Source, "group_policy")
	}
	return resolution
}

func OpenAIReasoningPolicyGroupFromContext(ctx context.Context) *Group {
	if ctx == nil {
		return nil
	}
	group, _ := ctx.Value(ctxkey.Group).(*Group)
	if IsGroupContextValid(group) {
		return group
	}
	return nil
}

func ApplyContextOpenAIReasoningPolicy(ctx context.Context, resolution GatewayEffortResolution, modelCandidates ...string) GatewayEffortResolution {
	return ApplyGroupOpenAIReasoningPolicy(OpenAIReasoningPolicyGroupFromContext(ctx), resolution, modelCandidates...)
}

func compareOpenAIReasoningEffort(left, right string) int {
	l := openAIReasoningEffortRank[NormalizeOpenAIReasoningEffortSetting(left)]
	r := openAIReasoningEffortRank[NormalizeOpenAIReasoningEffortSetting(right)]
	switch {
	case l < r:
		return -1
	case l > r:
		return 1
	default:
		return 0
	}
}

func normalizeOpenAIReasoningEffortAlias(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "x-high", "x_high", "extra_high", "extra-high":
		return "xhigh"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func anyModelMatches(pattern string, models ...string) bool {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return false
	}
	for _, model := range models {
		if matchModelPattern(pattern, strings.TrimSpace(model)) {
			return true
		}
	}
	return false
}

func reasoningStringPtr(value string) *string {
	return &value
}

func NormalizeCompositeModelRoutes(parentGroupID int64, routes []CompositeModelRoute) ([]CompositeModelRoute, error) {
	out := make([]CompositeModelRoute, 0, len(routes))
	seen := make(map[string]struct{}, len(routes))
	for _, route := range routes {
		displayModelID := strings.TrimSpace(route.DisplayModelID)
		if displayModelID == "" || route.TargetGroupID <= 0 {
			return nil, ErrCompositeRouteInvalid
		}
		if parentGroupID > 0 && route.TargetGroupID == parentGroupID {
			return nil, ErrCompositeRouteCycleForbidden
		}
		key := strings.ToLower(displayModelID)
		if _, exists := seen[key]; exists {
			return nil, ErrCompositeRouteInvalid
		}
		seen[key] = struct{}{}
		priority := route.Priority
		if priority <= 0 {
			priority = 50
		}
		out = append(out, CompositeModelRoute{
			ParentGroupID:  parentGroupID,
			DisplayModelID: displayModelID,
			TargetGroupID:  route.TargetGroupID,
			TargetModelID:  strings.TrimSpace(route.TargetModelID),
			Priority:       priority,
			Enabled:        route.Enabled,
			Notes:          strings.TrimSpace(route.Notes),
		})
	}
	return out, nil
}

func SelectCompositeModelRoute(routes []CompositeModelRoute, displayModelID string) *CompositeModelRoute {
	model := strings.TrimSpace(displayModelID)
	if model == "" {
		return nil
	}
	var best *CompositeModelRoute
	for i := range routes {
		route := &routes[i]
		if !route.Enabled || strings.TrimSpace(route.DisplayModelID) == "" {
			continue
		}
		if !matchModelPattern(strings.TrimSpace(route.DisplayModelID), model) {
			continue
		}
		if best == nil ||
			route.Priority < best.Priority ||
			(route.Priority == best.Priority && route.ID < best.ID) {
			best = route
		}
	}
	return best
}
