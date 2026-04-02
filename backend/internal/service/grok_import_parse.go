package service

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type GrokImportCandidate struct {
	Index            int            `json:"index"`
	Name             string         `json:"name"`
	Notes            string         `json:"notes,omitempty"`
	Type             string         `json:"type"`
	DetectedKind     string         `json:"detected_kind"`
	Credential       string         `json:"-"`
	CredentialKey    string         `json:"credential_key"`
	CredentialMasked string         `json:"credential_masked"`
	SourcePool       string         `json:"source_pool,omitempty"`
	Tier             string         `json:"grok_tier"`
	Priority         int            `json:"priority"`
	Concurrency      int            `json:"concurrency"`
	Credentials      map[string]any `json:"credentials"`
	Extra            map[string]any `json:"extra"`
}

type GrokImportParseError struct {
	Index      int    `json:"index"`
	SourcePool string `json:"source_pool,omitempty"`
	Message    string `json:"message"`
}

type GrokImportParseResult struct {
	DetectedKind string                 `json:"detected_kind,omitempty"`
	Candidates   []GrokImportCandidate  `json:"candidates"`
	Errors       []GrokImportParseError `json:"errors,omitempty"`
}

func ParseGrokImportPayload(content string) (*GrokImportParseResult, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, fmt.Errorf("import content is empty")
	}

	decoder := json.NewDecoder(strings.NewReader(content))
	decoder.UseNumber()

	var payload any
	if err := decoder.Decode(&payload); err == nil {
		return parseGrokImportJSON(payload)
	}

	return parseGrokImportText(content), nil
}

func parseGrokImportJSON(payload any) (*GrokImportParseResult, error) {
	switch root := payload.(type) {
	case []any:
		return parseGrokImportArray(root, ""), nil
	case map[string]any:
		if items, ok := firstNestedArray(root, "accounts", "items", "data"); ok {
			return parseGrokImportArray(items, ""), nil
		}
		if looksLikeGrokImportItem(root) {
			return parseGrokImportArray([]any{root}, ""), nil
		}
		return parseGrokImportLegacyPools(root), nil
	default:
		return nil, fmt.Errorf("unsupported import JSON structure")
	}
}

func parseGrokImportText(content string) *GrokImportParseResult {
	lines := strings.Split(content, "\n")
	items := make([]any, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, ";") {
			continue
		}
		items = append(items, trimmed)
	}
	return parseGrokImportArray(items, "")
}

