export default {
    oauthType: {
        builtInTitle: "Built-in OAuth (Gemini CLI / Code Assist)",
        builtInDesc: "Uses Google built-in client ID. No admin configuration required.",
        builtInRequirement: "Requires a GCP project and Project ID.",
        googleOneDesc: "Authorize with a personal Google account, ideal for Google One users.",
        gcpProjectLink: "Create project",
        advancedToggleShow: "Show AI Studio OAuth advanced options (custom OAuth client)",
        advancedToggleHide: "Hide AI Studio OAuth advanced options (custom OAuth client)",
        customTitle: "Custom OAuth (AI Studio OAuth)",
        customDesc: "Uses admin-configured OAuth client for org management.",
        customRequirement: "Admin must configure Client ID and add you as a test user.",
        vertexTitle: "Vertex AI",
        vertexDesc: "Manually configure GCP project, location, and Bearer token to call Vertex AI Publisher Models directly.",
        vertexRequirement: "Tokens are not refreshed automatically. Update the Access Token manually after expiry.",
        badges: {
            recommended: "Recommended",
            highConcurrency: "High concurrency",
            noAdmin: "No admin setup",
            orgManaged: "Org managed",
            adminRequired: "Admin required",
            manualToken: "Manual token",
        },
    }
}
