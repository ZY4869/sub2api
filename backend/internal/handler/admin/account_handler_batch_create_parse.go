package admin

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type batchCreateLineOverrides struct {
	Name                    string
	Notes                   *string
	Credentials             map[string]any
	Extra                   map[string]any
	ProxyID                 *int64
	Concurrency             *int
	LoadFactor              *int
	Priority                *int
	RateMultiplier          *float64
	GroupIDs                *[]int64
	ExpiresAt               *int64
	AutoPauseOnExpired      *bool
	ConfirmMixedChannelRisk *bool
	RawPreview              string
}

var (
	batchCreateKiroTokenContainerKeys = []string{"credentials", "credential", "token", "tokens", "auth", "oauth"}
	batchCreateKiroUserContainerKeys  = []string{"user", "profile", "account", "identity"}
)

func parseBatchCreateLine(raw string, platform string, accountType string) (*batchCreateLineOverrides, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, fmt.Errorf("line is empty")
	}
	if !batchCreateLooksLikeJSONObject(trimmed) {
		return &batchCreateLineOverrides{
			Credentials: batchCreateRawCredentials(platform, accountType, trimmed),
			RawPreview:  batchCreatePreview(trimmed),
		}, nil
	}

	root, err := batchCreateParseJSONObject(trimmed)
	if err != nil {
		return nil, err
	}
	return buildBatchCreateJSONLine(root, trimmed, platform, accountType), nil
}

func buildBatchCreateJSONLine(root map[string]any, raw string, platform string, accountType string) *batchCreateLineOverrides {
	line := &batchCreateLineOverrides{
		Name:                    batchCreateString(root, "name"),
		Notes:                   batchCreateOptionalString(root, "notes"),
		ProxyID:                 batchCreateOptionalInt64Pointer(root, "proxy_id", "proxyId"),
		Concurrency:             batchCreateOptionalIntPointer(root, "concurrency"),
		LoadFactor:              batchCreateOptionalIntPointer(root, "load_factor", "loadFactor"),
		Priority:                batchCreateOptionalIntPointer(root, "priority"),
		RateMultiplier:          batchCreateOptionalFloat64Pointer(root, "rate_multiplier", "rateMultiplier"),
		GroupIDs:                batchCreateOptionalInt64SlicePointer(root, "group_ids", "groupIds"),
		ExpiresAt:               batchCreateOptionalInt64Pointer(root, "expires_at", "expiresAt"),
		AutoPauseOnExpired:      batchCreateOptionalBoolPointer(root, "auto_pause_on_expired", "autoPauseOnExpired"),
		ConfirmMixedChannelRisk: batchCreateOptionalBoolPointer(root, "confirm_mixed_channel_risk", "confirmMixedChannelRisk"),
		RawPreview:              batchCreatePreview(raw),
	}

	line.Credentials = batchCreateExtractCredentials(root, platform, accountType)
	line.Extra = batchCreateExtractExtra(root, platform)
	return line
}

func batchCreateExtractCredentials(root map[string]any, platform string, accountType string) map[string]any {
	if strings.EqualFold(strings.TrimSpace(platform), service.PlatformKiro) {
		return service.NormalizeKiroCredentialsForStorage(batchCreateExtractKiroCredentials(root))
	}

	derived := batchCreateRawCredentialsFromJSONObject(root, accountType)
	nested := batchCreateMap(root, "credentials")
	return service.MergeCredentials(derived, nested)
}

func batchCreateExtractExtra(root map[string]any, platform string) map[string]any {
	extra := batchCreateMap(root, "extra")
	if strings.EqualFold(strings.TrimSpace(platform), service.PlatformKiro) {
		derived := batchCreateExtractKiroExtra(root)
		return service.MergeStringAnyMap(derived, extra)
	}
	return extra
}

func batchCreateExtractKiroCredentials(root map[string]any) map[string]any {
	sources := batchCreateCollectKiroSources(root)
	credentials := make(map[string]any)

	if accessToken := batchCreatePickString(sources, "access_token", "accessToken"); accessToken != "" {
		credentials["access_token"] = accessToken
	}
	batchCreateAssignIfPresent(credentials, "refresh_token", batchCreatePickString(sources, "refresh_token", "refreshToken"))
	batchCreateAssignIfPresent(credentials, "expires_at", batchCreatePickString(sources, "expires_at", "expiresAt", "expiration", "expires"))
	batchCreateAssignIfPresent(credentials, "auth_method", batchCreatePickString(sources, "auth_method", "authMethod"))
	batchCreateAssignIfPresent(credentials, "client_id", batchCreatePickString(sources, "client_id", "clientId"))
	batchCreateAssignIfPresent(credentials, "client_secret", batchCreatePickString(sources, "client_secret", "clientSecret"))
	batchCreateAssignIfPresent(credentials, "client_id_hash", batchCreatePickString(sources, "client_id_hash", "clientIdHash"))
	batchCreateAssignIfPresent(credentials, "start_url", batchCreatePickString(sources, "start_url", "startUrl"))
	batchCreateAssignIfPresent(credentials, "api_region", batchCreatePickString(sources, "api_region", "apiRegion", "region"))
	batchCreateAssignIfPresent(credentials, "profile_arn", batchCreatePickString(sources, "profile_arn", "profileArn"))

	return credentials
}

