package service

import "context"

func (s *GatewayService) ResolveChannelState(ctx context.Context, group *Group, requestedModel string) (*GatewayChannelState, error) {
	if s == nil || s.channelService == nil || group == nil || group.ID <= 0 {
		return nil, nil
	}
	return s.channelService.ResolveGatewayState(ctx, group.ID, group.Platform, requestedModel)
}

func (s *OpenAIGatewayService) ResolveChannelState(ctx context.Context, group *Group, requestedModel string) (*GatewayChannelState, error) {
	if s == nil || s.channelService == nil || group == nil || group.ID <= 0 {
		return nil, nil
	}
	return s.channelService.ResolveGatewayState(ctx, group.ID, group.Platform, requestedModel)
}
