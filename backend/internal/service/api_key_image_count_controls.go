package service

func sanitizeAPIKeyImageCountBillingForActor(apiKey *APIKey, actor *User) {
	if apiKey == nil || actor == nil || actor.IsAdmin() {
		return
	}
	apiKey.ImageCountBillingEnabled = false
	apiKey.ImageMaxCount = 0
	apiKey.ImageCountUsed = 0
	apiKey.ImageCountWeights = DefaultAPIKeyImageCountWeights()
}
