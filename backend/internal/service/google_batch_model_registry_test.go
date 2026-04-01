package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type googleBatchBindingRepoStub struct {
	items map[string]*UpstreamResourceBinding
}

func (s *googleBatchBindingRepoStub) Upsert(_ context.Context, binding *UpstreamResourceBinding) error {
	if s.items == nil {
		s.items = map[string]*UpstreamResourceBinding{}
	}
	if binding != nil {
		cloned := *binding
		s.items[strings.ToLower(strings.TrimSpace(binding.ResourceName))] = &cloned
	}
	return nil
}

func (s *googleBatchBindingRepoStub) Get(_ context.Context, _ string, resourceName string) (*UpstreamResourceBinding, error) {
	return s.items[strings.ToLower(strings.TrimSpace(resourceName))], nil
}

func (s *googleBatchBindingRepoStub) GetByNames(_ context.Context, _ string, resourceNames []string) ([]*UpstreamResourceBinding, error) {
	result := make([]*UpstreamResourceBinding, 0, len(resourceNames))
	for _, name := range resourceNames {
		if binding := s.items[strings.ToLower(strings.TrimSpace(name))]; binding != nil {
			result = append(result, binding)
		}
	}
	return result, nil
}

func (s *googleBatchBindingRepoStub) SoftDelete(_ context.Context, _, _ string) error {
	return nil
}

func TestGoogleBatchModelRegistryFamilyRecognizesAliases(t *testing.T) {
	family, ok := googleBatchModelRegistryFamily("publishers/google/models/gemini-3.1-flash-lite-preview")
	require.True(t, ok)
	require.Equal(t, "gemini_flash_lite", family)
	require.Equal(t, "gemini_pro", normalizeGoogleBatchModelFamily("models/gemini-2.5-pro-preview-tts"))
}

func TestResolveGoogleBatchInputMetadataPrefersExplicitModelAndBindingTokenEstimate(t *testing.T) {
	svc := &GeminiMessagesCompatService{
		resourceBindingRepo: &googleBatchBindingRepoStub{
			items: map[string]*UpstreamResourceBinding{
				"files/input-a": {
					ResourceName: "files/input-a",
					AccountID:    10,
					MetadataJSON: map[string]any{
						googleBatchBindingMetadataEstimatedTokens: int64(120),
						googleBatchBindingMetadataModelFamily:     "gemini_flash_lite",
						googleBatchBindingMetadataSourceProtocol:  UpstreamProviderAIStudio,
					},
				},
				"files/input-b": {
					ResourceName: "files/input-b",
					AccountID:    10,
					MetadataJSON: map[string]any{
						googleBatchBindingMetadataEstimatedTokens: int64(80),
						googleBatchBindingMetadataModelFamily:     "gemini_flash_lite",
						googleBatchBindingMetadataSourceProtocol:  UpstreamProviderAIStudio,
					},
				},
			},
		},
	}

	input := GoogleBatchForwardInput{
		Method: http.MethodPost,
		Path:   "/v1beta/models/gemini-2.5-pro:batchGenerateContent",
		Body: []byte(`{
			"batch": {
				"input_config": {
					"requests": [{"fileName": "files/input-a"}, {"fileName": "files/input-b"}]
				}
			}
		}`),
	}

	metadata, err := svc.resolveGoogleBatchInputMetadata(context.Background(), input)
	require.NoError(t, err)
	require.Equal(t, "gemini-2.5-pro", metadata.requestedModel)
	require.Equal(t, "gemini_pro", metadata.modelFamily)
	require.Equal(t, int64(200), metadata.estimatedTokens)
	require.Equal(t, UpstreamProviderAIStudio, metadata.sourceProtocol)
	require.Equal(t, []string{"files/input-a", "files/input-b"}, metadata.sourceResourceNames)
}

func TestBuildGoogleBatchFileBindingMetadataIncludesDigestAndUploadFields(t *testing.T) {
	payload := `{"key":"line-1","request":{"model":"publishers/google/models/gemini-2.5-flash"},"estimated_batch_tokens":33}` + "\n"
	svc := &GeminiMessagesCompatService{}
	input := GoogleBatchForwardInput{
		Method:   http.MethodPost,
		Path:     "/upload/v1beta/files",
		OpenBody: func() (io.ReadCloser, error) { return io.NopCloser(strings.NewReader(payload)), nil },
	}

	metadata := svc.buildGoogleBatchFileBindingMetadata(input)
	sum := sha256.Sum256([]byte(payload))

	require.Equal(t, UpstreamProviderAIStudio, metadata[googleBatchBindingMetadataSourceProtocol])
	require.Equal(t, "gemini-2.5-flash", metadata[googleBatchBindingMetadataRequestedModel])
	require.Equal(t, "gemini_flash", metadata[googleBatchBindingMetadataModelFamily])
	require.Equal(t, int64(33), metadata[googleBatchBindingMetadataEstimatedTokens])
	require.Equal(t, hex.EncodeToString(sum[:]), metadata[googleBatchBindingMetadataContentDigest])
	uploadedAt, ok := metadata[googleBatchBindingMetadataUploadedAt].(string)
	require.True(t, ok)
	_, err := time.Parse(time.RFC3339, uploadedAt)
	require.NoError(t, err)
}
