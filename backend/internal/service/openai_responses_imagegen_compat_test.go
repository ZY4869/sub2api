package service

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeOpenAIResponsesImageGenCompat_JSONShorthand(t *testing.T) {
	t.Parallel()

	body := mustMarshalCompatJSON(t, map[string]any{
		"model": "gpt-5.4-mini",
		"input": "$imagegen cinematic city skyline",
		"image_generation": map[string]any{
			"size": "1536x1024",
		},
		"reference_images": []any{
			map[string]any{"image_url": "https://example.com/reference.png"},
			map[string]any{"image_url": "data:image/png;base64,AAAA"},
		},
	})

	result, err := NormalizeOpenAIResponsesImageGenCompat(body, "application/json")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Metadata.Enabled)
	require.Equal(t, OpenAIResponsesImagegenCompatSourceJSONShorthand, result.Metadata.Source)
	require.Equal(t, 2, result.Metadata.ReferenceImageCount)
	require.Equal(t, "1536x1024", result.Metadata.ImageGenerationSize)

	tool := firstCompatTool(t, result.ParsedBody)
	require.Equal(t, "image_generation", tool["type"])
	require.Equal(t, "1536x1024", tool["size"])

	require.Equal(t, map[string]any{"type": "image_generation"}, result.ParsedBody["tool_choice"])

	content := firstCompatMessageContent(t, result.ParsedBody)
	require.Equal(t, "input_text", content[0]["type"])
	require.Equal(t, "cinematic city skyline", content[0]["text"])
	require.Equal(t, "input_image", content[1]["type"])
	require.Equal(t, "https://example.com/reference.png", content[1]["image_url"])
	require.Equal(t, "input_image", content[2]["type"])
	require.Equal(t, "data:image/png;base64,AAAA", content[2]["image_url"])
}

func TestNormalizeOpenAIResponsesImageGenCompat_JSONShorthandNormalizesSizeShorthand(t *testing.T) {
	t.Parallel()

	body := mustMarshalCompatJSON(t, map[string]any{
		"model": "gpt-5.4-mini",
		"input": "$imagegen test",
		"image_generation": map[string]any{
			"size": "2K 16:9",
		},
	})

	result, err := NormalizeOpenAIResponsesImageGenCompat(body, "application/json")
	require.NoError(t, err)
	require.NotNil(t, result)

	tool := firstCompatTool(t, result.ParsedBody)
	require.Equal(t, "2048x1152", tool["size"])
}

func TestNormalizeOpenAIResponsesImageGenCompat_JSONShorthandNormalizesSizeParts(t *testing.T) {
	t.Parallel()

	body := mustMarshalCompatJSON(t, map[string]any{
		"model": "gpt-5.4-mini",
		"input": "$imagegen test",
		"image_generation": map[string]any{
			"image_size":   "2K",
			"aspect_ratio": "16:9",
		},
	})

	result, err := NormalizeOpenAIResponsesImageGenCompat(body, "application/json")
	require.NoError(t, err)
	require.NotNil(t, result)

	tool := firstCompatTool(t, result.ParsedBody)
	require.Equal(t, "2048x1152", tool["size"])
}

func TestNormalizeOpenAIResponsesImageGenCompat_JSONShorthandNormalizesIntOptions(t *testing.T) {
	t.Parallel()

	body := mustMarshalCompatJSON(t, map[string]any{
		"model": "gpt-5.4-mini",
		"input": "$imagegen test",
		"image_generation": map[string]any{
			"n":                  "1",
			"partial_images":     "1",
			"output_compression": "80",
		},
	})

	result, err := NormalizeOpenAIResponsesImageGenCompat(body, "application/json")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Metadata.Enabled)

	tool := firstCompatTool(t, result.ParsedBody)
	_, hasN := tool["n"]
	require.False(t, hasN)
	require.Equal(t, 1, tool["partial_images"])
	require.Equal(t, 80, tool["output_compression"])
}

func TestNormalizeOpenAIResponsesImageGenCompat_JSONShorthandRejectsUnsupportedN(t *testing.T) {
	t.Parallel()

	body := mustMarshalCompatJSON(t, map[string]any{
		"model": "gpt-5.4-mini",
		"input": "$imagegen test",
		"image_generation": map[string]any{
			"n": "2",
		},
	})

	result, err := NormalizeOpenAIResponsesImageGenCompat(body, "application/json")
	require.Nil(t, result)
	var compatErr *OpenAIResponsesCompatError
	require.ErrorAs(t, err, &compatErr)
	require.Equal(t, 400, compatErr.Status)
	require.Equal(t, "invalid_request_error", compatErr.Type)
	require.Equal(t, "image_n_not_supported", compatErr.Code)
	require.Contains(t, compatErr.Message, "image_generation.n")
}

