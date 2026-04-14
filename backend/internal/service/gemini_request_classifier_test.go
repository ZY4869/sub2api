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
	require.Equal(t, BillingServiceTierStandard, standard.RequestedServiceTier)
	require.Equal(t, BillingServiceTierStandard, standard.ServiceTier)
	require.Equal(t, BillingBatchModeRealtime, standard.BatchMode)

	flex := classifier.ClassifyRequest(GeminiBillingCalculationInput{
		InboundEndpoint: "/v1/models/gemini-2.5-pro:streamGenerateContent",
		RequestBody:     []byte(`{"service_tier":"flex","contents":[{"parts":[{"text":"hi"}]}]}`),
	})
	require.Equal(t, "generate_content", flex.OperationType)
	require.Equal(t, BillingServiceTierFlex, flex.RequestedServiceTier)
	require.Equal(t, BillingServiceTierFlex, flex.ServiceTier)
}

func TestGeminiRequestClassifier_ClassifyRequest_BatchAndCacheModesDoNotKeepRealtimeTier(t *testing.T) {
	classifier := NewGeminiRequestClassifier()

	batch := classifier.ClassifyRequest(GeminiBillingCalculationInput{
		InboundEndpoint:      "/v1beta/models/gemini-2.5-pro:batchGenerateContent",
		RequestedServiceTier: BillingServiceTierFlex,
		RequestBody:          []byte(`{"contents":[{"parts":[{"text":"hi"}]}]}`),
	})
	require.Equal(t, "generate_content", batch.OperationType)
	require.Equal(t, BillingServiceTierFlex, batch.RequestedServiceTier)
	require.Equal(t, BillingServiceTierStandard, batch.ServiceTier)
	require.Equal(t, BillingBatchModeBatch, batch.BatchMode)

	cache := classifier.ClassifyRequest(GeminiBillingCalculationInput{
		InboundEndpoint: "/v1beta/cachedContents/cache-1",
		RequestBody:     []byte(`{"service_tier":"priority"}`),
	})
	require.Equal(t, BillingServiceTierPriority, cache.RequestedServiceTier)
	require.Equal(t, BillingServiceTierStandard, cache.ServiceTier)
	require.Equal(t, "read", cache.CachePhase)
	require.Equal(t, "cache", cache.ChargeSource)
}
