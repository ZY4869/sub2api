package service

import (
	"context"
	"regexp"
	"sort"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/model"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

var ErrChannelModelNotAllowed = infraerrors.BadRequest("CHANNEL_MODEL_NOT_ALLOWED", "requested model not allowed by channel")

type gatewayChannelStateContextKey struct{}

type GatewayChannelState struct {
	Channel        *model.Channel
	GroupID        int64
	Platform       string
	RequestedModel string
	SelectionModel string
}

type GatewayChannelUsage struct {
	TotalTokens       int
	ImageOutputTokens int
	ImageCount        int
}

type GatewayChannelResolvedPricing struct {
	PricingID        int64
	BillingMode      string
	BillingTier      string
	BillingModel     string
	InputPrice       *float64
	OutputPrice      *float64
	CacheWritePrice  *float64
	CacheReadPrice   *float64
	ImageOutputPrice *float64
	PerRequestPrice  *float64
}

func WithGatewayChannelState(ctx context.Context, state *GatewayChannelState) context.Context {
	if ctx == nil {
		return ctx
	}
	return context.WithValue(ctx, gatewayChannelStateContextKey{}, state)
}

func GatewayChannelStateFromContext(ctx context.Context) (*GatewayChannelState, bool) {
	if ctx == nil {
		return nil, false
	}
	state, ok := ctx.Value(gatewayChannelStateContextKey{}).(*GatewayChannelState)
	if !ok || state == nil {
		return nil, false
	}
	return state, true
}

func (s *ChannelService) ResolveGatewayState(ctx context.Context, groupID int64, platform, requestedModel string) (*GatewayChannelState, error) {
	if s == nil || s.repo == nil || groupID <= 0 {
		return nil, nil
	}

	channel, err := s.repo.GetActiveByGroupID(ctx, groupID)
	if err != nil || channel == nil {
		return channelStateResult(channel, groupID, platform, requestedModel), err
	}

	normalizedPlatform := normalizeChannelRuntimePlatform(platform)
	normalizedRequested := strings.TrimSpace(requestedModel)
	selectionModel := resolveChannelMappingTarget(channel, normalizedPlatform, normalizedRequested)
	if selectionModel == "" {
		selectionModel = normalizedRequested
	}

	state := &GatewayChannelState{
		Channel:        channel,
		GroupID:        groupID,
		Platform:       normalizedPlatform,
		RequestedModel: normalizedRequested,
		SelectionModel: selectionModel,
	}

	if normalizedRequested != "" && channel.RestrictModels && !channelAllowsModel(channel, normalizedPlatform, normalizedRequested, selectionModel) {
		return nil, ErrChannelModelNotAllowed
	}

	return state, nil
}

func channelStateResult(channel *model.Channel, groupID int64, platform, requestedModel string) *GatewayChannelState {
	if channel == nil {
		return nil
	}
	normalizedRequested := strings.TrimSpace(requestedModel)
	return &GatewayChannelState{
		Channel:        channel,
		GroupID:        groupID,
		Platform:       normalizeChannelRuntimePlatform(platform),
		RequestedModel: normalizedRequested,
		SelectionModel: normalizedRequested,
	}
}

func (s *GatewayChannelState) ChannelIDPtr() *int64 {
	if s == nil || s.Channel == nil || s.Channel.ID <= 0 {
		return nil
	}
	id := s.Channel.ID
	return &id
}

func (s *GatewayChannelState) ChannelName() string {
	if s == nil || s.Channel == nil {
		return ""
	}
	return strings.TrimSpace(s.Channel.Name)
}

func (s *GatewayChannelState) ResolveBillingModel(upstreamModel string) string {
	if s == nil || s.Channel == nil {
		return strings.TrimSpace(upstreamModel)
	}
	source := strings.TrimSpace(strings.ToLower(s.Channel.BillingModelSource))
	requestedModel := strings.TrimSpace(s.RequestedModel)
	selectionModel := strings.TrimSpace(s.SelectionModel)
	upstreamModel = strings.TrimSpace(upstreamModel)

	switch source {
	case model.ChannelBillingModelSourceRequested:
		if requestedModel != "" {
			return requestedModel
		}
	case model.ChannelBillingModelSourceUpstream:
		if upstreamModel != "" {
			return upstreamModel
		}
	}

	if selectionModel != "" {
		return selectionModel
	}
	if upstreamModel != "" {
		return upstreamModel
	}
	return requestedModel
}

func (s *GatewayChannelState) BuildModelMappingChain(upstreamModel string) *string {
	if s == nil {
		return nil
	}
	parts := make([]string, 0, 3)
	appendPart := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		if len(parts) > 0 && parts[len(parts)-1] == value {
			return
		}
		parts = append(parts, value)
	}

	appendPart(s.RequestedModel)
	appendPart(s.SelectionModel)
	appendPart(upstreamModel)
	if len(parts) == 0 {
		return nil
	}

	result := strings.Join(parts, " -> ")
	return &result
}

