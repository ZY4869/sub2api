export default {
    // OpenAI specific hints
    openai: {
        baseUrlHint: "留空使用官方 OpenAI API",
        apiKeyHint: "您的 OpenAI API Key",
        oauthPassthrough: "自动透传（仅替换认证）",
        oauthPassthroughDesc: "开启后，该 OpenAI 账号将自动透传请求与响应，仅替换认证并保留计费/并发/审计及必要安全过滤；如遇兼容性问题可随时关闭回滚。",
        responsesWebsocketsV2: "Responses WebSocket v2",
        responsesWebsocketsV2Desc: "默认关闭。开启后可启用 responses_websockets_v2 协议能力（受网关全局开关与账号类型开关约束）。",
        wsMode: "WS mode",
        wsModeDesc: "仅对当前 OpenAI 账号类型生效。",
        wsModeOff: "关闭（off）",
        wsModeCtxPool: "上下文池（ctx_pool）",
        wsModePassthrough: "透传（passthrough）",
        wsModeShared: "共享（shared）",
        wsModeDedicated: "独享（dedicated）",
        wsModeConcurrencyHint: "启用 WS mode 后，该账号并发数将作为该账号 WS 连接池上限。",
        wsModePassthroughHint: "passthrough 模式不使用 WS 连接池。",
        oauthResponsesWebsocketsV2: "OAuth WebSocket Mode",
        oauthResponsesWebsocketsV2Desc: "仅对 OpenAI OAuth 生效。开启后该账号才允许使用 OpenAI WebSocket Mode 协议。",
        apiKeyResponsesWebsocketsV2: "API Key WebSocket Mode",
        apiKeyResponsesWebsocketsV2Desc: "仅对 OpenAI API Key 生效。开启后该账号才允许使用 OpenAI WebSocket Mode 协议。",
        responsesWebsocketsV2PassthroughHint: "当前已开启自动透传：仅影响 HTTP 透传链路，不影响 WS mode。",
        codexCLIOnly: "仅允许 Codex 官方客户端",
        codexCLIOnlyDesc: "仅对 OpenAI OAuth 生效。开启后仅允许 Codex 官方客户端家族访问；关闭后完全绕过并保持原逻辑。",
        modelRestrictionDisabledByPassthrough: "已开启自动透传：模型白名单/映射不会生效。",
        imageProtocol: {
            label: "图片协议模式",
            description: "控制当前 OpenAI 账号默认走原生图片链路还是兼容图片链路；分组若强制指定，将优先覆盖这里。",
            compatUnavailableHint: "当前账号计划默认不开放兼容生图，请先升级计划或改用原生生图。",
            options: {
                native: "原生生图",
                compat: "兼容生图",
            },
        },
    },
    deepseek: {
        baseUrlHint: "留空使用官方 DeepSeek API（https://api.deepseek.com）",
        apiKeyHint: "您的 DeepSeek API Key",
    },
    anthropic: {
        apiKeyPassthrough: "自动透传（仅替换认证）",
        apiKeyPassthroughDesc: "仅对 Anthropic API Key 生效。开启后，messages/count_tokens 请求将透传上游并仅替换认证，保留计费/并发/审计及必要安全过滤；关闭即可回滚到现有兼容链路。",
    }
}
