package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestMergeAnthropicBetaDropping_FiltersExplicitRedactThinkingViaPolicy(t *testing.T) {
	got := mergeAnthropicBetaDropping(
		[]string{"oauth-2025-04-20"},
		"redact-thinking-2025-10-01,foo-beta",
		map[string]struct{}{"redact-thinking-2025-10-01": {}},
	)
	require.Equal(t, "oauth-2025-04-20,foo-beta", got)
}

func TestGatewayServiceEvaluateBetaPolicy_FiltersExplicitRedactThinkingWhenConfigured(t *testing.T) {
	settings := &BetaPolicySettings{
		Rules: []BetaPolicyRule{
			{
				BetaToken: "redact-thinking-2025-10-01",
				Action:    BetaPolicyActionFilter,
				Scope:     BetaPolicyScopeAll,
			},
		},
	}
	raw, err := json.Marshal(settings)
	require.NoError(t, err)

	svc := &GatewayService{
		settingService: NewSettingService(
			&betaPolicySettingRepoStub{values: map[string]string{
				SettingKeyBetaPolicySettings: string(raw),
			}},
			&config.Config{},
		),
	}

	result := svc.evaluateBetaPolicy(
		context.Background(),
		"redact-thinking-2025-10-01,oauth-2025-04-20",
		&Account{Type: AccountTypeOAuth},
	)

	require.Nil(t, result.blockErr)
	require.Contains(t, result.filterSet, "redact-thinking-2025-10-01")

	filtered := mergeAnthropicBetaDropping(
		[]string{"oauth-2025-04-20"},
		"redact-thinking-2025-10-01",
		result.filterSet,
	)
	require.Equal(t, "oauth-2025-04-20", filtered)
}
