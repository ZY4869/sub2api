package service

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateOpenAIImageCapabilities_GPTImage2Profiles(t *testing.T) {
	t.Parallel()

	t.Run("allows transparent background", func(t *testing.T) {
		profile, err := ValidateOpenAIImageCapabilities(&NormalizedImageRequest{
			Operation:    "generation",
			Prompt:       "poster",
			Background:   "transparent",
			OutputFormat: "png",
		}, OpenAIImageProtocolModeCompat, OpenAICompatImageTargetModel)
		require.NoError(t, err)
		require.Equal(t, "openai_image.compat.gpt-image-2.transparent_on.custom_resolution_on", profile.ID)
	})

	t.Run("allows 4k custom size on gpt-image-2", func(t *testing.T) {
		profile, err := ValidateOpenAIImageCapabilities(&NormalizedImageRequest{
			Operation: "generation",
			Prompt:    "poster",
			Size:      "3840x2160",
		}, OpenAIImageProtocolModeCompat, OpenAICompatImageTargetModel)
		require.NoError(t, err)
		require.Equal(t, "openai_image.compat.gpt-image-2.transparent_on.custom_resolution_on", profile.ID)
	})

	t.Run("requires stream for partial images", func(t *testing.T) {
		_, err := ValidateOpenAIImageCapabilities(&NormalizedImageRequest{
			Operation:     "generation",
			Prompt:        "poster",
			PartialImages: intPtrTest(1),
		}, OpenAIImageProtocolModeCompat, OpenAICompatImageTargetModel)
		var requestErr *OpenAIImageRequestError
		require.ErrorAs(t, err, &requestErr)
		require.Equal(t, "image_partial_images_requires_stream", requestErr.Code)
	})

	t.Run("rejects transparent jpeg even on gpt-image-2", func(t *testing.T) {
		_, err := ValidateOpenAIImageCapabilities(&NormalizedImageRequest{
			Operation:    "generation",
			Prompt:       "poster",
			Background:   "transparent",
			OutputFormat: "jpeg",
		}, OpenAIImageProtocolModeCompat, OpenAICompatImageTargetModel)
		var requestErr *OpenAIImageRequestError
		require.ErrorAs(t, err, &requestErr)
		require.Equal(t, "image_output_format_not_supported", requestErr.Code)
	})

	t.Run("rejects oversize custom size even when custom resolution is enabled", func(t *testing.T) {
		_, err := ValidateOpenAIImageCapabilities(&NormalizedImageRequest{
			Operation: "generation",
			Prompt:    "poster",
			Size:      "4096x2160",
		}, OpenAIImageProtocolModeCompat, OpenAICompatImageTargetModel)
		var requestErr *OpenAIImageRequestError
		require.ErrorAs(t, err, &requestErr)
		require.Equal(t, "image_size_too_large", requestErr.Code)
	})

	t.Run("uses display model as fallback for mapped native targets", func(t *testing.T) {
		profile, err := ValidateOpenAIImageCapabilities(&NormalizedImageRequest{
			Operation:      "generation",
			DisplayModelID: "gpt-image-2",
			TargetModelID:  "openai/image-prod",
			Prompt:         "poster",
			Background:     "transparent",
			OutputFormat:   "png",
			Size:           "2048x2048",
		}, OpenAIImageProtocolModeNative, "openai/image-prod")
		require.NoError(t, err)
		require.Equal(t, "openai_image.native.openai_image-prod.transparent_on.custom_resolution_on", profile.ID)
	})

	t.Run("treats versioned gpt image models as the same profile", func(t *testing.T) {
		profile, err := ValidateOpenAIImageCapabilities(&NormalizedImageRequest{
			Operation:    "generation",
			Prompt:       "poster",
			Background:   "transparent",
			OutputFormat: "webp",
			Size:         "2048x2048",
		}, OpenAIImageProtocolModeNative, "gpt-image-1.5")
		require.NoError(t, err)
		require.Equal(t, "openai_image.native.gpt-image-1.5.transparent_on.custom_resolution_on", profile.ID)
	})
}

