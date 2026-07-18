package securityaudit

import "time"

func AggregateResults(results []*NormalizedResult, latency time.Duration) (*NormalizedResult, error) {
	if len(results) == 0 {
		return nil, &GuardError{Code: ErrorCodeUnavailable}
	}
	aggregate := &NormalizedResult{
		Decision: EventPass, RiskLevel: RiskLow, Action: ActionAllow,
		ScannerScores: map[string]float64{}, ScannerEvidence: map[string]string{},
		PolicyID: DefaultStrategy, PolicyVersion: 1, LatencyMS: int(latency.Milliseconds()),
	}
	for _, result := range results {
		if result == nil {
			return nil, &GuardError{Code: ErrorCodeInvalidResponse}
		}
		mergeResult(aggregate, result)
	}
	return aggregate, nil
}

func mergeResult(dst, src *NormalizedResult) {
	if severityRank(src.Decision) > severityRank(dst.Decision) {
		dst.Decision, dst.RiskLevel, dst.Action = src.Decision, src.RiskLevel, src.Action
		dst.Safety, dst.GuardEndpointID, dst.ScannerVersion = src.Safety, src.GuardEndpointID, src.ScannerVersion
		dst.ScannerBackend, dst.PolicyID, dst.PolicyVersion = src.ScannerBackend, src.PolicyID, src.PolicyVersion
	}
	dst.Categories = mergeStrings(dst.Categories, src.Categories)
	dst.MatchedScanners = mergeStrings(dst.MatchedScanners, src.MatchedScanners)
	dst.UnknownCategories = mergeStrings(dst.UnknownCategories, src.UnknownCategories)
	for k, v := range src.ScannerScores {
		if _, ok := dst.ScannerScores[k]; !ok {
			dst.ScannerScores[k] = v
		}
	}
	for k, v := range src.ScannerEvidence {
		if _, ok := dst.ScannerEvidence[k]; !ok {
			dst.ScannerEvidence[k] = v
		}
	}
}

func severityRank(decision EventDecision) int {
	switch decision {
	case EventCritical:
		return 3
	case EventFlag:
		return 2
	case EventPass:
		return 1
	default:
		return 0
	}
}
