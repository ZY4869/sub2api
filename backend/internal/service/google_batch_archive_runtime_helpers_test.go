package service

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestNormalizeAIStudioJSONLToVertexWrapsRequestLines(t *testing.T) {
	input := strings.Join([]string{
		`{"contents":[{"parts":[{"text":"hello"}]}]}`,
		`{"key":"custom-key","request":{"contents":[{"parts":[{"text":"world"}]}]}}`,
		"",
	}, "\n")

	output, err := normalizeAIStudioJSONLToVertex([]byte(input))
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	require.Len(t, lines, 2)
	require.Equal(t, "request-1", gjson.Get(lines[0], "key").String())
	require.True(t, gjson.Get(lines[0], "request.contents.0.parts.0.text").Exists())
	require.Equal(t, "custom-key", gjson.Get(lines[1], "key").String())
	require.Equal(t, "world", gjson.Get(lines[1], "request.contents.0.parts.0.text").String())
}

func TestBuildArchiveBatchPayloadIncludesArchiveObject(t *testing.T) {
	svc := &GeminiMessagesCompatService{}
	retention := time.Date(2026, 4, 7, 10, 0, 0, 0, time.UTC)
	job := &GoogleBatchArchiveJob{
		PublicBatchName:    "batches/test-batch",
		State:              GoogleBatchArchiveStateSucceeded,
		RequestedModel:     "gemini-2.5-flash",
		ArchiveState:       GoogleBatchArchiveLifecycleArchived,
		RetentionExpiresAt: &retention,
		MetadataJSON: map[string]any{
			googleBatchBindingMetadataPublicResultFileName: "files/test-results",
		},
	}

	body := svc.buildArchiveBatchPayload(job, nil, googleBatchArchiveResponse{
		State:          GoogleBatchArchiveLifecycleArchived,
		ResultFileName: "files/test-results",
		Downloadable:   true,
		Source:         googleBatchArchiveSourceLocal,
		DownloadPath:   "/google/batch/archive/v1beta/files/test-results:download",
	})

	require.Equal(t, "batches/test-batch", gjson.GetBytes(body, "name").String())
	require.Equal(t, "files/test-results", gjson.GetBytes(body, "archive.result_file_name").String())
	require.True(t, gjson.GetBytes(body, "archive.downloadable").Bool())
	require.Equal(t, googleBatchArchiveSourceLocal, gjson.GetBytes(body, "archive.source").String())
}

func TestIsGoogleBatchQuotaFallbackResponse(t *testing.T) {
	require.True(t, isGoogleBatchQuotaFallbackResponse(&UpstreamHTTPResult{StatusCode: 429}))
	require.True(t, isGoogleBatchQuotaFallbackResponse(&UpstreamHTTPResult{StatusCode: 403, Body: []byte(`{"error":"quota exceeded"}`)}))
	require.False(t, isGoogleBatchQuotaFallbackResponse(&UpstreamHTTPResult{StatusCode: 403, Body: []byte(`{"error":"permission denied"}`)}))
	require.False(t, isGoogleBatchQuotaFallbackResponse(&UpstreamHTTPResult{StatusCode: 500, Body: []byte(`{"error":"quota exceeded"}`)}))
}

func TestSelectGoogleBatchResultObjectPrefersJSONL(t *testing.T) {
	items := []googleBatchGCSObject{
		{Name: "prefix/input.jsonl"},
		{Name: "prefix/metadata.json"},
		{Name: "prefix/predictions_0001.jsonl"},
	}

	selected := selectGoogleBatchResultObject(items, "prefix/input.jsonl")
	require.NotNil(t, selected)
	require.Equal(t, "prefix/predictions_0001.jsonl", selected.Name)
}

func TestGoogleBatchForwardInputOpenRequestBodyReplaysBufferedBody(t *testing.T) {
	input := GoogleBatchForwardInput{
		Method:        http.MethodPost,
		Path:          "/v1beta/batches",
		Body:          []byte(`{"ok":true}`),
		ContentLength: int64(len(`{"ok":true}`)),
	}

	first, err := input.OpenRequestBody()
	require.NoError(t, err)
	defer func() { _ = first.Close() }()
	firstBody, err := io.ReadAll(first)
	require.NoError(t, err)

	second, err := input.OpenRequestBody()
	require.NoError(t, err)
	defer func() { _ = second.Close() }()
	secondBody, err := io.ReadAll(second)
	require.NoError(t, err)

	require.Equal(t, firstBody, secondBody)
	require.Equal(t, `{"ok":true}`, string(firstBody))
}

func TestNormalizeAIStudioJSONLToVertexStreamPreservesRawLineIndexKeys(t *testing.T) {
	input := "\n" + strings.Join([]string{
		`{"contents":[{"parts":[{"text":"hello"}]}]}`,
		`{"contents":[{"parts":[{"text":"world"}]}]}`,
	}, "\n") + "\n"

	var output bytes.Buffer
	err := normalizeAIStudioJSONLToVertexStream(strings.NewReader(input), &output)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	require.Len(t, lines, 2)
	require.Equal(t, "request-2", gjson.Get(lines[0], "key").String())
	require.Equal(t, "request-3", gjson.Get(lines[1], "key").String())
}

func TestMergeGoogleBatchNamedListItemsStableDedupeAndSort(t *testing.T) {
	merged := mergeGoogleBatchNamedListItems([][]map[string]any{
		{
			{"name": "files/zeta", "source": "first"},
			{"name": "files/alpha", "source": "first"},
		},
		{
			{"name": "files/alpha", "source": "second"},
			{"name": "files/beta", "source": "second"},
		},
	})

	require.Len(t, merged, 3)
	require.Equal(t, "files/alpha", merged[0]["name"])
	require.Equal(t, "first", merged[0]["source"])
	require.Equal(t, "files/beta", merged[1]["name"])
	require.Equal(t, "files/zeta", merged[2]["name"])
}
