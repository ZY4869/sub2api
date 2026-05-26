//go:build unit

package service

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestValidateUpstreamBaseURLWhenAllowlistDisabledStillRejectsPrivateLiteral(t *testing.T) {
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.Enabled = false
	cfg.Security.URLAllowlist.AllowInsecureHTTP = true

	_, err := validateUpstreamBaseURLWithConfig(cfg, "http://127.0.0.1:8080/v1")
	require.Error(t, err)

	_, err = validateUpstreamBaseURLWithConfig(cfg, "http://172.17.0.1:8080/v1")
	require.Error(t, err)
}

func TestValidateUpstreamBaseURLWhenAllowlistDisabledAllowsUnresolvedPublicName(t *testing.T) {
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.Enabled = false
	cfg.Security.URLAllowlist.AllowInsecureHTTP = true

	normalized, err := validateUpstreamBaseURLWithConfig(cfg, "http://gateway.local.test/v1")
	require.NoError(t, err)
	require.Equal(t, "http://gateway.local.test/v1", normalized)
}

func TestValidateUpstreamBaseURLWhenAllowlistDisabledAllowsPrivateByExplicitConfig(t *testing.T) {
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.Enabled = false
	cfg.Security.URLAllowlist.AllowInsecureHTTP = true
	cfg.Security.URLAllowlist.AllowPrivateHosts = true

	normalized, err := validateUpstreamBaseURLWithConfig(cfg, "http://127.0.0.1:8080/v1")
	require.NoError(t, err)
	require.Equal(t, "http://127.0.0.1:8080/v1", normalized)
}

func TestValidateUpstreamBaseURLAllowsScopedPrivateException(t *testing.T) {
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.Enabled = false
	cfg.Security.URLAllowlist.AllowInsecureHTTP = true
	cfg.Security.URLAllowlist.PrivateHostExceptions = []config.PrivateHostExceptionConfig{
		{
			Scope:   PrivateHostExceptionScopeUpstreamBaseURL,
			Hosts:   []string{"127.0.0.1"},
			Ports:   []int{9000},
			Schemes: []string{"http"},
		},
	}

	normalized, err := validateUpstreamBaseURLWithConfig(cfg, "http://127.0.0.1:9000/v1")
	require.NoError(t, err)
	require.Equal(t, "http://127.0.0.1:9000/v1", normalized)
}

func TestValidateUpstreamBaseURLAllowsScopedCIDRPrivateException(t *testing.T) {
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.Enabled = false
	cfg.Security.URLAllowlist.AllowInsecureHTTP = true
	cfg.Security.URLAllowlist.PrivateHostExceptions = []config.PrivateHostExceptionConfig{
		{
			Scope:   PrivateHostExceptionScopeUpstreamBaseURL,
			CIDRs:   []string{"172.17.0.0/16"},
			Ports:   []int{9000},
			Schemes: []string{"http"},
		},
	}

	normalized, err := validateUpstreamBaseURLWithConfig(cfg, "http://172.17.0.1:9000/v1")
	require.NoError(t, err)
	require.Equal(t, "http://172.17.0.1:9000/v1", normalized)
}

func TestValidateUpstreamBaseURLAllowlistEnabledAllowsScopedPrivateException(t *testing.T) {
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.Enabled = true
	cfg.Security.URLAllowlist.UpstreamHosts = []string{"api.openai.com"}
	cfg.Security.URLAllowlist.AllowInsecureHTTP = true
	cfg.Security.URLAllowlist.PrivateHostExceptions = []config.PrivateHostExceptionConfig{
		{
			Scope:   PrivateHostExceptionScopeUpstreamBaseURL,
			Hosts:   []string{"127.0.0.1"},
			Ports:   []int{9000},
			Schemes: []string{"http"},
		},
	}

	normalized, err := validateUpstreamBaseURLWithConfig(cfg, "http://127.0.0.1:9000/v1")
	require.NoError(t, err)
	require.Equal(t, "http://127.0.0.1:9000/v1", normalized)
}

func TestValidateUpstreamBaseURLRejectsWrongPrivateExceptionScope(t *testing.T) {
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.Enabled = false
	cfg.Security.URLAllowlist.AllowInsecureHTTP = true
	cfg.Security.URLAllowlist.PrivateHostExceptions = []config.PrivateHostExceptionConfig{
		{
			Scope:   PrivateHostExceptionScopeProxyHost,
			Hosts:   []string{"127.0.0.1"},
			Ports:   []int{9000},
			Schemes: []string{"http"},
		},
	}

	_, err := validateUpstreamBaseURLWithConfig(cfg, "http://127.0.0.1:9000/v1")
	require.Error(t, err)
}

func TestValidateUpstreamBaseURLRejectsWrongPrivateExceptionPortOrScheme(t *testing.T) {
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.Enabled = false
	cfg.Security.URLAllowlist.AllowInsecureHTTP = true
	cfg.Security.URLAllowlist.PrivateHostExceptions = []config.PrivateHostExceptionConfig{
		{
			Scope:   PrivateHostExceptionScopeUpstreamBaseURL,
			Hosts:   []string{"127.0.0.1"},
			Ports:   []int{9000},
			Schemes: []string{"https"},
		},
	}

	_, err := validateUpstreamBaseURLWithConfig(cfg, "http://127.0.0.1:9000/v1")
	require.Error(t, err)

	cfg.Security.URLAllowlist.PrivateHostExceptions[0].Schemes = []string{"http"}
	_, err = validateUpstreamBaseURLWithConfig(cfg, "http://127.0.0.1:9001/v1")
	require.Error(t, err)
}

func TestValidateUpstreamBaseURLPrivateExceptionDoesNotBypassPublicAllowlist(t *testing.T) {
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.Enabled = true
	cfg.Security.URLAllowlist.UpstreamHosts = []string{"api.openai.com"}
	cfg.Security.URLAllowlist.PrivateHostExceptions = []config.PrivateHostExceptionConfig{
		{
			Scope:   PrivateHostExceptionScopeUpstreamBaseURL,
			Hosts:   []string{"93.184.216.34"},
			Ports:   []int{443},
			Schemes: []string{"https"},
		},
	}

	_, err := validateUpstreamBaseURLWithConfig(cfg, "https://93.184.216.34/v1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "host is not allowed")
}

func TestNormalizeBaseURLRejectsMulticastLiteral(t *testing.T) {
	_, err := normalizeBaseURL("https://224.0.0.1/api", nil, false)
	require.Error(t, err)
}
