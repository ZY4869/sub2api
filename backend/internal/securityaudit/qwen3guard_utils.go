package securityaudit

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

type GuardError struct {
	Code       string
	HTTPStatus int
	Retryable  bool
	Timeout    bool
	Cause      error
}

func (e *GuardError) Error() string {
	if e == nil {
		return "<nil>"
	}
	return e.Code
}

func (e *GuardError) Unwrap() error { return e.Cause }

func NormalizeCategory(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.NewReplacer("_", " ", "&", " and ", "/", " ", "-", " ", "–", " ", "—", " ").Replace(normalized)
	normalized = strings.Join(strings.Fields(normalized), " ")
	if canonical, ok := categoryAliases[normalized]; ok {
		return canonical
	}
	return strings.ReplaceAll(normalized, " ", "_")
}

func canonicalScannerIDs(values []string) []string {
	seen := map[string]struct{}{}
	for _, value := range values {
		id := NormalizeCategory(value)
		if _, ok := ScannerCatalog[id]; ok {
			seen[id] = struct{}{}
		}
	}
	return orderedScannerKeys(seen)
}

func BuildIssueSummaries(result NormalizedResult) []IssueSummary {
	values := mergeStrings(result.Categories, result.UnknownCategories)
	out := make([]IssueSummary, 0, len(values))
	for _, code := range values {
		title, description := code, "Prompt audit finding"
		if def, ok := ScannerCatalog[code]; ok {
			title, description = def.Label, def.Description
		}
		out = append(out, IssueSummary{Code: code, Title: title, Description: description, EvidenceHash: evidenceHash(result.ScannerEvidence[code])})
	}
	return out
}

func extractOpenAIContent(body []byte) (string, error) {
	var response struct {
		Choices []struct {
			Message struct {
				Content any `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &response); err != nil || len(response.Choices) == 0 {
		return "", errors.New("prompt guard response envelope invalid")
	}
	return openAIMessageContentText(response.Choices[0].Message.Content)
}

func openAIMessageContentText(content any) (string, error) {
	switch typed := content.(type) {
	case string:
		if strings.TrimSpace(typed) == "" {
			return "", errors.New("prompt guard response content empty")
		}
		return typed, nil
	case []any:
		return openAIContentPartsText(typed)
	default:
		return "", errors.New("prompt guard response content invalid")
	}
}

func openAIContentPartsText(content []any) (string, error) {
	parts := make([]string, 0, len(content))
	for _, item := range content {
		object, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if text, ok := object["text"].(string); ok && strings.TrimSpace(text) != "" {
			parts = append(parts, text)
		}
	}
	if len(parts) == 0 {
		return "", errors.New("prompt guard response content empty")
	}
	return strings.Join(parts, "\n"), nil
}

func unknownCategoryID(value string) string {
	digest := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(value))))
	return fmt.Sprintf("unknown:%x", digest[:8])
}

func orderedScannerKeys(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for _, id := range AllScannerIDs {
		if _, ok := set[id]; ok {
			out = append(out, id)
		}
	}
	return out
}

func sortedKeys(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func enabledCategoryIntersection(categories, enabled []string) []string {
	enabledSet := map[string]struct{}{}
	for _, value := range enabled {
		enabledSet[NormalizeCategory(value)] = struct{}{}
	}
	out := make([]string, 0, len(categories))
	for _, category := range categories {
		if _, ok := enabledSet[category]; ok {
			out = append(out, category)
		}
	}
	return out
}

func isElevatedControversial(category string) bool {
	return category == "jailbreak" || category == "pii" || category == "suicide_and_self_harm"
}

func mergeStrings(a, b []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(a)+len(b))
	for _, list := range [][]string{a, b} {
		for _, value := range list {
			value = strings.TrimSpace(value)
			if value == "" {
				continue
			}
			if _, ok := seen[value]; ok {
				continue
			}
			seen[value] = struct{}{}
			out = append(out, value)
		}
	}
	return out
}

func evidenceHash(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	digest := sha256.Sum256([]byte(value))
	return fmt.Sprintf("%x", digest[:8])
}
