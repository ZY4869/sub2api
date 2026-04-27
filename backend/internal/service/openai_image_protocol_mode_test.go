package service

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeOpenAIAccountImageExtra_OAuthDefaultsFollowPlan(t *testing.T) {
	t.Parallel()

	freeExtra := NormalizeOpenAIAccountImageExtra(PlatformOpenAI, AccountTypeOAuth, map[string]any{
		"plan_type": "free",
	}, nil)
	require.Equal(t, OpenAIImageProtocolModeNative, freeExtra[openAIImageProtocolModeExtraKey])
	require.Equal(t, false, freeExtra[openAIImageCompatAllowedExtraKey])

	paidExtra := NormalizeOpenAIAccountImageExtra(PlatformOpenAI, AccountTypeOAuth, map[string]any{
		"plan_type": "pro",
	}, nil)
	require.Equal(t, OpenAIImageProtocolModeCompat, paidExtra[openAIImageProtocolModeExtraKey])
	require.Equal(t, true, paidExtra[openAIImageCompatAllowedExtraKey])
}

func TestNormalizeOpenAIAccountImageExtra_OAuthFreeClampsCompatToNative(t *testing.T) {
	t.Parallel()

	extra := NormalizeOpenAIAccountImageExtra(PlatformOpenAI, AccountTypeOAuth, map[string]any{
		"plan_type": "free",
	}, map[string]any{
		openAIImageProtocolModeExtraKey: OpenAIImageProtocolModeCompat,
	})

	require.Equal(t, OpenAIImageProtocolModeNative, extra[openAIImageProtocolModeExtraKey])
	require.Equal(t, false, extra[openAIImageCompatAllowedExtraKey])
}

func TestNormalizeOpenAIAccountImageExtra_ProtocolGatewayDefaultsToNative(t *testing.T) {
	t.Parallel()

	extra := NormalizeOpenAIAccountImageExtra(PlatformProtocolGateway, AccountTypeAPIKey, nil, map[string]any{
		gatewayExtraProtocolKey: PlatformOpenAI,
	})

	require.Equal(t, OpenAIImageProtocolModeNative, extra[gatewayOpenAIImageProtocolModeExtraKey])
}

func TestResolveEffectiveOpenAIImageProtocolMode_GroupOverrideWins(t *testing.T) {
	t.Parallel()

	account := &Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Extra: map[string]any{
			openAIImageProtocolModeExtraKey:  OpenAIImageProtocolModeNative,
			openAIImageCompatAllowedExtraKey: true,
		},
	}
	group := &Group{ImageProtocolMode: OpenAIImageProtocolModeCompat}

	require.Equal(t, OpenAIImageProtocolModeCompat, ResolveEffectiveOpenAIImageProtocolMode(group, account))
	require.True(t, IsOpenAIImageCompatAllowed(account))
}

func TestForceOpenAIResponsesImageToolModel(t *testing.T) {
	t.Parallel()

	body := []byte(`{"model":"gpt-5.4-mini","tools":[{"type":"image_generation","size":"1024x1024"}]}`)
	rewritten, err := ForceOpenAIResponsesImageToolModel(body, OpenAICompatImageTargetModel)
	require.NoError(t, err)
	require.Equal(t, OpenAICompatImageTargetModel, firstOpenAIResponsesImageTool(rewritten)["model"])
}

func TestRewriteOpenAIResponsesImageToolModelPreservesInputFidelity(t *testing.T) {
	t.Parallel()

	body := []byte(`{"model":"gpt-5.4-mini","tools":[{"type":"image_generation","model":"friendly-image","input_fidelity":"high"}]}`)
	rewritten, err := RewriteOpenAIResponsesImageToolModel(body, OpenAICompatImageTargetModel)
	require.NoError(t, err)
	tool := firstOpenAIResponsesImageTool(rewritten)
	require.Equal(t, OpenAICompatImageTargetModel, tool["model"])
	require.Equal(t, "high", tool["input_fidelity"])
}

func TestNormalizeOpenAIImageRequest_EditJSON(t *testing.T) {
	t.Parallel()

	requestBody := []byte(`{
		"model":"gpt-image-2",
		"prompt":"replace the background",
		"images":[{"image_url":"https://example.com/source.png"}],
		"mask":{"image_url":"data:image/png;base64,AAAA"},
		"size":"2048x1152",
		"output_format":"webp",
		"output_compression":80,
		"n":2,
		"input_fidelity":"high"
	}`)

	req, err := NormalizeOpenAIImageRequest(requestBody, "application/json", "edits")
	require.NoError(t, err)
	require.Equal(t, "replace the background", req.Prompt)
	require.Len(t, req.Images, 1)
	require.Equal(t, "data:image/png;base64,AAAA", req.Mask)
	require.Equal(t, "2048x1152", req.Size)
	require.Equal(t, "webp", req.OutputFormat)
	require.NotNil(t, req.OutputCompression)
	require.Equal(t, 80, *req.OutputCompression)
	require.NotNil(t, req.N)
	require.Equal(t, 2, *req.N)
	require.Equal(t, "high", req.InputFidelity)
	require.Equal(t, OpenAIImageSizeTier2K, ResolveOpenAIImageSizeTier(req.Size))
}

