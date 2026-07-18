package securityaudit

import (
	"strings"
)

func ParseQwen3Guard(content string, enabledScanners []string) (*NormalizedResult, error) {
	safety, categories, err := parseGuardLines(content)
	if err != nil {
		return nil, err
	}
	known, unknown := normalizeGuardCategories(categories)
	matched := enabledCategoryIntersection(known, enabledScanners)
	result := &NormalizedResult{
		Safety: safety, Categories: known, MatchedScanners: matched,
		UnknownCategories: unknown, ScannerScores: map[string]float64{},
		ScannerEvidence: map[string]string{}, ScannerBackend: "qwen3guard-openai",
		ScannerVersion: "qwen3guard", PolicyID: DefaultStrategy, PolicyVersion: 1,
		Decision: EventPass, RiskLevel: RiskLow, Action: ActionAllow,
	}
	applyGuardDecision(result)
	return result, nil
}

func parseGuardLines(content string) (string, string, error) {
	lines := make([]string, 0, 2)
	for _, line := range strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			lines = append(lines, line)
		}
	}
	if len(lines) != 2 {
		return "", "", &GuardError{Code: ErrorCodeInvalidResponse}
	}
	return parseSafetyAndCategories(lines)
}

func parseSafetyAndCategories(lines []string) (string, string, error) {
	safety, categories := "", ""
	for _, line := range lines {
		lower := strings.ToLower(line)
		switch {
		case strings.HasPrefix(lower, "safety:"):
			if safety != "" {
				return "", "", &GuardError{Code: ErrorCodeInvalidResponse}
			}
			safety = strings.TrimSpace(line[len("safety:"):])
		case strings.HasPrefix(lower, "categories:"):
			if categories != "" {
				return "", "", &GuardError{Code: ErrorCodeInvalidResponse}
			}
			categories = strings.TrimSpace(line[len("categories:"):])
		default:
			return "", "", &GuardError{Code: ErrorCodeInvalidResponse}
		}
	}
	return normalizeGuardSafety(safety, categories)
}

func normalizeGuardSafety(safety string, categories string) (string, string, error) {
	switch strings.ToLower(safety) {
	case "safe":
		safety = "Safe"
	case "controversial":
		safety = "Controversial"
	case "unsafe":
		safety = "Unsafe"
	default:
		return "", "", &GuardError{Code: ErrorCodeInvalidResponse}
	}
	if categories == "" {
		return "", "", &GuardError{Code: ErrorCodeInvalidResponse}
	}
	return safety, categories, nil
}

func normalizeGuardCategories(raw string) ([]string, []string) {
	knownSet := map[string]struct{}{}
	unknownSet := map[string]struct{}{}
	for _, value := range strings.Split(raw, ",") {
		value = strings.TrimSpace(value)
		if value == "" || strings.EqualFold(value, "none") || strings.EqualFold(value, "n/a") {
			continue
		}
		id := NormalizeCategory(value)
		if _, ok := ScannerCatalog[id]; ok {
			knownSet[id] = struct{}{}
			continue
		}
		unknownSet[unknownCategoryID(id)] = struct{}{}
	}
	return orderedScannerKeys(knownSet), sortedKeys(unknownSet)
}

func applyGuardDecision(result *NormalizedResult) {
	score := 0.0
	switch result.Safety {
	case "Controversial":
		score = 0.5
		result.Decision, result.RiskLevel, result.Action = EventFlag, RiskMedium, ActionWarn
	case "Unsafe":
		score = 1
		applyUnsafeDecision(result)
	}
	applyScannerEvidence(result, score)
}

func applyUnsafeDecision(result *NormalizedResult) {
	if len(result.MatchedScanners) > 0 || len(result.UnknownCategories) > 0 || len(result.Categories) == 0 {
		result.Decision, result.RiskLevel, result.Action = EventCritical, RiskCritical, ActionBlock
		return
	}
	result.Decision, result.RiskLevel, result.Action = EventFlag, RiskHigh, ActionWarn
}

func applyScannerEvidence(result *NormalizedResult, score float64) {
	for _, category := range result.MatchedScanners {
		result.ScannerScores[category] = score
		result.ScannerEvidence[category] = ScannerCatalog[category].Label
		if result.Safety == "Controversial" && isElevatedControversial(category) {
			result.Decision, result.RiskLevel, result.Action = EventCritical, RiskCritical, ActionBlock
		}
	}
}