func batchCreateExtractKiroExtra(root map[string]any) map[string]any {
	sources := batchCreateCollectKiroSources(root)
	extra := map[string]any{}
	batchCreateAssignIfPresent(extra, "email", batchCreatePickString(sources, "email"))
	batchCreateAssignIfPresent(extra, "username", batchCreatePickString(sources, "username", "login", "user_name"))
	batchCreateAssignIfPresent(extra, "display_name", batchCreatePickString(sources, "display_name", "displayName", "name"))
	batchCreateAssignIfPresent(extra, "provider", batchCreatePickString(sources, "provider"))
	batchCreateAssignIfPresent(extra, "source", batchCreatePickString(sources, "source"))
	if _, ok := extra["provider"]; !ok {
		extra["provider"] = "kiro"
	}
	if _, ok := extra["source"]; !ok {
		extra["source"] = "kiro_import"
	}
	return extra
}

func batchCreateCollectKiroSources(root map[string]any) []map[string]any {
	sources := []map[string]any{root}
	for _, key := range batchCreateKiroTokenContainerKeys {
		if nested := batchCreateMap(root, key); len(nested) > 0 {
			sources = append(sources, nested)
		}
	}
	for _, key := range batchCreateKiroUserContainerKeys {
		if nested := batchCreateMap(root, key); len(nested) > 0 {
			sources = append(sources, nested)
		}
	}
	return sources
}

func batchCreateRawCredentialsFromJSONObject(root map[string]any, accountType string) map[string]any {
	credentials := map[string]any{}
	switch strings.ToLower(strings.TrimSpace(accountType)) {
	case service.AccountTypeAPIKey, service.AccountTypeUpstream:
		if apiKey := batchCreateString(root, "api_key", "apiKey", "key"); apiKey != "" {
			credentials["api_key"] = apiKey
		}
	default:
		if accessToken := batchCreateString(root, "access_token", "accessToken"); accessToken != "" {
			credentials["access_token"] = accessToken
		}
	}
	return credentials
}

func batchCreateRawCredentials(platform string, accountType string, raw string) map[string]any {
	trimmed := strings.TrimSpace(raw)
	if strings.EqualFold(strings.TrimSpace(platform), service.PlatformKiro) {
		return map[string]any{"access_token": trimmed}
	}
	switch strings.ToLower(strings.TrimSpace(accountType)) {
	case service.AccountTypeAPIKey, service.AccountTypeUpstream:
		return map[string]any{"api_key": trimmed}
	default:
		return map[string]any{"access_token": trimmed}
	}
}

func validateBatchCreateCredentials(platform string, accountType string, credentials map[string]any) error {
	if strings.EqualFold(strings.TrimSpace(platform), service.PlatformKiro) {
		if strings.TrimSpace(batchCreateNormalizeScalar(credentials["access_token"])) == "" {
			return fmt.Errorf("missing Kiro access_token")
		}
		return nil
	}

	switch strings.ToLower(strings.TrimSpace(accountType)) {
	case service.AccountTypeAPIKey, service.AccountTypeUpstream:
		if strings.TrimSpace(batchCreateNormalizeScalar(credentials["api_key"])) == "" {
			return fmt.Errorf("missing api_key")
		}
	default:
		if strings.TrimSpace(batchCreateNormalizeScalar(credentials["access_token"])) == "" {
			return fmt.Errorf("missing access_token")
		}
	}
	return nil
}

func batchCreateLooksLikeJSONObject(value string) bool {
	return strings.HasPrefix(value, "{") || strings.HasPrefix(value, "[")
}

func batchCreateParseJSONObject(value string) (map[string]any, error) {
	var parsed any
	if err := json.Unmarshal([]byte(value), &parsed); err != nil {
		return nil, fmt.Errorf("invalid JSON line: %w", err)
	}
	if list, ok := parsed.([]any); ok {
		if len(list) != 1 {
			return nil, fmt.Errorf("JSON line array must contain exactly one object")
		}
		parsed = list[0]
	}
	record, ok := parsed.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("JSON line must be an object")
	}
	return record, nil
}

