package service

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type OpenAIImageCapabilityProfile struct {
	ID                           string
	ProtocolMode                 string
	TargetModelID                string
	SupportsGenerate             bool
	SupportsEdit                 bool
	SupportsMask                 bool
	SupportsMultiImage           bool
	SupportsStream               bool
	CustomResolutionEnabled      bool
	TransparentBackgroundEnabled bool
}

type OpenAIImageRequestError struct {
	Status  int
	Type    string
	Code    string
	Message string
}

func (e *OpenAIImageRequestError) Error() string {
	if e == nil {
		return ""
	}
	return strings.TrimSpace(e.Message)
}

var openAIStableImageSizes = map[string]struct{}{
	"auto":      {},
	"1024x1024": {},
	"1536x1024": {},
	"1024x1536": {},
}

func ResolveOpenAIImageCapabilityProfile(protocolMode string, targetModel string) OpenAIImageCapabilityProfile {
	mode := NormalizeOpenAIImageProtocolMode(protocolMode)
	if mode == "" {
		mode = OpenAIImageProtocolModeNative
	}
	modelID := strings.TrimSpace(strings.ToLower(targetModel))
	if modelID == "" {
		modelID = "default"
	}

	profile := OpenAIImageCapabilityProfile{
		ProtocolMode:                 mode,
		TargetModelID:                modelID,
		SupportsGenerate:             true,
		SupportsEdit:                 true,
		SupportsMask:                 false,
		SupportsMultiImage:           false,
		SupportsStream:               false,
		CustomResolutionEnabled:      false,
		TransparentBackgroundEnabled: false,
	}
	if isOpenAIGPTImageProfileModelID(modelID) {
		applyOpenAIGPTImageProfile(&profile)
	}
	profile.ID = buildOpenAIImageCapabilityProfileID(profile)
	return profile
}

func ValidateOpenAIImageCapabilities(req *NormalizedImageRequest, protocolMode string, targetModel string) (OpenAIImageCapabilityProfile, error) {
	profile := resolveOpenAIImageCapabilityProfileForRequest(req, protocolMode, targetModel)
	if req == nil {
		return profile, newOpenAIImageRequestError("image_request_missing", "image request is required")
	}

	switch normalizeOpenAIImageOperation(req.Operation) {
	case "generate":
		if !profile.SupportsGenerate {
			return profile, newOpenAIImageRequestError("image_operation_not_supported", "image generation is not supported for this model")
		}
	case "edit":
		if !profile.SupportsEdit {
			return profile, newOpenAIImageRequestError("image_operation_not_supported", "image editing is not supported for this model")
		}
	default:
		return profile, newOpenAIImageRequestError("image_operation_invalid", "image operation must be generate or edit")
	}

	if req.Stream && !profile.SupportsStream {
		return profile, newOpenAIImageRequestError("image_stream_not_supported", "stream=true is not supported for this image profile")
	}
	if req.PartialImages != nil {
		if *req.PartialImages < 0 || *req.PartialImages > 3 {
			return profile, newOpenAIImageRequestError("image_partial_images_invalid", "partial_images must be between 0 and 3")
		}
		if !req.Stream {
			return profile, newOpenAIImageRequestError("image_partial_images_requires_stream", "partial_images requires stream=true")
		}
	}
	if strings.TrimSpace(req.Mask) != "" && !profile.SupportsMask {
		return profile, newOpenAIImageRequestError("image_mask_not_supported", "mask is not supported for this image profile")
	}
	if len(req.Images) > 1 && !profile.SupportsMultiImage {
		return profile, newOpenAIImageRequestError("image_multi_image_not_supported", "multiple input images are not supported for this image profile")
	}

	if background := normalizeOpenAIImageBackground(req.Background); background == "" {
		if strings.TrimSpace(req.Background) != "" {
			return profile, newOpenAIImageRequestError("image_background_invalid", "background must be auto, opaque, or transparent")
		}
	} else if background == "transparent" && !profile.TransparentBackgroundEnabled {
		return profile, newOpenAIImageRequestError("image_background_not_supported", fmt.Sprintf("background=transparent is not supported for model %q", profile.TargetModelID))
	}

	outputFormat := normalizeOpenAIImageOutputFormat(req.OutputFormat)
	if outputFormat == "" && strings.TrimSpace(req.OutputFormat) != "" {
		return profile, newOpenAIImageRequestError("image_output_format_invalid", "output_format must be png, jpeg, or webp")
	}
	if req.OutputCompression != nil {
		if *req.OutputCompression < 0 || *req.OutputCompression > 100 {
			return profile, newOpenAIImageRequestError("image_output_compression_invalid", "output_compression must be between 0 and 100")
		}
		if outputFormat != "jpeg" && outputFormat != "webp" {
			return profile, newOpenAIImageRequestError("image_output_compression_not_supported", "output_compression requires output_format=jpeg or webp")
		}
	}
	if normalizeOpenAIImageBackground(req.Background) == "transparent" && outputFormat == "jpeg" {
		return profile, newOpenAIImageRequestError("image_output_format_not_supported", "background=transparent requires output_format=png or webp")
	}

	size := strings.TrimSpace(strings.ToLower(req.Size))
	if size == "" {
		return profile, nil
	}
	if _, ok := openAIStableImageSizes[size]; ok {
		return profile, nil
	}

	width, height, ok := parseOpenAIImageSizeDimensions(size)
	if !ok {
		return profile, newOpenAIImageRequestError("image_size_invalid", "size must be auto or WIDTHxHEIGHT")
	}
	if width <= 0 || height <= 0 {
		return profile, newOpenAIImageRequestError("image_size_invalid", "size dimensions must be positive integers")
	}
	if !profile.CustomResolutionEnabled {
		return profile, newOpenAIImageRequestError("image_size_not_supported", fmt.Sprintf("size %q is not supported for model %q", strings.TrimSpace(req.Size), profile.TargetModelID))
	}
	if width > 3840 || height > 3840 {
		return profile, newOpenAIImageRequestError("image_size_too_large", "custom image size cannot exceed 3840px on either side")
	}
	return profile, nil
}

