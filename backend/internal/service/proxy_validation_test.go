//go:build unit

package service

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestValidateProxyEndpointRejectsPrivateHosts(t *testing.T) {
	cfg := &config.Config{}
	for _, host := range []string{
		"127.0.0.1",
		"localhost",
		"10.0.0.1",
		"172.16.0.1",
		"192.168.1.1",
		"169.254.169.254",
		"0.0.0.0",
		"224.0.0.1",
		"::1",
		"fc00::1",
		"ff02::1",
	} {
		t.Run(host, func(t *testing.T) {
			err := ValidateProxyEndpointWithConfig(cfg, "socks5h", host, 8080)
			require.Error(t, err)
			require.Contains(t, err.Error(), proxyInvalidHostCode)
		})
	}
}

func TestValidateProxyEndpointAllowsPublicHost(t *testing.T) {
	cfg := &config.Config{}
	err := ValidateProxyEndpointWithConfig(cfg, "http", "example.com", 8080)
	require.NoError(t, err)
}

func TestValidateProxyEndpointAllowsPrivateWhenConfigured(t *testing.T) {
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.AllowPrivateHosts = true
	err := ValidateProxyEndpointWithConfig(cfg, "socks5h", "127.0.0.1", 8080)
	require.NoError(t, err)
}

func TestValidateProxyEndpointAllowsScopedPrivateException(t *testing.T) {
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.PrivateHostExceptions = []config.PrivateHostExceptionConfig{
		{
			Scope:   PrivateHostExceptionScopeProxyHost,
			Hosts:   []string{"127.0.0.1"},
			Ports:   []int{8080},
			Schemes: []string{"socks5h"},
		},
	}

	err := ValidateProxyEndpointWithConfig(cfg, "socks5h", "127.0.0.1", 8080)
	require.NoError(t, err)
}

func TestValidateProxyEndpointAllowsScopedCIDRPrivateException(t *testing.T) {
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.PrivateHostExceptions = []config.PrivateHostExceptionConfig{
		{
			Scope:   PrivateHostExceptionScopeProxyHost,
			CIDRs:   []string{"172.17.0.0/16"},
			Ports:   []int{8080},
			Schemes: []string{"http"},
		},
	}

	err := ValidateProxyEndpointWithConfig(cfg, "http", "172.17.0.1", 8080)
	require.NoError(t, err)
}

func TestValidateProxyEndpointRejectsUpstreamOnlyPrivateException(t *testing.T) {
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.PrivateHostExceptions = []config.PrivateHostExceptionConfig{
		{
			Scope:   PrivateHostExceptionScopeUpstreamBaseURL,
			Hosts:   []string{"127.0.0.1"},
			Ports:   []int{8080},
			Schemes: []string{"socks5h"},
		},
	}

	err := ValidateProxyEndpointWithConfig(cfg, "socks5h", "127.0.0.1", 8080)
	require.Error(t, err)
	require.Contains(t, err.Error(), proxyInvalidHostCode)
}

func TestValidateProxyEndpointRejectsURLHost(t *testing.T) {
	cfg := &config.Config{}
	err := ValidateProxyEndpointWithConfig(cfg, "http", "http://example.com", 8080)
	require.Error(t, err)
	require.Contains(t, err.Error(), proxyInvalidHostCode)
}