func TestNormalizeOpenAIImageRequest_GenerationNormalizesSizeShorthand(t *testing.T) {
	t.Parallel()

	requestBody := []byte(`{
		"model":"gpt-image-2",
		"prompt":"a poster",
		"size":"2K 16:9"
	}`)

	req, err := NormalizeOpenAIImageRequest(requestBody, "application/json", "generations")
	require.NoError(t, err)
	require.Equal(t, "2048x1152", req.Size)
	require.Equal(t, OpenAIImageSizeTier2K, ResolveOpenAIImageSizeTier(req.Size))
}

func TestNormalizeOpenAIImageRequest_GenerationNormalizesSizeParts(t *testing.T) {
	t.Parallel()

	requestBody := []byte(`{
		"model":"gpt-image-2",
		"prompt":"a poster",
		"image_size":"2K",
		"aspect_ratio":"16:9"
	}`)

	req, err := NormalizeOpenAIImageRequest(requestBody, "application/json", "generations")
	require.NoError(t, err)
	require.Equal(t, "2048x1152", req.Size)
	require.Equal(t, OpenAIImageSizeTier2K, ResolveOpenAIImageSizeTier(req.Size))
}

func TestBuildOpenAIImageCompatResponsesBody_EditCarriesMaskAndAction(t *testing.T) {
	t.Parallel()

	req := &NormalizedImageRequest{
		Operation:         "edits",
		DisplayModelID:    "gpt-image-2",
		TargetModelID:     OpenAICompatImageTargetModel,
		Prompt:            "replace the sky",
		Images:            []string{"https://example.com/source.png"},
		Mask:              "data:image/png;base64,AAAA",
		Size:              "3840x2160",
		OutputFormat:      "png",
		OutputCompression: intPtrTest(90),
		PartialImages:     intPtrTest(2),
		InputFidelity:     "high",
	}

	body, err := BuildOpenAIImageCompatResponsesBody(req, OpenAICompatImageTargetModel, OpenAICompatImageTargetModel)
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	tool := firstOpenAIResponsesImageTool(body)
	require.Equal(t, "edit", tool["action"])
	require.Equal(t, "data:image/png;base64,AAAA", tool["input_image_mask"])
	require.Equal(t, "3840x2160", tool["size"])
	require.Equal(t, float64(2), tool["partial_images"])
	require.Equal(t, OpenAICompatImageTargetModel, tool["model"])
	_, hasInputFidelity := tool["input_fidelity"]
	require.False(t, hasInputFidelity)
	require.Equal(t, OpenAIImageSizeTier4K, ResolveOpenAIResponsesImageToolSizeTier(body))
}

func TestBuildOpenAIImageCompatResponsesBody_DoesNotForwardN(t *testing.T) {
	t.Parallel()

	req := &NormalizedImageRequest{
		Operation:      "generations",
		DisplayModelID: "gpt-image-2",
		TargetModelID:  OpenAICompatImageTargetModel,
		Prompt:         "a poster",
		Size:           "1024x1024",
		N:              intPtrTest(1),
	}

	body, err := BuildOpenAIImageCompatResponsesBody(req, OpenAICompatImageHostModel, OpenAICompatImageTargetModel)
	require.NoError(t, err)
	tool := firstOpenAIResponsesImageTool(body)
	_, hasN := tool["n"]
	require.False(t, hasN)
}

func TestBuildOpenAIImageCompatResponsesBody_RejectsUnsupportedN(t *testing.T) {
	t.Parallel()

	req := &NormalizedImageRequest{
		Operation:      "generations",
		DisplayModelID: "gpt-image-2",
		TargetModelID:  OpenAICompatImageTargetModel,
		Prompt:         "a poster",
		Size:           "1024x1024",
		N:              intPtrTest(2),
	}

	body, err := BuildOpenAIImageCompatResponsesBody(req, OpenAICompatImageHostModel, OpenAICompatImageTargetModel)
	require.Nil(t, body)
	var requestErr *OpenAIImageRequestError
	require.ErrorAs(t, err, &requestErr)
	require.Equal(t, "image_n_not_supported", requestErr.Code)
}

func TestBuildOpenAIImagesCompatResponse_ExtractsDataURIImages(t *testing.T) {
	t.Parallel()

	responseBody := []byte(`{
		"output":[
			{"type":"message","content":[
				{"type":"output_image","image_url":"data:image/png;base64,ZmFrZQ=="},
				{"type":"output_text","text":"done"}
			]}
		],
		"usage":{"input_tokens":4,"output_tokens":2}
	}`)

	out, count := buildOpenAIImagesCompatResponse(responseBody, &NormalizedImageRequest{OutputFormat: "png"})
	require.Equal(t, 1, count)
	require.Len(t, out.Data, 1)
	require.Equal(t, "ZmFrZQ==", out.Data[0]["b64_json"])
	require.Equal(t, "png", out.OutputFormat)
	require.Equal(t, float64(4), out.Usage["input_tokens"])
}

func intPtrTest(value int) *int {
	return &value
}
