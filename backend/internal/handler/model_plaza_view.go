package handler

import "github.com/Wei-Shaw/sub2api/internal/service"

type publicModelPlazaResponse struct {
	Description string                  `json:"description"`
	Groups      []publicModelPlazaGroup `json:"groups"`
}

type publicModelPlazaGroup struct {
	ID                    int64                   `json:"id"`
	Name                  string                  `json:"name"`
	Description           string                  `json:"description"`
	Platform              string                  `json:"platform"`
	SubscriptionType      string                  `json:"subscription_type"`
	RateMultiplier        float64                 `json:"rate_multiplier"`
	PeakRateEnabled       bool                    `json:"peak_rate_enabled"`
	PeakStart             string                  `json:"peak_start"`
	PeakEnd               string                  `json:"peak_end"`
	PeakRateMultiplier    float64                 `json:"peak_rate_multiplier"`
	IsExclusive           bool                    `json:"is_exclusive"`
	ImageRateIndependent  bool                    `json:"image_rate_independent"`
	ImageRateMultiplier   float64                 `json:"image_rate_multiplier"`
	ImagePrice1K          *float64                `json:"image_price_1k"`
	ImagePrice2K          *float64                `json:"image_price_2k"`
	ImagePrice4K          *float64                `json:"image_price_4k"`
	WebSearchPricePerCall *float64                `json:"web_search_price_per_call"`
	Models                []publicModelPlazaModel `json:"models"`
}

type publicModelPlazaModel struct {
	DisplayModelID  string                     `json:"display_model_id"`
	Platform        string                     `json:"platform"`
	Pricing         *userSupportedModelPricing `json:"pricing"`
	OfficialPricing *userSupportedModelPricing `json:"official_pricing"`
}

func toPublicModelPlazaResponse(input *service.ModelPlazaResponse) publicModelPlazaResponse {
	if input == nil {
		return publicModelPlazaResponse{Groups: []publicModelPlazaGroup{}}
	}
	groups := make([]publicModelPlazaGroup, 0, len(input.Groups))
	for i := range input.Groups {
		groups = append(groups, toPublicModelPlazaGroup(input.Groups[i]))
	}
	return publicModelPlazaResponse{
		Description: input.Description,
		Groups:      groups,
	}
}

func toPublicModelPlazaGroup(input service.ModelPlazaGroup) publicModelPlazaGroup {
	models := make([]publicModelPlazaModel, 0, len(input.Models))
	for i := range input.Models {
		models = append(models, publicModelPlazaModel{
			DisplayModelID:  input.Models[i].Name,
			Platform:        input.Models[i].Platform,
			Pricing:         toUserSupportedModelPricing(input.Models[i].Pricing),
			OfficialPricing: toUserSupportedModelPricing(input.Models[i].OfficialPricing),
		})
	}
	return publicModelPlazaGroup{
		ID:                    input.ID,
		Name:                  input.Name,
		Description:           input.Description,
		Platform:              input.Platform,
		SubscriptionType:      input.SubscriptionType,
		RateMultiplier:        input.RateMultiplier,
		PeakRateEnabled:       input.PeakRateEnabled,
		PeakStart:             input.PeakStart,
		PeakEnd:               input.PeakEnd,
		PeakRateMultiplier:    input.PeakRateMultiplier,
		IsExclusive:           input.IsExclusive,
		ImageRateIndependent:  input.ImageRateIndependent,
		ImageRateMultiplier:   input.ImageRateMultiplier,
		ImagePrice1K:          input.ImagePrice1K,
		ImagePrice2K:          input.ImagePrice2K,
		ImagePrice4K:          input.ImagePrice4K,
		WebSearchPricePerCall: input.WebSearchPricePerCall,
		Models:                models,
	}
}