func TestNormalizeOpenAIResponsesImageGenCompat_JSONShorthandRejectsInvalidIntOptions(t *testing.T) {
	t.Parallel()

	body := mustMarshalCompatJSON(t, map[string]any{
		"model": "gpt-5.4-mini",
		"input": "$imagegen test",
		"image_generation": map[string]any{
			"n": "abc",
		},
	})

	result, err := NormalizeOpenAIResponsesImageGenCompat(body, "application/json")
	require.Nil(t, result)
	var compatErr *OpenAIResponsesCompatError
	require.ErrorAs(t, err, &compatErr)
	require.Equal(t, 400, compatErr.Status)
	require.Equal(t, "invalid_request_error", compatErr.Type)
	require.Contains(t, compatErr.Message, "image_generation.n")
}

func TestNormalizeOpenAIResponsesImageGenCompat_ModelShorthandNormalizesIntOptionsInExistingTool(t *testing.T) {
	t.Parallel()

	body := mustMarshalCompatJSON(t, map[string]any{
		"model": "gpt-image-2",
		"input": "$imagegen test",
		"tools": []any{
			map[string]any{
				"type":               "image_generation",
				"n":                  "1",
				"partial_images":     "1",
				"output_compression": "80",
			},
		},
	})

	result, err := NormalizeOpenAIResponsesImageGenCompat(body, "application/json")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Metadata.Enabled)
	require.Equal(t, OpenAIResponsesImagegenCompatSourceModelShorthand, result.Metadata.Source)

	tool := firstCompatTool(t, result.ParsedBody)
	_, hasN := tool["n"]
	require.False(t, hasN)
	require.Equal(t, 1, tool["partial_images"])
	require.Equal(t, 80, tool["output_compression"])
}

func TestNormalizeOpenAIResponsesImageGenCompat_ModelShorthandRejectsUnsupportedNInExistingTool(t *testing.T) {
	t.Parallel()

	body := mustMarshalCompatJSON(t, map[string]any{
		"model": "gpt-image-2",
		"input": "$imagegen test",
		"tools": []any{
			map[string]any{
				"type": "image_generation",
				"n":    "2",
			},
		},
	})

	result, err := NormalizeOpenAIResponsesImageGenCompat(body, "application/json")
	require.Nil(t, result)
	var compatErr *OpenAIResponsesCompatError
	require.ErrorAs(t, err, &compatErr)
	require.Equal(t, 400, compatErr.Status)
	require.Equal(t, "invalid_request_error", compatErr.Type)
	require.Equal(t, "image_n_not_supported", compatErr.Code)
	require.Contains(t, compatErr.Message, "image_generation.n")
}

func TestNormalizeOpenAIResponsesImageGenCompat_ModelShorthandRejectsInvalidIntOptionsInExistingTool(t *testing.T) {
	t.Parallel()

	body := mustMarshalCompatJSON(t, map[string]any{
		"model": "gpt-image-2",
		"input": "$imagegen test",
		"tools": []any{
			map[string]any{
				"type": "image_generation",
				"n":    "abc",
			},
		},
	})

	result, err := NormalizeOpenAIResponsesImageGenCompat(body, "application/json")
	require.Nil(t, result)
	var compatErr *OpenAIResponsesCompatError
	require.ErrorAs(t, err, &compatErr)
	require.Equal(t, 400, compatErr.Status)
	require.Equal(t, "invalid_request_error", compatErr.Type)
	require.Contains(t, compatErr.Message, "image_generation.n")
}

func TestNormalizeOpenAIResponsesImageGenCompat_ModelShorthandInjectsTool(t *testing.T) {
	t.Parallel()

	body := mustMarshalCompatJSON(t, map[string]any{
		"model": "gpt-image-2",
		"input": "a poster",
	})

	result, err := NormalizeOpenAIResponsesImageGenCompat(body, "application/json")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Metadata.Enabled)
	require.Equal(t, OpenAIResponsesImagegenCompatSourceModelShorthand, result.Metadata.Source)

	tool := firstCompatTool(t, result.ParsedBody)
	require.Equal(t, "image_generation", tool["type"])
	require.Equal(t, OpenAICompatImageTargetModel, tool["model"])
	require.Equal(t, map[string]any{"type": "image_generation"}, result.ParsedBody["tool_choice"])
	require.Equal(t, "a poster", result.ParsedBody["input"])
}

