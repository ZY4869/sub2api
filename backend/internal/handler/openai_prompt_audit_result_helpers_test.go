package handler

import "github.com/Wei-Shaw/sub2api/internal/securityaudit"

func wsPromptAuditPassResult() *securityaudit.NormalizedResult {
	return &securityaudit.NormalizedResult{
		Decision: securityaudit.EventPass, RiskLevel: securityaudit.RiskLow, Action: securityaudit.ActionAllow, Safety: "Safe",
		ScannerScores: map[string]float64{}, ScannerEvidence: map[string]string{},
		ScannerBackend: "qwen3guard-openai", ScannerVersion: securityaudit.DefaultGuardModel,
		PolicyID: securityaudit.DefaultStrategy, PolicyVersion: 1,
	}
}

func wsPromptAuditBlockResult() *securityaudit.NormalizedResult {
	return &securityaudit.NormalizedResult{
		Decision: securityaudit.EventCritical, RiskLevel: securityaudit.RiskCritical, Action: securityaudit.ActionBlock, Safety: "Unsafe",
		Categories: []string{"jailbreak"}, MatchedScanners: []string{"jailbreak"},
		ScannerScores: map[string]float64{"jailbreak": 1}, ScannerEvidence: map[string]string{"jailbreak": "Jailbreak"},
		ScannerBackend: "qwen3guard-openai", ScannerVersion: securityaudit.DefaultGuardModel,
		PolicyID: securityaudit.DefaultStrategy, PolicyVersion: 1,
	}
}
