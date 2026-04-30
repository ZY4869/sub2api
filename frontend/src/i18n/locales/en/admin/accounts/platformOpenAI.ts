export default {
    // OpenAI specific hints
    openai: {
        baseUrlHint: "Leave default for official OpenAI API",
        apiKeyHint: "Your OpenAI API Key",
        oauthPassthrough: "Auto passthrough (auth only)",
        oauthPassthroughDesc: "When enabled, this OpenAI account uses automatic passthrough: the gateway forwards request/response as-is and only swaps auth, while keeping billing/concurrency/audit and necessary safety filtering.",
        responsesWebsocketsV2: "Responses WebSocket v2",
        responsesWebsocketsV2Desc: "Disabled by default. Enable to allow responses_websockets_v2 capability (still gated by global and account-type switches).",
        wsMode: "WS mode",
        wsModeDesc: "Only applies to the current OpenAI account type.",
        wsModeOff: "Off (off)",
        wsModeCtxPool: "Context Pool (ctx_pool)",
        wsModePassthrough: "Passthrough (passthrough)",
        wsModeShared: "Shared (shared)",
        wsModeDedicated: "Dedicated (dedicated)",
        wsModeConcurrencyHint: "When WS mode is enabled, account concurrency becomes the WS connection pool limit for this account.",
        wsModePassthroughHint: "Passthrough mode does not use the WS connection pool.",
        oauthResponsesWebsocketsV2: "OAuth WebSocket Mode",
        oauthResponsesWebsocketsV2Desc: "Only applies to OpenAI OAuth. This account can use OpenAI WebSocket Mode only when enabled.",
        apiKeyResponsesWebsocketsV2: "API Key WebSocket Mode",
        apiKeyResponsesWebsocketsV2Desc: "Only applies to OpenAI API Key. This account can use OpenAI WebSocket Mode only when enabled.",
        responsesWebsocketsV2PassthroughHint: "Automatic passthrough is currently enabled: it only affects HTTP passthrough and does not disable WS mode.",
        codexCLIOnly: "Codex official clients only",
        codexCLIOnlyDesc: "Only applies to OpenAI OAuth. When enabled, only Codex official client families are allowed; when disabled, the gateway bypasses this restriction and keeps existing behavior.",
        modelRestrictionDisabledByPassthrough: "Automatic passthrough is enabled: model whitelist/mapping will not take effect.",
        imageProtocol: {
            label: "Image Protocol Mode",
            description: "Controls whether this OpenAI account defaults to the native image chain or the compat image chain. Group-level overrides take precedence.",
            compatUnavailableHint: "Compat image generation is not enabled for the current plan. Upgrade the plan or switch back to native image generation.",
            options: {
                native: "Native Images",
                compat: "Compat Images",
            },
        },
    },
    deepseek: {
        baseUrlHint: "Leave default for the official DeepSeek API (https://api.deepseek.com)",
        apiKeyHint: "Your DeepSeek API Key",
    },
    anthropic: {
        apiKeyPassthrough: "Auto passthrough (auth only)",
        apiKeyPassthroughDesc: "Only applies to Anthropic API Key accounts. When enabled, messages/count_tokens are forwarded in passthrough mode with auth replacement only, while billing/concurrency/audit and safety filtering are preserved. Disable to roll back immediately.",
    }
}
