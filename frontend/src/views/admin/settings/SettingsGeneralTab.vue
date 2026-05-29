<template>
        <div class="space-y-6">
        <div class="card">
          <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('admin.settings.realtimeCountdown.title') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.settings.realtimeCountdown.description') }}
            </p>
          </div>
          <div class="space-y-5 p-6">
            <div class="flex items-center justify-between">
              <div>
                <label class="font-medium text-gray-900 dark:text-white">
                  {{ t('admin.settings.realtimeCountdown.globalEnabled') }}
                </label>
                <p class="text-sm text-gray-500 dark:text-gray-400">
                  {{ t('admin.settings.realtimeCountdown.globalEnabledHint') }}
                </p>
              </div>
              <Toggle v-model="globalRealtimeCountdownEnabled" />
            </div>

            <div class="rounded-xl border border-sky-200 bg-sky-50 px-4 py-3 text-xs text-sky-800 dark:border-sky-700/40 dark:bg-sky-950/30 dark:text-sky-200">
              {{ t('admin.settings.realtimeCountdown.scopeHint') }}
            </div>

            <div class="flex justify-end border-t border-gray-100 pt-4 dark:border-dark-700">
              <button
                type="button"
                class="btn btn-primary btn-sm"
                :disabled="savingGlobalRealtimeCountdown"
                @click="saveGlobalRealtimeCountdownPreference"
              >
                <svg
                  v-if="savingGlobalRealtimeCountdown"
                  class="mr-1 h-4 w-4 animate-spin"
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
                {{ savingGlobalRealtimeCountdown ? t('common.saving') : t('common.save') }}
              </button>
            </div>
          </div>
        </div>

        <!-- Site Settings -->
        <div class="card">
          <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('admin.settings.site.title') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.settings.site.description') }}
            </p>
          </div>
          <div class="space-y-6 p-6">
            <div class="grid grid-cols-1 gap-6 md:grid-cols-2">
              <div>
                <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.site.siteName') }}
                </label>
                <input
                  v-model="form.site_name"
                  type="text"
                  class="input"
                  :placeholder="t('admin.settings.site.siteNamePlaceholder')"
                />
                <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.settings.site.siteNameHint') }}
                </p>
              </div>
              <div>
                <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.site.siteSubtitle') }}
                </label>
                <input
                  v-model="form.site_subtitle"
                  type="text"
                  class="input"
                  :placeholder="t('admin.settings.site.siteSubtitlePlaceholder')"
                />
                <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.settings.site.siteSubtitleHint') }}
                </p>
              </div>
            </div>

            <div>
              <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.site.visualPresetDefault') }}
              </label>
              <div class="grid grid-cols-1 gap-2 sm:grid-cols-2">
                <button
                  type="button"
                  class="rounded-xl border px-3 py-2 text-sm font-medium transition"
                  :class="
                    form.visual_preset_default === 'classic'
                      ? 'border-primary-500 bg-primary-50 text-primary-700 dark:border-primary-400 dark:bg-primary-500/15 dark:text-primary-200'
                      : 'border-gray-200 bg-white text-gray-700 hover:border-gray-300 hover:bg-gray-50 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-200 dark:hover:bg-dark-700'
                  "
                  @click="form.visual_preset_default = 'classic'"
                >
                  {{ t('admin.settings.site.visualPresetClassic') }}
                </button>
                <button
                  type="button"
                  class="rounded-xl border px-3 py-2 text-sm font-medium transition"
                  :class="
                    form.visual_preset_default === 'airy'
                      ? 'border-primary-500 bg-primary-50 text-primary-700 dark:border-primary-400 dark:bg-primary-500/15 dark:text-primary-200'
                      : 'border-gray-200 bg-white text-gray-700 hover:border-gray-300 hover:bg-gray-50 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-200 dark:hover:bg-dark-700'
                  "
                  @click="form.visual_preset_default = 'airy'"
                >
                  {{ t('admin.settings.site.visualPresetAiry') }}
                </button>
              </div>
              <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.settings.site.visualPresetDefaultHint') }}
              </p>
            </div>

            <div>
              <div class="flex items-center justify-between gap-4 rounded-2xl border border-gray-200 bg-gray-50 px-4 py-3 dark:border-dark-600 dark:bg-dark-800/80">
                <div>
                  <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                    {{ t('admin.settings.site.accountAiryWhiteSurfaceEnabled') }}
                  </label>
                  <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                    {{ t('admin.settings.site.accountAiryWhiteSurfaceEnabledHint') }}
                  </p>
                </div>
                <Toggle v-model="form.account_airy_white_surface_enabled" />
              </div>
            </div>

            <!-- API Base URL -->
            <div>
              <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.site.apiBaseUrl') }}
              </label>
              <input
                v-model="form.api_base_url"
                type="text"
                class="input font-mono text-sm"
                :placeholder="t('admin.settings.site.apiBaseUrlPlaceholder')"
              />
              <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.settings.site.apiBaseUrlHint') }}
              </p>
            </div>

            <!-- Contact Info -->
            <div>
              <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.site.contactInfo') }}
              </label>
              <input
                v-model="form.contact_info"
                type="text"
                class="input"
                :placeholder="t('admin.settings.site.contactInfoPlaceholder')"
              />
              <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.settings.site.contactInfoHint') }}
              </p>
            </div>

            <!-- Doc URL -->
            <div>
              <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.site.docUrl') }}
              </label>
              <input
                v-model="form.doc_url"
                type="url"
                class="input font-mono text-sm"
                :placeholder="t('admin.settings.site.docUrlPlaceholder')"
              />
              <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.settings.site.docUrlHint') }}
              </p>
            </div>

            <!-- Site Logo Upload -->
            <div>
              <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.site.siteLogo') }}
              </label>
              <ImageUpload
                v-model="form.site_logo"
                mode="image"
                :upload-label="t('admin.settings.site.uploadImage')"
                :remove-label="t('admin.settings.site.remove')"
                :hint="t('admin.settings.site.logoHint')"
                :max-size="300 * 1024"
              />
            </div>

            <!-- Home Content -->
            <div>
              <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.site.homeContent') }}
              </label>
              <textarea
                v-model="form.home_content"
                rows="6"
                class="input font-mono text-sm"
                :placeholder="t('admin.settings.site.homeContentPlaceholder')"
              ></textarea>
              <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.settings.site.homeContentHint') }}
              </p>
              <!-- iframe CSP Warning -->
              <p class="mt-2 text-xs text-amber-600 dark:text-amber-400">
                {{ t('admin.settings.site.homeContentIframeWarning') }}
              </p>
            </div>

            <!-- Hide CCS Import Button -->
            <div
              class="flex items-center justify-between border-t border-gray-100 pt-4 dark:border-dark-700"
            >
              <div>
                <label class="font-medium text-gray-900 dark:text-white">{{
                  t('admin.settings.site.hideCcsImportButton')
                }}</label>
                <p class="text-sm text-gray-500 dark:text-gray-400">
                  {{ t('admin.settings.site.hideCcsImportButtonHint') }}
                </p>
              </div>
              <Toggle v-model="form.hide_ccs_import_button" />
            </div>

            <div
              class="flex items-center justify-between border-t border-gray-100 pt-4 dark:border-dark-700"
            >
              <div>
                <label class="font-medium text-gray-900 dark:text-white">{{
                  t('admin.settings.site.availableChannelsEnabled')
                }}</label>
                <p class="text-sm text-gray-500 dark:text-gray-400">
                  {{ t('admin.settings.site.availableChannelsEnabledHint') }}
                </p>
              </div>
              <Toggle v-model="form.available_channels_enabled" />
            </div>

            <div
              class="flex items-center justify-between border-t border-gray-100 pt-4 dark:border-dark-700"
            >
              <div>
                <label class="font-medium text-gray-900 dark:text-white">{{
                  t('admin.settings.site.channelMonitorEnabled')
                }}</label>
                <p class="text-sm text-gray-500 dark:text-gray-400">
                  {{ t('admin.settings.site.channelMonitorEnabledHint') }}
                </p>
              </div>
              <Toggle v-model="form.channel_monitor_enabled" />
            </div>

            <div
              class="flex items-center justify-between border-t border-gray-100 pt-4 dark:border-dark-700"
            >
              <div>
                <label class="font-medium text-gray-900 dark:text-white">{{
                  t('admin.settings.site.channelMonitorDefaultIntervalSeconds')
                }}</label>
                <p class="text-sm text-gray-500 dark:text-gray-400">
                  {{ t('admin.settings.site.channelMonitorDefaultIntervalSecondsHint') }}
                </p>
              </div>
              <input
                v-model.number="form.channel_monitor_default_interval_seconds"
                type="number"
                class="input w-32 text-right font-mono text-sm"
                min="15"
                max="3600"
                step="1"
                :disabled="!form.channel_monitor_enabled"
              />
            </div>

            <div
              class="flex items-center justify-between border-t border-gray-100 pt-4 dark:border-dark-700"
            >
              <div>
                <label class="font-medium text-gray-900 dark:text-white">{{
                  t('admin.settings.site.publicModelCatalogEnabled')
                }}</label>
                <p class="text-sm text-gray-500 dark:text-gray-400">
                  {{ t('admin.settings.site.publicModelCatalogEnabledHint') }}
                </p>
              </div>
              <Toggle v-model="form.public_model_catalog_enabled" />
            </div>
          </div>
        </div>

        <PaymentSettingsCard
          v-model:enabled="form.purchase_subscription_enabled"
          v-model:purchase-url="form.purchase_subscription_url"
          v-model:airwallex-enabled="form.payment_provider_airwallex_enabled"
          v-model:airwallex-env="form.airwallex_env"
          v-model:airwallex-client-id="form.airwallex_client_id"
          v-model:airwallex-api-key="form.airwallex_api_key"
          v-model:airwallex-webhook-secret="form.airwallex_webhook_secret"
          v-model:mobile-force-qrcode-enabled="form.payment_mobile_force_qrcode_enabled"
          v-model:allowed-currencies="form.payment_allowed_currencies"
          v-model:default-currency="form.payment_default_currency"
          v-model:min-topup-amount="form.payment_min_topup_amount"
          v-model:max-topup-amount="form.payment_max_topup_amount"
          v-model:subscription-plans="form.payment_subscription_plans"
          v-model:antigravity-user-agent-version="form.antigravity_user_agent_version"
          v-model:codex-oauth-user-agent-mode="form.codex_oauth_user_agent_mode"
          v-model:codex-oauth-user-agent-override="form.codex_oauth_user_agent_override"
          :api-key-configured="form.airwallex_api_key_configured"
          :webhook-secret-configured="form.airwallex_webhook_secret_configured"
          :effective-enabled="form.payment_provider_airwallex_effective"
        />

        <CustomMenuSettingsCard v-model="form.custom_menu_items" />

        <div class="card">
          <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('admin.settings.loginAgreement.title') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.settings.loginAgreement.description') }}
            </p>
          </div>
          <div class="space-y-6 p-6">
            <div class="flex items-center justify-between rounded-2xl border border-gray-100 p-5 dark:border-dark-700">
              <div>
                <label class="font-medium text-gray-900 dark:text-white">
                  {{ t('admin.settings.loginAgreement.enabled') }}
                </label>
                <p class="text-sm text-gray-500 dark:text-gray-400">
                  {{ t('admin.settings.loginAgreement.enabledHint') }}
                </p>
              </div>
              <Toggle v-model="form.login_agreement_enabled" />
            </div>

            <div v-if="form.login_agreement_enabled" class="space-y-4">
              <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
                <div>
                  <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                    {{ t('admin.settings.loginAgreement.mode') }}
                  </label>
                  <Select
                    v-model="form.login_agreement_mode"
                    :options="loginAgreementModeOptions"
                  />
                </div>
                <div>
                  <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                    {{ t('admin.settings.loginAgreement.updatedAt') }}
                  </label>
                  <input
                    v-model="form.login_agreement_updated_at"
                    type="text"
                    class="input"
                    placeholder="2026-05-07"
                  />
                </div>
              </div>

              <div>
                <div class="mb-2 flex items-center justify-between gap-3">
                  <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                    {{ t('admin.settings.loginAgreement.documents') }}
                  </label>
                  <button
                    type="button"
                    class="btn btn-secondary btn-sm"
                    :disabled="publishedMarkdownPageOptions.length === 0"
                    @click="addLoginAgreementDocument"
                  >
                    {{ t('admin.settings.loginAgreement.addDocument') }}
                  </button>
                </div>
                <p
                  v-if="publishedMarkdownPageOptions.length === 0"
                  class="text-sm text-amber-600 dark:text-amber-400"
                >
                  {{ t('admin.settings.loginAgreement.noPublishedPages') }}
                </p>
                <div v-else class="space-y-3">
                  <div
                    v-for="(doc, index) in form.login_agreement_documents"
                    :key="`${doc.id}-${index}`"
                    class="grid grid-cols-1 gap-3 rounded-lg border border-gray-100 p-3 dark:border-dark-700 md:grid-cols-[1fr,1fr,auto]"
                  >
                    <Select
                      v-model="doc.page_slug"
                      :options="publishedMarkdownPageOptions"
                      @update:modelValue="syncLoginAgreementDocument(index)"
                    />
                    <input
                      v-model="doc.title"
                      type="text"
                      class="input"
                      :placeholder="t('admin.settings.loginAgreement.documentTitle')"
                    />
                    <button
                      type="button"
                      class="btn btn-secondary btn-sm"
                      @click="removeLoginAgreementDocument(index)"
                    >
                      {{ t('admin.settings.loginAgreement.removeDocument') }}
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        </div><!-- /Tab: General -->
</template>

<script setup lang="ts">
import Select from '@/components/common/Select.vue'
import Toggle from '@/components/common/Toggle.vue'
import ImageUpload from '@/components/common/ImageUpload.vue'
import CustomMenuSettingsCard from '@/components/settings/CustomMenuSettingsCard.vue'
import PaymentSettingsCard from '@/components/settings/PaymentSettingsCard.vue'
const props = defineProps<{ ctx: any }>()
const {
  t,
  form,
  globalRealtimeCountdownEnabled,
  savingGlobalRealtimeCountdown,
  saveGlobalRealtimeCountdownPreference,
  loginAgreementModeOptions,
  publishedMarkdownPageOptions,
  syncLoginAgreementDocument,
  addLoginAgreementDocument,
  removeLoginAgreementDocument,
} = props.ctx
</script>

