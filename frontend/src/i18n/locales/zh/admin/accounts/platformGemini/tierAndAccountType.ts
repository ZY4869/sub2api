export default {
    tier: {
        label: "账号等级",
        hint: "提示：系统会优先尝试自动识别账号等级；若自动识别不可用或失败，则使用你选择的等级作为回退（本地模拟配额）。",
        aiStudioHint: "AI Studio 账号按官方 Free / Tier 1 / Tier 2 / Tier 3 管理。旧的按量付费账号会自动兼容映射到 Tier 1。",
        googleOne: {
            free: "Google One Free",
            pro: "Google One Pro",
            ultra: "Google One Ultra",
        },
        gcp: {
            standard: "GCP Standard",
            enterprise: "GCP Enterprise",
        },
        aiStudio: {
            free: "Google AI Free",
            paid: "Google AI Pay-as-you-go",
            tier1: "Google AI Tier 1",
            tier2: "Google AI Tier 2",
            tier3: "Google AI Tier 3",
        },
    },
    accountType: {
        oauthTitle: "OAuth 授权（Google）",
        oauthDesc: "使用 Google 账号授权，并选择 OAuth 登录方式。",
        apiKeyTitle: "AI Studio API Key",
        apiKeyDesc: "最快接入方式，使用 Google AI Studio 的 AIza API Key。",
        apiKeyNote: "适合轻量测试。免费层限流严格，数据可能用于训练。",
        vertexHint: "如果你要接入 Vertex AI，请直接切换到「Vertex AI」。",
        apiKeyLink: "获取 API Key",
        quotaLink: "配额说明",
    },
    batchCapability: {
        title: "Batch 能力说明",
        aiStudio: "AI Studio API Key：默认支持 Gemini 原生 Files API + Batch API。",
        vertex: "Vertex AI：默认支持标准 Vertex batchPredictionJobs。",
        vertexExpress: "Vertex Express API Key：仅保留在线推理，不伪装成真 Batch。",
    }
}
