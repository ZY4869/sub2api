export default {
    linuxdo: {
        title: "LinuxDo Connect 登录",
        description: "配置 LinuxDo Connect OAuth，用于 Sub2API 用户登录",
        enable: "启用 LinuxDo 登录",
        enableHint: "在登录/注册页面显示 LinuxDo 登录入口",
        clientId: "Client ID",
        clientIdPlaceholder: "例如：hprJ5pC3...",
        clientIdHint: "从 Connect.Linux.Do 后台获取",
        clientSecret: "Client Secret",
        clientSecretPlaceholder: "********",
        clientSecretHint: "用于后端交换 token（请保密）",
        clientSecretConfiguredPlaceholder: "********",
        clientSecretConfiguredHint: "密钥已配置，留空以保留当前值。",
        redirectUrl: "回调地址（Redirect URL）",
        redirectUrlPlaceholder: "https://your-domain.com/api/v1/auth/oauth/linuxdo/callback",
        redirectUrlHint: "需与 Connect.Linux.Do 中配置的回调地址一致（必须是 http(s) 完整 URL）",
        quickSetCopy: "使用当前站点生成并复制",
        redirectUrlSetAndCopied: "已使用当前站点生成回调地址并复制到剪贴板",
    }
}
