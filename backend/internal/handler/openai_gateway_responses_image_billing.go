package handler

import "github.com/Wei-Shaw/sub2api/internal/service"

func shouldReserveResponsesImageCount(apiKey *service.APIKey, hasImageTool bool) bool {
	return apiKey != nil && apiKey.EffectiveImageCountBillingEnabled() && hasImageTool
}