func TestValidateOpenAIImageCapabilities_UnknownModelsStayConservative(t *testing.T) {
	t.Parallel()

	t.Run("rejects stream for unknown model", func(t *testing.T) {
		_, err := ValidateOpenAIImageCapabilities(&NormalizedImageRequest{
			Operation: "generation",
			Prompt:    "poster",
			Stream:    true,
		}, OpenAIImageProtocolModeNative, "gpt-image-legacy")
		var requestErr *OpenAIImageRequestError
		require.ErrorAs(t, err, &requestErr)
		require.Equal(t, "image_stream_not_supported", requestErr.Code)
	})

	t.Run("rejects mask for unknown model", func(t *testing.T) {
		_, err := ValidateOpenAIImageCapabilities(&NormalizedImageRequest{
			Operation: "edit",
			Prompt:    "poster",
			Images:    []string{"https://example.com/source.png"},
			Mask:      "data:image/png;base64,AAAA",
		}, OpenAIImageProtocolModeNative, "gpt-image-legacy")
		var requestErr *OpenAIImageRequestError
		require.ErrorAs(t, err, &requestErr)
		require.Equal(t, "image_mask_not_supported", requestErr.Code)
	})

	t.Run("rejects multi image for unknown model", func(t *testing.T) {
		_, err := ValidateOpenAIImageCapabilities(&NormalizedImageRequest{
			Operation: "edit",
			Prompt:    "poster",
			Images: []string{
				"https://example.com/source.png",
				"https://example.com/style.png",
			},
		}, OpenAIImageProtocolModeNative, "gpt-image-legacy")
		var requestErr *OpenAIImageRequestError
		require.ErrorAs(t, err, &requestErr)
		require.Equal(t, "image_multi_image_not_supported", requestErr.Code)
	})

	t.Run("rejects custom size when capability disabled", func(t *testing.T) {
		_, err := ValidateOpenAIImageCapabilities(&NormalizedImageRequest{
			Operation: "generation",
			Prompt:    "poster",
			Size:      "2048x2048",
		}, OpenAIImageProtocolModeNative, "gpt-image-legacy")
		var requestErr *OpenAIImageRequestError
		require.ErrorAs(t, err, &requestErr)
		require.Equal(t, "image_size_not_supported", requestErr.Code)
	})
}

func TestNormalizeOpenAIResponsesImageToolRequestAndForceCompat(t *testing.T) {
	t.Parallel()

	body := []byte(`{
		"model":"gpt-5.4-mini",
		"input":[{"type":"message","role":"user","content":[
			{"type":"input_text","text":"replace the background"},
			{"type":"input_image","image_url":"https://example.com/source.png"},
			{"type":"input_image","image_url":"https://example.com/style.png"}
		]}],
		"stream":true,
		"tools":[{
			"type":"image_generation",
			"model":"gpt-image-2",
			"action":"edit",
			"size":"2048x2048",
			"partial_images":2,
			"input_fidelity":"high",
			"input_image_mask":"data:image/png;base64,AAAA"
		}]
	}`)

	req, err := NormalizeOpenAIResponsesImageToolRequest(body)
	require.NoError(t, err)
	require.Equal(t, "edit", normalizeOpenAIImageOperation(req.Operation))
	require.Equal(t, "replace the background", req.Prompt)
	require.Len(t, req.Images, 2)
	require.True(t, req.Stream)
	require.NotNil(t, req.PartialImages)
	require.Equal(t, 2, *req.PartialImages)
	require.Equal(t, "high", req.InputFidelity)

	rewritten, err := ForceOpenAIResponsesImageToolModel(body, OpenAICompatImageTargetModel)
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(rewritten, &payload))
	tool := firstOpenAIResponsesImageTool(rewritten)
	require.Equal(t, OpenAICompatImageTargetModel, tool["model"])
	_, hasInputFidelity := tool["input_fidelity"]
	require.False(t, hasInputFidelity)
}
