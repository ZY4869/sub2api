package service

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/jpeg"
	"image/png"
	"net/http"
	"net/url"
	"strings"

	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

func parseResponsesCompatSingleImageSource(raw any) (string, *OpenAIResponsesCompatError) {
	switch typed := raw.(type) {
	case string:
		return validateResponsesCompatReferenceImageURL(typed)
	case map[string]any:
		for _, key := range []string{"image_url", "input_image_mask", "mask"} {
			if value := scalarString(typed[key]); value != "" {
				return validateResponsesCompatReferenceImageURL(value)
			}
		}
		return "", newOpenAIResponsesCompatError(http.StatusBadRequest, "invalid_request_error", "", "mask must be a valid image_url object")
	default:
		return "", newOpenAIResponsesCompatError(http.StatusBadRequest, "invalid_request_error", "", "mask must be a string or {image_url} object")
	}
}

func validateResponsesCompatReferenceImageURL(raw string) (string, *OpenAIResponsesCompatError) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", newOpenAIResponsesCompatError(http.StatusBadRequest, "invalid_request_error", "", "reference image_url must not be empty")
	}
	lowerValue := strings.ToLower(value)
	if strings.HasPrefix(lowerValue, "data:image/") {
		if strings.Contains(lowerValue, ",") {
			return value, nil
		}
		return "", newOpenAIResponsesCompatError(http.StatusBadRequest, "invalid_request_error", "", "reference image data URI is invalid")
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed == nil || parsed.Host == "" {
		return "", newOpenAIResponsesCompatError(http.StatusBadRequest, "invalid_request_error", "", "reference image_url must be an http(s) URL or data URI")
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
		return value, nil
	default:
		return "", newOpenAIResponsesCompatError(http.StatusBadRequest, "invalid_request_error", "", "reference image_url must be an http(s) URL or data URI")
	}
}

func normalizeResponsesCompatReferenceImage(rawBytes []byte) (string, int64, *OpenAIResponsesCompatError) {
	mediaType := strings.ToLower(strings.TrimSpace(http.DetectContentType(rawBytes)))
	switch mediaType {
	case "image/jpeg", "image/png", "image/webp":
	default:
		return "", 0, newOpenAIResponsesCompatError(http.StatusBadRequest, "invalid_request_error", "unsupported_reference_image_type", "reference_image only supports JPEG, PNG, or WebP uploads")
	}

	img, _, err := image.Decode(bytes.NewReader(rawBytes))
	if err != nil {
		return "", 0, newOpenAIResponsesCompatError(http.StatusBadRequest, "invalid_request_error", "unsupported_reference_image_type", "reference_image could not be decoded as JPEG, PNG, or WebP")
	}

	bounds := img.Bounds()
	targetWidth, targetHeight := fitResponsesCompatImageSize(bounds.Dx(), bounds.Dy(), openAIResponsesReferenceImageMaxDimension)
	resized := img
	if targetWidth != bounds.Dx() || targetHeight != bounds.Dy() {
		if imageHasTransparentPixels(img) {
			dst := image.NewNRGBA(image.Rect(0, 0, targetWidth, targetHeight))
			xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, xdraw.Over, nil)
			resized = dst
		} else {
			dst := image.NewRGBA(image.Rect(0, 0, targetWidth, targetHeight))
			xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, xdraw.Src, nil)
			resized = dst
		}
	}

	var (
		buffer     bytes.Buffer
		outputType string
		encodeErr  error
		hasAlpha   = imageHasTransparentPixels(resized)
	)
	if hasAlpha {
		outputType = "image/png"
		encodeErr = png.Encode(&buffer, resized)
	} else {
		outputType = "image/jpeg"
		encodeErr = jpeg.Encode(&buffer, resized, &jpeg.Options{Quality: 82})
	}
	if encodeErr != nil {
		return "", 0, newOpenAIResponsesCompatError(http.StatusBadRequest, "invalid_request_error", "", "failed to normalize reference_image")
	}
	if int64(buffer.Len()) > openAIResponsesReferenceImageNormalizedLimitBytes {
		return "", 0, newOpenAIResponsesCompatError(http.StatusBadRequest, "invalid_request_error", "reference_image_too_large_after_normalization", "reference_image is still larger than 8MB after normalization")
	}

	encoded := base64.StdEncoding.EncodeToString(buffer.Bytes())
	return "data:" + outputType + ";base64," + encoded, int64(buffer.Len()), nil
}

func fitResponsesCompatImageSize(width int, height int, maxDimension int) (int, int) {
	if width <= 0 || height <= 0 || maxDimension <= 0 {
		return width, height
	}
	if width <= maxDimension && height <= maxDimension {
		return width, height
	}
	if width >= height {
		nextWidth := maxDimension
		nextHeight := int(float64(height) * (float64(maxDimension) / float64(width)))
		if nextHeight < 1 {
			nextHeight = 1
		}
		return nextWidth, nextHeight
	}
	nextHeight := maxDimension
	nextWidth := int(float64(width) * (float64(maxDimension) / float64(height)))
	if nextWidth < 1 {
		nextWidth = 1
	}
	return nextWidth, nextHeight
}

func imageHasTransparentPixels(img image.Image) bool {
	if img == nil {
		return false
	}
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			_, _, _, alpha := img.At(x, y).RGBA()
			if alpha != 0xffff {
				return true
			}
		}
	}
	return false
}
