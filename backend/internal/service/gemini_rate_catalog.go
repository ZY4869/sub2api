package service

import "context"

const (
	geminiRateCatalogEffectiveDate     = "2026-03-31"
	googleBatchInputFileSizeLimitBytes = 2 * 1024 * 1024 * 1024
	googleBatchFileStorageLimitBytes   = 20 * 1024 * 1024 * 1024
)

func (s *SettingService) GetGeminiRateCatalog(_ context.Context) *GeminiRateCatalog {
	return defaultGeminiRateCatalog()
}

func defaultGeminiRateCatalog() *GeminiRateCatalog {
	return &GeminiRateCatalog{
		EffectiveDate:              geminiRateCatalogEffectiveDate,
		RemainingQuotaAPISupported: false,
		AIStudioTiers: []GeminiRateCatalogTier{
			{
				TierID:         GeminiTierAIStudioFree,
				DisplayName:    "Free",
				Qualification:  "Active project or free trial",
				BillingTierCap: "N/A",
				ModelFamilies: []GeminiRateCatalogModelRow{
					{ModelFamily: "gemini_pro", DisplayName: "Gemini Pro family", RPM: 2, TPM: 0, RPD: 50, Notes: "Exact TPM/RPM per model may vary in AI Studio console."},
					{ModelFamily: "gemini_flash", DisplayName: "Gemini Flash family", RPM: 15, TPM: 0, RPD: 1500, Notes: "Exact TPM/RPM per model may vary in AI Studio console."},
				},
			},
			{
				TierID:         GeminiTierAIStudioTier1,
				DisplayName:    "Tier 1",
				Qualification:  "Set up and link an active billing account",
				BillingTierCap: "$250",
				ModelFamilies: []GeminiRateCatalogModelRow{
					{ModelFamily: "gemini_pro", DisplayName: "Gemini Pro family", RPM: 1000, TPM: 0, RPD: -1, Notes: "Exact per-model paid limits are shown in AI Studio rate limits."},
					{ModelFamily: "gemini_flash", DisplayName: "Gemini Flash family", RPM: 2000, TPM: 0, RPD: -1, Notes: "Exact per-model paid limits are shown in AI Studio rate limits."},
				},
			},
			{
				TierID:         GeminiTierAIStudioTier2,
				DisplayName:    "Tier 2",
				Qualification:  "Paid $100 + 3 days from first successful payment",
				BillingTierCap: "$2,000",
				ModelFamilies: []GeminiRateCatalogModelRow{
					{ModelFamily: "gemini_pro", DisplayName: "Gemini Pro family", RPM: 1000, TPM: 0, RPD: -1, Notes: "Exact per-model paid limits are shown in AI Studio rate limits."},
					{ModelFamily: "gemini_flash", DisplayName: "Gemini Flash family", RPM: 2000, TPM: 0, RPD: -1, Notes: "Exact per-model paid limits are shown in AI Studio rate limits."},
				},
			},
			{
				TierID:         GeminiTierAIStudioTier3,
				DisplayName:    "Tier 3",
				Qualification:  "Paid $1,000 + 30 days from first successful payment",
				BillingTierCap: "$20,000 - $100,000+",
				ModelFamilies: []GeminiRateCatalogModelRow{
					{ModelFamily: "gemini_pro", DisplayName: "Gemini Pro family", RPM: 1000, TPM: 0, RPD: -1, Notes: "Exact per-model paid limits are shown in AI Studio rate limits."},
					{ModelFamily: "gemini_flash", DisplayName: "Gemini Flash family", RPM: 2000, TPM: 0, RPD: -1, Notes: "Exact per-model paid limits are shown in AI Studio rate limits."},
				},
			},
		},
		BatchLimits: GeminiRateCatalogBatchLimits{
			ConcurrentBatchRequests: 100,
			InputFileSizeLimitBytes: googleBatchInputFileSizeLimitBytes,
			FileStorageLimitBytes:   googleBatchFileStorageLimitBytes,
			ByTier: []GeminiRateCatalogBatchTier{
				{
					TierID: GeminiTierAIStudioFree,
					Entries: []GeminiRateCatalogBatchRow{
						{ModelFamily: "gemini_pro", DisplayName: "Gemini 2.5 Pro", EnqueuedTokens: 5000000},
						{ModelFamily: "gemini_flash", DisplayName: "Gemini 2.5 Flash", EnqueuedTokens: 3000000},
						{ModelFamily: "gemini_flash_lite", DisplayName: "Gemini 2.5 Flash-Lite", EnqueuedTokens: 10000000},
						{ModelFamily: "gemini_2_flash", DisplayName: "Gemini 2.0 Flash", EnqueuedTokens: 10000000},
					},
				},
				{
					TierID: GeminiTierAIStudioTier2,
					Entries: []GeminiRateCatalogBatchRow{
						{ModelFamily: "gemini_pro", DisplayName: "Gemini 2.5 Pro", EnqueuedTokens: 500000000},
						{ModelFamily: "gemini_flash", DisplayName: "Gemini 2.5 Flash", EnqueuedTokens: 400000000},
						{ModelFamily: "gemini_flash_lite", DisplayName: "Gemini 2.5 Flash-Lite", EnqueuedTokens: 500000000},
						{ModelFamily: "gemini_2_flash", DisplayName: "Gemini 2.0 Flash", EnqueuedTokens: 1000000000},
					},
				},
				{
					TierID: GeminiTierAIStudioTier3,
					Entries: []GeminiRateCatalogBatchRow{
						{ModelFamily: "gemini_pro", DisplayName: "Gemini 2.5 Pro", EnqueuedTokens: 1000000000},
						{ModelFamily: "gemini_flash", DisplayName: "Gemini 2.5 Flash", EnqueuedTokens: 1000000000},
						{ModelFamily: "gemini_flash_lite", DisplayName: "Gemini 2.5 Flash-Lite", EnqueuedTokens: 1000000000},
						{ModelFamily: "gemini_2_flash", DisplayName: "Gemini 2.0 Flash", EnqueuedTokens: 5000000000},
					},
				},
			},
		},
		Links: []GeminiRateCatalogLink{
			{Label: "AI Studio rate limits", URL: "https://ai.google.dev/gemini-api/docs/rate-limits"},
			{Label: "AI Studio batch API", URL: "https://ai.google.dev/gemini-api/docs/batch-api"},
			{Label: "AI Studio files API", URL: "https://ai.google.dev/gemini-api/docs/files"},
			{Label: "AI Studio billing", URL: "https://ai.google.dev/gemini-api/docs/billing"},
			{Label: "AI Studio projects", URL: "https://aistudio.google.com/projects"},
			{Label: "Vertex AI quotas", URL: "https://cloud.google.com/vertex-ai/generative-ai/docs/quotas"},
		},
		Notes: []string{
			"AI Studio Tier qualification and Batch API limits are based on the official public docs as of 2026-03-31.",
			"Exact per-model synchronous RPM/TPM/RPD may vary by model and are surfaced in AI Studio rate limit pages and console.",
			"No public API for real-time remaining tier balance or remaining quota was found in official docs.",
		},
	}
}
