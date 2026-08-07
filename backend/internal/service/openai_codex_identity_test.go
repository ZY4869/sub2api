package service

import (
	"context"
	"net/http"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestEnforceCodexIdentityHeaders_PairsUserAgentOriginatorAndVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		userAgent      string
		originator     string
		version        string
		wantUserAgent  string
		wantOriginator string
		wantVersion    string
	}{
		{name: "default cli identity", wantUserAgent: codexCLIUserAgent, wantOriginator: "codex_cli_rs", wantVersion: codexCLIVersion},
		{name: "codex cli", userAgent: "codex_cli_rs/0.1.0", originator: "codex_vscode", version: "0.1.0", wantUserAgent: "codex_cli_rs/0.1.0", wantOriginator: "codex_cli_rs", wantVersion: codexCLIVersion},
		{name: "vscode", userAgent: "codex_vscode/1.2.3", originator: "codex_cli_rs", version: "0.1.0", wantUserAgent: "codex_vscode/1.2.3", wantOriginator: "codex_vscode", wantVersion: codexCLIVersion},
		{name: "app", userAgent: "codex_app/2.1.0", originator: "codex_cli_rs", version: codexCLIVersion, wantUserAgent: "codex_app/2.1.0", wantOriginator: "codex_app", wantVersion: codexCLIVersion},
		{name: "chatgpt desktop", userAgent: "Codex Desktop/1.2.3", originator: "codex_cli_rs", version: "9.9.9", wantUserAgent: "Codex Desktop/1.2.3", wantOriginator: "codex_chatgpt_desktop", wantVersion: "9.9.9"},
		{name: "atlas", userAgent: "codex_atlas/3.0.0", wantUserAgent: "codex_atlas/3.0.0", wantOriginator: "codex_atlas", wantVersion: codexCLIVersion},
		{name: "exec", userAgent: "codex_exec/0.8.0", wantUserAgent: "codex_exec/0.8.0", wantOriginator: "codex_exec", wantVersion: codexCLIVersion},
		{name: "sdk ts", userAgent: "codex_sdk_ts/0.7.0", wantUserAgent: "codex_sdk_ts/0.7.0", wantOriginator: "codex_sdk_ts", wantVersion: codexCLIVersion},
		{name: "unknown ua preserves sanitized originator", userAgent: "curl/8.0", originator: "codex_atlas", wantUserAgent: "curl/8.0", wantOriginator: "codex_atlas", wantVersion: codexCLIVersion},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			headers := http.Header{}
			if tt.userAgent != "" {
				headers.Set("user-agent", tt.userAgent)
			}
			if tt.originator != "" {
				headers.Set("originator", tt.originator)
			}
			if tt.version != "" {
				headers.Set("version", tt.version)
			}
			account := &Account{Platform: PlatformOpenAI, Type: AccountTypeOAuth}

			enforceCodexIdentityHeaders(context.Background(), headers, account)

			require.Equal(t, tt.wantUserAgent, headers.Get("user-agent"))
			require.Equal(t, tt.wantOriginator, headers.Get("originator"))
			require.Equal(t, tt.wantVersion, headers.Get("version"))
		})
	}
}

func TestEnforceCodexIdentityHeaders_IgnoresNonChatGPTOAuth(t *testing.T) {
	t.Parallel()

	headers := http.Header{}
	headers.Set("user-agent", "codex_vscode/1.2.3")
	account := &Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey}

	enforceCodexIdentityHeaders(context.Background(), headers, account)

	require.Empty(t, headers.Get("originator"))
	require.Empty(t, headers.Get("version"))
}

func TestEnforceCodexIdentityHeadersWithConfig_CanDisableOriginatorNormalization(t *testing.T) {
	t.Parallel()

	headers := http.Header{}
	headers.Set("user-agent", "codex_vscode/1.2.3")
	headers.Set("originator", "codex_cli_rs")
	headers.Set("version", "0.1.0")
	account := &Account{Platform: PlatformOpenAI, Type: AccountTypeOAuth}
	cfg := &config.Config{Gateway: config.GatewayConfig{DisableCodexOriginatorNormalization: true}}

	enforceCodexIdentityHeadersWithConfig(context.Background(), headers, account, cfg)

	require.Equal(t, "codex_vscode/1.2.3", headers.Get("user-agent"))
	require.Equal(t, "codex_cli_rs", headers.Get("originator"))
	require.Equal(t, codexCLIVersion, headers.Get("version"))
}
