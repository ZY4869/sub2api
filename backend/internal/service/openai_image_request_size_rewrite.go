package service

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func RewriteOpenAIImageRequestSizeAndDropExtras(body []byte, contentType string) ([]byte, string, string, error) {
	if len(body) == 0 {
		return body, contentType, "", nil
	}

	mediaType, params, err := mime.ParseMediaType(strings.TrimSpace(contentType))
	if err != nil {
		mediaType = strings.TrimSpace(contentType)
	}
	if strings.HasPrefix(strings.ToLower(mediaType), "multipart/form-data") {
		boundary := strings.TrimSpace(params["boundary"])
		if boundary == "" {
			return nil, "", "", fmt.Errorf("missing multipart boundary")
		}

		sizeValue, hasSize, imageSizeValue, hasImageSize, aspectValue, hasAspect, err := scanOpenAIImageSizeMultipartFields(body, boundary)
		if err != nil {
			return nil, "", "", err
		}

		normalizedSize, _, errCode, errMessage := normalizeOpenAIImageSizeWithAspect(sizeValue, imageSizeValue, aspectValue)
		if errCode != "" {
			return nil, "", "", newOpenAIImageRequestError(errCode, errMessage)
		}

		shouldRewrite := hasImageSize || hasAspect
		if strings.TrimSpace(normalizedSize) != "" {
			if !hasSize || strings.TrimSpace(sizeValue) != strings.TrimSpace(normalizedSize) {
				shouldRewrite = true
			}
		}
		if !shouldRewrite {
			return body, contentType, strings.TrimSpace(normalizedSize), nil
		}

		rewritten, rewrittenType, err := rewriteOpenAIImageSizeMultipart(body, boundary, normalizedSize)
		if err != nil {
			return nil, "", "", err
		}
		return rewritten, rewrittenType, strings.TrimSpace(normalizedSize), nil
	}

	if !gjson.ValidBytes(body) {
		return nil, "", "", fmt.Errorf("invalid json body")
	}

	sizeValue := strings.TrimSpace(gjson.GetBytes(body, "size").String())
	imageSizeValue := strings.TrimSpace(gjson.GetBytes(body, "image_size").String())
	aspectValue := strings.TrimSpace(gjson.GetBytes(body, "aspect_ratio").String())

	normalizedSize, _, errCode, errMessage := normalizeOpenAIImageSizeWithAspect(sizeValue, imageSizeValue, aspectValue)
	if errCode != "" {
		return nil, "", "", newOpenAIImageRequestError(errCode, errMessage)
	}

	updated := body
	if strings.TrimSpace(normalizedSize) != "" && strings.TrimSpace(normalizedSize) != sizeValue {
		next, err := sjson.SetBytes(updated, "size", normalizedSize)
		if err != nil {
			return nil, "", "", err
		}
		updated = next
	}

	if hasImageSizeField := gjson.GetBytes(updated, "image_size").Exists(); hasImageSizeField {
		next, err := sjson.DeleteBytes(updated, "image_size")
		if err != nil {
			return nil, "", "", err
		}
		updated = next
	}
	if hasAspectField := gjson.GetBytes(updated, "aspect_ratio").Exists(); hasAspectField {
		next, err := sjson.DeleteBytes(updated, "aspect_ratio")
		if err != nil {
			return nil, "", "", err
		}
		updated = next
	}

	nextType := contentType
	if strings.TrimSpace(nextType) == "" {
		nextType = "application/json"
	}
	return updated, nextType, strings.TrimSpace(normalizedSize), nil
}

func scanOpenAIImageSizeMultipartFields(body []byte, boundary string) (string, bool, string, bool, string, bool, error) {
	reader := multipart.NewReader(bytes.NewReader(body), boundary)
	var (
		sizeValue      string
		imageSizeValue string
		aspectValue    string
		hasSize        bool
		hasImageSize   bool
		hasAspect      bool
	)

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", false, "", false, "", false, fmt.Errorf("read multipart image request: %w", err)
		}

		name := strings.TrimSpace(part.FormName())
		if name == "" {
			_ = part.Close()
			continue
		}
		if part.FileName() != "" {
			_ = part.Close()
			continue
		}

		switch name {
		case "size", "image_size", "aspect_ratio":
			raw, readErr := io.ReadAll(io.LimitReader(part, 1<<20))
			_ = part.Close()
			if readErr != nil {
				return "", false, "", false, "", false, fmt.Errorf("read multipart field %q: %w", name, readErr)
			}
			value := strings.TrimSpace(string(raw))
			switch name {
			case "size":
				sizeValue = value
				hasSize = true
			case "image_size":
				imageSizeValue = value
				hasImageSize = true
			case "aspect_ratio":
				aspectValue = value
				hasAspect = true
			}
		default:
			_ = part.Close()
		}
	}

	return sizeValue, hasSize, imageSizeValue, hasImageSize, aspectValue, hasAspect, nil
}

func rewriteOpenAIImageSizeMultipart(body []byte, boundary string, normalizedSize string) ([]byte, string, error) {
	reader := multipart.NewReader(bytes.NewReader(body), boundary)
	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)
	replacedSize := false

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, "", fmt.Errorf("read multipart image request: %w", err)
		}

		name := strings.TrimSpace(part.FormName())
		if name == "" {
			_ = part.Close()
			continue
		}

		if part.FileName() == "" && (name == "image_size" || name == "aspect_ratio") {
			_, _ = io.Copy(io.Discard, part)
			_ = part.Close()
			continue
		}

		headers := cloneMIMEHeader(part.Header)
		partWriter, createErr := writer.CreatePart(headers)
		if createErr != nil {
			_ = part.Close()
			return nil, "", createErr
		}

		if part.FileName() == "" && name == "size" && strings.TrimSpace(normalizedSize) != "" {
			if _, writeErr := io.WriteString(partWriter, normalizedSize); writeErr != nil {
				_ = part.Close()
				return nil, "", writeErr
			}
			replacedSize = true
			_ = part.Close()
			continue
		}

		if _, copyErr := io.Copy(partWriter, part); copyErr != nil {
			_ = part.Close()
			return nil, "", copyErr
		}
		_ = part.Close()
	}

	if !replacedSize && strings.TrimSpace(normalizedSize) != "" {
		if err := writer.WriteField("size", normalizedSize); err != nil {
			return nil, "", err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, "", err
	}

	return buffer.Bytes(), writer.FormDataContentType(), nil
}
