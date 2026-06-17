export default {
    claudeCode: {
        title: "Claude Code 设置",
        description: "控制 Claude Code 客户端访问要求",
        minVersion: "最低版本号",
        minVersionPlaceholder: "例如 2.1.63",
        minVersionHint: "拒绝低于此版本的 Claude Code 客户端请求（semver 格式）。留空则不检查版本。",
        maxVersion: "最高版本号",
        maxVersionPlaceholder: "例如 2.3.0",
        maxVersionHint: "拒绝高于此版本的 Claude Code 客户端请求（semver 格式）。留空则不检查版本。",
        allowCodexPlugin: "允许 Claude Code Codex 插件",
        allowCodexPluginHint: "默认关闭；开启后仅放行 originator 为 Claude Code 且 User-Agent 含 Claude Code/ 的 Codex 插件请求。",
        allowedClients: "允许的客户端",
        allowedClientsHint: "只允许已知且受支持的例外客户端；未识别的客户端不会被保存或放行。",
        allowedClientClaudeCodeLabel: "Claude Code",
        allowedClientClaudeCode: "允许 Claude Code Codex 插件在开启 codex_cli_only 的 OpenAI 账号上通过。",
        oauthPromptBlocks: "Claude OAuth System Prompt Blocks",
        oauthPromptBlocksHint: "为 Claude OAuth 转发请求追加站点级 system 文本块，默认关闭。",
        oauthPromptBlocksPlaceholder: "每行一个 system prompt block",
        oauthPromptBlocksPriorityHint: "这些文本块会追加到现有 system 后面，优先级低于本地强制安全提示与 Claude Code 规范化规则。",
    }
}
