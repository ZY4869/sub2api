package routes

import "github.com/Wei-Shaw/sub2api/internal/securityaudit"

func promptAuditRouteConfig() securityaudit.UpdateConfigRequest {
	return securityaudit.UpdateConfigRequest{
		Enabled: true, BlockingEnabled: false, StorePassEvents: false,
		Strategy: securityaudit.DefaultStrategy, WorkerCount: 1, QueueCapacity: 8,
		Scanners: securityaudit.AllScannerIDs, AllGroups: true,
		Endpoints: []securityaudit.UpdateEndpoint{{
			ID: "primary", Name: "Primary", Protocol: "openai_compatible",
			BaseURL: "https://guard.example.com/v1", Model: securityaudit.DefaultGuardModel,
			TimeoutMS: securityaudit.DefaultTimeoutMS, InputLimit: securityaudit.DefaultInputLimit, Enabled: true,
		}},
	}
}

func promptAuditRoutePassResult() *securityaudit.NormalizedResult {
	return &securityaudit.NormalizedResult{
		Decision: securityaudit.EventPass, RiskLevel: securityaudit.RiskLow, Action: securityaudit.ActionAllow, Safety: "Safe",
		ScannerBackend: "qwen3guard-openai", ScannerVersion: securityaudit.DefaultGuardModel,
		PolicyID: securityaudit.DefaultStrategy, PolicyVersion: 1,
		ScannerScores: map[string]float64{}, ScannerEvidence: map[string]string{},
	}
}
