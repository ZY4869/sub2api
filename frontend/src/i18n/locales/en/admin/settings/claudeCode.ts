export default {
    claudeCode: {
        title: "Claude Code Settings",
        description: "Control Claude Code client access requirements",
        minVersion: "Minimum Version",
        minVersionPlaceholder: "e.g. 2.1.63",
        minVersionHint: "Reject Claude Code clients below this version (semver format). Leave empty to disable version check.",
        maxVersion: "Maximum Version",
        maxVersionPlaceholder: "e.g. 2.3.0",
        maxVersionHint: "Reject Claude Code clients above this version (semver format). Leave empty to disable version check.",
        allowCodexPlugin: "Allow Claude Code Codex plugin",
        allowCodexPluginHint: "Off by default. When enabled, only Codex plugin requests with originator Claude Code and a Claude Code/ User-Agent are allowed.",
        allowedClients: "Allowed clients",
        allowedClientsHint: "Only known and supported exception clients are saved or allowed.",
        allowedClientClaudeCodeLabel: "Claude Code",
        allowedClientClaudeCode: "Allow the Claude Code Codex plugin to pass on OpenAI accounts with codex_cli_only enabled.",
        oauthPromptBlocks: "Claude OAuth System Prompt Blocks",
        oauthPromptBlocksHint: "Append site-level system text blocks to Claude OAuth forwarded requests. Disabled by default.",
        oauthPromptBlocksPlaceholder: "One system prompt block per line",
        oauthPromptBlocksPriorityHint: "These blocks are appended after existing system content and remain lower priority than local forced safety prompts and Claude Code normalization.",
    }
}
