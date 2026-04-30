export default {
    oauthType: {
        builtInTitle: "内置授权（Gemini CLI / Code Assist）",
        builtInDesc: "使用 Google 内置客户端 ID，无需管理员配置。",
        builtInRequirement: "需要 GCP 项目并填写 Project ID。",
        googleOneDesc: "使用个人 Google 账号授权，适合 Google One 用户。",
        gcpProjectLink: "创建项目",
        advancedToggleShow: "显示 AI Studio OAuth 高级选项（自建 OAuth Client）",
        advancedToggleHide: "隐藏 AI Studio OAuth 高级选项（自建 OAuth Client）",
        customTitle: "自定义授权（AI Studio OAuth）",
        customDesc: "使用管理员预设的 OAuth 客户端，适合组织管理。",
        customRequirement: "需管理员配置 Client ID 并加入测试用户白名单。",
        vertexTitle: "Vertex AI",
        vertexDesc: "手动填写 GCP 项目、区域和 Bearer Token，直接走 Vertex AI Publisher Models。",
        vertexRequirement: "不会自动刷新 token，过期后请手动更新 Access Token。",
        badges: {
            recommended: "推荐",
            highConcurrency: "高并发",
            noAdmin: "无需管理员配置",
            orgManaged: "组织管理",
            adminRequired: "需要管理员",
            manualToken: "手动 Token",
        },
    }
}