func TestNormalizeOpenAIResponsesImageGenCompat_ModelShorthandAcceptsCompatFieldsWithoutPrefix(t *testing.T) {
	t.Parallel()

	body := mustMarshalCompatJSON(t, map[string]any{
		"model": "gpt-image-2",
		"input": "a poster",
		"image_generation": map[string]any{
			"size": "1536x1024",
		},
		"reference_images": []any{
			map[string]any{"image_url": "https://example.com/reference.png"},
		},
		"mask": map[string]any{
			"image_url": "data:image/png;base64,AAAA",
		},
	})

	result, err := NormalizeOpenAIResponsesImageGenCompat(body, "application/json")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Metadata.Enabled)
	require.Equal(t, OpenAIResponsesImagegenCompatSourceModelShorthand, result.Metadata.Source)
	require.Equal(t, 1, result.Metadata.ReferenceImageCount)
	require.Equal(t, "1536x1024", result.Metadata.ImageGenerationSize)

	tool := firstCompatTool(t, result.ParsedBody)
	require.Equal(t, "image_generation", tool["type"])
	require.Equal(t, OpenAICompatImageTargetModel, tool["model"])
	require.Equal(t, "1536x1024", tool["size"])
	require.Equal(t, "data:image/png;base64,AAAA", tool["input_image_mask"])

	content := firstCompatMessageContent(t, result.ParsedBody)
	require.Len(t, content, 2)
	require.Equal(t, "a poster", content[0]["text"])
	require.Equal(t, "https://example.com/reference.png", content[1]["image_url"])
}

func TestNormalizeOpenAIResponsesImageGenCompat_ModelShorthandRejectsToolChoiceConflict(t *testing.T) {
	t.Parallel()

	body := mustMarshalCompatJSON(t, map[string]any{
		"model":       "gpt-image-2",
		"input":       "a poster",
		"tool_choice": "auto",
	})

	result, err := NormalizeOpenAIResponsesImageGenCompat(body, "application/json")
	require.Nil(t, result)
	var compatErr *OpenAIResponsesCompatError
	require.ErrorAs(t, err, &compatErr)
	require.Equal(t, "imagegen_compat_tool_choice_conflict", compatErr.Code)
}

func TestNormalizeOpenAIResponsesImageGenCompat_ModelShorthandAddsImageToolWhenOtherToolsPresent(t *testing.T) {
	t.Parallel()

	body := mustMarshalCompatJSON(t, map[string]any{
		"model": "gpt-image-2",
		"input": "a poster",
		"tools": []any{
			map[string]any{"type": "function", "name": "lookup"},
		},
	})

	result, err := NormalizeOpenAIResponsesImageGenCompat(body, "application/json")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Metadata.Enabled)
	require.Equal(t, OpenAIResponsesImagegenCompatSourceModelShorthand, result.Metadata.Source)

	tool := findCompatToolByType(t, result.ParsedBody, "image_generation")
	require.Equal(t, OpenAICompatImageTargetModel, tool["model"])
	require.Equal(t, map[string]any{"type": "image_generation"}, result.ParsedBody["tool_choice"])
}

func TestNormalizeOpenAIResponsesImageGenCompat_StructuredInputText(t *testing.T) {
	t.Parallel()

	body := mustMarshalCompatJSON(t, map[string]any{
		"model": "gpt-5.4-mini",
		"input": []any{
			map[string]any{
				"type": "message",
				"role": "user",
				"content": []any{
					map[string]any{"type": "input_text", "text": "$imagegen watercolor fox"},
				},
			},
		},
		"image_generation": map[string]any{
			"background": "transparent",
		},
	})

	result, err := NormalizeOpenAIResponsesImageGenCompat(body, "application/json")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Metadata.Enabled)
	require.Equal(t, OpenAIResponsesImagegenCompatSourceStructured, result.Metadata.Source)

	content := firstCompatMessageContent(t, result.ParsedBody)
	require.Equal(t, "watercolor fox", content[0]["text"])
	tool := firstCompatTool(t, result.ParsedBody)
	require.Equal(t, "transparent", tool["background"])
}

func TestNormalizeOpenAIResponsesImageGenCompat_JSONEditCarriesMaskAndAction(t *testing.T) {
	t.Parallel()

	body := mustMarshalCompatJSON(t, map[string]any{
		"model": "gpt-5.4-mini",
		"input": "$imagegen replace the background",
		"image_generation": map[string]any{
			"action":         "edit",
			"size":           "2048x2048",
			"input_fidelity": "high",
		},
		"reference_images": []any{
			map[string]any{"image_url": "https://example.com/source.png"},
			map[string]any{"image_url": "https://example.com/style.png"},
		},
		"mask": map[string]any{
			"image_url": "data:image/png;base64,AAAA",
		},
	})

	result, err := NormalizeOpenAIResponsesImageGenCompat(body, "application/json")
	require.NoError(t, err)
	require.NotNil(t, result)

	tool := firstCompatTool(t, result.ParsedBody)
	require.Equal(t, "edit", tool["action"])
	require.Equal(t, "2048x2048", tool["size"])
	_, hasInputFidelity := tool["input_fidelity"]
	require.False(t, hasInputFidelity)
	require.Equal(t, "data:image/png;base64,AAAA", tool["input_image_mask"])
	require.Equal(t, "high", result.TraceTool["input_fidelity"])

	content := firstCompatMessageContent(t, result.ParsedBody)
	require.Len(t, content, 3)
	require.Equal(t, "replace the background", content[0]["text"])
	require.Equal(t, "https://example.com/source.png", content[1]["image_url"])
	require.Equal(t, "https://example.com/style.png", content[2]["image_url"])
}

