package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestImageBatchGroupAllowsNormalizesProviderAliases(t *testing.T) {
	settings := &ImageBatchGroupSettings{
		Enabled:          true,
		AllowedProviders: []string{ImageBatchProviderGemini},
		AllowedModels:    []string{"imagen-*"},
		MaxItems:         2,
	}

	require.True(t, imageBatchGroupAllows(settings, "google", "imagen-4", 2))
	require.True(t, imageBatchGroupAllows(settings, "ai_studio", "imagen-4", 1))
	require.False(t, imageBatchGroupAllows(settings, "openai", "imagen-4", 1))
	require.False(t, imageBatchGroupAllows(settings, "google", "gemini-text", 1))
	require.False(t, imageBatchGroupAllows(settings, "google", "imagen-4", 3))
}

func TestImageBatchGroupAllowsVertexProviderAliases(t *testing.T) {
	settings := &ImageBatchGroupSettings{
		Enabled:          true,
		AllowedProviders: []string{ImageBatchProviderVertex},
		AllowedModels:    []string{"imagen-*"},
		MaxItems:         2,
	}

	require.Equal(t, ImageBatchProviderVertex, normalizeImageBatchProvider("vertex_ai"))
	require.Equal(t, PlatformGemini, imageBatchProviderPlatform("vertexai"))
	require.True(t, imageBatchGroupAllows(settings, "vertexai", "imagen-4", 1))
	require.True(t, imageBatchProviderModelAllowed(settings, ImageBatchProviderVertex, "imagen-4"))
	require.False(t, imageBatchGroupAllows(settings, "gemini_api", "imagen-4", 1))
}

func TestImageBatchDownloadLimitPrefersGroupThenGlobal(t *testing.T) {
	svc := &ImageBatchService{cfg: &config.Config{ImageBatch: config.ImageBatchConfig{DownloadMaxBytes: 42}}}

	require.Equal(t, int64(42), svc.imageBatchDownloadLimitBytes(nil))
	require.Equal(t, int64(17), svc.imageBatchDownloadLimitBytes(&ImageBatchGroupSettings{MaxDownloadBytes: 17}))
}

func TestImageBatchDownloadConcurrencyBusy(t *testing.T) {
	svc := &ImageBatchService{cfg: &config.Config{ImageBatch: config.ImageBatchConfig{DownloadConcurrency: 1}}}

	release, err := svc.acquireImageBatchDownload(context.Background(), &ImageBatchJob{})
	require.NoError(t, err)
	defer release()

	_, err = svc.acquireImageBatchDownload(context.Background(), &ImageBatchJob{})
	require.True(t, errors.Is(err, ErrImageBatchDownloadBusy))
}

func TestReadImageBatchProviderResultRejectsOversizedPayload(t *testing.T) {
	body, err := readImageBatchProviderResult(strings.NewReader("abcd"), 4)
	require.NoError(t, err)
	require.Equal(t, "abcd", string(body))

	_, err = readImageBatchProviderResult(strings.NewReader("abcde"), 4)
	require.True(t, errors.Is(err, ErrImageBatchDownloadTooLarge))
}

func TestImageBatchOutputCleanupAfterDisabledByDefault(t *testing.T) {
	require.Zero(t, (&ImageBatchService{}).outputCleanupAfter())
	require.Equal(t, 2*time.Hour, (&ImageBatchService{cfg: &config.Config{ImageBatch: config.ImageBatchConfig{OutputCleanupAfterHours: 2}}}).outputCleanupAfter())
}

func TestImageBatchJobResponseDoesNotExposeInternalRoutingFields(t *testing.T) {
	now := time.Now().UTC()
	resp := ImageBatchJobToResponse(&ImageBatchJob{
		ID:                "job_1",
		UserID:            7,
		APIKeyID:          9,
		Provider:          ImageBatchProviderGemini,
		DisplayModelID:    "public-imagen",
		TargetModelID:     "internal-imagen",
		ProviderBatchName: "providers/internal/batches/1",
		Status:            ImageBatchJobCompleted,
		ItemCount:         1,
		SuccessCount:      1,
		CreatedAt:         now,
		UpdatedAt:         now,
	})

	raw, err := json.Marshal(resp)
	require.NoError(t, err)
	payload := string(raw)
	require.Contains(t, payload, `"display_model_id":"public-imagen"`)
	require.NotContains(t, payload, "target_model_id")
	require.NotContains(t, payload, "internal-imagen")
	require.NotContains(t, payload, "provider_batch")
	require.NotContains(t, payload, "api_key")
}

func TestImageBatchJobResponseMapsAsyncTaskStatusesWithoutProviderTaskID(t *testing.T) {
	for _, status := range []ImageBatchJobStatus{
		ImageBatchJobSubmitted,
		ImageBatchJobRunning,
		ImageBatchJobCompleted,
		ImageBatchJobFailed,
		ImageBatchJobCancelled,
	} {
		resp := ImageBatchJobToResponse(&ImageBatchJob{
			ID:                "batch_" + string(status),
			Provider:          ImageBatchProviderGemini,
			DisplayModelID:    "imagen",
			Status:            status,
			ProviderBatchName: "providers/internal/tasks/" + string(status),
			CreatedAt:         time.Now().UTC(),
			UpdatedAt:         time.Now().UTC(),
		})

		raw, err := json.Marshal(resp)
		require.NoError(t, err)
		payload := string(raw)
		require.Contains(t, payload, `"status":"`+string(status)+`"`)
		require.NotContains(t, payload, "providers/internal/tasks")
		require.NotContains(t, payload, "task_id")
		require.NotContains(t, payload, "provider_batch")
	}
}
