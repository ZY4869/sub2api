package service

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"strings"
	"time"
)

type googleBatchBodyInspection struct {
	requestedModel  string
	modelFamily     string
	estimatedTokens int64
	contentDigest   string
}

func (s *GeminiMessagesCompatService) buildGoogleBatchFileBindingMetadata(input GoogleBatchForwardInput) map[string]any {
	metadata := map[string]any{
		googleBatchBindingMetadataSourceProtocol: publicGoogleBatchProtocol(input.Path),
		googleBatchBindingMetadataUploadedAt:     time.Now().UTC().Format(time.RFC3339),
	}
	inspection, err := inspectGoogleBatchForwardInputBody(input)
	if err != nil {
		return metadata
	}
	metadata[googleBatchBindingMetadataEstimatedTokens] = inspection.estimatedTokens
	if inspection.requestedModel != "" {
		metadata[googleBatchBindingMetadataRequestedModel] = inspection.requestedModel
	}
	if inspection.modelFamily != "" {
		metadata[googleBatchBindingMetadataModelFamily] = inspection.modelFamily
	}
	if inspection.contentDigest != "" {
		metadata[googleBatchBindingMetadataContentDigest] = inspection.contentDigest
	}
	return metadata
}

func inspectGoogleBatchForwardInputBody(input GoogleBatchForwardInput) (googleBatchBodyInspection, error) {
	if len(input.Body) > 0 {
		return inspectGoogleBatchBodyReader(bytes.NewReader(input.Body))
	}
	body, err := input.OpenRequestBody()
	if err != nil {
		return googleBatchBodyInspection{}, err
	}
	defer func() { _ = body.Close() }()
	return inspectGoogleBatchBodyReader(body)
}

func inspectGoogleBatchBodyReader(reader io.Reader) (googleBatchBodyInspection, error) {
	if reader == nil {
		return googleBatchBodyInspection{}, nil
	}
	var inspection googleBatchBodyInspection
	hasher := sha256.New()
	teeReader := io.TeeReader(reader, hasher)
	models := make([]string, 0, 1)
	if err := walkJSONLLines(teeReader, func(_ int, line []byte) error {
		if model := strings.TrimSpace(extractGoogleBatchModelID("", line)); model != "" {
			models = append(models, model)
		}
		inspection.estimatedTokens += estimateGoogleBatchTokensFromPayload(line)
		return nil
	}); err != nil {
		return googleBatchBodyInspection{}, err
	}
	inspection.contentDigest = hex.EncodeToString(hasher.Sum(nil))
	inspection.requestedModel = stableGoogleBatchModelValue(models)
	if inspection.requestedModel != "" {
		inspection.modelFamily = normalizeGoogleBatchModelFamily(inspection.requestedModel)
	}
	return inspection, nil
}

func stableGoogleBatchModelValue(values []string) string {
	var stable string
	for _, value := range values {
		current := strings.TrimSpace(value)
		if current == "" {
			continue
		}
		if stable == "" {
			stable = current
			continue
		}
		if !strings.EqualFold(stable, current) {
			return ""
		}
	}
	return stable
}
