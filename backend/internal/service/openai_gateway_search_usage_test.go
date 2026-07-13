package service

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestResolveWebSearchPricePerCallUSD(t *testing.T) {
	negative := -0.5
	free := 0.0
	override := 0.25

	tests := []struct {
		name  string
		group *Group
		want  float64
	}{
		{name: "nil group uses default", group: nil, want: DefaultWebSearchPricePerCallUSD},
		{name: "nil price uses default", group: &Group{}, want: DefaultWebSearchPricePerCallUSD},
		{name: "negative price uses default", group: &Group{WebSearchPricePerCall: &negative}, want: DefaultWebSearchPricePerCallUSD},
		{name: "zero price is free", group: &Group{WebSearchPricePerCall: &free}, want: 0},
		{name: "positive price overrides default", group: &Group{WebSearchPricePerCall: &override}, want: override},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, ResolveWebSearchPricePerCallUSD(tt.group))
		})
	}
}

func TestOpenAIGatewayServiceRecordWebSearchUsage_BillsSuccessful2xxOnce(t *testing.T) {
	groupID := int64(7101)
	groupRate := 1.5
	userRate := 2.0
	accountRate := 3.0
	price := 0.025
	payloadHash := HashUsageRequestPayload([]byte(`{"query":"cleanroom alpha search"}`))

	usageRepo := &openAIRecordUsageLogRepoStub{}
	billingRepo := &openAIRecordUsageBillingRepoStub{result: &UsageBillingApplyResult{Applied: true}}
	quotaSvc := &openAIRecordUsageAPIKeyQuotaStub{}
	rateRepo := &openAIUserGroupRateRepoStub{rate: &userRate}
	svc := newOpenAIRecordUsageServiceWithBillingRepoForTest(
		usageRepo,
		billingRepo,
		&openAIRecordUsageUserRepoStub{},
		&openAIRecordUsageSubRepoStub{},
		rateRepo,
	)

	err := svc.RecordWebSearchUsage(context.Background(), &OpenAIRecordWebSearchUsageInput{
		Result: &OpenAIAlphaSearchForwardResult{
			RequestID:  "search_2xx_once",
			StatusCode: http.StatusCreated,
			Duration:   1500 * time.Millisecond,
		},
		APIKey: &APIKey{
			ID:          8101,
			GroupID:     &groupID,
			Quota:       100,
			RateLimit1d: 100,
			Group: &Group{
				ID:                    groupID,
				Platform:              PlatformOpenAI,
				RateMultiplier:        groupRate,
				PeakRateEnabled:       true,
				PeakStart:             "00:00",
				PeakEnd:               "23:59",
				PeakRateMultiplier:    10,
				WebSearchPricePerCall: &price,
			},
		},
		User:               &User{ID: 9101},
		Account:            &Account{ID: 10101, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, RateMultiplier: &accountRate},
		InboundEndpoint:    " /v1/alpha/search ",
		UpstreamEndpoint:   " /v1/alpha/search ",
		UpstreamURL:        " https://api.openai.com/v1/alpha/search ",
		UpstreamService:    " openai ",
		UserAgent:          " test-agent ",
		IPAddress:          "203.0.113.9",
		RequestPayloadHash: payloadHash,
		APIKeyService:      quotaSvc,
	})

	require.NoError(t, err)
	require.Equal(t, 1, rateRepo.calls)
	require.Equal(t, 1, billingRepo.calls)
	require.Equal(t, 1, usageRepo.calls)
	require.Equal(t, 0, quotaSvc.quotaCalls, "billing repo path applies the atomic billing command instead of legacy quota writes")

	log := usageRepo.lastLog
	require.NotNil(t, log)
	require.Equal(t, int64(9101), log.UserID)
	require.Equal(t, int64(8101), log.APIKeyID)
	require.Equal(t, int64(10101), log.AccountID)
	require.NotNil(t, log.GroupID)
	require.Equal(t, groupID, *log.GroupID)
	require.Equal(t, OpenAIAlphaSearchUsageModel, log.Model)
	require.Equal(t, OpenAIAlphaSearchUsageModel, log.RequestedModel)
	require.Equal(t, BillingTypeBalance, log.BillingType)
	require.Equal(t, RequestTypeSync, log.RequestType)
	require.Equal(t, UsageLogStatusSucceeded, log.Status)
	require.NotNil(t, log.HTTPStatus)
	require.Equal(t, http.StatusCreated, *log.HTTPStatus)
	require.NotNil(t, log.DurationMs)
	require.Equal(t, 1500, *log.DurationMs)
	require.Equal(t, 0, log.InputTokens)
	require.Equal(t, 0, log.OutputTokens)
	require.Equal(t, 0, log.CacheCreationTokens)
	require.Equal(t, 0, log.CacheReadTokens)
	require.Equal(t, 0, log.ImageCount)
	require.Equal(t, ModelPricingCurrencyUSD, log.BillingCurrency)
	require.InDelta(t, price, log.TotalCost, 1e-12)
	require.InDelta(t, price*userRate, log.ActualCost, 1e-12)
	require.Equal(t, userRate, log.RateMultiplier, "web search uses the base user/group multiplier and does not apply peak multiplier")
	require.NotNil(t, log.AccountRateMultiplier)
	require.Equal(t, accountRate, *log.AccountRateMultiplier)
	require.NotNil(t, log.InboundEndpoint)
	require.Equal(t, "/v1/alpha/search", *log.InboundEndpoint)
	require.NotNil(t, log.UpstreamEndpoint)
	require.Equal(t, "/v1/alpha/search", *log.UpstreamEndpoint)
	require.NotNil(t, log.UpstreamURL)
	require.Equal(t, "https://api.openai.com/v1/alpha/search", *log.UpstreamURL)
	require.NotNil(t, log.UpstreamService)
	require.Equal(t, "openai", *log.UpstreamService)

	cmd := billingRepo.lastCmd
	require.NotNil(t, cmd)
	require.Equal(t, "search_2xx_once", cmd.RequestID)
	require.Equal(t, payloadHash, cmd.RequestPayloadHash)
	require.Equal(t, OpenAIAlphaSearchUsageModel, cmd.Model)
	require.Equal(t, BillingTypeBalance, cmd.BillingType)
	require.Equal(t, ModelPricingCurrencyUSD, cmd.BillingCurrency)
	require.InDelta(t, price*userRate, cmd.BalanceCost, 1e-12)
	require.InDelta(t, price*userRate, cmd.UserPlatformCost, 1e-12)
	require.InDelta(t, price*userRate, cmd.APIKeyQuotaCost, 1e-12)
	require.InDelta(t, price*userRate, cmd.APIKeyGroupQuotaCost, 1e-12)
	require.InDelta(t, price*userRate, cmd.APIKeyRateLimitCost, 1e-12)
	require.Equal(t, 0, cmd.InputTokens)
	require.Equal(t, 0, cmd.OutputTokens)
	require.Equal(t, 0, cmd.CacheCreationTokens)
	require.Equal(t, 0, cmd.CacheReadTokens)
	require.Equal(t, 0, cmd.ImageCount)
}

