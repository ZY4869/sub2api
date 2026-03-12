package service

import "strings"

func mapAntigravityModel(account *Account, requestedModel string) string {
	if account == nil {
		return ""
	}
	mapping := account.GetModelMapping()
	if len(mapping) == 0 {
		return ""
	}
	mapped := account.GetMappedModel(requestedModel)
	if mapped != requestedModel {
		return mapped
	}
	if account.IsModelSupported(requestedModel) {
		return requestedModel
	}
	return ""
}
func (s *AntigravityGatewayService) getMappedModel(account *Account, requestedModel string) string {
	return mapAntigravityModel(account, requestedModel)
}
func applyThinkingModelSuffix(mappedModel string, thinkingEnabled bool) string {
	if !thinkingEnabled {
		return mappedModel
	}
	if mappedModel == "claude-sonnet-4-5" {
		return "claude-sonnet-4-5-thinking"
	}
	return mappedModel
}
func (s *AntigravityGatewayService) IsModelSupported(requestedModel string) bool {
	return strings.HasPrefix(requestedModel, "claude-") || strings.HasPrefix(requestedModel, "gemini-")
}
