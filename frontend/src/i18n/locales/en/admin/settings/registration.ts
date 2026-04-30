export default {
    registration: {
        title: "Registration Settings",
        description: "Control user registration and verification",
        enableRegistration: "Enable Registration",
        enableRegistrationHint: "Allow new users to register",
        emailVerification: "Email Verification",
        emailVerificationHint: "Require email verification for new registrations",
        emailSuffixWhitelist: "Email Domain Whitelist",
        emailSuffixWhitelistHint: "Only email addresses from the specified domains can register (for example, {'@'}qq.com, {'@'}gmail.com)",
        emailSuffixWhitelistPlaceholder: "example.com",
        emailSuffixWhitelistInputHint: "Leave empty for no restriction",
        promoCode: "Promo Code",
        promoCodeHint: "Allow users to use promo codes during registration",
        invitationCode: "Invitation Code Registration",
        invitationCodeHint: "When enabled, users must enter a valid invitation code to register",
        passwordReset: "Password Reset",
        passwordResetHint: "Allow users to reset their password via email",
        totp: "Two-Factor Authentication (2FA)",
        totpHint: "Allow users to use authenticator apps like Google Authenticator",
        totpKeyNotConfigured: "Please configure TOTP_ENCRYPTION_KEY in environment variables first. Generate a key with: openssl rand -hex 32",
    }
}