func TestNormalizeOpenAIResponsesImageGenCompat_PassthroughAndValidation(t *testing.T) {
	t.Parallel()

	t.Run("explicit tools passthrough", func(t *testing.T) {
		body := mustMarshalCompatJSON(t, map[string]any{
			"model": "gpt-5.4-mini",
			"input": "$imagegen keep prefix untouched",
			"tools": []any{
				map[string]any{"type": "image_generation"},
			},
		})

		result, err := NormalizeOpenAIResponsesImageGenCompat(body, "application/json")
		require.NoError(t, err)
		require.NotNil(t, result)
		require.False(t, result.Metadata.Enabled)
		require.Equal(t, body, result.Body)
		require.Equal(t, "$imagegen keep prefix untouched", result.ParsedBody["input"])
	})

	t.Run("compat fields require prefix", func(t *testing.T) {
		body := mustMarshalCompatJSON(t, map[string]any{
			"model": "gpt-5.4-mini",
			"input": "plain text request",
			"image_generation": map[string]any{
				"size": "1024x1024",
			},
		})

		result, err := NormalizeOpenAIResponsesImageGenCompat(body, "application/json")
		require.Nil(t, result)
		var compatErr *OpenAIResponsesCompatError
		require.ErrorAs(t, err, &compatErr)
		require.Equal(t, "imagegen_compat_requires_prefix", compatErr.Code)
	})

	t.Run("explicit tools conflict with compat fields", func(t *testing.T) {
		body := mustMarshalCompatJSON(t, map[string]any{
			"model": "gpt-5.4-mini",
			"input": "$imagegen poster",
			"tools": []any{
				map[string]any{"type": "image_generation"},
			},
			"image_generation": map[string]any{
				"size": "1024x1024",
			},
		})

		result, err := NormalizeOpenAIResponsesImageGenCompat(body, "application/json")
		require.Nil(t, result)
		var compatErr *OpenAIResponsesCompatError
		require.ErrorAs(t, err, &compatErr)
		require.Equal(t, "imagegen_compat_conflict", compatErr.Code)
	})
}

func TestNormalizeOpenAIResponsesImageGenCompat_MultipartBuildsStandardJSON(t *testing.T) {
	t.Parallel()

	alphaPNG := mustEncodeCompatPNG(t, newCompatAlphaImage())
	opaquePNG := mustEncodeCompatPNG(t, newCompatOpaqueImage())

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "gpt-5.4-mini"))
	require.NoError(t, writer.WriteField("input", "$imagegen matte poster"))
	require.NoError(t, writer.WriteField("size", "1536x1024"))

	alphaPart, err := writer.CreateFormFile("reference_image", "alpha.png")
	require.NoError(t, err)
	_, err = alphaPart.Write(alphaPNG)
	require.NoError(t, err)

	require.NoError(t, writer.WriteField("reference_image_url", "https://example.com/remote.png"))

	opaquePart, err := writer.CreateFormFile("reference_image", "opaque.png")
	require.NoError(t, err)
	_, err = opaquePart.Write(opaquePNG)
	require.NoError(t, err)

	require.NoError(t, writer.Close())

	result, err := NormalizeOpenAIResponsesImageGenCompat(body.Bytes(), writer.FormDataContentType())
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "application/json", result.ContentType)
	require.True(t, result.Metadata.Enabled)
	require.Equal(t, OpenAIResponsesImagegenCompatSourceMultipart, result.Metadata.Source)
	require.Equal(t, 3, result.Metadata.ReferenceImageCount)
	require.Equal(t, int64(len(alphaPNG)+len(opaquePNG)), result.Metadata.ReferenceImageBytesBefore)
	require.True(t, result.Metadata.ReferenceImagesNormalized)
	require.Greater(t, result.Metadata.ReferenceImageBytesAfter, int64(0))
	require.Equal(t, "1536x1024", result.Metadata.ImageGenerationSize)

	content := firstCompatMessageContent(t, result.ParsedBody)
	require.Equal(t, "matte poster", content[0]["text"])
	firstImageURL, ok := content[1]["image_url"].(string)
	require.True(t, ok)
	require.True(t, strings.HasPrefix(firstImageURL, "data:image/png;base64,"))
	require.Equal(t, "https://example.com/remote.png", content[2]["image_url"])
	secondImageURL, ok := content[3]["image_url"].(string)
	require.True(t, ok)
	require.True(t, strings.HasPrefix(secondImageURL, "data:image/jpeg;base64,"))

	tool := firstCompatTool(t, result.ParsedBody)
	require.Equal(t, "1536x1024", tool["size"])
	require.Equal(t, map[string]any{"type": "image_generation"}, result.ParsedBody["tool_choice"])
}

