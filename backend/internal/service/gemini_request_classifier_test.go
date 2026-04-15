package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGeminiRequestClassifier_ClassifyRequest_FileSearchEmbeddingAndRetrieval(t *testing.T) {
	classifier := NewGeminiRequestClassifier()

	embedding := classifier.ClassifyRequest(GeminiBillingCalculationInput{
		Model:           "gemini-2.5-pro",
		InboundEndpoint: "/v1beta/fileSearchStores/default/documents:import",
		RequestBody:     []byte(`{"documents":[{"id":"doc-1"}]}`),
	})
	require.Equal(t, "file_search_embedding", embedding.OperationType)
	require.Equal(t, BillingSurfaceGeminiNative, embedding.Surface)

	retrieval := classifier.ClassifyRequest(GeminiBillingCalculationInput{
		Model:           "gemini-2.5-pro",
		InboundEndpoint: "/v1beta/fileSearchStores/default:search",
		RequestBody:     []byte(`{"query":"hello","tools":[{"fileSearch":{}}]}`),
	})
	require.Equal(t, "file_search_retrieval", retrieval.OperationType)
	require.Equal(t, "file_search_retrieval", retrieval.ChargeSource)
}

func TestGeminiRequestClassifier_ClassifyRequest_GroundingAndAudioModalities(t *testing.T) {
	classifier := NewGeminiRequestClassifier()

	search := classifier.ClassifyRequest(GeminiBillingCalculationInput{
		Model:           "gemini-2.5-pro",
		InboundEndpoint: "/v1beta/models/gemini-2.5-pro:generateContent",
		RequestBody:     []byte(`{"tools":[{"googleSearch":{}}]}`),
	})
	require.Equal(t, "search", search.GroundingKind)
	require.Equal(t, "grounding", search.ChargeSource)

	maps := classifier.ClassifyRequest(GeminiBillingCalculationInput{
		Model:           "gemini-2.5-pro",
		InboundEndpoint: "/v1beta/models/gemini-2.5-pro:generateContent",
		RequestBody:     []byte(`{"tools":[{"googleMaps":{}}]}`),
	})
	require.Equal(t, "maps", maps.GroundingKind)

	audio := classifier.ClassifyRequest(GeminiBillingCalculationInput{
		Model:           "gemini-2.5-tts",
		InboundEndpoint: "/v1beta/models/gemini-2.5-tts:generateContent",
		RequestBody:     []byte(`{"generationConfig":{"responseModalities":["AUDIO"]}}`),
	})
	require.Equal(t, "audio", audio.OutputModality)
	require.Equal(t, BillingServiceTierStandard, audio.ServiceTier)
}

func TestGeminiRequestClassifier_ClassifyRequest_AuthTokensUsesLiveSurface(t *testing.T) {
	classifier := NewGeminiRequestClassifier()

	authTokens := classifier.ClassifyRequest(GeminiBillingCalculationInput{
		InboundEndpoint: "/v1alpha/authTokens",
		RequestBody:     []byte(`{"ttl":60}`),
	})

	require.Equal(t, BillingSurfaceGeminiLive, authTokens.Surface)
	require.Equal(t, "auth_tokens", authTokens.OperationType)
}

func TestGeminiRequestClassifier_ClassifyRequest_ServiceTierModes(t *testing.T) {
	classifier := NewGeminiRequestClassifier()

	standard := classifier.ClassifyRequest(GeminiBillingCalculationInput{
		InboundEndpoint: "/v1/models/gemini-2.5-pro:generateContent",
		RequestBody:     []byte(`{"contents":[{"parts":[{"text":"hi"}]}]}`),
	})
	require.Equal(t, "generate_content", standard.OperationType)
	require.Equal(t, BillingServiceTierStandard, standard.RequestedMode)
	require.Equal(t, BillingServiceTierStandard, standard.ResolvedMode)
	require.Equal(t, BillingServiceTierStandard, standard.RequestedServiceTier)
	require.Equal(t, BillingServiceTierStandard, standard.ServiceTier)
	require.False(t, standard.ServiceTierExplicit)
	require.Equal(t, BillingBatchModeRealtime, standard.BatchMode)

	flex := classifier.ClassifyRequest(GeminiBillingCalculationInput{
		InboundEndpoint: "/v1/models/gemini-2.5-pro:streamGenerateContent",
		RequestBody:     []byte(`{"service_tier":"flex","contents":[{"parts":[{"text":"hi"}]}]}`),
	})
	require.Equal(t, "generate_content", flex.OperationType)
	require.Equal(t, BillingServiceTierFlex, flex.RequestedMode)
	require.Equal(t, BillingServiceTierFlex, flex.ResolvedMode)
	require.Equal(t, BillingServiceTierFlex, flex.RequestedServiceTier)
	require.Equal(t, BillingServiceTierFlex, flex.ServiceTier)
	require.True(t, flex.ServiceTierExplicit)

	interactionFlex := classifier.ClassifyRequest(GeminiBillingCalculationInput{
		InboundEndpoint: "/v1beta/interactions",
		RequestBody:     []byte(`{"service_tier":"flex","model":"gemini-2.5-pro","input":{"text":"hi"}}`),
	})
	require.Equal(t, BillingSurfaceInteractions, interactionFlex.Surface)
	require.Equal(t, "interaction", interactionFlex.OperationType)
	require.Equal(t, BillingServiceTierFlex, interactionFlex.RequestedServiceTier)
	require.Equal(t, BillingServiceTierFlex, interactionFlex.ServiceTier)
	require.Equal(t, BillingServiceTierFlex, interactionFlex.ResolvedMode)

	answer := classifier.ClassifyRequest(GeminiBillingCalculationInput{
		InboundEndpoint: "/v1beta/models/gemini-2.5-pro:generateAnswer",
		RequestBody:     []byte(`{"contents":[{"parts":[{"text":"hi"}]}]}`),
	})
	require.Equal(t, "generate_content", answer.OperationType)
	require.Equal(t, BillingBatchModeRealtime, answer.BatchMode)
}

