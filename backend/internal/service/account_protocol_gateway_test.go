package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGatewayClientProfileDescriptorByID(t *testing.T) {
	descriptor, ok := GatewayClientProfileDescriptorByID("codex")
	require.True(t, ok)
	require.Equal(t, GatewayClientProfileCodex, descriptor.ID)
	require.Equal(t, "Codex", descriptor.DisplayName)
	require.Equal(t, "gpt-5.1-codex", descriptor.DefaultTestModel)
	require.Equal(t, []string{PlatformOpenAI}, descriptor.CompatibleProtocols)

	_, ok = GatewayClientProfileDescriptorByID("unknown")
	require.False(t, ok)
}

func TestNormalizeGatewayAcceptedProtocolsMixed(t *testing.T) {
	accepted := NormalizeGatewayAcceptedProtocols(GatewayProtocolMixed, map[string]any{
		"gateway_accepted_protocols": []any{" openai ", "gemini", "gemini", "mixed", "invalid"},
	})
	require.Equal(t, []string{PlatformOpenAI, PlatformGemini}, accepted)

	fallback := NormalizeGatewayAcceptedProtocols(GatewayProtocolMixed, map[string]any{})
	require.Equal(t, []string{PlatformOpenAI, PlatformAnthropic, PlatformGemini}, fallback)
}

func TestNormalizeGatewayClientRoutesFiltersUnsupportedRoutes(t *testing.T) {
	routes := NormalizeGatewayClientRoutes(GatewayProtocolMixed, map[string]any{
		"gateway_accepted_protocols": []any{"openai", "gemini"},
		"gateway_client_routes": []any{
			map[string]any{
				"protocol":       "openai",
				"match_type":     "exact",
				"match_value":    "gpt-5",
				"client_profile": "codex",
			},
			map[string]any{
				"protocol":       "openai",
				"match_type":     "exact",
				"match_value":    "gpt-4.1",
				"client_profile": "gemini_cli",
			},
			map[string]any{
				"protocol":       "anthropic",
				"match_type":     "prefix",
				"match_value":    "claude",
				"client_profile": "codex",
			},
		},
	})

	require.Equal(t, []GatewayClientRoute{
		{
			Protocol:      PlatformOpenAI,
			MatchType:     "exact",
			MatchValue:    "gpt-5",
			ClientProfile: GatewayClientProfileCodex,
		},
	}, routes)
}

func TestMatchGatewayClientRoutePrefersLongestPrefix(t *testing.T) {
	account := &Account{
		Platform: PlatformProtocolGateway,
		Extra: map[string]any{
			"gateway_protocol":           GatewayProtocolMixed,
			"gateway_accepted_protocols": []any{"openai"},
			"gateway_client_routes": []any{
				map[string]any{
					"protocol":       "openai",
					"match_type":     "prefix",
					"match_value":    "gpt-",
					"client_profile": "codex",
				},
				map[string]any{
					"protocol":       "openai",
					"match_type":     "prefix",
					"match_value":    "gpt-5.1-",
					"client_profile": "codex",
				},
			},
		},
	}

	route := MatchGatewayClientRoute(account, PlatformOpenAI, "gpt-5.1-codex")
	require.NotNil(t, route)
	require.Equal(t, "gpt-5.1-", route.MatchValue)
	require.Equal(t, GatewayClientProfileCodex, route.ClientProfile)
}

func TestResolveProtocolGatewayInboundAccountNarrowsMixedAccount(t *testing.T) {
	account := &Account{
		Platform: PlatformProtocolGateway,
		Extra: map[string]any{
			"gateway_protocol":           GatewayProtocolMixed,
			"gateway_accepted_protocols": []any{"openai", "gemini"},
		},
	}

	resolved := ResolveProtocolGatewayInboundAccount(account, PlatformGemini)
	require.NotNil(t, resolved)
	require.NotSame(t, account, resolved)
	require.Equal(t, GatewayProtocolGemini, GetAccountGatewayProtocol(resolved))
	require.Equal(t, []string{PlatformGemini}, GetAccountGatewayAcceptedProtocols(resolved))

	require.Equal(t, GatewayProtocolMixed, GetAccountGatewayProtocol(account))
	require.Equal(t, []string{PlatformOpenAI, PlatformGemini}, GetAccountGatewayAcceptedProtocols(account))
}

func TestRoutingPlatformsFromValuesForMixedProtocolGateway(t *testing.T) {
	platforms := RoutingPlatformsFromValues(PlatformProtocolGateway, map[string]any{
		"gateway_protocol":           GatewayProtocolMixed,
		"gateway_accepted_protocols": []any{"gemini", "openai", "openai"},
	})

	require.Equal(t, []string{PlatformGemini, PlatformOpenAI}, platforms)
}