func TestNormalizeOpenAIResponsesImageGenCompat_MultipartStreamUnsupported(t *testing.T) {
	t.Parallel()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "gpt-5.4-mini"))
	require.NoError(t, writer.WriteField("input", "$imagegen prompt"))
	require.NoError(t, writer.WriteField("stream", "true"))
	require.NoError(t, writer.Close())

	result, err := NormalizeOpenAIResponsesImageGenCompat(body.Bytes(), writer.FormDataContentType())
	require.Nil(t, result)
	var compatErr *OpenAIResponsesCompatError
	require.ErrorAs(t, err, &compatErr)
	require.Equal(t, "multipart_stream_unsupported", compatErr.Code)
}

func TestNormalizeOpenAIResponsesImageGenCompat_MultipartSingleReferenceImage(t *testing.T) {
	t.Parallel()

	alphaPNG := mustEncodeCompatPNG(t, newCompatAlphaImage())

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "gpt-5.4-mini"))
	require.NoError(t, writer.WriteField("input", "$imagegen studio portrait"))
	part, err := writer.CreateFormFile("reference_image", "alpha.png")
	require.NoError(t, err)
	_, err = part.Write(alphaPNG)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	result, err := NormalizeOpenAIResponsesImageGenCompat(body.Bytes(), writer.FormDataContentType())
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 1, result.Metadata.ReferenceImageCount)

	content := firstCompatMessageContent(t, result.ParsedBody)
	require.Len(t, content, 2)
	require.Equal(t, "studio portrait", content[0]["text"])
	imageURL, ok := content[1]["image_url"].(string)
	require.True(t, ok)
	require.True(t, strings.HasPrefix(imageURL, "data:image/png;base64,"))
}

func TestNormalizeOpenAIResponsesImageGenCompat_MultipartAliasFieldsRespectImageGenerationPriority(t *testing.T) {
	t.Parallel()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "gpt-5.4-mini"))
	require.NoError(t, writer.WriteField("input", "$imagegen scenic valley"))
	require.NoError(t, writer.WriteField("image_generation", `{"size":"1024x1024","background":"transparent"}`))
	require.NoError(t, writer.WriteField("size", "1536x1024"))
	require.NoError(t, writer.WriteField("background", "opaque"))
	require.NoError(t, writer.WriteField("quality", "high"))
	require.NoError(t, writer.Close())

	result, err := NormalizeOpenAIResponsesImageGenCompat(body.Bytes(), writer.FormDataContentType())
	require.NoError(t, err)
	require.NotNil(t, result)

	tool := firstCompatTool(t, result.ParsedBody)
	require.Equal(t, "1024x1024", tool["size"])
	require.Equal(t, "transparent", tool["background"])
	require.Equal(t, "high", tool["quality"])
}

func TestNormalizeOpenAIResponsesImageGenCompat_MultipartEditAliasesCarryMask(t *testing.T) {
	t.Parallel()

	alphaPNG := mustEncodeCompatPNG(t, newCompatAlphaImage())

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "gpt-5.4-mini"))
	require.NoError(t, writer.WriteField("input", "$imagegen replace the sky"))
	require.NoError(t, writer.WriteField("action", "edit"))
	require.NoError(t, writer.WriteField("input_fidelity", "high"))
	require.NoError(t, writer.WriteField("reference_image_url", "https://example.com/source.png"))

	maskPart, err := writer.CreateFormFile("mask", "mask.png")
	require.NoError(t, err)
	_, err = maskPart.Write(alphaPNG)
	require.NoError(t, err)

	require.NoError(t, writer.Close())

	result, err := NormalizeOpenAIResponsesImageGenCompat(body.Bytes(), writer.FormDataContentType())
	require.NoError(t, err)
	require.NotNil(t, result)

	tool := firstCompatTool(t, result.ParsedBody)
	require.Equal(t, "edit", tool["action"])
	_, hasInputFidelity := tool["input_fidelity"]
	require.False(t, hasInputFidelity)
	maskImageURL, ok := tool["input_image_mask"].(string)
	require.True(t, ok)
	require.True(t, strings.HasPrefix(maskImageURL, "data:image/png;base64,"))
	require.Equal(t, "high", result.TraceTool["input_fidelity"])

	content := firstCompatMessageContent(t, result.ParsedBody)
	require.Len(t, content, 2)
	require.Equal(t, "replace the sky", content[0]["text"])
	require.Equal(t, "https://example.com/source.png", content[1]["image_url"])
}