func resolveOpenAIImageCapabilityProfileForRequest(req *NormalizedImageRequest, protocolMode string, targetModel string) OpenAIImageCapabilityProfile {
	profile := ResolveOpenAIImageCapabilityProfile(
		protocolMode,
		firstNonEmptyString(strings.TrimSpace(targetModel), stringOrEmpty(req, func(r *NormalizedImageRequest) string { return r.TargetModelID }), stringOrEmpty(req, func(r *NormalizedImageRequest) string { return r.DisplayModelID })),
	)
	if isOpenAIGPTImageProfileModelID(profile.TargetModelID) {
		return profile
	}
	if req == nil {
		return profile
	}
	if !isOpenAIGPTImageProfileModelID(req.TargetModelID) && !isOpenAIGPTImageProfileModelID(req.DisplayModelID) {
		return profile
	}
	applyOpenAIGPTImageProfile(&profile)
	profile.ID = buildOpenAIImageCapabilityProfileID(profile)
	return profile
}

func applyOpenAIGPTImageProfile(profile *OpenAIImageCapabilityProfile) {
	if profile == nil {
		return
	}
	profile.SupportsGenerate = true
	profile.SupportsEdit = true
	profile.SupportsMask = true
	profile.SupportsMultiImage = true
	profile.SupportsStream = true
	profile.CustomResolutionEnabled = true
	profile.TransparentBackgroundEnabled = true
}

func isOpenAIGPTImageProfileModelID(value string) bool {
	normalized := strings.TrimSpace(strings.ToLower(value))
	if normalized == "" {
		return false
	}
	if normalized == "chatgpt-image-latest" {
		return true
	}
	if !strings.HasPrefix(normalized, "gpt-image-") {
		return false
	}
	remainder := strings.TrimPrefix(normalized, "gpt-image-")
	if remainder == "" {
		return false
	}
	first := remainder[0]
	return first >= '0' && first <= '9'
}

func stringOrEmpty(req *NormalizedImageRequest, selector func(*NormalizedImageRequest) string) string {
	if req == nil || selector == nil {
		return ""
	}
	return strings.TrimSpace(selector(req))
}

func normalizeOpenAIImageOperation(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "generate", "generation", "generations":
		return "generate"
	case "edit", "edits":
		return "edit"
	default:
		return ""
	}
}

func parseOpenAIImageSizeDimensions(value string) (int, int, bool) {
	parts := strings.Split(strings.TrimSpace(strings.ToLower(value)), "x")
	if len(parts) != 2 {
		return 0, 0, false
	}
	width, errWidth := strconv.Atoi(strings.TrimSpace(parts[0]))
	height, errHeight := strconv.Atoi(strings.TrimSpace(parts[1]))
	if errWidth != nil || errHeight != nil {
		return 0, 0, false
	}
	return width, height, true
}

func normalizeOpenAIImageBackground(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "auto":
		return "auto"
	case "opaque":
		return "opaque"
	case "transparent":
		return "transparent"
	default:
		return ""
	}
}

func normalizeOpenAIImageOutputFormat(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "png":
		return "png"
	case "jpeg", "jpg":
		return "jpeg"
	case "webp":
		return "webp"
	default:
		return ""
	}
}

func buildOpenAIImageCapabilityProfileID(profile OpenAIImageCapabilityProfile) string {
	mode := firstNonEmptyString(strings.TrimSpace(profile.ProtocolMode), OpenAIImageProtocolModeNative)
	modelID := firstNonEmptyString(strings.TrimSpace(profile.TargetModelID), "default")
	modelID = strings.NewReplacer("/", "_", " ", "_").Replace(modelID)
	return fmt.Sprintf(
		"openai_image.%s.%s.transparent_%s.custom_resolution_%s",
		mode,
		modelID,
		boolOnOff(profile.TransparentBackgroundEnabled),
		boolOnOff(profile.CustomResolutionEnabled),
	)
}

func boolOnOff(value bool) string {
	if value {
		return "on"
	}
	return "off"
}

func newOpenAIImageRequestError(code string, message string) *OpenAIImageRequestError {
	return &OpenAIImageRequestError{
		Status:  http.StatusBadRequest,
		Type:    "invalid_request_error",
		Code:    strings.TrimSpace(code),
		Message: strings.TrimSpace(message),
	}
}
