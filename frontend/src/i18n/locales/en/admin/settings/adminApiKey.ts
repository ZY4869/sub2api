export default {
    adminApiKey: {
        title: "Admin API Key",
        description: "Global API key for external system integration with full admin access",
        notConfigured: "Admin API key not configured",
        configured: "Admin API key is active",
        currentKey: "Current Key",
        regenerate: "Regenerate",
        regenerating: "Regenerating...",
        delete: "Delete",
        deleting: "Deleting...",
        create: "Create Key",
        creating: "Creating...",
        regenerateConfirm: "Are you sure? The current key will be immediately invalidated.",
        deleteConfirm: "Are you sure you want to delete the admin API key? External integrations will stop working.",
        keyGenerated: "New admin API key generated",
        keyDeleted: "Admin API key deleted",
        copyKey: "Copy Key",
        keyCopied: "Key copied to clipboard",
        keyWarning: "This key will only be shown once. Please copy it now.",
        securityWarning: "Warning: This key provides full admin access. Keep it secure.",
        usage: "Usage: Add to request header - x-api-key: <your-admin-api-key>",
    }
}
