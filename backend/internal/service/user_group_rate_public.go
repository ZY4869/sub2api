package service

import "context"

// ResolveUserGroupRateMultiplier returns the same user/group multiplier used by
// the gateway billing path. It deliberately falls back to the group default on
// lookup failures, matching runtime billing behavior.
func (s *GatewayService) ResolveUserGroupRateMultiplier(ctx context.Context, userID, groupID int64, groupDefaultMultiplier float64) float64 {
	return s.getUserGroupRateMultiplier(ctx, userID, groupID, groupDefaultMultiplier)
}

// ResolveUserGroupRateMultiplier returns the OpenAI gateway billing multiplier.
func (s *OpenAIGatewayService) ResolveUserGroupRateMultiplier(ctx context.Context, userID, groupID int64, groupDefaultMultiplier float64) float64 {
	if s == nil || s.userGroupRateResolver == nil {
		return groupDefaultMultiplier
	}
	return s.userGroupRateResolver.Resolve(ctx, userID, groupID, groupDefaultMultiplier)
}