func TestNormalizeOpenAIResponsesImageGenCompat_MultipartPreservesReferenceImageOrder(t *testing.T) {
	t.Parallel()

	alphaPNG := mustEncodeCompatPNG(t, newCompatAlphaImage())
	opaquePNG := mustEncodeCompatPNG(t, newCompatOpaqueImage())

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "gpt-5.4-mini"))
	require.NoError(t, writer.WriteField("input", "$imagegen order test"))

	firstPart, err := writer.CreateFormFile("reference_image", "alpha.png")
	require.NoError(t, err)
	_, err = firstPart.Write(alphaPNG)
	require.NoError(t, err)

	require.NoError(t, writer.WriteField("reference_image_url", "https://example.com/middle.png"))

	secondPart, err := writer.CreateFormFile("reference_image", "opaque.png")
	require.NoError(t, err)
	_, err = secondPart.Write(opaquePNG)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	result, err := NormalizeOpenAIResponsesImageGenCompat(body.Bytes(), writer.FormDataContentType())
	require.NoError(t, err)
	require.NotNil(t, result)

	content := firstCompatMessageContent(t, result.ParsedBody)
	require.Len(t, content, 4)
	firstImageURL, ok := content[1]["image_url"].(string)
	require.True(t, ok)
	require.True(t, strings.HasPrefix(firstImageURL, "data:image/png;base64,"))
	require.Equal(t, "https://example.com/middle.png", content[2]["image_url"])
	secondImageURL, ok := content[3]["image_url"].(string)
	require.True(t, ok)
	require.True(t, strings.HasPrefix(secondImageURL, "data:image/jpeg;base64,"))
}

func TestNormalizeOpenAIResponsesImageGenCompat_InvalidReferenceImageURLs(t *testing.T) {
	t.Parallel()

	t.Run("json rejects non-http url", func(t *testing.T) {
		body := mustMarshalCompatJSON(t, map[string]any{
			"model": "gpt-5.4-mini",
			"input": "$imagegen poster",
			"reference_images": []any{
				map[string]any{"image_url": "ftp://example.com/file.png"},
			},
		})

		result, err := NormalizeOpenAIResponsesImageGenCompat(body, "application/json")
		require.Nil(t, result)
		var compatErr *OpenAIResponsesCompatError
		require.ErrorAs(t, err, &compatErr)
		require.True(t, compatErr.Metadata.Rejected)
		require.Equal(t, OpenAIResponsesImagegenCompatSourceJSONShorthand, compatErr.Metadata.SourceGuess)
	})

	t.Run("multipart rejects malformed data uri", func(t *testing.T) {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		require.NoError(t, writer.WriteField("model", "gpt-5.4-mini"))
		require.NoError(t, writer.WriteField("input", "$imagegen poster"))
		require.NoError(t, writer.WriteField("reference_image_url", "data:image/png;base64"))
		require.NoError(t, writer.Close())

		result, err := NormalizeOpenAIResponsesImageGenCompat(body.Bytes(), writer.FormDataContentType())
		require.Nil(t, result)
		var compatErr *OpenAIResponsesCompatError
		require.ErrorAs(t, err, &compatErr)
		require.True(t, compatErr.Metadata.Rejected)
		require.Equal(t, OpenAIResponsesImagegenCompatSourceMultipart, compatErr.Metadata.SourceGuess)
	})
}

