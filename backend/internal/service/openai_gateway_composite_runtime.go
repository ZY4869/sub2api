package service

import (
	"context"
	"strings"
)

type CompositeRouteRuntimeResolution struct {
	Matched          bool
	APIKey           *APIKey
	Subscription     *UserSubscription
	DisplayModelID   string
	RuntimeModelID   string
	TargetGroup      *Group
	TargetGroupID    int64
	Route            *CompositeModelRoute
	ParentGroupID    int64
	ParentGroupName  string
	TargetModelIDSet bool
}

func (s *OpenAIGatewayService) ResolveCompositeRouteRuntime(
	ctx context.Context,
	apiKey *APIKey,
	subscription *UserSubscription,
	displayModelID string,
) (*CompositeRouteRuntimeResolution, error) {
	if s == nil || s.groupRepo == nil || apiKey == nil || apiKey.Group == nil {
		return &CompositeRouteRuntimeResolution{
			APIKey:         apiKey,
			Subscription:   subscription,
			DisplayModelID: strings.TrimSpace(displayModelID),
			RuntimeModelID: strings.TrimSpace(displayModelID),
		}, nil
	}
	parent := apiKey.Group
	displayModelID = strings.TrimSpace(displayModelID)
	if CanonicalizePlatformValue(parent.Platform) != PlatformComposite {
		return &CompositeRouteRuntimeResolution{
			APIKey:         apiKey,
			Subscription:   subscription,
			DisplayModelID: displayModelID,
			RuntimeModelID: displayModelID,
		}, nil
	}
	route, err := s.groupRepo.FindCompositeRoute(ctx, parent.ID, displayModelID)
	if err != nil {
		return nil, err
	}
	if route == nil {
		return nil, ErrCompositeRouteNotFound
	}
	targetGroup := route.TargetGroup
	if targetGroup == nil || targetGroup.ID <= 0 || !targetGroup.Hydrated {
		targetGroup, err = s.groupRepo.GetByIDLite(ctx, route.TargetGroupID)
		if err != nil {
			return nil, err
		}
	}
	if targetGroup == nil || targetGroup.ID <= 0 || CanonicalizePlatformValue(targetGroup.Platform) == PlatformComposite {
		return nil, ErrCompositeTargetGroupInvalid
	}
	targetModelID := strings.TrimSpace(route.TargetModelID)
	runtimeModelID := displayModelID
	targetModelIDSet := targetModelID != ""
	if targetModelIDSet {
		runtimeModelID = targetModelID
	}
	if runtimeModelID == "" {
		runtimeModelID = displayModelID
	}
	cloned := *apiKey
	targetGroupCopy := *targetGroup
	cloned.GroupID = &targetGroupCopy.ID
	cloned.Group = &targetGroupCopy
	cloned.SelectedGroupBinding = nil
	for i := range cloned.GroupBindings {
		if cloned.GroupBindings[i].GroupID != targetGroupCopy.ID {
			continue
		}
		cloned.GroupBindings[i].Group = &targetGroupCopy
		cloned.SelectedGroupBinding = &cloned.GroupBindings[i]
		break
	}
	return &CompositeRouteRuntimeResolution{
		Matched:          true,
		APIKey:           &cloned,
		Subscription:     subscription,
		DisplayModelID:   displayModelID,
		RuntimeModelID:   runtimeModelID,
		TargetGroup:      &targetGroupCopy,
		TargetGroupID:    targetGroupCopy.ID,
		Route:            route,
		ParentGroupID:    parent.ID,
		ParentGroupName:  parent.Name,
		TargetModelIDSet: targetModelIDSet,
	}, nil
}
