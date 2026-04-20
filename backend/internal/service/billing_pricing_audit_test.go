//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestBillingCenterService_GetPricingAuditAggregatesStatusAndIssueSummaries(t *testing.T) {
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	repo.values[SettingKeyBillingPricingCatalogSnapshot] = mustModelCatalogJSON(t, BillingPricingCatalogSnapshot{
		UpdatedAt: time.Date(2026, 4, 20, 9, 0, 0, 0, time.UTC),
		Models: []BillingPricingPersistedModel{
			{Model: "gpt-5.4"},
			{Model: "legacy-snapshot-only-model"},
		},
	})

	pricingService := &PricingService{
		pricingData: map[string]*LiteLLMModelPricing{
			"gemini-3-flash-preview": {
				InputCostPerToken:  5e-7,
				OutputCostPerToken: 3e-6,
				LiteLLMProvider:    PlatformGemini,
				Mode:               "chat",
			},
			"gpt-5.4-pro": {
				InputCostPerToken:  3e-5,
				OutputCostPerToken: 1.8e-4,
				LiteLLMProvider:    PlatformOpenAI,
				Mode:               "responses",
			},
		},
	}
	svc := NewModelCatalogService(repo, nil, NewBillingService(&config.Config{}, pricingService), pricingService, &config.Config{})

	records, err := svc.buildCatalogRecords(context.Background())
	require.NoError(t, err)

	audit, err := svc.GetBillingPricingAudit(context.Background())
	require.NoError(t, err)
	require.NotNil(t, audit)

	require.Equal(t, len(records), audit.TotalModels)
	require.Equal(t, 0, len(audit.DuplicateModelIDs))
	expectedCollisionCounts := BillingPricingCollisionCountsBySource{}
	for _, collision := range audit.AuxIdentifierCollisions {
		switch collision.Source {
		case "aliases":
			expectedCollisionCounts.Aliases++
		case "protocol_ids":
			expectedCollisionCounts.ProtocolIDs++
		case "pricing_lookup_ids":
			expectedCollisionCounts.PricingLookupIDs++
		}
	}
	require.Equal(t, expectedCollisionCounts, audit.CollisionCountsBySource)
	require.Greater(t, audit.CollisionCountsBySource.PricingLookupIDs, 0)

	statusTotal := audit.PricingStatusCounts.OK + audit.PricingStatusCounts.Fallback + audit.PricingStatusCounts.Conflict + audit.PricingStatusCounts.Missing
	require.Equal(t, audit.TotalModels, statusTotal)
	require.Greater(t, audit.PricingStatusCounts.OK, 0)
	require.Greater(t, audit.PricingStatusCounts.Missing, 0)

	require.Greater(t, audit.MissingInSnapshotCount, 0)
	require.Equal(t, 1, audit.SnapshotOnlyCount)
	require.Equal(t, []string{"legacy-snapshot-only-model"}, audit.SnapshotOnlyModels)
	require.True(t, audit.RefreshRequired)
	require.NotNil(t, audit.SnapshotUpdatedAt)

	require.NotEmpty(t, audit.ProviderIssueCounts)
	for _, item := range audit.ProviderIssueCounts {
		require.Equal(t, item.Total, item.Fallback+item.Conflict+item.Missing)
	}
	for index := 1; index < len(audit.ProviderIssueCounts); index++ {
		require.GreaterOrEqual(t, audit.ProviderIssueCounts[index-1].Total, audit.ProviderIssueCounts[index].Total)
	}

	require.NotEmpty(t, audit.PricingIssueExamples)
	require.LessOrEqual(t, len(audit.PricingIssueExamples), billingPricingAuditIssueExampleLimit)
	for _, example := range audit.PricingIssueExamples {
		require.NotEqual(t, BillingPricingStatusOK, example.PricingStatus)
		require.NotEmpty(t, example.Model)
	}
	for index := 1; index < len(audit.PricingIssueExamples); index++ {
		prev := billingPricingIssuePriority(audit.PricingIssueExamples[index-1].PricingStatus)
		curr := billingPricingIssuePriority(audit.PricingIssueExamples[index].PricingStatus)
		require.LessOrEqual(t, prev, curr)
	}
}
