package openai

import (
	"strings"
	"testing"
)

func TestIsCodexCLIRequest(t *testing.T) {
	tests := []struct {
		name string
		ua   string
		want bool
	}{
		{name: "codex_cli_rs 前缀", ua: "codex_cli_rs/0.1.0", want: true},
		{name: "codex_vscode 前缀", ua: "codex_vscode/1.2.3", want: true},
		{name: "大小写混合", ua: "Codex_CLI_Rs/0.1.0", want: true},
		{name: "复合 UA 包含 codex", ua: "Mozilla/5.0 codex_cli_rs/0.1.0", want: true},
		{name: "空白包裹", ua: "  codex_vscode/1.2.3  ", want: true},
		{name: "非 codex", ua: "curl/8.0.1", want: false},
		{name: "空字符串", ua: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsCodexCLIRequest(tt.ua)
			if got != tt.want {
				t.Fatalf("IsCodexCLIRequest(%q) = %v, want %v", tt.ua, got, tt.want)
			}
		})
	}
}

func TestIsCodexOfficialClientRequest(t *testing.T) {
	tests := []struct {
		name string
		ua   string
		want bool
	}{
		{name: "codex_cli_rs 前缀", ua: "codex_cli_rs/0.98.0", want: true},
		{name: "codex_vscode 前缀", ua: "codex_vscode/1.0.0", want: true},
		{name: "codex_app 前缀", ua: "codex_app/0.1.0", want: true},
		{name: "codex_chatgpt_desktop 前缀", ua: "codex_chatgpt_desktop/1.0.0", want: true},
		{name: "codex_atlas 前缀", ua: "codex_atlas/1.0.0", want: true},
		{name: "codex_exec 前缀", ua: "codex_exec/0.1.0", want: true},
		{name: "codex_sdk_ts 前缀", ua: "codex_sdk_ts/0.1.0", want: true},
		{name: "Codex 桌面 UA", ua: "Codex Desktop/1.2.3", want: true},
		{name: "复合 UA 包含 codex_app", ua: "Mozilla/5.0 codex_app/0.1.0", want: true},
		{name: "大小写混合", ua: "Codex_VSCode/1.2.3", want: true},
		{name: "非 codex", ua: "curl/8.0.1", want: false},
		{name: "空字符串", ua: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsCodexOfficialClientRequest(tt.ua)
			if got != tt.want {
				t.Fatalf("IsCodexOfficialClientRequest(%q) = %v, want %v", tt.ua, got, tt.want)
			}
		})
	}
}

func TestIsCodexOfficialClientOriginator(t *testing.T) {
	tests := []struct {
		name       string
		originator string
		want       bool
	}{
		{name: "codex_cli_rs", originator: "codex_cli_rs", want: true},
		{name: "codex_vscode", originator: "codex_vscode", want: true},
		{name: "codex_app", originator: "codex_app", want: true},
		{name: "codex_chatgpt_desktop", originator: "codex_chatgpt_desktop", want: true},
		{name: "codex_atlas", originator: "codex_atlas", want: true},
		{name: "codex_exec", originator: "codex_exec", want: true},
		{name: "codex_sdk_ts", originator: "codex_sdk_ts", want: true},
		{name: "Codex 前缀", originator: "Codex Desktop", want: true},
		{name: "空白包裹", originator: "  codex_vscode  ", want: true},
		{name: "非 codex", originator: "my_client", want: false},
		{name: "空字符串", originator: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsCodexOfficialClientOriginator(tt.originator)
			if got != tt.want {
				t.Fatalf("IsCodexOfficialClientOriginator(%q) = %v, want %v", tt.originator, got, tt.want)
			}
		})
	}
}

func TestIsCodexOfficialClientByHeaders(t *testing.T) {
	tests := []struct {
		name       string
		ua         string
		originator string
		want       bool
	}{
		{name: "仅 originator 命中 desktop", originator: "Codex Desktop", want: true},
		{name: "仅 originator 命中 vscode", originator: "codex_vscode", want: true},
		{name: "仅 ua 命中 desktop", ua: "Codex Desktop/1.2.3", want: true},
		{name: "ua 与 originator 都未命中", ua: "curl/8.0.1", originator: "my_client", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsCodexOfficialClientByHeaders(tt.ua, tt.originator)
			if got != tt.want {
				t.Fatalf("IsCodexOfficialClientByHeaders(%q, %q) = %v, want %v", tt.ua, tt.originator, got, tt.want)
			}
		})
	}
}

func TestNormalizeAllowedClientIDs(t *testing.T) {
	got := NormalizeAllowedClientIDs([]string{" CLAUDE_CODE ", "unknown", "claude_code", ""})
	if len(got) != 1 || got[0] != AllowedClientClaudeCode {
		t.Fatalf("NormalizeAllowedClientIDs returned %#v, want [%q]", got, AllowedClientClaudeCode)
	}

	if got := NormalizeAllowedClientIDs([]string{"unknown"}); len(got) != 0 {
		t.Fatalf("NormalizeAllowedClientIDs returned %#v, want empty", got)
	}
}

func TestCodexOriginatorForUserAgent(t *testing.T) {
	tests := []struct {
		name string
		ua   string
		want string
	}{
		{name: "cli", ua: "codex_cli_rs/0.104.0", want: "codex_cli_rs"},
		{name: "vscode", ua: "Mozilla/5.0 codex_vscode/1.2.3", want: "codex_vscode"},
		{name: "desktop legacy", ua: "Codex Desktop/1.2.3", want: "codex_chatgpt_desktop"},
		{name: "unknown", ua: "curl/8.0.1", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CodexOriginatorForUserAgent(tt.ua); got != tt.want {
				t.Fatalf("CodexOriginatorForUserAgent(%q) = %q, want %q", tt.ua, got, tt.want)
			}
		})
	}
}

func TestSanitizeCodexOriginator(t *testing.T) {
	if got := SanitizeCodexOriginator(" codex_cli_rs "); got != "codex_cli_rs" {
		t.Fatalf("SanitizeCodexOriginator returned %q", got)
	}
	if got := SanitizeCodexOriginator("codex_cli_rs\nbad"); got != "" {
		t.Fatalf("SanitizeCodexOriginator accepted newline: %q", got)
	}
	if got := SanitizeCodexOriginator(strings.Repeat("a", 65)); got != "" {
		t.Fatalf("SanitizeCodexOriginator accepted overlong value: %q", got)
	}
}
