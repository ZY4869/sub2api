package hostexceptions

import (
	"net"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestPrivateHostExceptionsRequireExplicitPortAndScheme(t *testing.T) {
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.PrivateHostExceptions = []config.PrivateHostExceptionConfig{
		{
			Scope: ScopeUpstreamBaseURL,
			Hosts: []string{"127.0.0.1"},
		},
	}

	_, ok := IsAllowed(cfg, ScopeUpstreamBaseURL, "http", "127.0.0.1", 9000)
	require.False(t, ok)

	cfg.Security.URLAllowlist.PrivateHostExceptions[0].Ports = []int{9000}
	_, ok = IsAllowed(cfg, ScopeUpstreamBaseURL, "http", "127.0.0.1", 9000)
	require.False(t, ok)

	cfg.Security.URLAllowlist.PrivateHostExceptions[0].Schemes = []string{"http"}
	_, ok = IsAllowed(cfg, ScopeUpstreamBaseURL, "http", "127.0.0.1", 9000)
	require.True(t, ok)
}

func TestPrivateHostExceptionsOnlyApplyToBlockedResolvedIPs(t *testing.T) {
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.PrivateHostExceptions = []config.PrivateHostExceptionConfig{
		{
			Scope:   ScopeUpstreamBaseURL,
			Hosts:   []string{"93.184.216.34"},
			Ports:   []int{443},
			Schemes: []string{"https"},
		},
	}

	_, ok, err := ResolvedMatch(t.Context(), cfg, ScopeUpstreamBaseURL, "https", "93.184.216.34", 443)
	require.NoError(t, err)
	require.False(t, ok)
}

func TestPrivateHostExceptionsRequireEveryBlockedResolvedIPToMatch(t *testing.T) {
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.PrivateHostExceptions = []config.PrivateHostExceptionConfig{
		{
			Scope:   ScopeUpstreamBaseURL,
			CIDRs:   []string{"127.0.0.0/8"},
			Ports:   []int{9000},
			Schemes: []string{"http"},
		},
	}

	_, ok := validateResolvedHostExceptionIPs(
		cfg,
		ScopeUpstreamBaseURL,
		"http",
		"local-upstream.test",
		9000,
		[]net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("172.17.0.1")},
	)
	require.False(t, ok)

	cfg.Security.URLAllowlist.PrivateHostExceptions[0].CIDRs = []string{"127.0.0.0/8", "172.17.0.0/16"}
	match, ok := validateResolvedHostExceptionIPs(
		cfg,
		ScopeUpstreamBaseURL,
		"http",
		"local-upstream.test",
		9000,
		[]net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("172.17.0.1")},
	)
	require.True(t, ok)
	require.Equal(t, ScopeUpstreamBaseURL, match.Scope)
}