func TestOpenAIGatewayServiceRecordWebSearchUsage_PriceSemantics(t *testing.T) {
	negative := -0.25
	free := 0.0
	custom := 0.05
	groupRate := 1.25

	tests := []struct {
		name      string
		price     *float64
		wantPrice float64
	}{
		{name: "nil price uses default", price: nil, wantPrice: DefaultWebSearchPricePerCallUSD},
		{name: "negative price uses default", price: &negative, wantPrice: DefaultWebSearchPricePerCallUSD},
		{name: "zero price is free", price: &free, wantPrice: 0},
		{name: "positive price overrides default", price: &custom, wantPrice: custom},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groupID := int64(7200 + i)
			usageRepo := &openAIRecordUsageLogRepoStub{}
			billingRepo := &openAIRecordUsageBillingRepoStub{result: &UsageBillingApplyResult{Applied: true}}
			svc := newOpenAIRecordUsageServiceWithBillingRepoForTest(
				usageRepo,
				billingRepo,
				&openAIRecordUsageUserRepoStub{},
				&openAIRecordUsageSubRepoStub{},
				nil,
			)

			err := svc.RecordWebSearchUsage(context.Background(), &OpenAIRecordWebSearchUsageInput{
				Result: &OpenAIAlphaSearchForwardResult{
					RequestID:  "search_price_" + tt.name,
					StatusCode: http.StatusOK,
					Duration:   time.Second,
				},
				APIKey: &APIKey{
					ID:      int64(8200 + i),
					GroupID: &groupID,
					Group: &Group{
						ID:                    groupID,
						Platform:              PlatformOpenAI,
						RateMultiplier:        groupRate,
						WebSearchPricePerCall: tt.price,
					},
				},
				User:    &User{ID: int64(9200 + i)},
				Account: &Account{ID: int64(10200 + i), Platform: PlatformOpenAI},
			})

			require.NoError(t, err)
			require.Equal(t, 1, billingRepo.calls)
			require.Equal(t, 1, usageRepo.calls)
			require.NotNil(t, usageRepo.lastLog)
			require.InDelta(t, tt.wantPrice, usageRepo.lastLog.TotalCost, 1e-12)
			require.InDelta(t, tt.wantPrice*groupRate, usageRepo.lastLog.ActualCost, 1e-12)
			require.Equal(t, groupRate, usageRepo.lastLog.RateMultiplier)
			require.NotNil(t, billingRepo.lastCmd)
			require.InDelta(t, tt.wantPrice*groupRate, billingRepo.lastCmd.BalanceCost, 1e-12)
			require.InDelta(t, tt.wantPrice*groupRate, billingRepo.lastCmd.UserPlatformCost, 1e-12)
		})
	}
}