func (s *ChannelService) ResolveUsagePricing(state *GatewayChannelState, billingModel string, usage GatewayChannelUsage) *GatewayChannelResolvedPricing {
	if s == nil || state == nil || state.Channel == nil {
		return nil
	}

	billingModel = strings.TrimSpace(billingModel)
	if billingModel == "" {
		return nil
	}

	platform := normalizeChannelRuntimePlatform(state.Platform)
	for i := range state.Channel.ModelPricing {
		pricing := &state.Channel.ModelPricing[i]
		if !matchesChannelPricingPlatform(pricing.Platform, platform) {
			continue
		}
		if !matchAnyChannelPattern(pricing.Models, billingModel) {
			continue
		}

		resolved := &GatewayChannelResolvedPricing{
			PricingID:        pricing.ID,
			BillingMode:      pricing.BillingMode,
			BillingModel:     billingModel,
			InputPrice:       cloneNullableFloat(pricing.InputPrice),
			OutputPrice:      cloneNullableFloat(pricing.OutputPrice),
			CacheWritePrice:  cloneNullableFloat(pricing.CacheWritePrice),
			CacheReadPrice:   cloneNullableFloat(pricing.CacheReadPrice),
			ImageOutputPrice: cloneNullableFloat(pricing.ImageOutputPrice),
			PerRequestPrice:  cloneNullableFloat(pricing.PerRequestPrice),
		}

		tokenBasis := usage.TotalTokens
		if resolved.BillingMode == model.ChannelBillingModeImage {
			tokenBasis = usage.ImageOutputTokens
			if tokenBasis <= 0 {
				tokenBasis = usage.ImageCount
			}
		}

		if interval := matchChannelPricingInterval(pricing.Intervals, tokenBasis); interval != nil {
			if strings.TrimSpace(interval.TierLabel) != "" {
				resolved.BillingTier = strings.TrimSpace(interval.TierLabel)
			}
			if interval.InputPrice != nil {
				resolved.InputPrice = cloneNullableFloat(interval.InputPrice)
			}
			if interval.OutputPrice != nil {
				resolved.OutputPrice = cloneNullableFloat(interval.OutputPrice)
			}
			if interval.CacheWritePrice != nil {
				resolved.CacheWritePrice = cloneNullableFloat(interval.CacheWritePrice)
			}
			if interval.CacheReadPrice != nil {
				resolved.CacheReadPrice = cloneNullableFloat(interval.CacheReadPrice)
			}
			if interval.PerRequestPrice != nil {
				resolved.PerRequestPrice = cloneNullableFloat(interval.PerRequestPrice)
			}
		}

		return resolved
	}

	return nil
}

func normalizeChannelRuntimePlatform(platform string) string {
	return strings.TrimSpace(strings.ToLower(platform))
}

func resolveChannelMappingTarget(channel *model.Channel, platform, requestedModel string) string {
	if channel == nil {
		return ""
	}
	requestedModel = strings.TrimSpace(requestedModel)
	if requestedModel == "" {
		return ""
	}

	for _, key := range []string{platform, "*"} {
		mapping := channel.ModelMapping[key]
		target := resolveChannelMappingFromMap(mapping, requestedModel)
		if target != "" {
			return target
		}
	}

	return ""
}

func resolveChannelMappingFromMap(mapping map[string]string, requestedModel string) string {
	if len(mapping) == 0 {
		return ""
	}
	if target := strings.TrimSpace(mapping[requestedModel]); target != "" {
		return target
	}

	patterns := make([]string, 0, len(mapping))
	for pattern := range mapping {
		if strings.Contains(pattern, "*") {
			patterns = append(patterns, pattern)
		}
	}
	sort.SliceStable(patterns, func(i, j int) bool {
		return channelPatternSpecificity(patterns[i]) > channelPatternSpecificity(patterns[j])
	})
	for _, pattern := range patterns {
		if !matchChannelPattern(pattern, requestedModel) {
			continue
		}
		if target := strings.TrimSpace(mapping[pattern]); target != "" {
			return target
		}
	}
	return ""
}

func channelAllowsModel(channel *model.Channel, platform, requestedModel, selectionModel string) bool {
	if channel == nil {
		return true
	}
	if resolveChannelMappingTarget(channel, platform, requestedModel) != "" {
		return true
	}

	candidates := []string{strings.TrimSpace(selectionModel), strings.TrimSpace(requestedModel)}
	for i := range channel.ModelPricing {
		pricing := channel.ModelPricing[i]
		if !matchesChannelPricingPlatform(pricing.Platform, platform) {
			continue
		}
		for _, candidate := range candidates {
			if candidate == "" {
				continue
			}
			if matchAnyChannelPattern(pricing.Models, candidate) {
				return true
			}
		}
	}

	return false
}

func matchesChannelPricingPlatform(entryPlatform, platform string) bool {
	entryPlatform = normalizeChannelRuntimePlatform(entryPlatform)
	platform = normalizeChannelRuntimePlatform(platform)
	return entryPlatform == "" || entryPlatform == "*" || entryPlatform == platform
}

func matchAnyChannelPattern(patterns []string, value string) bool {
	for _, pattern := range patterns {
		if matchChannelPattern(pattern, value) {
			return true
		}
	}
	return false
}

func matchChannelPricingInterval(intervals []model.ChannelPricingInterval, tokenCount int) *model.ChannelPricingInterval {
	for i := range intervals {
		interval := &intervals[i]
		if tokenCount < int(interval.MinTokens) {
			continue
		}
		if interval.MaxTokens != nil && tokenCount >= int(*interval.MaxTokens) {
			continue
		}
		return interval
	}
	return nil
}

func cloneNullableFloat(value *float64) *float64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func channelPatternSpecificity(pattern string) int {
	return len(strings.ReplaceAll(pattern, "*", ""))
}

func matchChannelPattern(pattern, value string) bool {
	pattern = strings.TrimSpace(pattern)
	value = strings.TrimSpace(value)
	if pattern == "" || value == "" {
		return false
	}
	if pattern == "*" {
		return true
	}
	if !strings.Contains(pattern, "*") {
		return strings.EqualFold(pattern, value)
	}

	quoted := regexp.QuoteMeta(pattern)
	quoted = strings.ReplaceAll(quoted, "\\*", ".*")
	matched, err := regexp.MatchString("(?i)^"+quoted+"$", value)
	return err == nil && matched
}
