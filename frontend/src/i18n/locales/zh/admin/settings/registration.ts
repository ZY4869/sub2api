export default {
    registration: {
        title: "注册设置",
        description: "控制用户注册和验证",
        enableRegistration: "开放注册",
        enableRegistrationHint: "允许新用户注册",
        emailVerification: "邮箱验证",
        emailVerificationHint: "新用户注册时需要验证邮箱",
        emailSuffixWhitelist: "邮箱域名白名单",
        emailSuffixWhitelistHint: "仅允许使用指定域名的邮箱注册账号（例如 {'@'}qq.com, {'@'}gmail.com）",
        emailSuffixWhitelistPlaceholder: "example.com",
        emailSuffixWhitelistInputHint: "留空则不限制",
        promoCode: "优惠码",
        promoCodeHint: "允许用户在注册时使用优惠码",
        invitationCode: "邀请码注册",
        invitationCodeHint: "开启后，用户注册时需要填写有效的邀请码",
        passwordReset: "忘记密码",
        passwordResetHint: "允许用户通过邮箱重置密码",
        totp: "双因素认证 (2FA)",
        totpHint: "允许用户使用 Google Authenticator 等应用进行二次验证",
        totpKeyNotConfigured: "请先在环境变量中配置 TOTP_ENCRYPTION_KEY。使用命令 openssl rand -hex 32 生成密钥。",
    }
}
