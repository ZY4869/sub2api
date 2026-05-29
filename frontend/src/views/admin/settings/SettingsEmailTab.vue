<template>
        <div class="space-y-6">
        <!-- Email disabled hint - show when email_verify_enabled is off -->
        <div v-if="!form.email_verify_enabled" class="card">
          <div class="p-6">
            <div class="flex items-start gap-3">
              <Icon name="mail" size="md" class="mt-0.5 flex-shrink-0 text-gray-400 dark:text-gray-500" />
              <div>
                <h3 class="font-medium text-gray-900 dark:text-white">
                  {{ t('admin.settings.emailTabDisabledTitle') }}
                </h3>
                <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                  {{ t('admin.settings.emailTabDisabledHint') }}
                </p>
              </div>
            </div>
          </div>
        </div>

        <!-- SMTP Settings - Only show when email verification is enabled -->
        <div v-if="form.email_verify_enabled" class="card">
          <div
            class="flex items-center justify-between border-b border-gray-100 px-6 py-4 dark:border-dark-700"
          >
            <div>
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ t('admin.settings.smtp.title') }}
              </h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ t('admin.settings.smtp.description') }}
              </p>
            </div>
            <button
              type="button"
              @click="testSmtpConnection"
              :disabled="testingSmtp"
              class="btn btn-secondary btn-sm"
            >
              <svg v-if="testingSmtp" class="h-4 w-4 animate-spin" fill="none" viewBox="0 0 24 24">
                <circle
                  class="opacity-25"
                  cx="12"
                  cy="12"
                  r="10"
                  stroke="currentColor"
                  stroke-width="4"
                ></circle>
                <path
                  class="opacity-75"
                  fill="currentColor"
                  d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                ></path>
              </svg>
              {{
                testingSmtp
                  ? t('admin.settings.smtp.testing')
                  : t('admin.settings.smtp.testConnection')
              }}
            </button>
          </div>
          <div class="space-y-6 p-6">
            <div class="grid grid-cols-1 gap-6 md:grid-cols-2">
              <div>
                <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.smtp.host') }}
                </label>
                <input
                  v-model="form.smtp_host"
                  type="text"
                  class="input"
                  :placeholder="t('admin.settings.smtp.hostPlaceholder')"
                />
              </div>
              <div>
                <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.smtp.port') }}
                </label>
                <input
                  v-model.number="form.smtp_port"
                  type="number"
                  min="1"
                  max="65535"
                  class="input"
                  :placeholder="t('admin.settings.smtp.portPlaceholder')"
                />
              </div>
              <div>
                <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.smtp.username') }}
                </label>
                <input
                  v-model="form.smtp_username"
                  type="text"
                  class="input"
                  :placeholder="t('admin.settings.smtp.usernamePlaceholder')"
                />
              </div>
              <div>
                <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.smtp.password') }}
                </label>
                <input
                  v-model="form.smtp_password"
                  type="password"
                  class="input"
                  :placeholder="
                    form.smtp_password_configured
                      ? t('admin.settings.smtp.passwordConfiguredPlaceholder')
                      : t('admin.settings.smtp.passwordPlaceholder')
                  "
                />
                <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                  {{
                    form.smtp_password_configured
                      ? t('admin.settings.smtp.passwordConfiguredHint')
                      : t('admin.settings.smtp.passwordHint')
                  }}
                </p>
              </div>
              <div>
                <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.smtp.fromEmail') }}
                </label>
                <input
                  v-model="form.smtp_from_email"
                  type="email"
                  class="input"
                  :placeholder="t('admin.settings.smtp.fromEmailPlaceholder')"
                />
              </div>
              <div>
                <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.smtp.fromName') }}
                </label>
                <input
                  v-model="form.smtp_from_name"
                  type="text"
                  class="input"
                  :placeholder="t('admin.settings.smtp.fromNamePlaceholder')"
                />
              </div>
            </div>

            <!-- Use TLS Toggle -->
            <div
              class="flex items-center justify-between border-t border-gray-100 pt-4 dark:border-dark-700"
            >
              <div>
                <label class="font-medium text-gray-900 dark:text-white">{{
                  t('admin.settings.smtp.useTls')
                }}</label>
                <p class="text-sm text-gray-500 dark:text-gray-400">
                  {{ t('admin.settings.smtp.useTlsHint') }}
                </p>
              </div>
              <Toggle v-model="form.smtp_use_tls" />
            </div>
          </div>
        </div>

        <!-- Send Test Email - Only show when email verification is enabled -->
        <div v-if="form.email_verify_enabled" class="card">
          <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('admin.settings.testEmail.title') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.settings.testEmail.description') }}
            </p>
          </div>
          <div class="p-6">
            <div class="flex items-end gap-4">
              <div class="flex-1">
                <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.testEmail.recipientEmail') }}
                </label>
                <input
                  v-model="testEmailAddress"
                  type="email"
                  class="input"
                  :placeholder="t('admin.settings.testEmail.recipientEmailPlaceholder')"
                />
              </div>
              <button
                type="button"
                @click="sendTestEmail"
                :disabled="sendingTestEmail || !testEmailAddress"
                class="btn btn-secondary"
              >
                <svg
                  v-if="sendingTestEmail"
                  class="h-4 w-4 animate-spin"
                  fill="none"
                  viewBox="0 0 24 24"
                >
                  <circle
                    class="opacity-25"
                    cx="12"
                    cy="12"
                    r="10"
                    stroke="currentColor"
                    stroke-width="4"
                  ></circle>
                  <path
                    class="opacity-75"
                    fill="currentColor"
                    d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                  ></path>
                </svg>
                {{
                  sendingTestEmail
                    ? t('admin.settings.testEmail.sending')
                    : t('admin.settings.testEmail.sendTestEmail')
                }}
              </button>
            </div>
          </div>
        </div>

        <div v-if="form.email_verify_enabled" class="card">
          <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('admin.settings.emailTemplates.title') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.settings.emailTemplates.description') }}
            </p>
          </div>
          <div class="space-y-5 p-6">
            <div v-if="emailTemplatesLoading" class="flex items-center gap-2 text-gray-500">
              <div class="h-4 w-4 animate-spin rounded-full border-b-2 border-primary-600"></div>
              {{ t('common.loading') }}
            </div>
            <template v-else>
              <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
                <div>
                  <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                    {{ t('admin.settings.emailTemplates.template') }}
                  </label>
                  <Select
                    :modelValue="selectedEmailTemplateKey"
                    @update:modelValue="selectEmailTemplate(String($event || ''))"
                    :options="emailTemplateOptions"
                  />
                </div>
                <div>
                  <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                    {{ t('admin.settings.emailTemplates.locale') }}
                  </label>
                  <Select
                    :modelValue="selectedEmailTemplateLocale"
                    @update:modelValue="selectEmailTemplateLocale(String($event || 'en'))"
                    :options="emailTemplateLocaleOptions"
                  />
                </div>
              </div>

              <div v-if="selectedEmailTemplate" class="space-y-4">
                <div class="flex flex-wrap gap-2">
                  <span
                    v-for="name in selectedEmailTemplate.variables"
                    :key="name"
                    class="rounded bg-gray-100 px-2 py-1 font-mono text-xs text-gray-600 dark:bg-dark-700 dark:text-gray-300"
                  >
                    {{ formatEmailTemplateVariable(name) }}
                  </span>
                </div>
                <div>
                  <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                    {{ t('admin.settings.emailTemplates.subject') }}
                  </label>
                  <input v-model="emailTemplateDraft.subject" type="text" class="input" />
                </div>
                <div>
                  <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                    {{ t('admin.settings.emailTemplates.body') }}
                  </label>
                  <textarea
                    v-model="emailTemplateDraft.body"
                    rows="10"
                    class="input font-mono text-sm"
                  />
                </div>
                <div class="flex items-center justify-between border-t border-gray-100 pt-4 dark:border-dark-700">
                  <div>
                    <label class="font-medium text-gray-900 dark:text-white">
                      {{ t('admin.settings.emailTemplates.enabled') }}
                    </label>
                    <p class="text-sm text-gray-500 dark:text-gray-400">
                      {{ t('admin.settings.emailTemplates.enabledHint') }}
                    </p>
                  </div>
                  <Toggle v-model="emailTemplateDraft.enabled" />
                </div>
                <div class="grid grid-cols-1 gap-4 border-t border-gray-100 pt-4 dark:border-dark-700 md:grid-cols-[1fr_auto_auto_auto]">
                  <input
                    v-model="emailTemplateTestAddress"
                    type="email"
                    class="input"
                    :placeholder="t('admin.settings.emailTemplates.testRecipientPlaceholder')"
                  />
                  <button
                    type="button"
                    class="btn btn-secondary"
                    :disabled="emailTemplateTesting || !emailTemplateTestAddress"
                    @click="testSelectedEmailTemplate"
                  >
                    {{ emailTemplateTesting ? t('admin.settings.emailTemplates.testing') : t('admin.settings.emailTemplates.test') }}
                  </button>
                  <button
                    type="button"
                    class="btn btn-secondary"
                    :disabled="emailTemplateSaving"
                    @click="resetSelectedEmailTemplate"
                  >
                    {{ t('admin.settings.emailTemplates.reset') }}
                  </button>
                  <button
                    type="button"
                    class="btn btn-primary"
                    :disabled="emailTemplateSaving"
                    @click="saveSelectedEmailTemplate"
                  >
                    {{ emailTemplateSaving ? t('common.saving') : t('common.save') }}
                  </button>
                </div>
              </div>
            </template>
          </div>
        </div>
        </div><!-- /Tab: Email -->
</template>

<script setup lang="ts">
import Select from '@/components/common/Select.vue'
import Toggle from '@/components/common/Toggle.vue'
import Icon from '@/components/icons/Icon.vue'
const props = defineProps<{ ctx: any }>()
const {
  t,
  form,
  testingSmtp,
  testSmtpConnection,
  sendingTestEmail,
  sendTestEmail,
  testEmailAddress,
  emailTemplatesLoading,
  emailTemplateOptions,
  selectedEmailTemplateKey,
  selectEmailTemplate,
  emailTemplateLocaleOptions,
  selectedEmailTemplateLocale,
  selectEmailTemplateLocale,
  selectedEmailTemplate,
  formatEmailTemplateVariable,
  emailTemplateDraft,
  emailTemplateTestAddress,
  emailTemplateTesting,
  testSelectedEmailTemplate,
  emailTemplateSaving,
  resetSelectedEmailTemplate,
  saveSelectedEmailTemplate,
} = props.ctx
</script>

