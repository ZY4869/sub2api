package service

import (
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
)

const (
	OpenAIKnownModelsSourceImportModels = "import_models"
	OpenAIKnownModelsSourceTestProbe    = "test_probe"
	OpenAIKnownModelsSourceModelMapping = "model_mapping"
)

func normalizeOpenAIPlanType(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	normalized := strings.ToLower(trimmed)
	normalized = strings.NewReplacer("-", "", "_", "", " ", "").Replace(normalized)

	switch normalized {
	case "chatgptplus", "plus":
		return "plus"
	case "chatgptteam", "team":
		return "team"
	case "chatgptpro", "pro":
		return "pro"
	case "chatgptfree", "free":
		return "free"
	default:
		return trimmed
	}
}

func cloneStringAnyMap(src map[string]any) map[string]any {
	if len(src) == 0 {
		return nil
	}
	out := make(map[string]any, len(src))
	for key, value := range src {
		out[key] = value
	}
	return out
}

func mergeStringAnyMap(base map[string]any, updates map[string]any) map[string]any {
	if len(base) == 0 && len(updates) == 0 {
		return nil
	}
	out := cloneStringAnyMap(base)
	if out == nil {
		out = make(map[string]any, len(updates))
	}
	for key, value := range updates {
		out[key] = value
	}
	return out
}

func MergeStringAnyMap(base map[string]any, updates map[string]any) map[string]any {
	return mergeStringAnyMap(base, updates)
}

func stringValueFromAny(value any) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	default:
		return ""
	}
}

func EnrichOpenAIOAuthCredentials(platform, accountType string, credentials map[string]any) (map[string]any, bool) {
	if len(credentials) == 0 {
		return credentials, false
	}

	normalizedPlatform := strings.ToLower(strings.TrimSpace(platform))
	if normalizedPlatform != PlatformOpenAI {
		return credentials, false
	}
	if strings.ToLower(strings.TrimSpace(accountType)) != AccountTypeOAuth {
		return credentials, false
	}

	out := credentials
	changed := false
	cloned := false

	planType := stringValueFromAny(credentials["plan_type"])
	if normalizedPlanType := normalizeOpenAIPlanType(planType); normalizedPlanType != "" && normalizedPlanType != planType {
		out = cloneStringAnyMap(credentials)
		out["plan_type"] = normalizedPlanType
		changed = true
		cloned = true
	}

	idToken := stringValueFromAny(credentials["id_token"])
	if idToken == "" {
		return out, changed
	}

	claims, err := openai.DecodeIDToken(idToken)
	if err != nil {
		return out, changed
	}
	userInfo := claims.GetUserInfo()
	if userInfo == nil {
		return out, changed
	}

	if !cloned {
		out = cloneStringAnyMap(credentials)
	}
	setIfMissing := func(key, value string) {
		if strings.TrimSpace(value) == "" {
			return
		}
		if stringValueFromAny(out[key]) != "" {
			return
		}
		out[key] = value
		changed = true
	}

	setIfMissing("email", userInfo.Email)
	setIfMissing("chatgpt_account_id", userInfo.ChatGPTAccountID)
	setIfMissing("chatgpt_user_id", userInfo.ChatGPTUserID)
	setIfMissing("organization_id", userInfo.OrganizationID)
	setIfMissing("plan_type", normalizeOpenAIPlanType(userInfo.PlanType))

	return out, changed
}

func BuildOpenAIKnownModelsExtra(models []string, updatedAt time.Time, source string) map[string]any {
	out := make(map[string]any, 3)

	uniqueModels := make([]string, 0, len(models))
	seen := make(map[string]struct{}, len(models))
	for _, model := range models {
		trimmed := strings.TrimSpace(model)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		uniqueModels = append(uniqueModels, trimmed)
	}
	out["openai_known_models"] = uniqueModels

	if !updatedAt.IsZero() {
		out["openai_known_models_updated_at"] = updatedAt.UTC().Format(time.RFC3339)
	}

	if trimmedSource := strings.TrimSpace(source); trimmedSource != "" {
		out["openai_known_models_source"] = trimmedSource
	}

	return out
}
