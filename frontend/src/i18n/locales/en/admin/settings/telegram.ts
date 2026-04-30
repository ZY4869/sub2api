export default {
    telegram: {
        title: "Telegram Notifications",
        description: "Configure a Telegram bot for scheduled test alerts.",
        botToken: "Bot Token",
        botTokenPlaceholder: "123456:ABCDEF...",
        botTokenHint: "Enter the Telegram bot token used to send scheduled test notifications.",
        botTokenConfiguredHint: "A bot token is already configured. Leave empty to keep the current token.",
        chatId: "Chat ID",
        chatIdPlaceholder: "e.g. -1001234567890",
        chatIdHint: "The destination chat or group ID that receives scheduled test notifications.",
        testConnection: "Test Connection",
        testing: "Testing...",
        testSuccess: "Telegram connection successful",
        testFailed: "Telegram connection failed",
    }
}
