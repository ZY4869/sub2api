package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCompareOpenAIAccountUsagePressure_UsesRequestedModelScope(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	proSparkHeavy := &Account{
		ID:       501,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"plan_type": "pro",
		},
		Extra: map[string]any{
			"codex_5h_used_percent":    10.0,
			"codex_5h_reset_at":        now.Add(2 * time.Hour).Format(time.RFC3339),
			codexSpark5hUsedPercentKey: 90.0,
			codexSpark5hResetAtKey:     now.Add(2 * time.Hour).Format(time.RFC3339),
		},
	}
	proNormalHeavy := &Account{
		ID:       502,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"plan_type": "pro",
		},
		Extra: map[string]any{
			"codex_5h_used_percent":    95.0,
			"codex_5h_reset_at":        now.Add(2 * time.Hour).Format(time.RFC3339),
			codexSpark5hUsedPercentKey: 20.0,
			codexSpark5hResetAtKey:     now.Add(2 * time.Hour).Format(time.RFC3339),
		},
	}

	require.Less(t, compareOpenAIAccountUsagePressure(proNormalHeavy, proSparkHeavy, "gpt-5.4", now), 0)
	require.Less(t, compareOpenAIAccountUsagePressure(proSparkHeavy, proNormalHeavy, "gpt-5.3-codex-spark", now), 0)
	require.Equal(t, openAICodexScopeNormal, resolveOpenAIAccountUsagePressureScope(proSparkHeavy, "gpt-5.4"))
	require.Equal(t, openAICodexScopeSpark, resolveOpenAIAccountUsagePressureScope(proSparkHeavy, "gpt-5.3-codex-spark"))
}

func TestCompareOpenAIAccountPlanRank_PrefersPlusThenTeamThenPro(t *testing.T) {
	plus := &Account{
		ID:       600,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"plan_type": "plus",
		},
	}
	team := &Account{
		ID:       601,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"plan_type": "team",
		},
	}
	pro := &Account{
		ID:       602,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"plan_type": "pro",
		},
	}
	free := &Account{
		ID:       603,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"plan_type": "free",
		},
	}
	unknown := &Account{
		ID:       604,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"plan_type": "mystery",
		},
	}

	cmp, ok := compareOpenAIAccountPlanRank(plus, team)
	require.True(t, ok)
	require.Equal(t, -1, cmp)

	cmp, ok = compareOpenAIAccountPlanRank(team, pro)
	require.True(t, ok)
	require.Equal(t, -1, cmp)

	cmp, ok = compareOpenAIAccountPlanRank(pro, free)
	require.True(t, ok)
	require.Equal(t, -1, cmp)

	cmp, ok = compareOpenAIAccountPlanRank(plus, unknown)
	require.False(t, ok)
	require.Equal(t, 0, cmp)
}

func TestCompareOpenAIAccountsForSelection_PrefersPlanRankBeforeConcurrency(t *testing.T) {
	teamHighConcurrency := &Account{
		ID:          701,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Concurrency: 10,
		Priority:    1,
		Credentials: map[string]any{
			"plan_type": "team",
		},
	}
	freeLowConcurrency := &Account{
		ID:          702,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Concurrency: 1,
		Priority:    1,
		Credentials: map[string]any{
			"plan_type": "free",
		},
	}

	require.Less(t, compareOpenAIAccountsForSelection(teamHighConcurrency, freeLowConcurrency, "gpt-5.4", time.Now()), 0)
	require.Greater(t, compareOpenAIAccountsForSelection(freeLowConcurrency, teamHighConcurrency, "gpt-5.4", time.Now()), 0)
}
