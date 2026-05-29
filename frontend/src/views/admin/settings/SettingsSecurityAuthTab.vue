<template>
        <div class="space-y-6">
        <!-- Registration Settings -->
        <div class="card">
          <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('admin.settings.registration.title') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.settings.registration.description') }}
            </p>
          </div>
          <div class="space-y-5 p-6">
            <!-- Enable Registration -->
            <div class="flex items-center justify-between">
              <div>
                <label class="font-medium text-gray-900 dark:text-white">{{
                  t('admin.settings.registration.enableRegistration')
                }}</label>
                <p class="text-sm text-gray-500 dark:text-gray-400">
                  {{ t('admin.settings.registration.enableRegistrationHint') }}
                </p>
              </div>
              <Toggle v-model="form.registration_enabled" />
            </div>

            <!-- Email Verification -->
            <div
              class="flex items-center justify-between border-t border-gray-100 pt-4 dark:border-dark-700"
            >
              <div>
                <label class="font-medium text-gray-900 dark:text-white">{{
                  t('admin.settings.registration.emailVerification')
                }}</label>
                <p class="text-sm text-gray-500 dark:text-gray-400">
                  {{ t('admin.settings.registration.emailVerificationHint') }}
                </p>
              </div>
              <Toggle v-model="form.email_verify_enabled" />
            </div>

            <!-- Email Suffix Whitelist -->
            <div class="border-t border-gray-100 pt-4 dark:border-dark-700">
              <label class="font-medium text-gray-900 dark:text-white">{{
                t('admin.settings.registration.emailSuffixWhitelist')
              }}</label>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ t('admin.settings.registration.emailSuffixWhitelistHint') }}
              </p>
              <div
                class="mt-3 rounded-lg border border-gray-300 bg-white p-2 dark:border-dark-500 dark:bg-dark-700"
              >
                <div class="flex flex-wrap items-center gap-2">
                  <span
                    v-for="suffix in registrationEmailSuffixWhitelistTags"
                    :key="suffix"
                    class="inline-flex items-center gap-1 rounded bg-gray-100 px-2 py-1 text-xs font-mono text-gray-700 dark:bg-dark-600 dark:text-gray-200"
                  >
                    <span class="text-gray-400 dark:text-gray-500">@</span>
                    <span>{{ suffix }}</span>
                    <button
                      type="button"
                      class="rounded-full text-gray-500 hover:bg-gray-200 hover:text-gray-700 dark:text-gray-300 dark:hover:bg-dark-500 dark:hover:text-white"
                      @click="removeRegistrationEmailSuffixWhitelistTag(suffix)"
                    >
                      <Icon name="x" size="xs" class="h-3.5 w-3.5" :stroke-width="2" />
                    </button>
                  </span>

                  <div
                    class="flex min-w-[220px] flex-1 items-center gap-1 rounded border border-transparent px-2 py-1 focus-within:border-primary-300 dark:focus-within:border-primary-700"
                  >
                    <span class="font-mono text-sm text-gray-400 dark:text-gray-500">@</span>
                    <input
                      v-model="registrationEmailSuffixWhitelistDraft"
                      type="text"
                      class="w-full bg-transparent text-sm font-mono text-gray-900 outline-none placeholder:text-gray-400 dark:text-white dark:placeholder:text-gray-500"
                      :placeholder="t('admin.settings.registration.emailSuffixWhitelistPlaceholder')"
                      @input="handleRegistrationEmailSuffixWhitelistDraftInput"
                      @keydown="handleRegistrationEmailSuffixWhitelistDraftKeydown"
                      @blur="commitRegistrationEmailSuffixWhitelistDraft"
                      @paste="handleRegistrationEmailSuffixWhitelistPaste"
                    />
                  </div>
                </div>
              </div>
              <p class="mt-2 text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.settings.registration.emailSuffixWhitelistInputHint') }}
              </p>
            </div>

            <!-- Promo Code -->
            <div
              class="flex items-center justify-between border-t border-gray-100 pt-4 dark:border-dark-700"
            >
              <div>
                <label class="font-medium text-gray-900 dark:text-white">{{
                  t('admin.settings.registration.promoCode')
                }}</label>
                <p class="text-sm text-gray-500 dark:text-gray-400">
                  {{ t('admin.settings.registration.promoCodeHint') }}
                </p>
              </div>
              <Toggle v-model="form.promo_code_enabled" />
            </div>

            <!-- Invitation Code -->
            <div
              class="flex items-center justify-between border-t border-gray-100 pt-4 dark:border-dark-700"
            >
              <div>
                <label class="font-medium text-gray-900 dark:text-white">{{
                  t('admin.settings.registration.invitationCode')
                }}</label>
                <p class="text-sm text-gray-500 dark:text-gray-400">
                  {{ t('admin.settings.registration.invitationCodeHint') }}
                </p>
              </div>
              <Toggle v-model="form.invitation_code_enabled" />
            </div>
            <!-- Password Reset - Only show when email verification is enabled -->
            <div
              v-if="form.email_verify_enabled"
              class="flex items-center justify-between border-t border-gray-100 pt-4 dark:border-dark-700"
            >
              <div>
                <label class="font-medium text-gray-900 dark:text-white">{{
                  t('admin.settings.registration.passwordReset')
                }}</label>
                <p class="text-sm text-gray-500 dark:text-gray-400">
                  {{ t('admin.settings.registration.passwordResetHint') }}
                </p>
              </div>
              <Toggle v-model="form.password_reset_enabled" />
            </div>

            <!-- TOTP 2FA -->
            <div
              class="flex items-center justify-between border-t border-gray-100 pt-4 dark:border-dark-700"
            >
              <div>
                <label class="font-medium text-gray-900 dark:text-white">{{
                  t('admin.settings.registration.totp')
                }}</label>
                <p class="text-sm text-gray-500 dark:text-gray-400">
                  {{ t('admin.settings.registration.totpHint') }}
                </p>
                <!-- Warning when encryption key not configured -->
                <p
                  v-if="!form.totp_encryption_key_configured"
                  class="mt-2 text-sm text-amber-600 dark:text-amber-400"
                >
                  {{ t('admin.settings.registration.totpKeyNotConfigured') }}
                </p>
              </div>
              <Toggle
                v-model="form.totp_enabled"
                :disabled="!form.totp_encryption_key_configured"
              />
            </div>
          </div>
        </div>

        <!-- Cloudflare Turnstile Settings -->
        <div class="card">
          <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('admin.settings.turnstile.title') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.settings.turnstile.description') }}
            </p>
          </div>
          <div class="space-y-5 p-6">
            <!-- Enable Turnstile -->
            <div class="flex items-center justify-between">
              <div>
                <label class="font-medium text-gray-900 dark:text-white">{{
                  t('admin.settings.turnstile.enableTurnstile')
                }}</label>
                <p class="text-sm text-gray-500 dark:text-gray-400">
                  {{ t('admin.settings.turnstile.enableTurnstileHint') }}
                </p>
              </div>
              <Toggle v-model="form.turnstile_enabled" />
            </div>

            <!-- Turnstile Keys - Only show when enabled -->
            <div
              v-if="form.turnstile_enabled"
              class="border-t border-gray-100 pt-4 dark:border-dark-700"
            >
              <div class="grid grid-cols-1 gap-6">
                <div>
                  <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                    {{ t('admin.settings.turnstile.siteKey') }}
                  </label>
                  <input
                    v-model="form.turnstile_site_key"
                    type="text"
                    class="input font-mono text-sm"
                    placeholder="0x4AAAAAAA..."
                  />
                  <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                    {{ t('admin.settings.turnstile.siteKeyHint') }}
                    <a
                      href="https://dash.cloudflare.com/"
                      target="_blank"
                      class="text-primary-600 hover:text-primary-500"
                      >{{ t('admin.settings.turnstile.cloudflareDashboard') }}</a
                    >
                  </p>
                </div>
                <div>
                  <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                    {{ t('admin.settings.turnstile.secretKey') }}
                  </label>
                  <input
                    v-model="form.turnstile_secret_key"
                    type="password"
                    class="input font-mono text-sm"
                    placeholder="0x4AAAAAAA..."
                  />
                  <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                    {{
                      form.turnstile_secret_key_configured
                        ? t('admin.settings.turnstile.secretKeyConfiguredHint')
                        : t('admin.settings.turnstile.secretKeyHint')
                    }}
                  </p>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- LinuxDo Connect OAuth 登录 -->
        <div class="card">
          <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('admin.settings.linuxdo.title') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.settings.linuxdo.description') }}
            </p>
          </div>
          <div class="space-y-5 p-6">
            <div class="flex items-center justify-between">
              <div>
                <label class="font-medium text-gray-900 dark:text-white">{{
                  t('admin.settings.linuxdo.enable')
                }}</label>
                <p class="text-sm text-gray-500 dark:text-gray-400">
                  {{ t('admin.settings.linuxdo.enableHint') }}
                </p>
              </div>
              <Toggle v-model="form.linuxdo_connect_enabled" />
            </div>

            <div
              v-if="form.linuxdo_connect_enabled"
              class="border-t border-gray-100 pt-4 dark:border-dark-700"
            >
              <div class="grid grid-cols-1 gap-6">
                <div>
                  <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                    {{ t('admin.settings.linuxdo.clientId') }}
                  </label>
                  <input
                    v-model="form.linuxdo_connect_client_id"
                    type="text"
                    class="input font-mono text-sm"
                    :placeholder="t('admin.settings.linuxdo.clientIdPlaceholder')"
                  />
                  <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                    {{ t('admin.settings.linuxdo.clientIdHint') }}
                  </p>
                </div>

                <div>
                  <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                    {{ t('admin.settings.linuxdo.clientSecret') }}
                  </label>
                  <input
                    v-model="form.linuxdo_connect_client_secret"
                    type="password"
                    class="input font-mono text-sm"
                    :placeholder="
                      form.linuxdo_connect_client_secret_configured
                        ? t('admin.settings.linuxdo.clientSecretConfiguredPlaceholder')
                        : t('admin.settings.linuxdo.clientSecretPlaceholder')
                    "
                  />
                  <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                    {{
                      form.linuxdo_connect_client_secret_configured
                        ? t('admin.settings.linuxdo.clientSecretConfiguredHint')
                        : t('admin.settings.linuxdo.clientSecretHint')
                    }}
                  </p>
                </div>

                <div>
                  <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                    {{ t('admin.settings.linuxdo.redirectUrl') }}
                  </label>
                  <input
                    v-model="form.linuxdo_connect_redirect_url"
                    type="url"
                    class="input font-mono text-sm"
                    :placeholder="t('admin.settings.linuxdo.redirectUrlPlaceholder')"
                  />
                  <div class="mt-2 flex flex-col gap-2 sm:flex-row sm:items-center sm:gap-3">
                    <button
                      type="button"
                      class="btn btn-secondary btn-sm w-fit"
                      @click="setAndCopyLinuxdoRedirectUrl"
                    >
                      {{ t('admin.settings.linuxdo.quickSetCopy') }}
                    </button>
                    <code
                      v-if="linuxdoRedirectUrlSuggestion"
                      class="select-all break-all rounded bg-gray-50 px-2 py-1 font-mono text-xs text-gray-600 dark:bg-dark-800 dark:text-gray-300"
                    >
                      {{ linuxdoRedirectUrlSuggestion }}
                    </code>
                  </div>
                  <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                    {{ t('admin.settings.linuxdo.redirectUrlHint') }}
                  </p>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div class="card">
          <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('admin.settings.socialOAuth.title') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.settings.socialOAuth.description') }}
            </p>
          </div>
          <div class="space-y-6 p-6">
            <div class="space-y-5 rounded-2xl border border-gray-100 p-5 dark:border-dark-700">
              <div class="flex items-center justify-between">
                <div>
                  <label class="font-medium text-gray-900 dark:text-white">
                    {{ t('admin.settings.socialOAuth.githubEnable') }}
                  </label>
                  <p class="text-sm text-gray-500 dark:text-gray-400">
                    {{ t('admin.settings.socialOAuth.githubEnableHint') }}
                  </p>
                </div>
                <Toggle v-model="form.github_oauth_enabled" />
              </div>

              <div v-if="form.github_oauth_enabled" class="grid grid-cols-1 gap-6 border-t border-gray-100 pt-4 dark:border-dark-700">
                <div>
                  <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                    {{ t('admin.settings.socialOAuth.clientId') }}
                  </label>
                  <input
                    v-model="form.github_oauth_client_id"
                    type="text"
                    class="input font-mono text-sm"
                    :placeholder="t('admin.settings.socialOAuth.githubClientIdPlaceholder')"
                  />
                </div>
                <div>
                  <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                    {{ t('admin.settings.socialOAuth.clientSecret') }}
                  </label>
                  <input
                    v-model="form.github_oauth_client_secret"
                    type="password"
                    class="input font-mono text-sm"
                    :placeholder="
                      form.github_oauth_client_secret_configured
                        ? t('admin.settings.socialOAuth.clientSecretConfiguredPlaceholder')
                        : t('admin.settings.socialOAuth.githubClientSecretPlaceholder')
                    "
                  />
                </div>
                <div>
                  <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                    {{ t('admin.settings.socialOAuth.redirectUrl') }}
                  </label>
                  <input
                    v-model="form.github_oauth_redirect_url"
                    type="url"
                    class="input font-mono text-sm"
                    :placeholder="t('admin.settings.socialOAuth.githubRedirectUrlPlaceholder')"
                  />
                </div>
              </div>
            </div>

            <div class="space-y-5 rounded-2xl border border-gray-100 p-5 dark:border-dark-700">
              <div class="flex items-center justify-between">
                <div>
                  <label class="font-medium text-gray-900 dark:text-white">
                    {{ t('admin.settings.socialOAuth.googleEnable') }}
                  </label>
                  <p class="text-sm text-gray-500 dark:text-gray-400">
                    {{ t('admin.settings.socialOAuth.googleEnableHint') }}
                  </p>
                </div>
                <Toggle v-model="form.google_oauth_enabled" />
              </div>

              <div v-if="form.google_oauth_enabled" class="grid grid-cols-1 gap-6 border-t border-gray-100 pt-4 dark:border-dark-700">
                <div>
                  <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                    {{ t('admin.settings.socialOAuth.clientId') }}
                  </label>
                  <input
                    v-model="form.google_oauth_client_id"
                    type="text"
                    class="input font-mono text-sm"
                    :placeholder="t('admin.settings.socialOAuth.googleClientIdPlaceholder')"
                  />
                </div>
                <div>
                  <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                    {{ t('admin.settings.socialOAuth.clientSecret') }}
                  </label>
                  <input
                    v-model="form.google_oauth_client_secret"
                    type="password"
                    class="input font-mono text-sm"
                    :placeholder="
                      form.google_oauth_client_secret_configured
                        ? t('admin.settings.socialOAuth.clientSecretConfiguredPlaceholder')
                        : t('admin.settings.socialOAuth.googleClientSecretPlaceholder')
                    "
                  />
                </div>
                <div>
                  <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                    {{ t('admin.settings.socialOAuth.redirectUrl') }}
                  </label>
                  <input
                    v-model="form.google_oauth_redirect_url"
                    type="url"
                    class="input font-mono text-sm"
                    :placeholder="t('admin.settings.socialOAuth.googleRedirectUrlPlaceholder')"
                  />
                </div>
              </div>
            </div>

            <div class="space-y-5 rounded-2xl border border-gray-100 p-5 dark:border-dark-700">
              <div class="flex items-center justify-between">
                <div>
                  <label class="font-medium text-gray-900 dark:text-white">
                    {{ t('admin.settings.socialOAuth.dingtalkEnable') }}
                  </label>
                  <p class="text-sm text-gray-500 dark:text-gray-400">
                    {{ t('admin.settings.socialOAuth.dingtalkEnableHint') }}
                  </p>
                </div>
                <Toggle v-model="form.dingtalk_oauth_enabled" />
              </div>

              <div v-if="form.dingtalk_oauth_enabled" class="grid grid-cols-1 gap-6 border-t border-gray-100 pt-4 dark:border-dark-700">
                <div>
                  <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                    {{ t('admin.settings.socialOAuth.clientId') }}
                  </label>
                  <input
                    v-model="form.dingtalk_oauth_client_id"
                    type="text"
                    class="input font-mono text-sm"
                    :placeholder="t('admin.settings.socialOAuth.dingtalkClientIdPlaceholder')"
                  />
                </div>
                <div>
                  <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                    {{ t('admin.settings.socialOAuth.clientSecret') }}
                  </label>
                  <input
                    v-model="form.dingtalk_oauth_client_secret"
                    type="password"
                    class="input font-mono text-sm"
                    :placeholder="
                      form.dingtalk_oauth_client_secret_configured
                        ? t('admin.settings.socialOAuth.clientSecretConfiguredPlaceholder')
                        : t('admin.settings.socialOAuth.dingtalkClientSecretPlaceholder')
                    "
                  />
                </div>
                <div>
                  <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                    {{ t('admin.settings.socialOAuth.redirectUrl') }}
                  </label>
                  <input
                    v-model="form.dingtalk_oauth_redirect_url"
                    type="url"
                    class="input font-mono text-sm"
                    :placeholder="t('admin.settings.socialOAuth.dingtalkRedirectUrlPlaceholder')"
                  />
                </div>
              </div>
            </div>
          </div>
        </div>

        <div class="card">
          <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('admin.settings.moderation.title') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.settings.moderation.description') }}
            </p>
          </div>
          <div class="space-y-6 p-6">
            <div class="flex items-center justify-between rounded-2xl border border-gray-100 p-5 dark:border-dark-700">
              <div>
                <label class="font-medium text-gray-900 dark:text-white">
                  {{ t('admin.settings.moderation.enabled') }}
                </label>
                <p class="text-sm text-gray-500 dark:text-gray-400">
                  {{ t('admin.settings.moderation.enabledHint') }}
                </p>
              </div>
              <Toggle v-model="form.content_moderation_enabled" />
            </div>

            <div
              v-if="form.content_moderation_enabled"
              class="grid grid-cols-1 gap-6 rounded-2xl border border-gray-100 p-5 dark:border-dark-700 md:grid-cols-2"
            >
              <div>
                <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.moderation.provider') }}
                </label>
                <input
                  v-model="form.content_moderation_provider"
                  type="text"
                  class="input"
                  placeholder="openai"
                />
                <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.settings.moderation.providerHint') }}
                </p>
              </div>
              <div>
                <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.moderation.model') }}
                </label>
                <input
                  v-model="form.content_moderation_model"
                  type="text"
                  class="input"
                  placeholder="omni-moderation-latest"
                />
                <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.settings.moderation.modelHint') }}
                </p>
              </div>
              <div class="md:col-span-2">
                <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.moderation.baseUrl') }}
                </label>
                <input
                  v-model="form.content_moderation_base_url"
                  type="url"
                  class="input font-mono text-sm"
                  placeholder="https://api.example.com"
                />
                <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.settings.moderation.baseUrlHint') }}
                </p>
              </div>
              <div class="md:col-span-2">
                <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.moderation.addApiKey') }}
                </label>
                <input
                  v-model="form.content_moderation_api_key"
                  type="password"
                  class="input font-mono text-sm"
                  :placeholder="
                    form.content_moderation_api_key_configured
                      ? t('admin.settings.moderation.apiKeyConfiguredPlaceholder')
                      : 'sk-...'
                  "
                />
                <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.settings.moderation.apiKeyHint') }}
                </p>
              </div>
              <div
                v-if="form.content_moderation_api_key_statuses.length > 0"
                class="md:col-span-2"
              >
                <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.moderation.configuredKeys') }}
                </label>
                <div class="space-y-2">
                  <div
                    v-for="keyStatus in form.content_moderation_api_key_statuses"
                    :key="keyStatus.hash"
                    class="flex items-center justify-between gap-3 rounded-lg border border-gray-100 px-3 py-2 dark:border-dark-700"
                  >
                    <div class="min-w-0">
                      <div class="truncate font-mono text-sm text-gray-900 dark:text-white">
                        {{ keyStatus.masked }}
                      </div>
                      <div class="truncate text-xs text-gray-500 dark:text-gray-400">
                        {{ keyStatus.hash }}
                      </div>
                      <div
                        v-if="keyStatus.frozen_until"
                        class="mt-1 text-xs text-amber-600 dark:text-amber-400"
                      >
                        {{
                          t('admin.settings.moderation.frozenUntil', {
                            time: keyStatus.frozen_until
                          })
                        }}
                      </div>
                    </div>
                    <button
                      type="button"
                      class="btn btn-secondary btn-sm shrink-0"
                      @click="deleteContentModerationKey(keyStatus.hash)"
                    >
                      {{ t('admin.settings.moderation.deleteKey') }}
                    </button>
                  </div>
                </div>
              </div>
              <div>
                <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.moderation.timeoutMs') }}
                </label>
                <input
                  v-model.number="form.content_moderation_timeout_ms"
                  type="number"
                  min="100"
                  step="100"
                  class="input"
                />
                <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.settings.moderation.timeoutMsHint') }}
                </p>
              </div>
              <div>
                <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.moderation.dedupeWindowSeconds') }}
                </label>
                <input
                  v-model.number="form.content_moderation_dedupe_window_seconds"
                  type="number"
                  min="0"
                  step="10"
                  class="input"
                />
                <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.settings.moderation.dedupeWindowSecondsHint') }}
                </p>
              </div>
              <div class="md:col-span-2">
                <div class="flex items-center justify-between rounded-2xl border border-gray-100 p-4 dark:border-dark-700">
                  <div>
                    <label class="font-medium text-gray-900 dark:text-white">
                      {{ t('admin.settings.moderation.failOpen') }}
                    </label>
                    <p class="text-sm text-gray-500 dark:text-gray-400">
                      {{ t('admin.settings.moderation.failOpenHint') }}
                    </p>
                  </div>
                  <Toggle v-model="form.content_moderation_fail_open" />
                </div>
              </div>
              <div class="md:col-span-2">
                <div class="flex items-center justify-between rounded-2xl border border-gray-100 p-4 dark:border-dark-700">
                  <div>
                    <label class="font-medium text-gray-900 dark:text-white">
                      {{ t('admin.settings.moderation.keywordBlock') }}
                    </label>
                    <p class="text-sm text-gray-500 dark:text-gray-400">
                      {{ t('admin.settings.moderation.keywordBlockHint') }}
                    </p>
                  </div>
                  <Toggle v-model="form.content_moderation_keyword_block_enabled" />
                </div>
              </div>
              <div class="md:col-span-2">
                <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.moderation.modelFilterType') }}
                </label>
                <Select
                  v-model="form.content_moderation_model_filter.type"
                  :options="contentModerationModelFilterOptions"
                />
                <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.settings.moderation.modelFilterTypeHint') }}
                </p>
              </div>
              <ContentModerationThresholdsEditor
                v-model="form.content_moderation_category_thresholds"
              />
              <div
                v-if="form.content_moderation_model_filter.type !== 'all'"
                class="md:col-span-2"
              >
                <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.moderation.modelFilterModels') }}
                </label>
                <textarea
                  v-model="contentModerationModelFilterModelsText"
                  rows="4"
                  class="input font-mono text-sm"
                  :placeholder="t('admin.settings.moderation.modelFilterModelsPlaceholder')"
                />
                <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.settings.moderation.modelFilterModelsHint') }}
                </p>
              </div>
              <div class="md:col-span-2">
                <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.moderation.keywords') }}
                </label>
                <textarea
                  v-model="contentModerationKeywordsText"
                  rows="5"
                  class="input font-mono text-sm"
                  :placeholder="t('admin.settings.moderation.keywordsPlaceholder')"
                />
                <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.settings.moderation.keywordsHint') }}
                </p>
              </div>
            </div>
          </div>
        </div>
        </div><!-- /Tab: Security — Registration, Turnstile, LinuxDo -->
</template>

<script setup lang="ts">
import Select from '@/components/common/Select.vue'
import Toggle from '@/components/common/Toggle.vue'
import Icon from '@/components/icons/Icon.vue'
import ContentModerationThresholdsEditor from '@/components/admin/settings/ContentModerationThresholdsEditor.vue'
const props = defineProps<{ ctx: any }>()
const {
  t,
  form,
  registrationEmailSuffixWhitelistTags,
  registrationEmailSuffixWhitelistDraft,
  removeRegistrationEmailSuffixWhitelistTag,
  handleRegistrationEmailSuffixWhitelistDraftInput,
  handleRegistrationEmailSuffixWhitelistDraftKeydown,
  handleRegistrationEmailSuffixWhitelistPaste,
  commitRegistrationEmailSuffixWhitelistDraft,
  linuxdoRedirectUrlSuggestion,
  setAndCopyLinuxdoRedirectUrl,
  contentModerationModelFilterOptions,
  contentModerationKeywordsText,
  contentModerationModelFilterModelsText,
  deleteContentModerationKey,
} = props.ctx
</script>

