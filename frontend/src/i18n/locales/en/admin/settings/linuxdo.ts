export default {
    linuxdo: {
        title: "LinuxDo Connect Login",
        description: "Configure LinuxDo Connect OAuth for Sub2API end-user login",
        enable: "Enable LinuxDo Login",
        enableHint: "Show LinuxDo login on the login/register pages",
        clientId: "Client ID",
        clientIdPlaceholder: "e.g., hprJ5pC3...",
        clientIdHint: "Get this from Connect.Linux.Do",
        clientSecret: "Client Secret",
        clientSecretPlaceholder: "********",
        clientSecretHint: "Used by backend to exchange tokens (keep it secret)",
        clientSecretConfiguredPlaceholder: "********",
        clientSecretConfiguredHint: "Secret configured. Leave empty to keep the current value.",
        redirectUrl: "Redirect URL",
        redirectUrlPlaceholder: "https://your-domain.com/api/v1/auth/oauth/linuxdo/callback",
        redirectUrlHint: "Must match the redirect URL configured in Connect.Linux.Do (must be an absolute http(s) URL)",
        quickSetCopy: "Generate & Copy (current site)",
        redirectUrlSetAndCopied: "Redirect URL generated and copied to clipboard",
    }
}
