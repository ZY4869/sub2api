package service

import "testing"

import "github.com/stretchr/testify/require"

func TestIsGeminiNonBillablePassthroughEndpoint_StrictOfficialSecondStageSurfaces(t *testing.T) {
	tests := []string{
		"/v1beta/corpora",
		"/v1beta/corpora/corpus-1/operations/op-1",
		"/v1beta/corpora/corpus-1/permissions/perm-1",
		"/v1beta/dynamic/dynamic-1:generateContent",
		"/v1beta/generatedFiles",
		"/v1beta/generatedFiles/generated-file-1/operations/op-1",
		"/v1beta/models/gemini-2.5-pro/operations",
		"/v1beta/tunedModels",
		"/v1beta/tunedModels/tuned-model-1:asyncBatchEmbedContent",
		"/v1beta/tunedModels/tuned-model-1/permissions/perm-1",
		"/v1beta/tunedModels/tuned-model-1/operations/op-1",
	}

	for _, path := range tests {
		t.Run(path, func(t *testing.T) {
			require.True(t, isGeminiNonBillablePassthroughEndpoint(path))
			require.False(t, isGeminiBillingEndpoint(path))
		})
	}
}
