package service

import (
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"strings"
)

const (
	openAIResponsesImagegenPrefix = "$imagegen "

	OpenAIResponsesImagegenCompatSourceJSONShorthand  = "json_shorthand"
	OpenAIResponsesImagegenCompatSourceModelShorthand = "model_shorthand"
	OpenAIResponsesImagegenCompatSourceStructured     = "structured_json"
	OpenAIResponsesImagegenCompatSourceMultipart      = "multipart"

	openAIResponsesReferenceImageUploadLimitBytes      = 20 * 1024 * 1024
	openAIResponsesReferenceImageUploadTotalLimitBytes = 40 * 1024 * 1024
	openAIResponsesReferenceImageNormalizedLimitBytes  = 8 * 1024 * 1024
	openAIResponsesReferenceImageUploadLimitCount      = 4
	openAIResponsesReferenceImageMaxDimension          = 2048
)

var openAIResponsesImagegenToolOptionKeys = []string{
	"action",
	"size",
	"quality",
	"background",
	"output_format",
	"output_compression",
	"partial_images",
	"moderation",
	"input_image_mask",
}

var openAIResponsesImagegenAcceptedOptionKeys = []string{
	"action",
	"aspect_ratio",
	"image_size",
	"size",
	"quality",
	"background",
	"output_format",
	"output_compression",
	"partial_images",
	"moderation",
	"input_fidelity",
	"input_image_mask",
	"n",
}

type OpenAIResponsesCompatMetadata struct {
	Enabled                   bool
	Source                    string
	SourceGuess               string
	Rejected                  bool
	RejectCode                string
	ReferenceImageCount       int
	ReferenceImageBytesBefore int64
	ReferenceImageBytesAfter  int64
	ReferenceImagesNormalized bool
	ImageGenerationSize       string
}

type OpenAIResponsesCompatResult struct {
	Body        []byte
	ContentType string
	ParsedBody  map[string]any
	Metadata    OpenAIResponsesCompatMetadata
	TraceTool   map[string]any
}

type OpenAIResponsesCompatError struct {
	Status   int
	Type     string
	Code     string
	Message  string
	Metadata OpenAIResponsesCompatMetadata
}

func (e *OpenAIResponsesCompatError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func NormalizeOpenAIResponsesImageGenCompat(body []byte, contentType string) (*OpenAIResponsesCompatResult, error) {
	result := &OpenAIResponsesCompatResult{
		Body:        body,
		ContentType: strings.TrimSpace(contentType),
	}

	mediaType, params, err := mime.ParseMediaType(strings.TrimSpace(contentType))
	if err != nil {
		mediaType = strings.TrimSpace(contentType)
	}
	if strings.HasPrefix(strings.ToLower(mediaType), "multipart/form-data") {
		boundary := strings.TrimSpace(params["boundary"])
		if boundary == "" {
			return nil, newOpenAIResponsesCompatErrorWithMetadata(
				http.StatusBadRequest,
				"invalid_request_error",
				"",
				"missing multipart boundary",
				OpenAIResponsesCompatMetadata{
					Rejected:    true,
					SourceGuess: OpenAIResponsesImagegenCompatSourceMultipart,
				},
			)
		}
		return normalizeOpenAIResponsesMultipartImagegenCompat(body, boundary)
	}

	if len(body) == 0 || !json.Valid(body) {
		return result, nil
	}

	var reqBody map[string]any
	if err := json.Unmarshal(body, &reqBody); err != nil {
		return result, nil
	}
	if reqBody == nil {
		return result, nil
	}

	normalized, compatErr := normalizeOpenAIResponsesJSONImagegenCompat(reqBody)
	if compatErr != nil {
		enrichOpenAIResponsesCompatRejectMetadata(
			compatErr,
			OpenAIResponsesCompatMetadata{
				Rejected:            true,
				SourceGuess:         guessOpenAIResponsesCompatJSONSource(reqBody),
				ReferenceImageCount: countOpenAIResponsesCompatReferenceImageCandidates(reqBody),
			},
		)
		return nil, compatErr
	}
	if normalized == nil {
		result.ParsedBody = reqBody
		return result, nil
	}

	encoded, err := json.Marshal(normalized.ParsedBody)
	if err != nil {
		return nil, fmt.Errorf("marshal responses compat body: %w", err)
	}
	normalized.Body = encoded
	normalized.ContentType = "application/json"
	return normalized, nil
}