func TestNormalizeOpenAIResponsesImageGenCompat_MultipartReferenceImageLimits(t *testing.T) {
	t.Parallel()

	t.Run("rejects more than four references", func(t *testing.T) {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		require.NoError(t, writer.WriteField("model", "gpt-5.4-mini"))
		require.NoError(t, writer.WriteField("input", "$imagegen collage"))
		for idx := 0; idx < 5; idx++ {
			require.NoError(t, writer.WriteField("reference_image_url", "https://example.com/reference.png"))
		}
		require.NoError(t, writer.Close())

		result, err := NormalizeOpenAIResponsesImageGenCompat(body.Bytes(), writer.FormDataContentType())
		require.Nil(t, result)
		var compatErr *OpenAIResponsesCompatError
		require.ErrorAs(t, err, &compatErr)
		require.Equal(t, "reference_image_count_exceeded", compatErr.Code)
		require.Equal(t, 4, compatErr.Metadata.ReferenceImageCount)
	})

	t.Run("rejects single upload larger than 20MB", func(t *testing.T) {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		require.NoError(t, writer.WriteField("model", "gpt-5.4-mini"))
		require.NoError(t, writer.WriteField("input", "$imagegen oversized"))
		part, err := writer.CreateFormFile("reference_image", "big.png")
		require.NoError(t, err)
		_, err = part.Write(bytes.Repeat([]byte("a"), openAIResponsesReferenceImageUploadLimitBytes+1))
		require.NoError(t, err)
		require.NoError(t, writer.Close())

		result, err := NormalizeOpenAIResponsesImageGenCompat(body.Bytes(), writer.FormDataContentType())
		require.Nil(t, result)
		var compatErr *OpenAIResponsesCompatError
		require.ErrorAs(t, err, &compatErr)
		require.Equal(t, "reference_image_too_large", compatErr.Code)
		require.Equal(t, 1, compatErr.Metadata.ReferenceImageCount)
	})

	t.Run("rejects total upload larger than 40MB", func(t *testing.T) {
		largeValidPNG := mustEncodeCompatPNGBetweenSizes(
			t,
			newCompatHighEntropyOpaqueImage,
			10*1024*1024,
			19*1024*1024,
		)
		overflowBytes := int(openAIResponsesReferenceImageUploadTotalLimitBytes - int64(len(largeValidPNG)*2) + 1)
		require.Greater(t, overflowBytes, 0)
		require.LessOrEqual(t, overflowBytes, openAIResponsesReferenceImageUploadLimitBytes)

		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		require.NoError(t, writer.WriteField("model", "gpt-5.4-mini"))
		require.NoError(t, writer.WriteField("input", "$imagegen oversized total"))
		for index, payload := range [][]byte{
			largeValidPNG,
			largeValidPNG,
			bytes.Repeat([]byte("b"), overflowBytes),
		} {
			part, err := writer.CreateFormFile("reference_image", fmt.Sprintf("chunk-%d.png", index))
			require.NoError(t, err)
			_, err = part.Write(payload)
			require.NoError(t, err)
		}
		require.NoError(t, writer.Close())

		result, err := NormalizeOpenAIResponsesImageGenCompat(body.Bytes(), writer.FormDataContentType())
		require.Nil(t, result)
		var compatErr *OpenAIResponsesCompatError
		require.ErrorAs(t, err, &compatErr)
		require.Equal(t, "reference_image_total_too_large", compatErr.Code)
		require.Equal(t, 3, compatErr.Metadata.ReferenceImageCount)
	})
}

func TestNormalizeResponsesCompatReferenceImage_ReencodesAlphaAndOpaque(t *testing.T) {
	t.Parallel()

	t.Run("alpha png stays png", func(t *testing.T) {
		alphaPNG := mustEncodeCompatPNG(t, newCompatAlphaImage())
		dataURI, _, err := normalizeResponsesCompatReferenceImage(alphaPNG)
		require.Nil(t, err)
		require.True(t, strings.HasPrefix(dataURI, "data:image/png;base64,"))
	})

	t.Run("opaque png becomes jpeg", func(t *testing.T) {
		opaquePNG := mustEncodeCompatPNG(t, newCompatOpaqueImage())
		dataURI, _, err := normalizeResponsesCompatReferenceImage(opaquePNG)
		require.Nil(t, err)
		require.True(t, strings.HasPrefix(dataURI, "data:image/jpeg;base64,"))
	})
}

func TestNormalizeResponsesCompatReferenceImage_UnsupportedMime(t *testing.T) {
	t.Parallel()

	_, _, err := normalizeResponsesCompatReferenceImage([]byte("not-an-image"))
	var compatErr *OpenAIResponsesCompatError
	require.ErrorAs(t, err, &compatErr)
	require.Equal(t, "unsupported_reference_image_type", compatErr.Code)
}

func TestNormalizeResponsesCompatReferenceImage_WebPDecode(t *testing.T) {
	t.Parallel()

	const webpBase64 = "UklGRkAAAABXRUJQVlA4IDQAAADwAQCdASoBAAEAAQAcJaACdLoB+AAETAAA/vW4f/6aR40jxpHxcP/ugT90CfugT/3NoAAA"
	rawWebP, err := base64.StdEncoding.DecodeString(webpBase64)
	require.NoError(t, err)

	dataURI, normalizedBytes, compatErr := normalizeResponsesCompatReferenceImage(rawWebP)
	require.Nil(t, compatErr)
	require.True(t, strings.HasPrefix(dataURI, "data:image/jpeg;base64,"))
	require.Greater(t, normalizedBytes, int64(0))
}

func TestNormalizeResponsesCompatReferenceImage_ResizesLongestEdge(t *testing.T) {
	t.Parallel()

	largePNG := mustEncodeCompatPNG(t, newCompatOpaqueImageWithSize(4096, 2048))
	dataURI, _, compatErr := normalizeResponsesCompatReferenceImage(largePNG)
	require.Nil(t, compatErr)

	img := decodeCompatDataURIImage(t, dataURI)
	require.Equal(t, 2048, img.Bounds().Dx())
	require.Equal(t, 1024, img.Bounds().Dy())
}

func TestNormalizeResponsesCompatReferenceImage_TooLargeAfterNormalization(t *testing.T) {
	t.Parallel()

	noisyPNG := mustEncodeCompatPNGBetweenSizes(
		t,
		newCompatHighEntropyAlphaImage,
		openAIResponsesReferenceImageNormalizedLimitBytes+1,
		openAIResponsesReferenceImageUploadLimitBytes-1,
	)
	_, _, err := normalizeResponsesCompatReferenceImage(noisyPNG)
	require.Error(t, err)
	var compatErr *OpenAIResponsesCompatError
	require.ErrorAs(t, err, &compatErr)
	require.Equal(t, "reference_image_too_large_after_normalization", compatErr.Code)
}

