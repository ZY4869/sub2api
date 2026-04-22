package service

import (
	"bytes"
	"mime/multipart"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDetectAndRewriteOpenAIImageRequestModel_JSON(t *testing.T) {
	body := []byte(`{"model":"gpt-image-2","prompt":"hello"}`)

	model, err := DetectOpenAIImageRequestModel(body, "application/json")
	require.NoError(t, err)
	require.Equal(t, "gpt-image-2", model)

	rewritten, contentType, err := RewriteOpenAIImageRequestModel(body, "application/json", "gpt-image-2-preview")
	require.NoError(t, err)
	require.Equal(t, "application/json", contentType)
	require.JSONEq(t, `{"model":"gpt-image-2-preview","prompt":"hello"}`, string(rewritten))
}

func TestDetectAndRewriteOpenAIImageRequestModel_Multipart(t *testing.T) {
	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)
	require.NoError(t, writer.WriteField("model", "gpt-image-2"))
	require.NoError(t, writer.WriteField("size", "1024x1024"))
	require.NoError(t, writer.Close())

	contentType := writer.FormDataContentType()
	model, err := DetectOpenAIImageRequestModel(buffer.Bytes(), contentType)
	require.NoError(t, err)
	require.Equal(t, "gpt-image-2", model)
	require.Equal(t, "1024x1024", DetectOpenAIImageRequestSize(buffer.Bytes(), contentType))

	rewritten, rewrittenType, err := RewriteOpenAIImageRequestModel(buffer.Bytes(), contentType, "gpt-image-2-preview")
	require.NoError(t, err)
	require.Contains(t, rewrittenType, "multipart/form-data")

	model, err = DetectOpenAIImageRequestModel(rewritten, rewrittenType)
	require.NoError(t, err)
	require.Equal(t, "gpt-image-2-preview", model)
	require.Equal(t, "1024x1024", DetectOpenAIImageRequestSize(rewritten, rewrittenType))
}

func TestOpenAIGatewayService_DetectResponsesImageGenerationToolModel(t *testing.T) {
	toolModel, ok := DetectOpenAIResponsesImageGenerationToolModel([]byte(`{
		"model":"gpt-5.4-mini",
		"tools":[
			{"type":"web_search_preview"},
			{"type":"image_generation","model":"gpt-image-2"}
		]
	}`))

	require.True(t, ok)
	require.Equal(t, "gpt-image-2", toolModel)
}

func TestOpenAIGatewayService_DetectResponsesImageGenerationToolModel_IgnoresNonImageTools(t *testing.T) {
	toolModel, ok := DetectOpenAIResponsesImageGenerationToolModel([]byte(`{
		"model":"gpt-5.4-mini",
		"tools":[{"type":"web_search_preview"}]
	}`))

	require.False(t, ok)
	require.Empty(t, toolModel)
}
