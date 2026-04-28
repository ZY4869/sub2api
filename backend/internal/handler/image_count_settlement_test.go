package handler

import (
	"context"
	"errors"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

type imageCountQuotaServiceStub struct {
	reserveOK     bool
	reserveErr    error
	rollbackErr   error
	reserveCounts []int
	rollbackUnits []int
}

func (s *imageCountQuotaServiceStub) TryReserveImageCount(_ context.Context, _ int64, count int) (bool, error) {
	s.reserveCounts = append(s.reserveCounts, count)
	if s.reserveErr != nil {
		return false, s.reserveErr
	}
	return s.reserveOK, nil
}

func (s *imageCountQuotaServiceStub) RollbackImageCount(_ context.Context, _ int64, count int) error {
	s.rollbackUnits = append(s.rollbackUnits, count)
	return s.rollbackErr
}

func TestOpenAIImages_ImageCountSettleRollsBackNegativeDiff(t *testing.T) {
	quota := &imageCountQuotaServiceStub{reserveOK: true}
	core, logs := observer.New(zap.InfoLevel)

	settled := settleAPIKeyImageCountUnits(
		context.Background(),
		zap.New(core),
		quota,
		imageCountSettleTestAPIKey(),
		8,
		1,
		service.OpenAIImageSizeTier1K,
	)

	require.True(t, settled)
	require.Equal(t, []int{7}, quota.rollbackUnits)
	require.Empty(t, quota.reserveCounts)
	require.Equal(t, 1, logs.FilterMessage("api_key_image_count_settled").Len())
}

func TestOpenAIImages_ImageCountSettleReservesPositiveDiff(t *testing.T) {
	quota := &imageCountQuotaServiceStub{reserveOK: true}
	core, logs := observer.New(zap.InfoLevel)

	settled := settleAPIKeyImageCountUnits(
		context.Background(),
		zap.New(core),
		quota,
		imageCountSettleTestAPIKey(),
		2,
		2,
		service.OpenAIImageSizeTier4K,
	)

	require.True(t, settled)
	require.Equal(t, []int{6}, quota.reserveCounts)
	require.Empty(t, quota.rollbackUnits)
	entry := logs.FilterMessage("api_key_image_count_settled").All()[0]
	require.Equal(t, int64(6), entry.ContextMap()["extra_units"])
}

func TestOpenAIImages_ImageCountSettleLogsPositiveDiffReserveFailure(t *testing.T) {
	quota := &imageCountQuotaServiceStub{reserveOK: false, reserveErr: errors.New("quota store down")}
	core, logs := observer.New(zap.WarnLevel)

	settled := settleAPIKeyImageCountUnits(
		context.Background(),
		zap.New(core),
		quota,
		imageCountSettleTestAPIKey(),
		2,
		2,
		service.OpenAIImageSizeTier4K,
	)

	require.True(t, settled)
	require.Equal(t, []int{6}, quota.reserveCounts)
	require.Equal(t, 1, logs.FilterMessage("api_key_image_count_settle_reserve_failed").Len())
}

func TestOpenAIResponses_ImageOnlyRejectsModelOutsideConfiguredScope(t *testing.T) {
	apiKey := &service.APIKey{
		ImageOnlyEnabled: true,
		GroupBindings: []service.APIKeyGroupBinding{{
			GroupID: 1,
			Group: &service.Group{
				ID:       1,
				Platform: service.PlatformOpenAI,
				Status:   service.StatusActive,
			},
			ModelPatterns: []string{"gpt-image-2"},
		}},
	}

	require.True(t, service.IsOpenAINativeImageModelID("gpt-image-2"))
	require.True(t, service.APIKeyAllowsConfiguredModel(apiKey, "gpt-image-2"))
	require.True(t, service.IsOpenAINativeImageModelID("gpt-image-1.5"))
	require.False(t, service.APIKeyAllowsConfiguredModel(apiKey, "gpt-image-1.5"))
	require.False(t, service.IsOpenAINativeImageModelID("gpt-5.4"))
}

func TestPublicImageRoute_ImageOnlyScopeRequiresConfiguredModel(t *testing.T) {
	apiKey := imageCountSettleTestAPIKey()
	apiKey.ImageOnlyEnabled = true
	apiKey.GroupBindings = []service.APIKeyGroupBinding{{
		GroupID: 1,
		Group:   &service.Group{ID: 1, Platform: service.PlatformOpenAI, Status: service.StatusActive},
		ModelPatterns: []string{
			"gpt-image-2",
		},
	}}

	require.True(t, service.APIKeyAllowsConfiguredModel(apiKey, "gpt-image-2"))
	require.False(t, service.APIKeyAllowsConfiguredModel(apiKey, "gemini-2.5-flash-image"))
}

func imageCountSettleTestAPIKey() *service.APIKey {
	return &service.APIKey{
		ID: 9,
		ImageCountWeights: map[string]int{
			service.OpenAIImageSizeTier1K: 1,
			service.OpenAIImageSizeTier2K: 2,
			service.OpenAIImageSizeTier4K: 4,
		},
	}
}
