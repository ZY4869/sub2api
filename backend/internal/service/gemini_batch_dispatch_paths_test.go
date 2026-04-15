package service

import (
	"context"
	"net/http"
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestGeminiBatchDispatchPaths(t *testing.T) {
	t.Parallel()

	svc := &GeminiMessagesCompatService{}

	tests := []struct {
		name       string
		call       func() error
		wantReason string
	}{
		{
			name: "google files unsupported action path",
			call: func() error {
				_, _, err := svc.ForwardGoogleFiles(context.Background(), GoogleBatchForwardInput{
					Method: http.MethodPost,
					Path:   "/v1beta/files:publish",
				})
				return err
			},
			wantReason: "GOOGLE_FILES_PATH_UNSUPPORTED",
		},
		{
			name: "google batches unsupported root post path",
			call: func() error {
				_, _, err := svc.ForwardGoogleBatches(context.Background(), GoogleBatchForwardInput{
					Method: http.MethodPost,
					Path:   "/v1beta/batches",
				})
				return err
			},
			wantReason: "GOOGLE_BATCH_PATH_UNSUPPORTED",
		},
		{
			name: "google file download missing resource",
			call: func() error {
				_, _, err := svc.ForwardGoogleFileDownload(context.Background(), GoogleBatchForwardInput{
					Method: http.MethodGet,
					Path:   "/download/v1beta/files/",
				})
				return err
			},
			wantReason: "GOOGLE_FILE_DOWNLOAD_NOT_FOUND",
		},
		{
			name: "google archive batch missing resource",
			call: func() error {
				_, _, err := svc.ForwardGoogleArchiveBatch(context.Background(), GoogleBatchForwardInput{
					Method: http.MethodGet,
					Path:   "/google/batch/archive/v1beta/batches/",
				})
				return err
			},
			wantReason: "GOOGLE_BATCH_ARCHIVE_NOT_FOUND",
		},
		{
			name: "google archive file missing resource",
			call: func() error {
				_, _, err := svc.ForwardGoogleArchiveFileDownload(context.Background(), GoogleBatchForwardInput{
					Method: http.MethodGet,
					Path:   "/google/batch/archive/v1beta/files/",
				})
				return err
			},
			wantReason: "GOOGLE_ARCHIVE_FILE_NOT_FOUND",
		},
		{
			name: "vertex batch invalid path",
			call: func() error {
				_, _, err := svc.ForwardVertexBatchPredictionJobs(context.Background(), GoogleBatchForwardInput{
					Method: http.MethodGet,
					Path:   "/v1/projects/demo/locations/us-central1/jobs",
				})
				return err
			},
			wantReason: "VERTEX_BATCH_PATH_INVALID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call()
			require.Error(t, err)

			appErr := infraerrors.FromError(err)
			require.NotNil(t, appErr)
			require.Equal(t, tt.wantReason, appErr.Reason)
		})
	}
}

func TestGeminiBatchDispatchPaths_PatchUpdateRoutesUseBatchResourceFlow(t *testing.T) {
	t.Parallel()

	svc := &GeminiMessagesCompatService{}
	paths := []string{
		"/v1beta/batches/batch-1:updateGenerateContentBatch",
		"/v1beta/batches/batch-1:updateEmbedContentBatch",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			_, _, err := svc.ForwardGoogleBatches(context.Background(), GoogleBatchForwardInput{
				Method: http.MethodPatch,
				Path:   path,
			})
			require.Error(t, err)

			appErr := infraerrors.FromError(err)
			require.NotNil(t, appErr)
			require.NotEqual(t, "GOOGLE_BATCH_PATH_UNSUPPORTED", appErr.Reason)
		})
	}
}
