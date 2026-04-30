export default {
    helpButton: "使用帮助",
    helpDialog: {
        title: "Gemini 使用指南",
        apiKeySection: "API Key 相关链接",
    },
    gemini3Guide: {
        title: "Gemini 3 参数对齐",
        items: {
            stableDefault: "默认测试模型仍使用 gemini-2.5-flash；Gemini 3 参数请优先选择 Gemini 3 预览模型。",
            thinkingLevel: "Gemini 3 推荐使用 thinkingLevel；兼容层会把 reasoning_effort 映射为 MINIMAL / LOW / MEDIUM / HIGH。",
            thinkingBudget: "thinkingBudget 在 Gemini 3 中仅作为兼容模式保留；同时传 thinkingLevel 与 thinkingBudget 会直接报错。",
            mediaResolution: "Claude 兼容请求现已支持 media_resolution / mediaResolution，并统一映射到 generationConfig.mediaResolution。",
            urlContext: "URL Context 仅支持 Gemini API 通道，不支持 Vertex Gemini 通道。",
            toolCombination: "启用 includeServerSideToolInvocations 后，内建工具与函数调用会自动切到 VALIDATED 模式，避免 Gemini 3 组合工具 400。",
        },
        links: {
            gemini3: "Gemini 3 文档",
            mediaResolution: "媒体分辨率文档",
            toolCombination: "工具组合文档",
            vertexInference: "Vertex Inference 文档",
        },
    },
    modelPassthrough: "Gemini 直接转发模型",
    modelPassthroughDesc: "所有模型请求将直接转发至 Gemini API，不进行模型限制或映射。",
    baseUrlHint: "留空使用官方 Gemini API",
    apiKeyHint: "您的 Gemini API Key（以 AIza 开头）"
}
