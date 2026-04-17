package service

import (
	"context"
	"database/sql"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestGetUsageRequestPreviewForUsageReturnsUnavailableWhenTraceMissing(t *testing.T) {
	svc := NewOpsService(&opsRepoMock{
		GetUsageRequestPreviewFn: func(ctx context.Context, userID, apiKeyID int64, requestID string) (*UsageRequestPreview, error) {
			return nil, sql.ErrNoRows
		},
	}, nil, &config.Config{
		Ops: config.OpsConfig{
			Enabled: true,
			RequestDetails: config.OpsRequestDetailsConfig{
				Enabled: true,
			},
		},
	}, nil, nil, nil, nil, nil, nil, nil, nil)

	preview, err := svc.GetUsageRequestPreviewForUsage(context.Background(), &UsageLog{
		UserID:    42,
		APIKeyID:  11,
		RequestID: "req-missing",
	})

	require.NoError(t, err)
	require.False(t, preview.Available)
	require.Equal(t, "req-missing", preview.RequestID)
	require.Nil(t, preview.CapturedAt)
}

func TestGetUsageRequestPreviewForUsageReturnsUnavailableWhenCaptureDisabled(t *testing.T) {
	svc := NewOpsService(&opsRepoMock{}, nil, &config.Config{
		Ops: config.OpsConfig{
			Enabled: false,
			RequestDetails: config.OpsRequestDetailsConfig{
				Enabled: true,
			},
		},
	}, nil, nil, nil, nil, nil, nil, nil, nil)

	preview, err := svc.GetUsageRequestPreviewForUsage(context.Background(), &UsageLog{
		UserID:    42,
		APIKeyID:  11,
		RequestID: "req-disabled",
	})

	require.NoError(t, err)
	require.False(t, preview.Available)
	require.Equal(t, "req-disabled", preview.RequestID)
}