func TestGeminiRequestClassifier_ClassifyRequest_BatchAndCacheModesDoNotKeepRealtimeTier(t *testing.T) {
	classifier := NewGeminiRequestClassifier()

	batch := classifier.ClassifyRequest(GeminiBillingCalculationInput{
		InboundEndpoint:      "/v1beta/models/gemini-2.5-pro:batchGenerateContent",
		RequestedServiceTier: BillingServiceTierFlex,
		RequestBody:          []byte(`{"contents":[{"parts":[{"text":"hi"}]}]}`),
	})
	require.Equal(t, "generate_content", batch.OperationType)
	require.Equal(t, BillingServiceTierFlex, batch.RequestedMode)
	require.Equal(t, BillingBatchModeBatch, batch.ResolvedMode)
	require.Equal(t, BillingServiceTierFlex, batch.RequestedServiceTier)
	require.Equal(t, BillingServiceTierStandard, batch.ServiceTier)
	require.True(t, batch.ServiceTierExplicit)
	require.Equal(t, BillingBatchModeBatch, batch.BatchMode)
	require.Equal(t, "service_tier_ignored_for_batch", resolveGeminiModeFallbackReason(batch))

	cache := classifier.ClassifyRequest(GeminiBillingCalculationInput{
		InboundEndpoint: "/v1beta/cachedContents/cache-1",
		RequestBody:     []byte(`{"service_tier":"priority"}`),
	})
	require.Equal(t, BillingServiceTierPriority, cache.RequestedMode)
	require.Equal(t, "cache", cache.ResolvedMode)
	require.Equal(t, BillingServiceTierPriority, cache.RequestedServiceTier)
	require.Equal(t, BillingServiceTierStandard, cache.ServiceTier)
	require.True(t, cache.ServiceTierExplicit)
	require.Equal(t, "read", cache.CachePhase)
	require.Equal(t, "cache", cache.ChargeSource)
	require.Equal(t, "service_tier_ignored_for_cache", resolveGeminiModeFallbackReason(cache))
}

func TestGeminiRequestClassifier_ClassifyRequest_NonRealtimeServiceTierLeavesAuditSignal(t *testing.T) {
	classifier := NewGeminiRequestClassifier()

	countTokens := classifier.ClassifyRequest(GeminiBillingCalculationInput{
		InboundEndpoint: "/v1/models/gemini-2.5-pro:countTokens",
		RequestBody:     []byte(`{"serviceTier":"priority","contents":[{"parts":[{"text":"hi"}]}]}`),
	})

	require.Equal(t, "count_tokens", countTokens.OperationType)
	require.Equal(t, BillingServiceTierPriority, countTokens.RequestedMode)
	require.Equal(t, BillingServiceTierStandard, countTokens.ResolvedMode)
	require.Equal(t, BillingServiceTierPriority, countTokens.RequestedServiceTier)
	require.Equal(t, BillingServiceTierStandard, countTokens.ServiceTier)
	require.True(t, countTokens.ServiceTierExplicit)
	require.Equal(t, "service_tier_ignored_for_non_realtime_operation", resolveGeminiModeFallbackReason(countTokens))
}

func TestGeminiRequestClassifier_ClassifyRequest_ResolvedServiceTierCanDowngradeRequestedTier(t *testing.T) {
	classifier := NewGeminiRequestClassifier()

	interaction := classifier.ClassifyRequest(GeminiBillingCalculationInput{
		InboundEndpoint:      "/v1beta/interactions",
		RequestedServiceTier: BillingServiceTierPriority,
		ResolvedServiceTier:  BillingServiceTierStandard,
		RequestBody:          []byte(`{"model":"gemini-2.5-pro","input":{"text":"hi"}}`),
	})

	require.Equal(t, BillingServiceTierPriority, interaction.RequestedServiceTier)
	require.Equal(t, BillingServiceTierStandard, interaction.ServiceTier)
	require.True(t, interaction.ServiceTierDowngraded)
	require.Equal(t, BillingServiceTierPriority, interaction.RequestedMode)
	require.Equal(t, BillingServiceTierStandard, interaction.ResolvedMode)
	require.Equal(t, "service_tier_downgraded_by_upstream", resolveGeminiModeFallbackReason(interaction))
}
