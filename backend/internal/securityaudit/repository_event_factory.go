package securityaudit

func eventFromJob(job *Job, result *NormalizedResult, fullPrompt string) *Event {
	return &Event{
		JobID: job.ID, RequestID: job.RequestID, UserID: cloneInt64(job.UserID),
		Username: job.Username, UserEmail: job.UserEmail, APIKeyID: cloneInt64(job.APIKeyID),
		APIKeyName: job.APIKeyName, GroupID: cloneInt64(job.GroupID), GroupName: job.GroupName,
		Provider: job.Provider, Endpoint: job.Endpoint, Protocol: job.Protocol, Model: job.Model,
		PromptHash: job.PromptHash, RedactedPreview: job.RedactedPreview, FullPrompt: fullPrompt,
		Stage: job.Stage, Decision: result.Decision, RiskLevel: result.RiskLevel, Action: result.Action,
		Categories: mergeStrings(result.Categories, result.UnknownCategories), MatchedScanners: result.MatchedScanners,
		ScannerScores: result.ScannerScores, ScannerEvidence: result.ScannerEvidence,
		ScannerBackend: result.ScannerBackend, ScannerVersion: result.ScannerVersion,
		GuardEndpointID: result.GuardEndpointID, PolicyID: result.PolicyID, PolicyVersion: result.PolicyVersion,
		ConfigVersion: job.ConfigVersion, ChunkTotal: 1, LatencyMS: result.LatencyMS,
		IssueSummaries: BuildIssueSummaries(*result),
	}
}
