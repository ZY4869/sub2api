export default {
    tier: {
        label: "Account Tier",
        hint: "Tip: The system will try to auto-detect the tier first; if auto-detection is unavailable or fails, your selected tier is used as a fallback (simulated quota).",
        aiStudioHint: "AI Studio now follows the official Free / Tier 1 / Tier 2 / Tier 3 model. Legacy pay-as-you-go accounts are normalized to Tier 1.",
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
        oauthTitle: "OAuth (Google)",
        oauthDesc: "Authorize with your Google account and choose an OAuth sign-in type.",
        apiKeyTitle: "AI Studio API Key",
        apiKeyDesc: "Fastest setup. Use an AIza API key from Google AI Studio.",
        apiKeyNote: "Best for light testing. Free tier has strict rate limits and data may be used for training.",
        vertexHint: 'To use Vertex AI, switch directly to "Vertex AI".',
        apiKeyLink: "Get API Key",
        quotaLink: "Quota guide",
    },
    batchCapability: {
        title: "Batch capability",
        aiStudio: "AI Studio API key: native Gemini Files API + Batch API enabled by default.",
        vertex: "Vertex AI: standard Vertex batchPredictionJobs enabled by default.",
        vertexExpress: "Vertex Express API key: online inference only, not treated as true Batch.",
    }
}
