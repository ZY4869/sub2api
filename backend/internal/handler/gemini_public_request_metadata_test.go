package handler

import "testing"

import "github.com/stretchr/testify/require"

func TestDeriveGeminiPublicPathMetadata_StrictOfficialSecondStageResources(t *testing.T) {
	tests := []struct {
		path     string
		version  string
		resource string
	}{
		{path: "/v1beta/corpora/corpus-1/operations/op-1", version: "v1beta", resource: "corpora_operations"},
		{path: "/v1beta/generatedFiles/generated-file-1/operations/op-1", version: "v1beta", resource: "generated_files_operations"},
		{path: "/v1beta/models/gemini-2.5-pro/operations", version: "v1beta", resource: "model_operations"},
		{path: "/v1beta/models/gemini-2.5-pro:generateAnswer", version: "v1beta", resource: "models"},
		{path: "/v1beta/fileSearchStores/default-store:importFile", version: "v1beta", resource: "file_search_stores"},
		{path: "/v1beta/tunedModels/tuned-model-1/permissions/perm-1", version: "v1beta", resource: "tuned_models_permissions"},
		{path: "/v1beta/tunedModels/tuned-model-1/operations/op-1", version: "v1beta", resource: "tuned_models_operations"},
		{path: "/v1beta/tunedModels/tuned-model-1:asyncBatchEmbedContent", version: "v1beta", resource: "tuned_models"},
		{path: "/v1beta/dynamic/dynamic-1:generateContent", version: "v1beta", resource: "dynamic"},
	}

	for _, tt := range tests {
		t.Run(tt.resource, func(t *testing.T) {
			meta := deriveGeminiPublicPathMetadata(tt.path)
			require.Equal(t, tt.version, meta.version)
			require.Equal(t, tt.resource, meta.resource)
			require.False(t, meta.aliasUsed)
			require.Equal(t, tt.path, meta.upstreamPath)
		})
	}
}
