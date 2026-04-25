package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/model"
)

type AvailableGroupRef struct {
	ID               int64
	Name             string
	Platform         string
	SubscriptionType string
	RateMultiplier   float64
	IsExclusive      bool
}

type SupportedModelPricingInterval struct {
	MinTokens       int
	MaxTokens       *int
	TierLabel       string
	InputPrice      *float64
	OutputPrice     *float64
	CacheWritePrice *float64
	CacheReadPrice  *float64
	PerRequestPrice *float64
}

type SupportedModelPricing struct {
	BillingMode      string
	InputPrice       *float64
	OutputPrice      *float64
	CacheWritePrice  *float64
	CacheReadPrice   *float64
	ImageOutputPrice *float64
	PerRequestPrice  *float64
	Intervals        []SupportedModelPricingInterval
}

type SupportedModel struct {
	Name     string
	Platform string
	Pricing  *SupportedModelPricing
}

type AvailableChannel struct {
	ID                 int64
	Name               string
	Description        string
	Status             string
	BillingModelSource string
	RestrictModels     bool
	Groups             []AvailableGroupRef
	SupportedModels    []SupportedModel
}

func (s *ChannelService) ListAvailable(ctx context.Context) ([]AvailableChannel, error) {
	channels, err := s.repo.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("list channels: %w", err)
	}

	if s.groupRepo == nil {
		return nil, fmt.Errorf("group repository not configured")
	}
	groups, err := s.groupRepo.ListActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("list active groups: %w", err)
	}
	groupByID := make(map[int64]AvailableGroupRef, len(groups))
	for i := range groups {
		g := groups[i]
		groupByID[g.ID] = AvailableGroupRef{
			ID:               g.ID,
			Name:             g.Name,
			Platform:         g.Platform,
			SubscriptionType: g.SubscriptionType,
			RateMultiplier:   g.RateMultiplier,
			IsExclusive:      g.IsExclusive,
		}
	}

	out := make([]AvailableChannel, 0, len(channels))
	for _, ch := range channels {
		if ch == nil {
			continue
		}
		refs := make([]AvailableGroupRef, 0, len(ch.GroupIDs))
		for _, gid := range ch.GroupIDs {
			if ref, ok := groupByID[gid]; ok {
				refs = append(refs, ref)
			}
		}
		sort.SliceStable(refs, func(i, j int) bool {
			return strings.ToLower(refs[i].Name) < strings.ToLower(refs[j].Name)
		})

		supported := buildChannelSupportedModels(ch)
		s.fillGlobalPricingFallback(supported)

		out = append(out, AvailableChannel{
			ID:                 ch.ID,
			Name:               ch.Name,
			Description:        ch.Description,
			Status:             ch.Status,
			BillingModelSource: ch.BillingModelSource,
			RestrictModels:     ch.RestrictModels,
			Groups:             refs,
			SupportedModels:    supported,
		})
	}

	sort.SliceStable(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out, nil
}

func (s *ChannelService) fillGlobalPricingFallback(models []SupportedModel) {
	if s.pricingService == nil {
		return
	}
	for i := range models {
		if models[i].Pricing != nil {
			continue
		}
		lp := s.pricingService.GetModelPricing(models[i].Name)
		if lp == nil {
			continue
		}
		models[i].Pricing = synthesizeSupportedModelPricing(lp)
	}
}

func synthesizeSupportedModelPricing(lp *LiteLLMModelPricing) *SupportedModelPricing {
	if lp == nil {
		return nil
	}
	return &SupportedModelPricing{
		BillingMode:      model.ChannelBillingModeToken,
		InputPrice:       nonZeroFloatPtr(lp.InputCostPerToken),
		OutputPrice:      nonZeroFloatPtr(lp.OutputCostPerToken),
		CacheWritePrice:  nonZeroFloatPtr(lp.CacheCreationInputTokenCost),
		CacheReadPrice:   nonZeroFloatPtr(lp.CacheReadInputTokenCost),
		PerRequestPrice:  nonZeroFloatPtr(lp.OutputCostPerImage),
		ImageOutputPrice: nil,
		Intervals:        nil,
	}
}

func nonZeroFloatPtr(v float64) *float64 {
	if v == 0 {
		return nil
	}
	clone := v
	return &clone
}
