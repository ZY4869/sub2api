export default {
    helpButton: "Help",
    helpDialog: {
        title: "Gemini Usage Guide",
        apiKeySection: "API Key Links",
    },
    gemini3Guide: {
        title: "Gemini 3 Parameter Alignment",
        items: {
            stableDefault: "The default test model remains gemini-2.5-flash. Use Gemini 3 preview models when you need Gemini 3 specific parameters.",
            thinkingLevel: "Gemini 3 prefers thinkingLevel. The compatibility layer now maps reasoning_effort into MINIMAL / LOW / MEDIUM / HIGH.",
            thinkingBudget: "thinkingBudget is kept only as a legacy compatibility mode on Gemini 3. Sending both thinkingLevel and thinkingBudget now returns a validation error.",
            mediaResolution: "Claude-compatible requests now accept media_resolution / mediaResolution and normalize them into generationConfig.mediaResolution.",
            urlContext: "URL Context is available only on Gemini API channels and is rejected on Vertex Gemini channels.",
            toolCombination: "When includeServerSideToolInvocations is enabled, built-in tools plus function calling are normalized to VALIDATED mode to avoid Gemini 3 tool-combination 400s.",
        },
        links: {
            gemini3: "Gemini 3 docs",
            mediaResolution: "Media resolution docs",
            toolCombination: "Tool combination docs",
            vertexInference: "Vertex inference docs",
        },
    },
    modelPassthrough: "Gemini Model Passthrough",
    modelPassthroughDesc: "All model requests are forwarded directly to the Gemini API without model restrictions or mappings.",
    baseUrlHint: "Leave default for official Gemini API",
    apiKeyHint: "Your Gemini API Key (starts with AIza)"
}