func firstCompatTool(t *testing.T, parsedBody map[string]any) map[string]any {
	t.Helper()

	rawTools, ok := parsedBody["tools"].([]any)
	require.True(t, ok)
	require.Len(t, rawTools, 1)

	tool, ok := rawTools[0].(map[string]any)
	require.True(t, ok)
	return tool
}

func findCompatToolByType(t *testing.T, parsedBody map[string]any, toolType string) map[string]any {
	t.Helper()

	rawTools, ok := parsedBody["tools"].([]any)
	require.True(t, ok)
	require.NotEmpty(t, rawTools)

	for _, rawTool := range rawTools {
		tool, ok := rawTool.(map[string]any)
		require.True(t, ok)
		if strings.TrimSpace(fmt.Sprint(tool["type"])) == toolType {
			return tool
		}
	}

	t.Fatalf("missing tool type %q", toolType)
	return nil
}

func firstCompatMessageContent(t *testing.T, parsedBody map[string]any) []map[string]any {
	t.Helper()

	rawInput, ok := parsedBody["input"].([]any)
	require.True(t, ok)
	require.NotEmpty(t, rawInput)

	message, ok := rawInput[0].(map[string]any)
	require.True(t, ok)
	rawContent, ok := message["content"].([]any)
	require.True(t, ok)

	result := make([]map[string]any, 0, len(rawContent))
	for _, item := range rawContent {
		part, ok := item.(map[string]any)
		require.True(t, ok)
		result = append(result, part)
	}
	return result
}

func mustMarshalCompatJSON(t *testing.T, payload map[string]any) []byte {
	t.Helper()

	raw, err := json.Marshal(payload)
	require.NoError(t, err)
	return raw
}

func mustEncodeCompatPNG(t *testing.T, img image.Image) []byte {
	t.Helper()

	var out bytes.Buffer
	require.NoError(t, png.Encode(&out, img))
	return out.Bytes()
}

func newCompatAlphaImage() image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			alpha := uint8(255)
			if x == 0 && y == 0 {
				alpha = 128
			}
			img.Set(x, y, color.NRGBA{R: 10, G: 20, B: 30, A: alpha})
		}
	}
	return img
}

func newCompatOpaqueImage() image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			img.Set(x, y, color.NRGBA{R: 220, G: 30, B: 40, A: 255})
		}
	}
	return img
}

func newCompatOpaqueImageWithSize(width int, height int) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.NRGBA{R: 220, G: 30, B: 40, A: 255})
		}
	}
	return img
}

func newCompatHighEntropyAlphaImage(width int, height int) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			seed := uint32((x + 1) * 73856093)
			seed ^= uint32((y + 1) * 19349663)
			seed ^= seed << 13
			seed ^= seed >> 17
			seed ^= seed << 5
			img.Set(x, y, color.NRGBA{
				R: uint8(seed),
				G: uint8(seed >> 8),
				B: uint8(seed >> 16),
				A: uint8(32 + (seed % 223)),
			})
		}
	}
	return img
}

func newCompatHighEntropyOpaqueImage(width int, height int) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			seed := uint32((x + 1) * 83492791)
			seed ^= uint32((y + 1) * 2654435761)
			seed ^= seed << 13
			seed ^= seed >> 17
			seed ^= seed << 5
			img.Set(x, y, color.NRGBA{
				R: uint8(seed),
				G: uint8(seed >> 8),
				B: uint8(seed >> 16),
				A: 255,
			})
		}
	}
	return img
}

func mustEncodeCompatPNGBetweenSizes(
	t *testing.T,
	build func(width int, height int) image.Image,
	minBytes int64,
	maxBytes int64,
) []byte {
	t.Helper()

	for _, size := range []int{2304, 2176, 2048, 1984, 1920, 1856, 1792, 1728, 1664} {
		encoded := mustEncodeCompatPNG(t, build(size, size))
		if int64(len(encoded)) >= minBytes && int64(len(encoded)) <= maxBytes {
			return encoded
		}
	}

	t.Fatalf("failed to encode compat png within size range [%d, %d]", minBytes, maxBytes)
	return nil
}

func decodeCompatDataURIImage(t *testing.T, dataURI string) image.Image {
	t.Helper()

	parts := strings.SplitN(dataURI, ",", 2)
	require.Len(t, parts, 2)
	raw, err := base64.StdEncoding.DecodeString(parts[1])
	require.NoError(t, err)
	img, _, err := image.Decode(bytes.NewReader(raw))
	require.NoError(t, err)
	return img
}