func batchCreatePreview(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	if batchCreateLooksLikeJSONObject(trimmed) {
		root, err := batchCreateParseJSONObject(trimmed)
		if err == nil {
			if name := batchCreateString(root, "name"); name != "" {
				return "name=" + name
			}
			for _, key := range []string{"access_token", "accessToken", "api_key", "apiKey", "key"} {
				if token := batchCreateString(root, key); token != "" {
					return maskBatchCreateSecret(token)
				}
			}
		}
		return "{json}"
	}
	return maskBatchCreateSecret(trimmed)
}

func maskBatchCreateSecret(value string) string {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) <= 12 {
		return trimmed
	}
	return trimmed[:6] + "..." + trimmed[len(trimmed)-4:]
}

func batchCreatePickString(sources []map[string]any, keys ...string) string {
	for _, source := range sources {
		for _, key := range keys {
			if value := batchCreateNormalizeScalar(source[key]); value != "" {
				return value
			}
		}
	}
	return ""
}

func batchCreateAssignIfPresent(target map[string]any, key string, value string) {
	if strings.TrimSpace(value) == "" {
		return
	}
	target[key] = strings.TrimSpace(value)
}

func batchCreateMap(root map[string]any, keys ...string) map[string]any {
	for _, key := range keys {
		value, ok := root[key]
		if !ok {
			continue
		}
		record, ok := value.(map[string]any)
		if ok && len(record) > 0 {
			return record
		}
	}
	return nil
}

func batchCreateString(root map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := batchCreateNormalizeScalar(root[key]); value != "" {
			return value
		}
	}
	return ""
}

func batchCreateNormalizeScalar(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return ""
		}
		if math.Trunc(v) == v {
			return fmt.Sprintf("%.0f", v)
		}
		return fmt.Sprintf("%v", v)
	case json.Number:
		return strings.TrimSpace(v.String())
	default:
		return ""
	}
}

func batchCreateOptionalString(root map[string]any, keys ...string) *string {
	for _, key := range keys {
		value, ok := root[key]
		if !ok || value == nil {
			continue
		}
		switch v := value.(type) {
		case string:
			trimmed := strings.TrimSpace(v)
			return &trimmed
		}
	}
	return nil
}

func batchCreateOptionalInt64Pointer(root map[string]any, keys ...string) *int64 {
	for _, key := range keys {
		value, ok := root[key]
		if !ok || value == nil {
			continue
		}
		parsed, ok := batchCreateInt64(value)
		if !ok {
			continue
		}
		if parsed == 0 {
			return nil
		}
		result := parsed
		return &result
	}
	return nil
}

func batchCreateOptionalIntPointer(root map[string]any, keys ...string) *int {
	for _, key := range keys {
		value, ok := root[key]
		if !ok || value == nil {
			continue
		}
		parsed, ok := batchCreateInt64(value)
		if !ok {
			continue
		}
		result := int(parsed)
		return &result
	}
	return nil
}

func batchCreateOptionalFloat64Pointer(root map[string]any, keys ...string) *float64 {
	for _, key := range keys {
		value, ok := root[key]
		if !ok || value == nil {
			continue
		}
		switch v := value.(type) {
		case float64:
			result := v
			return &result
		case json.Number:
			if parsed, err := v.Float64(); err == nil {
				result := parsed
				return &result
			}
		}
	}
	return nil
}

func batchCreateOptionalBoolPointer(root map[string]any, keys ...string) *bool {
	for _, key := range keys {
		value, ok := root[key]
		if !ok || value == nil {
			continue
		}
		parsed, ok := value.(bool)
		if !ok {
			continue
		}
		result := parsed
		return &result
	}
	return nil
}

func batchCreateOptionalInt64SlicePointer(root map[string]any, keys ...string) *[]int64 {
	for _, key := range keys {
		value, ok := root[key]
		if !ok || value == nil {
			continue
		}
		list, ok := value.([]any)
		if !ok {
			continue
		}
		result := make([]int64, 0, len(list))
		for _, item := range list {
			if parsed, ok := batchCreateInt64(item); ok {
				result = append(result, parsed)
			}
		}
		out := append([]int64(nil), result...)
		return &out
	}
	return nil
}

func batchCreateInt64(value any) (int64, bool) {
	switch v := value.(type) {
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return 0, false
		}
		return int64(v), true
	case json.Number:
		parsed, err := v.Int64()
		if err != nil {
			return 0, false
		}
		return parsed, true
	case int64:
		return v, true
	case int:
		return int64(v), true
	default:
		return 0, false
	}
}