func parseGrokImportLegacyPools(root map[string]any) *GrokImportParseResult {
	keys := make([]string, 0, len(root))
	for key, value := range root {
		if _, ok := value.([]any); ok {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)

	result := &GrokImportParseResult{
		DetectedKind: GrokDetectedKindLegacyPool,
		Candidates:   []GrokImportCandidate{},
		Errors:       []GrokImportParseError{},
	}
	index := 0
	for _, key := range keys {
		items, _ := root[key].([]any)
		parsed := parseGrokImportArray(items, key)
		for _, candidate := range parsed.Candidates {
			candidate.Index = index
			index++
			result.Candidates = append(result.Candidates, candidate)
		}
		for _, parseErr := range parsed.Errors {
			parseErr.Index = index
			index++
			result.Errors = append(result.Errors, parseErr)
		}
	}
	if len(keys) == 0 {
		result.Errors = append(result.Errors, GrokImportParseError{
			Index:   0,
			Message: "legacy pool JSON does not contain any token arrays",
		})
	}
	return result
}

func parseGrokImportArray(items []any, sourcePool string) *GrokImportParseResult {
	result := &GrokImportParseResult{
		Candidates: []GrokImportCandidate{},
		Errors:     []GrokImportParseError{},
	}
	index := 0
	kindVotes := map[string]int{}

	for _, item := range items {
		candidate, err := parseSingleGrokImportItem(item, sourcePool, index)
		if err != nil {
			result.Errors = append(result.Errors, GrokImportParseError{
				Index:      index,
				SourcePool: sourcePool,
				Message:    err.Error(),
			})
			index++
			continue
		}
		result.Candidates = append(result.Candidates, candidate)
		kindVotes[candidate.DetectedKind]++
		index++
	}

	if sourcePool != "" {
		result.DetectedKind = GrokDetectedKindLegacyPool
		return result
	}

	switch len(kindVotes) {
	case 0:
		result.DetectedKind = ""
	case 1:
		for kind := range kindVotes {
			result.DetectedKind = kind
		}
	default:
		if kindVotes[GrokDetectedKindSSO] >= kindVotes[GrokDetectedKindAPIKey] {
			result.DetectedKind = GrokDetectedKindSSO
		} else {
			result.DetectedKind = GrokDetectedKindAPIKey
		}
	}
	return result
}

func parseSingleGrokImportItem(item any, sourcePool string, index int) (GrokImportCandidate, error) {
	switch value := item.(type) {
	case string:
		return buildGrokImportCandidate(
			index,
			sourcePool,
			InferGrokCredentialKind(value),
			value,
			"",
			"",
			0,
			0,
			"",
			nil,
			nil,
		)
	case map[string]any:
		credentials, _ := value["credentials"].(map[string]any)
		explicitKind := normalizeExplicitGrokImportKind(firstStringField(value, "type", "account_type", "detected_kind"))
		rawCredential := firstStringField(value, "api_key", "apikey", "key")
		if rawCredential == "" {
			rawCredential = firstStringField(credentials, "api_key", "apikey", "key")
		}
		if rawCredential != "" && explicitKind == "" {
			explicitKind = GrokDetectedKindAPIKey
		}
		if rawCredential == "" {
			rawCredential = firstStringField(value, "sso_token", "token", "access_token", "credential", "value")
		}
		if rawCredential == "" {
			rawCredential = firstStringField(credentials, "sso_token", "token", "access_token", "credential", "value")
		}
		if rawCredential == "" {
			return GrokImportCandidate{}, fmt.Errorf("missing credential")
		}
		if explicitKind == "" {
			explicitKind = InferGrokCredentialKind(rawCredential)
		}
		name := firstStringField(value, "name", "display_name", "title")
		notes := firstStringField(value, "notes")
		baseURL := firstStringField(value, "base_url")
		if baseURL == "" {
			baseURL = firstStringField(credentials, "base_url")
		}
		priority := ParseExtraInt(firstAnyField(value, "priority"))
		concurrency := ParseExtraInt(firstAnyField(value, "concurrency"))
		extra, _ := value["extra"].(map[string]any)
		return buildGrokImportCandidate(index, sourcePool, explicitKind, rawCredential, name, notes, priority, concurrency, baseURL, credentials, extra)
	default:
		return GrokImportCandidate{}, fmt.Errorf("unsupported item type")
	}
}

func buildGrokImportCandidate(index int, sourcePool string, kind string, rawCredential string, name string, notes string, priority int, concurrency int, baseURL string, rawCredentials map[string]any, rawExtra map[string]any) (GrokImportCandidate, error) {
	kind = normalizeExplicitGrokImportKind(kind)
	if kind == "" {
		kind = InferGrokCredentialKind(rawCredential)
	}
	credential := NormalizeGrokCredentialValue(kind, rawCredential)
	if credential == "" {
		return GrokImportCandidate{}, fmt.Errorf("credential is empty after normalization")
	}

	tier := GrokTierBasic
	if sourcePool != "" {
		tier = LegacyGrokPoolTier(sourcePool)
	}
	if rawExtra != nil {
		tier = ResolveGrokTier(rawExtra)
	}
	if rawCredentials != nil {
		if explicitTier := NormalizeGrokTierValue(firstStringField(rawCredentials, "grok_tier", "tier")); explicitTier != "" {
			tier = explicitTier
		}
	}

	if priority <= 0 {
		priority = DefaultGrokPriorityForTier(tier)
	}
	if concurrency <= 0 {
		concurrency = 1
	}
	if strings.TrimSpace(name) == "" {
		name = fmt.Sprintf("grok-%s-%s-%02d", kind, tier, index+1)
	}

	extra := map[string]any{
		"grok_tier":         tier,
		"grok_capabilities": DefaultGrokCapabilitiesForTier(tier).ToMap(),
	}
	for key, value := range rawExtra {
		extra[key] = value
	}
	extra["grok_tier"] = tier
	extra["grok_capabilities"] = ResolveGrokCapabilities(extra).ToMap()
	if sourcePool != "" {
		extra["grok_import_source_pool"] = sourcePool
	}

	credentials := map[string]any{}
	for key, value := range rawCredentials {
		credentials[key] = value
	}
	switch kind {
	case GrokDetectedKindAPIKey:
		credentials["api_key"] = credential
		if strings.TrimSpace(baseURL) == "" {
			baseURL = "https://api.x.ai"
		}
		credentials["base_url"] = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	case GrokDetectedKindSSO:
		credentials["sso_token"] = credential
		if _, ok := credentials["model_mapping"].(map[string]any); !ok {
			credentials["model_mapping"] = DefaultGrokModelMappingForTier(tier)
		}
	default:
		return GrokImportCandidate{}, fmt.Errorf("unsupported detected kind: %s", kind)
	}

	return GrokImportCandidate{
		Index:            index,
		Name:             strings.TrimSpace(name),
		Notes:            strings.TrimSpace(notes),
		Type:             kind,
		DetectedKind:     kind,
		Credential:       credential,
		CredentialKey:    kind + ":" + credential,
		CredentialMasked: MaskGrokCredentialValue(kind, credential),
		SourcePool:       strings.TrimSpace(sourcePool),
		Tier:             tier,
		Priority:         priority,
		Concurrency:      concurrency,
		Credentials:      credentials,
		Extra:            extra,
	}, nil
}

func looksLikeGrokImportItem(item map[string]any) bool {
	if item == nil {
		return false
	}
	if _, ok := item["credentials"].(map[string]any); ok {
		return true
	}
	for _, key := range []string{"api_key", "apikey", "key", "sso_token", "token", "access_token", "credential", "value"} {
		if text, _ := item[key].(string); strings.TrimSpace(text) != "" {
			return true
		}
	}
	return false
}

func firstNestedArray(item map[string]any, keys ...string) ([]any, bool) {
	for _, key := range keys {
		if values, ok := item[key].([]any); ok {
			return values, true
		}
	}
	return nil, false
}

func normalizeExplicitGrokImportKind(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "api_key", "apikey":
		return GrokDetectedKindAPIKey
	case "sso", "token", "bearer":
		return GrokDetectedKindSSO
	default:
		return ""
	}
}

func firstStringField(item map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, _ := item[key].(string); strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func firstAnyField(item map[string]any, keys ...string) any {
	for _, key := range keys {
		if value, ok := item[key]; ok {
			return value
		}
	}
	return nil
}