func TestOpenAIGatewayServiceRecordWebSearchUsage_Non2xxDoesNotBill(t *testing.T) {
	statuses := []int{
		http.StatusMultipleChoices,
		http.StatusFound,
		http.StatusBadRequest,
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
	}

	for _, status := range statuses {
		t.Run(http.StatusText(status), func(t *testing.T) {
			groupID := int64(7301)
			userRate := 4.0
			price := 0.03
			usageRepo := &openAIRecordUsageLogRepoStub{}
			billingRepo := &openAIRecordUsageBillingRepoStub{result: &UsageBillingApplyResult{Applied: true}}
			rateRepo := &openAIUserGroupRateRepoStub{rate: &userRate}
			svc := newOpenAIRecordUsageServiceWithBillingRepoForTest(
				usageRepo,
				billingRepo,
				&openAIRecordUsageUserRepoStub{},
				&openAIRecordUsageSubRepoStub{},
				rateRepo,
			)

			err := svc.RecordWebSearchUsage(context.Background(), &OpenAIRecordWebSearchUsageInput{
				Result: &OpenAIAlphaSearchForwardResult{
					RequestID:  "search_non_2xx",
					StatusCode: status,
					Duration:   time.Second,
				},
				APIKey: &APIKey{
					ID:      8301,
					GroupID: &groupID,
					Group: &Group{
						ID:                    groupID,
						Platform:              PlatformOpenAI,
						RateMultiplier:        1.5,
						WebSearchPricePerCall: &price,
					},
				},
				User:    &User{ID: 9301},
				Account: &Account{ID: 10301, Platform: PlatformOpenAI},
			})

			require.NoError(t, err)
			require.Equal(t, 0, rateRepo.calls)
			require.Equal(t, 0, billingRepo.calls)
			require.Equal(t, 0, usageRepo.calls)
		})
	}
}

func TestOpenAIGatewayServiceRecordWebSearchUsage_MissingDependenciesNoop(t *testing.T) {
	usageRepo := &openAIRecordUsageLogRepoStub{}
	billingRepo := &openAIRecordUsageBillingRepoStub{result: &UsageBillingApplyResult{Applied: true}}
	svc := newOpenAIRecordUsageServiceWithBillingRepoForTest(
		usageRepo,
		billingRepo,
		&openAIRecordUsageUserRepoStub{},
		&openAIRecordUsageSubRepoStub{},
		nil,
	)

	input := &OpenAIRecordWebSearchUsageInput{
		Result:  &OpenAIAlphaSearchForwardResult{StatusCode: http.StatusOK},
		APIKey:  &APIKey{ID: 1},
		User:    &User{ID: 2},
		Account: &Account{ID: 3},
	}
	require.NoError(t, svc.RecordWebSearchUsage(context.Background(), nil))
	require.NoError(t, svc.RecordWebSearchUsage(context.Background(), &OpenAIRecordWebSearchUsageInput{}))
	withoutResult := *input
	withoutResult.Result = nil
	require.NoError(t, svc.RecordWebSearchUsage(context.Background(), &withoutResult))
	withoutAPIKey := *input
	withoutAPIKey.APIKey = nil
	require.NoError(t, svc.RecordWebSearchUsage(context.Background(), &withoutAPIKey))
	withoutUser := *input
	withoutUser.User = nil
	require.NoError(t, svc.RecordWebSearchUsage(context.Background(), &withoutUser))
	withoutAccount := *input
	withoutAccount.Account = nil
	require.NoError(t, svc.RecordWebSearchUsage(context.Background(), &withoutAccount))

	require.Equal(t, 0, billingRepo.calls)
	require.Equal(t, 0, usageRepo.calls)
}
