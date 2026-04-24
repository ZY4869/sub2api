package service

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/textproto"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func DetectOpenAIImageRequestModel(body []byte, contentType string) (string, error) {
	if len(body) == 0 {
		return "", nil
	}
	mediaType, params, err := mime.ParseMediaType(strings.TrimSpace(contentType))
	if err != nil {
		mediaType = strings.TrimSpace(contentType)
	}
	if strings.HasPrefix(strings.ToLower(mediaType), "multipart/form-data") {
		boundary := strings.TrimSpace(params["boundary"])
		if boundary == "" {
			return "", fmt.Errorf("missing multipart boundary")
		}
		return detectMultipartImageField(body, boundary, "model")
	}
	if !gjson.ValidBytes(body) {
		return "", fmt.Errorf("invalid json body")
	}
	return strings.TrimSpace(gjson.GetBytes(body, "model").String()), nil
}

func DetectOpenAIImageRequestSize(body []byte, contentType string) string {
	if len(body) == 0 {
		return ""
	}
	mediaType, params, err := mime.ParseMediaType(strings.TrimSpace(contentType))
	if err != nil {
		mediaType = strings.TrimSpace(contentType)
	}
	if strings.HasPrefix(strings.ToLower(mediaType), "multipart/form-data") {
		boundary := strings.TrimSpace(params["boundary"])
		if boundary == "" {
			return ""
		}
		size, _ := detectMultipartImageField(body, boundary, "size")
		return strings.TrimSpace(size)
	}
	return strings.TrimSpace(gjson.GetBytes(body, "size").String())
}

func DetectOpenAIImageRequestN(body []byte, contentType string) int {
	if len(body) == 0 {
		return 1
	}
	mediaType, params, err := mime.ParseMediaType(strings.TrimSpace(contentType))
	if err != nil {
		mediaType = strings.TrimSpace(contentType)
	}
	if strings.HasPrefix(strings.ToLower(mediaType), "multipart/form-data") {
		boundary := strings.TrimSpace(params["boundary"])
		if boundary == "" {
			return 1
		}
		raw, _ := detectMultipartImageField(body, boundary, "n")
		n, err := strconv.Atoi(strings.TrimSpace(raw))
		if err == nil && n > 0 {
			return n
		}
		return 1
	}
	if !gjson.ValidBytes(body) {
		return 1
	}
	n := int(gjson.GetBytes(body, "n").Int())
	if n > 0 {
		return n
	}
	return 1
}

func RewriteOpenAIImageRequestModel(body []byte, contentType string, mappedModel string) ([]byte, string, error) {
	mappedModel = strings.TrimSpace(mappedModel)
	if mappedModel == "" || len(body) == 0 {
		return body, contentType, nil
	}

	mediaType, params, err := mime.ParseMediaType(strings.TrimSpace(contentType))
	if err != nil {
		mediaType = strings.TrimSpace(contentType)
	}
	if strings.HasPrefix(strings.ToLower(mediaType), "multipart/form-data") {
		boundary := strings.TrimSpace(params["boundary"])
		if boundary == "" {
			return nil, "", fmt.Errorf("missing multipart boundary")
		}
		return rewriteMultipartImageField(body, boundary, "model", mappedModel)
	}
	if !gjson.ValidBytes(body) {
		return nil, "", fmt.Errorf("invalid json body")
	}
	rewritten, err := sjson.SetBytes(body, "model", mappedModel)
	if err != nil {
		return nil, "", err
	}
	nextType := contentType
	if strings.TrimSpace(nextType) == "" {
		nextType = "application/json"
	}
	return rewritten, nextType, nil
}

func CountOpenAIImageResponse(body []byte) int {
	if len(body) == 0 || !gjson.ValidBytes(body) {
		return 0
	}
	count := int(gjson.GetBytes(body, "data.#").Int())
	if count > 0 {
		return count
	}
	if gjson.GetBytes(body, "data.0").Exists() {
		return 1
	}
	return 0
}

func DetectOpenAIResponsesImageGenerationToolModel(body []byte) (string, bool) {
	if len(body) == 0 || !gjson.ValidBytes(body) {
		return "", false
	}
	for _, tool := range gjson.GetBytes(body, "tools").Array() {
		if strings.TrimSpace(tool.Get("type").String()) != "image_generation" {
			continue
		}
		return strings.TrimSpace(tool.Get("model").String()), true
	}
	return "", false
}

func detectMultipartImageField(body []byte, boundary string, fieldName string) (string, error) {
	reader := multipart.NewReader(bytes.NewReader(body), boundary)
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			return "", nil
		}
		if err != nil {
			return "", err
		}
		if part.FormName() != fieldName {
			_ = part.Close()
			continue
		}
		value, readErr := io.ReadAll(io.LimitReader(part, 1<<20))
		_ = part.Close()
		if readErr != nil {
			return "", readErr
		}
		return strings.TrimSpace(string(value)), nil
	}
}

func rewriteMultipartImageField(body []byte, boundary string, fieldName string, replacement string) ([]byte, string, error) {
	reader := multipart.NewReader(bytes.NewReader(body), boundary)
	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)
	replaced := false

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, "", err
		}

		headers := cloneMIMEHeader(part.Header)
		partWriter, createErr := writer.CreatePart(headers)
		if createErr != nil {
			_ = part.Close()
			return nil, "", createErr
		}
		if part.FormName() == fieldName && part.FileName() == "" {
			if _, writeErr := io.WriteString(partWriter, replacement); writeErr != nil {
				_ = part.Close()
				return nil, "", writeErr
			}
			replaced = true
			_ = part.Close()
			continue
		}
		if _, copyErr := io.Copy(partWriter, part); copyErr != nil {
			_ = part.Close()
			return nil, "", copyErr
		}
		_ = part.Close()
	}

	if !replaced {
		if fieldErr := writer.WriteField(fieldName, replacement); fieldErr != nil {
			return nil, "", fieldErr
		}
	}
	if err := writer.Close(); err != nil {
		return nil, "", err
	}
	return buffer.Bytes(), writer.FormDataContentType(), nil
}

func cloneMIMEHeader(header textproto.MIMEHeader) textproto.MIMEHeader {
	cloned := make(textproto.MIMEHeader, len(header))
	for key, values := range header {
		copied := make([]string, len(values))
		copy(copied, values)
		cloned[key] = copied
	}
	return cloned
}
