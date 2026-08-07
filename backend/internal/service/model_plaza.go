package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

type ModelPlazaResponse struct {
	Description string            `json:"description"`
	Groups      []ModelPlazaGroup `json:"groups"`
}

type ModelPlazaGroup struct {
	ID                    int64             `json:"id"`
	Name                  string            `json:"name"`
	Description           string            `json:"description"`
	Platform              string            `json:"platform"`
	SubscriptionType      string            `json:"subscription_type"`
	RateMultiplier        float64           `json:"rate_multiplier"`
	PeakRateEnabled       bool              `json:"peak_rate_enabled"`
	PeakStart             string            `json:"peak_start"`
	PeakEnd               string            `json:"peak_end"`
	PeakRateMultiplier    float64           `json:"peak_rate_multiplier"`
	IsExclusive           bool              `json:"is_exclusive"`
	ImageRateIndependent  bool              `json:"image_rate_independent"`
	ImageRateMultiplier   float64           `json:"image_rate_multiplier"`
	ImagePrice1K          *float64          `json:"image_price_1k,omitempty"`
	ImagePrice2K          *float64          `json:"image_price_2k,omitempty"`
	ImagePrice4K          *float64          `json:"image_price_4k,omitempty"`
	WebSearchPricePerCall *float64          `json:"web_search_price_per_call,omitempty"`
	ProfitControlEnabled  bool              `json:"profit_control_enabled"`
	ProfitMinMargin       float64           `json:"profit_min_margin"`
	ProfitSafetyBuffer    float64           `json:"profit_safety_buffer"`
	Models                []ModelPlazaModel `json:"models"`
}

type ModelPlazaModel struct {
	Name            string                 `json:"name"`
	Platform        string                 `json:"platform"`
	Pricing         *SupportedModelPricing `json:"pricing,omitempty"`
	OfficialPricing *SupportedModelPricing `json:"official_pricing,omitempty"`
}

type ModelPlazaService struct {
	channelService *ChannelService
}

func NewModelPlazaService(channelService *ChannelService) *ModelPlazaService {
	return &ModelPlazaService{channelService: channelService}
}

func (s *ModelPlazaService) List(ctx context.Context, allowedGroupIDs map[int64]struct{}, authenticated bool) (*ModelPlazaResponse, error) {
	if s == nil || s.channelService == nil {
		return nil, fmt.Errorf("model plaza service not configured")
	}
	channels, err := s.channelService.ListAvailable(ctx)
	if err != nil {
		return nil, err
	}
	groups := BuildModelPlazaGroups(channels, allowedGroupIDs, authenticated)
	return &ModelPlazaResponse{
		Description: "Public model availability and pricing are projected from local channel, group and billing settings.",
		Groups:      groups,
	}, nil
}

func BuildModelPlazaGroups(channels []AvailableChannel, allowedGroupIDs map[int64]struct{}, authenticated bool) []ModelPlazaGroup {
	byID := make(map[int64]*ModelPlazaGroup)
	order := make([]int64, 0)
	for _, ch := range channels {
		if ch.Status != StatusActive {
			continue
		}
		for _, ref := range ch.Groups {
			if !modelPlazaGroupVisible(ref, allowedGroupIDs, authenticated) {
				continue
			}
			group := byID[ref.ID]
			if group == nil {
				next := modelPlazaGroupFromRef(ref)
				byID[ref.ID] = &next
				order = append(order, ref.ID)
				group = &next
			}
			group.Models = append(group.Models, modelPlazaModelsForGroup(ref, ch.SupportedModels)...)
			byID[ref.ID] = group
		}
	}
	out := make([]ModelPlazaGroup, 0, len(order))
	for _, id := range order {
		group := byID[id]
		if group == nil {
			continue
		}
		group.Models = dedupeModelPlazaModels(group.Models)
		if len(group.Models) == 0 {
			continue
		}
		out = append(out, *group)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Platform != out[j].Platform {
			return out[i].Platform < out[j].Platform
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out
}

func modelPlazaGroupVisible(ref AvailableGroupRef, allowedGroupIDs map[int64]struct{}, authenticated bool) bool {
	if !ref.IsExclusive {
		return true
	}
	if !authenticated {
		return false
	}
	_, ok := allowedGroupIDs[ref.ID]
	return ok
}

func modelPlazaGroupFromRef(ref AvailableGroupRef) ModelPlazaGroup {
	imageIndependent := ref.ImagePrice1K != nil || ref.ImagePrice2K != nil || ref.ImagePrice4K != nil
	return ModelPlazaGroup{
		ID:                    ref.ID,
		Name:                  ref.Name,
		Description:           ref.Description,
		Platform:              ref.Platform,
		SubscriptionType:      ref.SubscriptionType,
		RateMultiplier:        ref.RateMultiplier,
		PeakRateEnabled:       ref.PeakRateEnabled,
		PeakStart:             ref.PeakStart,
		PeakEnd:               ref.PeakEnd,
		PeakRateMultiplier:    ref.PeakRateMultiplier,
		IsExclusive:           ref.IsExclusive,
		ImageRateIndependent:  imageIndependent,
		ImageRateMultiplier:   1,
		ImagePrice1K:          ref.ImagePrice1K,
		ImagePrice2K:          ref.ImagePrice2K,
		ImagePrice4K:          ref.ImagePrice4K,
		WebSearchPricePerCall: ref.WebSearchPricePerCall,
		ProfitControlEnabled:  ref.ProfitControlEnabled,
		ProfitMinMargin:       ref.ProfitMinMargin,
		ProfitSafetyBuffer:    ref.ProfitSafetyBuffer,
		Models:                []ModelPlazaModel{},
	}
}

func modelPlazaModelsForGroup(ref AvailableGroupRef, models []SupportedModel) []ModelPlazaModel {
	out := make([]ModelPlazaModel, 0, len(models))
	for _, item := range models {
		if item.Name == "" {
			continue
		}
		if item.Platform != "" && ref.Platform != "" && item.Platform != ref.Platform {
			continue
		}
		out = append(out, ModelPlazaModel{
			Name:            item.Name,
			Platform:        firstNonEmptyString(item.Platform, ref.Platform),
			Pricing:         item.Pricing,
			OfficialPricing: item.Pricing,
		})
	}
	return out
}

func dedupeModelPlazaModels(items []ModelPlazaModel) []ModelPlazaModel {
	if len(items) == 0 {
		return []ModelPlazaModel{}
	}
	seen := make(map[string]struct{}, len(items))
	out := make([]ModelPlazaModel, 0, len(items))
	for _, item := range items {
		key := strings.ToLower(item.Platform) + "\x00" + strings.ToLower(item.Name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, item)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out
}
