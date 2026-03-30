package service

import "strings"

type AccountManualModel struct {
	ModelID        string `json:"model_id"`
	RequestAlias   string `json:"request_alias,omitempty"`
	SourceProtocol string `json:"source_protocol,omitempty"`
}

func NormalizeAccountManualModels(models []AccountManualModel, allowSourceProtocol bool) []AccountManualModel {
	if len(models) == 0 {
		return nil
	}
	normalized := make([]AccountManualModel, 0, len(models))
	seen := make(map[string]struct{}, len(models))
	for _, item := range models {
		normalizedItem, ok := normalizeAccountManualModel(item, allowSourceProtocol)
		if !ok {
			continue
		}
		key := accountManualModelKey(normalizedItem)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, normalizedItem)
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func AccountManualModelsFromExtra(extra map[string]any, allowSourceProtocol bool) []AccountManualModel {
	if len(extra) == 0 {
		return nil
	}
	raw, ok := extra["manual_models"]
	if !ok || raw == nil {
		return nil
	}
	items, ok := raw.([]any)
	if !ok {
		return nil
	}
	parsed := make([]AccountManualModel, 0, len(items))
	for _, item := range items {
		entry, ok := item.(map[string]any)
		if !ok || len(entry) == 0 {
			continue
		}
		parsed = append(parsed, AccountManualModel{
			ModelID:        stringValueFromAny(entry["model_id"]),
			RequestAlias:   stringValueFromAny(entry["request_alias"]),
			SourceProtocol: stringValueFromAny(entry["source_protocol"]),
		})
	}
	return NormalizeAccountManualModels(parsed, allowSourceProtocol)
}

func AccountManualModelsToExtraValue(models []AccountManualModel, allowSourceProtocol bool) []map[string]any {
	normalized := NormalizeAccountManualModels(models, allowSourceProtocol)
	if len(normalized) == 0 {
		return nil
	}
	values := make([]map[string]any, 0, len(normalized))
	for _, item := range normalized {
		entry := map[string]any{
			"model_id": item.ModelID,
		}
		if alias := strings.TrimSpace(item.RequestAlias); alias != "" && alias != item.ModelID {
			entry["request_alias"] = alias
		}
		if protocol := strings.TrimSpace(item.SourceProtocol); protocol != "" {
			entry["source_protocol"] = protocol
		}
		values = append(values, entry)
	}
	return values
}

func normalizeAccountManualModel(model AccountManualModel, allowSourceProtocol bool) (AccountManualModel, bool) {
	model.ModelID = strings.TrimSpace(model.ModelID)
	if model.ModelID == "" {
		return AccountManualModel{}, false
	}
	model.RequestAlias = strings.TrimSpace(model.RequestAlias)
	if model.RequestAlias == "" {
		model.RequestAlias = model.ModelID
	}
	if allowSourceProtocol {
		model.SourceProtocol = NormalizeGatewayProtocol(model.SourceProtocol)
	} else {
		model.SourceProtocol = ""
	}
	return model, true
}

func accountManualModelKey(model AccountManualModel) string {
	return normalizeRegistryID(model.ModelID) + "::" + NormalizeGatewayProtocol(model.SourceProtocol)
}
