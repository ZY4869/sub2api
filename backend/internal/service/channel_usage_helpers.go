package service

import (
	"context"
	"strings"
)

type gatewayChannelBillingResolution struct {
	State             *GatewayChannelState
	BillingModel      string
	ModelMappingChain *string
	Pricing           *GatewayChannelResolvedPricing
}

func ResolveGatewaySelectionModelFromContext(ctx context.Context, requestedModel string) string {
	model, _ := ResolveGatewaySelectionModelWithState(ctx, requestedModel)
	return model
}

func ResolveGatewaySelectionModelWithState(ctx context.Context, requestedModel string) (string, *GatewayChannelState) {
	requestedModel = strings.TrimSpace(requestedModel)
	state, ok := GatewayChannelStateFromContext(ctx)
	if !ok || state == nil {
		return requestedModel, nil
	}
	if selectionModel := strings.TrimSpace(state.SelectionModel); selectionModel != "" {
		return selectionModel, state
	}
	return requestedModel, state
}

func resolveGatewayChannelBilling(
	ctx context.Context,
	channelService *ChannelService,
	requestedModel string,
	upstreamModel string,
	usage GatewayChannelUsage,
) *gatewayChannelBillingResolution {
	_, state := ResolveGatewaySelectionModelWithState(ctx, requestedModel)
	if state == nil {
		return nil
	}

	resolution := &gatewayChannelBillingResolution{
		State:             state,
		ModelMappingChain: state.BuildModelMappingChain(upstreamModel),
	}
	resolution.BillingModel = strings.TrimSpace(state.ResolveBillingModel(upstreamModel))
	if resolution.BillingModel == "" {
		resolution.BillingModel = strings.TrimSpace(requestedModel)
	}
	if resolution.BillingModel == "" {
		resolution.BillingModel = strings.TrimSpace(upstreamModel)
	}
	if channelService != nil && resolution.BillingModel != "" {
		resolution.Pricing = channelService.ResolveUsagePricing(state, resolution.BillingModel, usage)
	}
	return resolution
}

func applyGatewayChannelUsageLogMetadata(
	usageLog *UsageLog,
	resolution *gatewayChannelBillingResolution,
	imageOutputTokens *int,
	imageOutputCost *float64,
) {
	if usageLog == nil || resolution == nil || resolution.State == nil {
		return
	}

	usageLog.ChannelID = resolution.State.ChannelIDPtr()
	usageLog.ModelMappingChain = resolution.ModelMappingChain
	if resolution.Pricing != nil {
		usageLog.BillingTier = optionalTrimmedStringPtr(resolution.Pricing.BillingTier)
		usageLog.BillingMode = optionalTrimmedStringPtr(resolution.Pricing.BillingMode)
	}
	usageLog.ImageOutputTokens = imageOutputTokens
	usageLog.ImageOutputCost = imageOutputCost
}
