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
