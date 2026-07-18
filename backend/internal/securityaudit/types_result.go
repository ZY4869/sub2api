package securityaudit

type NormalizedResult struct {
	Decision          EventDecision
	RiskLevel         RiskLevel
	Action            Action
	Safety            string
	Categories        []string
	MatchedScanners   []string
	ScannerScores     map[string]float64
	ScannerEvidence   map[string]string
	UnknownCategories []string
	ScannerBackend    string
	ScannerVersion    string
	GuardEndpointID   string
	PolicyID          string
	PolicyVersion     int
	LatencyMS         int
}
