package service

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestExtractOpenAIRequestMetaFromBody(t *testing.T) {
	tests := []struct {
		name          string
		body          []byte
		wantModel     string
		wantStream    bool
		wantPromptKey string
	}{
		{
			name:          "reads complete payload",
			body:          []byte(`{"model":"gpt-5","stream":true,"prompt_cache_key":" ses-1 "}`),
			wantModel:     "gpt-5",
			wantStream:    true,
			wantPromptKey: "ses-1",
		},
		{
			name:          "handles missing optional fields",
			body:          []byte(`{"model":"gpt-4"}`),
			wantModel:     "gpt-4",
			wantStream:    false,
			wantPromptKey: "",
		},
		{
			name:          "handles empty body",
			body:          nil,
			wantModel:     "",
			wantStream:    false,
			wantPromptKey: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, stream, promptKey := extractOpenAIRequestMetaFromBody(tt.body)
			require.Equal(t, tt.wantModel, model)
			require.Equal(t, tt.wantStream, stream)
			require.Equal(t, tt.wantPromptKey, promptKey)
		})
	}
}

func TestExtractOpenAIReasoningEffortFromBody(t *testing.T) {
	tests := []struct {
		name      string
		body      []byte
		model     string
		wantNil   bool
		wantValue string
	}{
		{
			name:      "prefers reasoning.effort",
			body:      []byte(`{"reasoning":{"effort":"medium"}}`),
			model:     "gpt-5-high",
			wantNil:   false,
			wantValue: "medium",
		},
		{
			name:      "supports reasoning_effort alias",
			body:      []byte(`{"reasoning_effort":"x-high"}`),
			model:     "",
			wantNil:   false,
			wantValue: "xhigh",
		},
		{
			name:      "maps max reasoning_effort alias to xhigh",
			body:      []byte(`{"reasoning_effort":"max"}`),
			model:     "",
			wantNil:   false,
			wantValue: "xhigh",
		},
		{
			name:      "keeps max reasoning_effort for gpt 5.6",
			body:      []byte(`{"reasoning_effort":"max"}`),
			model:     "gpt-5.6-sol",
			wantNil:   false,
			wantValue: "max",
		},
		{
			name:      "normalizes minimal to none",
			body:      []byte(`{"reasoning":{"effort":"minimal"}}`),
			model:     "gpt-5-high",
			wantNil:   false,
			wantValue: "none",
		},
		{
			name:      "derives from model suffix when body missing",
			body:      []byte(`{"input":"hi"}`),
			model:     "gpt-5-high",
			wantNil:   false,
			wantValue: "high",
		},
		{
			name:    "unknown suffix returns nil",
			body:    []byte(`{"input":"hi"}`),
			model:   "gpt-5-unknown",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractOpenAIReasoningEffortFromBody(tt.body, tt.model)
			if tt.wantNil {
				require.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			require.Equal(t, tt.wantValue, *got)
		})
	}
}

func TestNormalizeOpenAIRequestBodyEffort_UsesMappedGpt56Candidate(t *testing.T) {
	reqBody := map[string]any{
		"model":            "public-gpt56-alias",
		"reasoning_effort": "max",
	}

	resolution := normalizeOpenAIRequestBodyEffort(reqBody, "public-gpt56-alias", "gpt-5.6-terra")

	require.NotNil(t, resolution.Raw)
	require.NotNil(t, resolution.Effective)
	require.Equal(t, "max", *resolution.Raw)
	require.Equal(t, "max", *resolution.Effective)
	require.NotContains(t, reqBody, "reasoning_effort")
	reasoning, ok := reqBody["reasoning"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "max", reasoning["effort"])
}

func TestNormalizeOpenAIRequestBodyEffort_DerivesMaxFromMappedGpt56Suffix(t *testing.T) {
	reqBody := map[string]any{
		"model": "public-gpt56-alias",
	}

	resolution := normalizeOpenAIRequestBodyEffort(reqBody, "public-gpt56-alias", "gpt-5.6-sol-max")

	require.NotNil(t, resolution.Raw)
	require.NotNil(t, resolution.Effective)
	require.Equal(t, "max", *resolution.Raw)
	require.Equal(t, "max", *resolution.Effective)
	require.Equal(t, effortSourceModelSuffix, resolution.Source)
	reasoning, ok := reqBody["reasoning"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "max", reasoning["effort"])
}

func TestExtractOpenAIReasoningEffortFromBody_DerivesFromMappedCandidate(t *testing.T) {
	resolution := extractOpenAIReasoningEffortResolutionFromBody(
		[]byte(`{"model":"public-gpt56-alias","input":"hi"}`),
		"public-gpt56-alias",
		"gpt-5.6-luna-max",
	)

	require.NotNil(t, resolution.Raw)
	require.NotNil(t, resolution.Effective)
	require.Equal(t, "max", *resolution.Raw)
	require.Equal(t, "max", *resolution.Effective)
	require.Equal(t, effortSourceModelSuffix, resolution.Source)
}

func TestGetOpenAIRequestBodyMap_UsesContextCache(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	cached := map[string]any{"model": "cached-model", "stream": true}
	c.Set(OpenAIParsedRequestBodyKey, cached)

	got, err := getOpenAIRequestBodyMap(c, []byte(`{invalid-json`))
	require.NoError(t, err)
	require.Equal(t, cached, got)
}

func TestGetOpenAIRequestBodyMap_ParseErrorWithoutCache(t *testing.T) {
	_, err := getOpenAIRequestBodyMap(nil, []byte(`{invalid-json`))
	require.Error(t, err)
	require.Contains(t, err.Error(), "parse request")
}

func TestGetOpenAIRequestBodyMap_WriteBackContextCache(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	got, err := getOpenAIRequestBodyMap(c, []byte(`{"model":"gpt-5","stream":true}`))
	require.NoError(t, err)
	require.Equal(t, "gpt-5", got["model"])

	cached, ok := c.Get(OpenAIParsedRequestBodyKey)
	require.True(t, ok)
	cachedMap, ok := cached.(map[string]any)
	require.True(t, ok)
	require.Equal(t, got, cachedMap)
}

func TestExtractOpenAIReasoningEffortFromBody_IgnoresTopLevelEffortLevel(t *testing.T) {
	got := extractOpenAIReasoningEffortFromBody([]byte(`{"effortLevel":"max","input":"hi"}`), "gpt-5.4")
	require.Nil(t, got)
}
