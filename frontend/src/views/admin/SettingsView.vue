<template>
  <AppLayout>
    <div class="mx-auto max-w-4xl space-y-6">
      <!-- Loading State -->
      <div v-if="loading" class="flex items-center justify-center py-12">
        <div class="h-8 w-8 animate-spin rounded-full border-b-2 border-primary-600"></div>
      </div>

      <!-- Settings Form -->
      <form v-else @submit.prevent="saveSettings" class="space-y-6">
        <!-- Tab Navigation -->
        <div class="sticky top-0 z-10 overflow-x-auto scrollbar-hide">
          <nav class="settings-tabs">
            <button
              v-for="tab in settingsTabs"
              :key="tab.key"
              type="button"
              :class="['settings-tab', activeTab === tab.key && 'settings-tab-active']"
              @click="activateTab(tab.key)"
            >
              <span class="settings-tab-icon">
                <Icon :name="tab.icon" size="sm" />
              </span>
              <span>{{ t(`admin.settings.tabs.${tab.key}`) }}</span>
            </button>
          </nav>
        </div>

        <SettingsSecurityAdminTab v-show="activeTab === 'security'" :ctx="settingsViewContext" />
        <SettingsGatewayMainTab v-show="activeTab === 'gateway'" :ctx="settingsViewContext" />
        <SettingsSecurityAuthTab v-show="activeTab === 'security'" :ctx="settingsViewContext" />
        <SettingsUsersTab v-show="activeTab === 'users'" :ctx="settingsViewContext" />
        <SettingsGatewayExtraTab v-show="activeTab === 'gateway'" :ctx="settingsViewContext" />
        <SettingsGeneralTab v-show="activeTab === 'general'" :ctx="settingsViewContext" />
        <SettingsNotificationTab v-show="activeTab === 'notification'" :ctx="settingsViewContext" />
        <SettingsEmailTab v-show="activeTab === 'email'" :ctx="settingsViewContext" />

        <!-- Save Button -->
        <div class="flex justify-end">
          <button type="submit" :disabled="saving" class="btn btn-primary">
            <svg v-if="saving" class="h-4 w-4 animate-spin" fill="none" viewBox="0 0 24 24">
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
            {{ saving ? t('admin.settings.saving') : t('admin.settings.saveSettings') }}
          </button>
        </div>
      </form>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { adminAPI, userAPI } from '@/api'
import type {
  SystemSettings,
  UpdateSettingsRequest,
  DefaultSubscriptionSetting,
  ContentModerationAPIKeyStatus,
  ContentModerationModelFilterType
} from '@/api/admin/settings'
import type { AdminGroup, CustomMenuItem, LoginAgreementDocument } from '@/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import SettingsEmailTab from './settings/SettingsEmailTab.vue'
import SettingsGatewayExtraTab from './settings/SettingsGatewayExtraTab.vue'
import SettingsGatewayMainTab from './settings/SettingsGatewayMainTab.vue'
import SettingsGeneralTab from './settings/SettingsGeneralTab.vue'
import SettingsNotificationTab from './settings/SettingsNotificationTab.vue'
import SettingsSecurityAdminTab from './settings/SettingsSecurityAdminTab.vue'
import SettingsSecurityAuthTab from './settings/SettingsSecurityAuthTab.vue'
import SettingsUsersTab from './settings/SettingsUsersTab.vue'
import { useAdminApiKeySettings } from './settings/useAdminApiKeySettings'
import { useGatewaySettingsControls } from './settings/useGatewaySettingsControls'
import { useSettingsEmailServices } from './settings/useSettingsEmailServices'
import { useClipboard } from '@/composables/useClipboard'
import { useAppStore } from '@/stores'
import { useAuthStore } from '@/stores/auth'
import { useAdminSettingsStore } from '@/stores/adminSettings'
import { resolveSettingsTab, settingsTabs, type SettingsTab } from './settingsTabs'
import {
  isRegistrationEmailSuffixDomainValid,
  normalizeRegistrationEmailSuffixDomain,
  normalizeRegistrationEmailSuffixDomains,
  parseRegistrationEmailSuffixWhitelistInput
} from '@/utils/registrationEmailPolicy'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const appStore = useAppStore()
const adminSettingsStore = useAdminSettingsStore()
const authStore = useAuthStore()

const activeTab = ref<SettingsTab>(resolveSettingsTab(route.query.tab))
const { copyToClipboard } = useClipboard()

function activateTab(tab: SettingsTab) {
  activeTab.value = tab
  if (resolveSettingsTab(route.query.tab) === tab) return
  router.replace({ query: { ...route.query, tab } }).catch(() => undefined)
}

watch(
  () => route.query.tab,
  (tab) => {
    activeTab.value = resolveSettingsTab(tab)
  }
)

const loading = ref(true)
const saving = ref(false)
const globalRealtimeCountdownEnabled = ref(false)
const savingGlobalRealtimeCountdown = ref(false)
const registrationEmailSuffixWhitelistTags = ref<string[]>([])
const registrationEmailSuffixWhitelistDraft = ref('')
const subscriptionGroups = ref<AdminGroup[]>([])

interface DefaultSubscriptionGroupOption {
  value: number
  label: string
  description: string | null
  platform: AdminGroup['platform']
  subscriptionType: AdminGroup['subscription_type']
  rate: number
  [key: string]: unknown
}

type SettingsForm = SystemSettings & {
  smtp_password: string
  telegram_bot_token: string
  turnstile_secret_key: string
  linuxdo_connect_client_secret: string
  github_oauth_client_secret: string
  google_oauth_client_secret: string
  dingtalk_oauth_client_secret: string
  airwallex_api_key: string
  airwallex_webhook_secret: string
  content_moderation_api_key: string
  delete_content_moderation_api_key_hashes: string[]
}

const form = reactive<SettingsForm>({
  registration_enabled: true,
  email_verify_enabled: false,
  registration_email_suffix_whitelist: [],
  promo_code_enabled: true,
  invitation_code_enabled: false,
  password_reset_enabled: false,
  frontend_url: '',
  totp_enabled: false,
  totp_encryption_key_configured: false,
  default_balance: 0,
  default_concurrency: 1,
  default_subscriptions: [],
  site_name: 'Sub2API',
  site_logo: '',
  site_subtitle: 'Subscription to API Conversion Platform',
  visual_preset_default: 'classic',
  account_airy_white_surface_enabled: false,
  api_base_url: '',
  contact_info: '',
  doc_url: '',
  home_content: '',
  hide_ccs_import_button: false,
  available_channels_enabled: false,
  channel_monitor_enabled: false,
  channel_monitor_default_interval_seconds: 60,
  public_model_catalog_enabled: true,
  affiliate_enabled: false,
  affiliate_transfer_enabled: true,
  affiliate_rebate_on_usage_enabled: true,
  affiliate_rebate_on_topup_enabled: true,
  affiliate_rebate_rate: 20.0,
  affiliate_rebate_freeze_hours: 0,
  affiliate_rebate_duration_days: 0,
  affiliate_rebate_per_invitee_cap: 0,
  affiliate_aff_code_length: 10,
  purchase_subscription_enabled: false,
  purchase_subscription_url: '',
  payment_provider_airwallex_enabled: false,
  payment_provider_airwallex_effective: false,
  airwallex_env: 'demo',
  airwallex_client_id: '',
  airwallex_api_key: '',
  airwallex_api_key_configured: false,
  airwallex_webhook_secret: '',
  airwallex_webhook_secret_configured: false,
  payment_allowed_currencies: ['USD', 'CNY', 'HKD'],
  payment_default_currency: 'USD',
  payment_min_topup_amount: 1,
  payment_max_topup_amount: 5000,
  payment_subscription_plans: [],
  antigravity_user_agent_version: '',
  payment_mobile_force_qrcode_enabled: false,
  codex_oauth_user_agent_mode: 'default',
  codex_oauth_user_agent_override: '',
  openai_allow_claude_code_codex_plugin: false,
  backend_mode_enabled: false,
  maintenance_mode_enabled: false,
  custom_menu_items: [] as CustomMenuItem[],
  login_agreement_enabled: false,
  login_agreement_mode: 'checkbox',
  login_agreement_updated_at: '',
  login_agreement_documents: [] as LoginAgreementDocument[],
  smtp_host: '',
  smtp_port: 587,
  smtp_username: '',
  smtp_password: '',
  smtp_password_configured: false,
  smtp_from_email: '',
  smtp_from_name: '',
  smtp_use_tls: true,
  telegram_chat_id: '',
  telegram_bot_token: '',
  telegram_bot_token_configured: false,
  telegram_bot_token_masked: '',
  // Cloudflare Turnstile
  turnstile_enabled: false,
  turnstile_site_key: '',
  turnstile_secret_key: '',
  turnstile_secret_key_configured: false,
  // LinuxDo Connect OAuth 登录
  linuxdo_connect_enabled: false,
  linuxdo_connect_client_id: '',
  linuxdo_connect_client_secret: '',
  linuxdo_connect_client_secret_configured: false,
  linuxdo_connect_redirect_url: '',
  github_oauth_enabled: false,
  github_oauth_client_id: '',
  github_oauth_client_secret: '',
  github_oauth_client_secret_configured: false,
  github_oauth_redirect_url: '',
  google_oauth_enabled: false,
  google_oauth_client_id: '',
  google_oauth_client_secret: '',
  google_oauth_client_secret_configured: false,
  google_oauth_redirect_url: '',
  dingtalk_oauth_enabled: false,
  dingtalk_oauth_client_id: '',
  dingtalk_oauth_client_secret: '',
  dingtalk_oauth_client_secret_configured: false,
  dingtalk_oauth_redirect_url: '',
  content_moderation_enabled: false,
  content_moderation_provider: 'openai',
  content_moderation_base_url: '',
  content_moderation_api_key: '',
  content_moderation_api_key_configured: false,
  content_moderation_api_key_statuses: [] as ContentModerationAPIKeyStatus[],
  content_moderation_model: '',
  content_moderation_timeout_ms: 1500,
  content_moderation_dedupe_window_seconds: 300,
  content_moderation_fail_open: true,
  content_moderation_keyword_block_enabled: false,
  content_moderation_keywords: [],
  content_moderation_model_filter: {
    type: 'all',
    models: []
  },
  content_moderation_category_thresholds: {},
  // Model fallback
  enable_model_fallback: false,
  fallback_model_anthropic: 'claude-3-5-sonnet-20241022',
  fallback_model_openai: 'gpt-4o',
  fallback_model_gemini: 'gemini-2.5-pro',
  fallback_model_antigravity: 'gemini-2.5-pro',
  // Identity patch (Claude -> Gemini)
  enable_identity_patch: true,
  identity_patch_prompt: '',
  // Ops monitoring (vNext)
  ops_monitoring_enabled: true,
  ops_realtime_monitoring_enabled: true,
  ops_query_mode_default: 'auto',
  ops_metrics_interval_seconds: 60,
  // Gateway forwarding policies
  openai_fast_policy_settings: {
    rules: [
      { service_tier: 'priority', action: 'filter', scope: 'all' },
      { service_tier: 'fast', action: 'filter', scope: 'all' },
      { service_tier: 'flex', action: 'pass', scope: 'all' }
    ]
  },
  enable_anthropic_cache_ttl_1h_injection: false,
  // Claude Code version check
  min_claude_code_version: '',
  max_claude_code_version: '',
  // 分组隔离
  allow_ungrouped_key_scheduling: false,
  delete_content_moderation_api_key_hashes: []
})

const adminApiKeySettings = useAdminApiKeySettings(t, appStore)
const gatewaySettings = useGatewaySettingsControls(t, appStore)
const emailServices = useSettingsEmailServices(t, appStore, form)

const defaultSubscriptionGroupOptions = computed<DefaultSubscriptionGroupOption[]>(() =>
  subscriptionGroups.value.map((group) => ({
    value: group.id,
    label: group.name,
    description: group.description,
    platform: group.platform,
    subscriptionType: group.subscription_type,
    rate: group.rate_multiplier
  }))
)

const contentModerationModelFilterOptions = computed(() => [
  { value: 'all', label: t('admin.settings.moderation.modelFilterAll') },
  { value: 'include', label: t('admin.settings.moderation.modelFilterInclude') },
  { value: 'exclude', label: t('admin.settings.moderation.modelFilterExclude') }
])

const normalizeContentModerationModelNames = (value: string) => {
  const seen = new Set<string>()
  return value
    .split(/[\n,，;；]+/)
    .map((item) => item.trim())
    .filter((item) => {
      if (!item) return false
      const key = item.toLowerCase()
      if (seen.has(key)) return false
      seen.add(key)
      return true
    })
}

const ensureContentModerationModelFilter = () => {
  const filter = form.content_moderation_model_filter
  if (!filter || !['all', 'include', 'exclude'].includes(filter.type)) {
    form.content_moderation_model_filter = { type: 'all', models: [] }
    return
  }
  form.content_moderation_model_filter = {
    type: filter.type as ContentModerationModelFilterType,
    models: normalizeContentModerationModelNames((filter.models || []).join('\n'))
  }
}

const contentModerationThresholdCategories = [
  'hate',
  'hate/threatening',
  'harassment',
  'harassment/threatening',
  'self-harm',
  'self-harm/intent',
  'self-harm/instructions',
  'sexual',
  'sexual/minors',
  'violence',
  'violence/graphic',
  'illicit',
  'illicit/violent'
]

const clampContentModerationThreshold = (value: unknown): number => {
  const parsed = Number(value)
  if (!Number.isFinite(parsed)) return 1
  return Math.min(1, Math.max(0, Math.round(parsed * 100) / 100))
}

const normalizeContentModerationThresholds = (value: Record<string, number> | undefined) => {
  const normalized: Record<string, number> = {}
  for (const category of contentModerationThresholdCategories) {
    normalized[category] = clampContentModerationThreshold(value?.[category] ?? 1)
  }
  return normalized
}

const ensureContentModerationCategoryThresholds = () => {
  form.content_moderation_category_thresholds = normalizeContentModerationThresholds(
    form.content_moderation_category_thresholds
  )
}

const contentModerationKeywordsText = computed({
  get: () => form.content_moderation_keywords.join('\n'),
  set: (value: string) => {
    const seen = new Set<string>()
    form.content_moderation_keywords = value
      .split(/[\n,，;；]+/)
      .map((item) => item.trim())
      .filter((item) => {
        if (!item) return false
        const key = item.toLowerCase()
        if (seen.has(key)) return false
        seen.add(key)
        return true
      })
  }
})

const contentModerationModelFilterModelsText = computed({
  get: () => form.content_moderation_model_filter.models.join('\n'),
  set: (value: string) => {
    form.content_moderation_model_filter.models = normalizeContentModerationModelNames(value)
  }
})

const loginAgreementModeOptions = computed(() => [
  { value: 'checkbox', label: t('admin.settings.loginAgreement.checkboxMode') }
])

const publishedMarkdownPageOptions = computed(() =>
  form.custom_menu_items
    .filter((item) => item.page_mode === 'markdown' && item.page_published && item.page_slug)
    .map((item) => ({
      value: item.page_slug || '',
      label: item.label || item.page_slug || ''
    }))
)

const registrationEmailSuffixWhitelistSeparatorKeys = new Set([' ', ',', '，', 'Enter', 'Tab'])

function removeRegistrationEmailSuffixWhitelistTag(suffix: string) {
  registrationEmailSuffixWhitelistTags.value = registrationEmailSuffixWhitelistTags.value.filter(
    (item) => item !== suffix
  )
}

function addRegistrationEmailSuffixWhitelistTag(raw: string) {
  const suffix = normalizeRegistrationEmailSuffixDomain(raw)
  if (
    !isRegistrationEmailSuffixDomainValid(suffix) ||
    registrationEmailSuffixWhitelistTags.value.includes(suffix)
  ) {
    return
  }
  registrationEmailSuffixWhitelistTags.value = [
    ...registrationEmailSuffixWhitelistTags.value,
    suffix
  ]
}

function commitRegistrationEmailSuffixWhitelistDraft() {
  if (!registrationEmailSuffixWhitelistDraft.value) {
    return
  }
  addRegistrationEmailSuffixWhitelistTag(registrationEmailSuffixWhitelistDraft.value)
  registrationEmailSuffixWhitelistDraft.value = ''
}

function handleRegistrationEmailSuffixWhitelistDraftInput() {
  registrationEmailSuffixWhitelistDraft.value = normalizeRegistrationEmailSuffixDomain(
    registrationEmailSuffixWhitelistDraft.value
  )
}

function handleRegistrationEmailSuffixWhitelistDraftKeydown(event: KeyboardEvent) {
  if (event.isComposing) {
    return
  }

  if (registrationEmailSuffixWhitelistSeparatorKeys.has(event.key)) {
    event.preventDefault()
    commitRegistrationEmailSuffixWhitelistDraft()
    return
  }

  if (
    event.key === 'Backspace' &&
    !registrationEmailSuffixWhitelistDraft.value &&
    registrationEmailSuffixWhitelistTags.value.length > 0
  ) {
    registrationEmailSuffixWhitelistTags.value.pop()
  }
}

function handleRegistrationEmailSuffixWhitelistPaste(event: ClipboardEvent) {
  const text = event.clipboardData?.getData('text') || ''
  if (!text.trim()) {
    return
  }
  event.preventDefault()
  const tokens = parseRegistrationEmailSuffixWhitelistInput(text)
  for (const token of tokens) {
    addRegistrationEmailSuffixWhitelistTag(token)
  }
}

// LinuxDo OAuth redirect URL suggestion
const linuxdoRedirectUrlSuggestion = computed(() => {
  if (typeof window === 'undefined') return ''
  const origin =
    window.location.origin || `${window.location.protocol}//${window.location.host}`
  return `${origin}/api/v1/auth/oauth/linuxdo/callback`
})

async function setAndCopyLinuxdoRedirectUrl() {
  const url = linuxdoRedirectUrlSuggestion.value
  if (!url) return

  form.linuxdo_connect_redirect_url = url
  await copyToClipboard(url, t('admin.settings.linuxdo.redirectUrlSetAndCopied'))
}

async function loadSettings() {
  loading.value = true
  try {
    const settings = await adminAPI.settings.getSettings()
    Object.assign(form, settings)
    ensureContentModerationModelFilter()
    ensureContentModerationCategoryThresholds()
    form.default_subscriptions = Array.isArray(settings.default_subscriptions)
      ? settings.default_subscriptions
          .filter((item) => item.group_id > 0 && item.validity_days > 0)
          .map((item) => ({
            group_id: item.group_id,
            validity_days: item.validity_days
          }))
      : []
    registrationEmailSuffixWhitelistTags.value = normalizeRegistrationEmailSuffixDomains(
      settings.registration_email_suffix_whitelist
    )
    registrationEmailSuffixWhitelistDraft.value = ''
    form.smtp_password = ''
    form.telegram_bot_token = ''
    form.turnstile_secret_key = ''
    form.linuxdo_connect_client_secret = ''
    form.github_oauth_client_secret = ''
    form.google_oauth_client_secret = ''
    form.dingtalk_oauth_client_secret = ''
    form.airwallex_api_key = ''
    form.airwallex_webhook_secret = ''
    form.content_moderation_api_key = ''
    form.delete_content_moderation_api_key_hashes = []
    globalRealtimeCountdownEnabled.value =
      authStore.user?.global_realtime_countdown_enabled === true
  } catch (error: any) {
    appStore.showError(
      t('admin.settings.failedToLoad') + ': ' + (error.message || t('common.unknownError'))
    )
  } finally {
    loading.value = false
  }
}

watch(
  () => authStore.user?.global_realtime_countdown_enabled,
  (enabled) => {
    globalRealtimeCountdownEnabled.value = enabled === true
  },
  { immediate: true }
)

async function loadSubscriptionGroups() {
  try {
    const groups = await adminAPI.groups.getAll()
    subscriptionGroups.value = groups.filter(
      (group) => group.subscription_type === 'subscription' && group.status === 'active'
    )
  } catch (error) {
    console.error('Failed to load subscription groups:', error)
    subscriptionGroups.value = []
  }
}

function addDefaultSubscription() {
  if (subscriptionGroups.value.length === 0) return
  const existing = new Set(form.default_subscriptions.map((item) => item.group_id))
  const candidate = subscriptionGroups.value.find((group) => !existing.has(group.id))
  if (!candidate) return
  form.default_subscriptions.push({
    group_id: candidate.id,
    validity_days: 30
  })
}

function removeDefaultSubscription(index: number) {
  form.default_subscriptions.splice(index, 1)
}

function deleteContentModerationKey(hash: string) {
  if (!hash || form.delete_content_moderation_api_key_hashes.includes(hash)) {
    return
  }
  form.delete_content_moderation_api_key_hashes.push(hash)
  form.content_moderation_api_key_statuses = form.content_moderation_api_key_statuses.filter(
    (item) => item.hash !== hash
  )
}

function syncLoginAgreementDocument(index: number) {
  const doc = form.login_agreement_documents[index]
  if (!doc) return
  const option = publishedMarkdownPageOptions.value.find((item) => item.value === doc.page_slug)
  doc.page_slug = String(option?.value || doc.page_slug || '')
  if (!doc.id) {
    doc.id = doc.page_slug
  }
  if (!doc.title && option?.label) {
    doc.title = String(option.label)
  }
}

function addLoginAgreementDocument() {
  const existing = new Set(form.login_agreement_documents.map((item) => item.page_slug))
  const option = publishedMarkdownPageOptions.value.find((item) => !existing.has(String(item.value)))
  if (!option) return
  form.login_agreement_documents.push({
    id: String(option.value),
    title: String(option.label),
    page_slug: String(option.value)
  })
}

function removeLoginAgreementDocument(index: number) {
  form.login_agreement_documents.splice(index, 1)
}

async function saveSettings() {
  saving.value = true
  try {
    const normalizedDefaultSubscriptions = form.default_subscriptions
      .filter((item) => item.group_id > 0 && item.validity_days > 0)
      .map((item: DefaultSubscriptionSetting) => ({
        group_id: item.group_id,
        validity_days: Math.min(36500, Math.max(1, Math.floor(item.validity_days)))
      }))

    const seenGroupIDs = new Set<number>()
    const duplicateDefaultSubscription = normalizedDefaultSubscriptions.find((item) => {
      if (seenGroupIDs.has(item.group_id)) {
        return true
      }
      seenGroupIDs.add(item.group_id)
      return false
    })
    if (duplicateDefaultSubscription) {
      appStore.showError(
        t('admin.settings.defaults.defaultSubscriptionsDuplicate', {
          groupId: duplicateDefaultSubscription.group_id
        })
      )
      return
    }

    const contentModerationNewKeys = form.content_moderation_api_key
      ? [form.content_moderation_api_key]
      : undefined
    const contentModerationDeletedKeys =
      form.delete_content_moderation_api_key_hashes.length > 0
        ? form.delete_content_moderation_api_key_hashes
        : undefined

    const payload: UpdateSettingsRequest = {
      registration_enabled: form.registration_enabled,
      email_verify_enabled: form.email_verify_enabled,
      registration_email_suffix_whitelist: registrationEmailSuffixWhitelistTags.value.map(
        (suffix) => `@${suffix}`
      ),
      promo_code_enabled: form.promo_code_enabled,
      invitation_code_enabled: form.invitation_code_enabled,
      password_reset_enabled: form.password_reset_enabled,
      totp_enabled: form.totp_enabled,
      default_balance: form.default_balance,
      default_concurrency: form.default_concurrency,
      default_subscriptions: normalizedDefaultSubscriptions,
      site_name: form.site_name,
      site_logo: form.site_logo,
      site_subtitle: form.site_subtitle,
      visual_preset_default: form.visual_preset_default,
      account_airy_white_surface_enabled: form.account_airy_white_surface_enabled,
      api_base_url: form.api_base_url,
      contact_info: form.contact_info,
      doc_url: form.doc_url,
      home_content: form.home_content,
      hide_ccs_import_button: form.hide_ccs_import_button,
      available_channels_enabled: form.available_channels_enabled,
      channel_monitor_enabled: form.channel_monitor_enabled,
      channel_monitor_default_interval_seconds: form.channel_monitor_default_interval_seconds,
      public_model_catalog_enabled: form.public_model_catalog_enabled,
      affiliate_enabled: form.affiliate_enabled,
      affiliate_transfer_enabled: form.affiliate_transfer_enabled,
      affiliate_rebate_on_usage_enabled: form.affiliate_rebate_on_usage_enabled,
      affiliate_rebate_on_topup_enabled: form.affiliate_rebate_on_topup_enabled,
      affiliate_rebate_rate: form.affiliate_rebate_rate,
      affiliate_rebate_freeze_hours: form.affiliate_rebate_freeze_hours,
      affiliate_rebate_duration_days: form.affiliate_rebate_duration_days,
      affiliate_rebate_per_invitee_cap: form.affiliate_rebate_per_invitee_cap,
      affiliate_aff_code_length: form.affiliate_aff_code_length,
      purchase_subscription_enabled: form.purchase_subscription_enabled,
      purchase_subscription_url: form.purchase_subscription_url,
      payment_provider_airwallex_enabled: form.payment_provider_airwallex_enabled,
      airwallex_env: form.airwallex_env,
      airwallex_client_id: form.airwallex_client_id,
      airwallex_api_key: form.airwallex_api_key || undefined,
      airwallex_webhook_secret: form.airwallex_webhook_secret || undefined,
      payment_mobile_force_qrcode_enabled: form.payment_mobile_force_qrcode_enabled,
      payment_allowed_currencies: form.payment_allowed_currencies,
      payment_default_currency: form.payment_default_currency,
      payment_min_topup_amount: form.payment_min_topup_amount,
      payment_max_topup_amount: form.payment_max_topup_amount,
      payment_subscription_plans: form.payment_subscription_plans,
      antigravity_user_agent_version: form.antigravity_user_agent_version,
      codex_oauth_user_agent_mode: form.codex_oauth_user_agent_mode,
      codex_oauth_user_agent_override: form.codex_oauth_user_agent_override,
      openai_allow_claude_code_codex_plugin: form.openai_allow_claude_code_codex_plugin,
      maintenance_mode_enabled: form.maintenance_mode_enabled,
      custom_menu_items: form.custom_menu_items,
      login_agreement_enabled: form.login_agreement_enabled,
      login_agreement_mode: form.login_agreement_mode,
      login_agreement_updated_at: form.login_agreement_updated_at,
      login_agreement_documents: form.login_agreement_documents
        .filter((doc) => doc.page_slug)
        .map((doc) => ({
          id: doc.id || doc.page_slug,
          title: doc.title || doc.page_slug,
          page_slug: doc.page_slug
        })),
      smtp_host: form.smtp_host,
      smtp_port: form.smtp_port,
      smtp_username: form.smtp_username,
      smtp_password: form.smtp_password || undefined,
      smtp_from_email: form.smtp_from_email,
      smtp_from_name: form.smtp_from_name,
      smtp_use_tls: form.smtp_use_tls,
      telegram_chat_id: form.telegram_chat_id,
      telegram_bot_token: form.telegram_bot_token || undefined,
      turnstile_enabled: form.turnstile_enabled,
      turnstile_site_key: form.turnstile_site_key,
      turnstile_secret_key: form.turnstile_secret_key || undefined,
      linuxdo_connect_enabled: form.linuxdo_connect_enabled,
      linuxdo_connect_client_id: form.linuxdo_connect_client_id,
      linuxdo_connect_client_secret: form.linuxdo_connect_client_secret || undefined,
      linuxdo_connect_redirect_url: form.linuxdo_connect_redirect_url,
      github_oauth_enabled: form.github_oauth_enabled,
      github_oauth_client_id: form.github_oauth_client_id,
      github_oauth_client_secret: form.github_oauth_client_secret || undefined,
      github_oauth_redirect_url: form.github_oauth_redirect_url,
      google_oauth_enabled: form.google_oauth_enabled,
      google_oauth_client_id: form.google_oauth_client_id,
      google_oauth_client_secret: form.google_oauth_client_secret || undefined,
      google_oauth_redirect_url: form.google_oauth_redirect_url,
      dingtalk_oauth_enabled: form.dingtalk_oauth_enabled,
      dingtalk_oauth_client_id: form.dingtalk_oauth_client_id,
      dingtalk_oauth_client_secret: form.dingtalk_oauth_client_secret || undefined,
      dingtalk_oauth_redirect_url: form.dingtalk_oauth_redirect_url,
      content_moderation_enabled: form.content_moderation_enabled,
      content_moderation_provider: form.content_moderation_provider,
      content_moderation_base_url: form.content_moderation_base_url,
      content_moderation_api_keys: contentModerationNewKeys,
      content_moderation_api_keys_mode:
        contentModerationNewKeys || contentModerationDeletedKeys ? 'append' : undefined,
      delete_content_moderation_api_key_hashes: contentModerationDeletedKeys,
      content_moderation_model: form.content_moderation_model,
      content_moderation_timeout_ms: form.content_moderation_timeout_ms,
      content_moderation_dedupe_window_seconds: form.content_moderation_dedupe_window_seconds,
      content_moderation_fail_open: form.content_moderation_fail_open,
      content_moderation_keyword_block_enabled: form.content_moderation_keyword_block_enabled,
      content_moderation_keywords: form.content_moderation_keywords,
      content_moderation_model_filter: {
        type: form.content_moderation_model_filter.type,
        models:
          form.content_moderation_model_filter.type === 'all'
            ? []
            : form.content_moderation_model_filter.models
      },
      content_moderation_category_thresholds: normalizeContentModerationThresholds(
        form.content_moderation_category_thresholds
      ),
      enable_model_fallback: form.enable_model_fallback,
      fallback_model_anthropic: form.fallback_model_anthropic,
      fallback_model_openai: form.fallback_model_openai,
      fallback_model_gemini: form.fallback_model_gemini,
      fallback_model_antigravity: form.fallback_model_antigravity,
      enable_identity_patch: form.enable_identity_patch,
      identity_patch_prompt: form.identity_patch_prompt,
      min_claude_code_version: form.min_claude_code_version,
      max_claude_code_version: form.max_claude_code_version,
      allow_ungrouped_key_scheduling: form.allow_ungrouped_key_scheduling,
      openai_fast_policy_settings: form.openai_fast_policy_settings,
      enable_anthropic_cache_ttl_1h_injection: form.enable_anthropic_cache_ttl_1h_injection
    }
    const updated = await adminAPI.settings.updateSettings(payload)
    Object.assign(form, updated)
    ensureContentModerationModelFilter()
    ensureContentModerationCategoryThresholds()
    registrationEmailSuffixWhitelistTags.value = normalizeRegistrationEmailSuffixDomains(
      updated.registration_email_suffix_whitelist
    )
    registrationEmailSuffixWhitelistDraft.value = ''
    form.smtp_password = ''
    form.telegram_bot_token = ''
    form.turnstile_secret_key = ''
    form.linuxdo_connect_client_secret = ''
    form.github_oauth_client_secret = ''
    form.google_oauth_client_secret = ''
    form.dingtalk_oauth_client_secret = ''
    form.content_moderation_api_key = ''
    form.delete_content_moderation_api_key_hashes = []
    // Refresh cached settings so sidebar/header update immediately
    await appStore.fetchPublicSettings(true)
    await adminSettingsStore.fetch(true)
    appStore.showSuccess(t('admin.settings.settingsSaved'))
  } catch (error: any) {
    appStore.showError(
      t('admin.settings.failedToSave') + ': ' + (error.message || t('common.unknownError'))
    )
  } finally {
    saving.value = false
  }
}

async function saveGlobalRealtimeCountdownPreference() {
  savingGlobalRealtimeCountdown.value = true
  try {
    const updatedUser = await userAPI.updateProfile({
      global_realtime_countdown_enabled: globalRealtimeCountdownEnabled.value,
    })
    authStore.setCurrentUser(updatedUser)
    appStore.showSuccess(t('admin.settings.realtimeCountdown.saved'))
  } catch (error: any) {
    globalRealtimeCountdownEnabled.value =
      authStore.user?.global_realtime_countdown_enabled === true
    appStore.showError(
      t('admin.settings.realtimeCountdown.saveFailed') +
        ': ' +
        (error.message || t('common.unknownError'))
    )
  } finally {
    savingGlobalRealtimeCountdown.value = false
  }
}

const settingsViewContext = {
  t,
  form,
  ...adminApiKeySettings,
  ...gatewaySettings,
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
  contentModerationThresholdCategories,
  deleteContentModerationKey,
  subscriptionGroups,
  defaultSubscriptionGroupOptions,
  addDefaultSubscription,
  removeDefaultSubscription,
  globalRealtimeCountdownEnabled,
  savingGlobalRealtimeCountdown,
  saveGlobalRealtimeCountdownPreference,
  loginAgreementModeOptions,
  publishedMarkdownPageOptions,
  syncLoginAgreementDocument,
  addLoginAgreementDocument,
  removeLoginAgreementDocument,
  ...emailServices
}

onMounted(() => {
  loadSettings()
  loadSubscriptionGroups()
  adminApiKeySettings.loadAdminApiKey()
  gatewaySettings.loadOverloadCooldownSettings()
  gatewaySettings.loadStreamTimeoutSettings()
  gatewaySettings.loadRectifierSettings()
  gatewaySettings.loadBetaPolicySettings()
  emailServices.loadEmailTemplates()
})
</script>

<style scoped>
.default-sub-group-select :deep(.select-trigger) {
  @apply h-[42px];
}

.default-sub-delete-btn {
  @apply h-[42px];
}

/* ============ Settings Tab Navigation ============ */
.settings-tabs {
  @apply inline-flex min-w-full gap-1 rounded-2xl
         border border-gray-100 bg-white/80 p-1.5 backdrop-blur-sm
         dark:border-dark-700/50 dark:bg-dark-800/80;
  box-shadow: 0 1px 3px rgb(0 0 0 / 0.04), 0 1px 2px rgb(0 0 0 / 0.02);
}

@media (min-width: 640px) {
  .settings-tabs {
    @apply flex;
  }
}

.settings-tab {
  @apply relative flex flex-1 items-center justify-center gap-2
         whitespace-nowrap rounded-xl px-4 py-2.5
         text-sm font-medium
         text-gray-500 dark:text-dark-400
         transition-all duration-200 ease-out;
}

.settings-tab:hover:not(.settings-tab-active) {
  @apply text-gray-700 dark:text-gray-300;
  background: rgb(0 0 0 / 0.03);
}

:root.dark .settings-tab:hover:not(.settings-tab-active) {
  background: rgb(255 255 255 / 0.04);
}

.settings-tab-active {
  @apply text-primary-600 dark:text-primary-400;
  background: linear-gradient(135deg, rgba(20, 184, 166, 0.08), rgba(20, 184, 166, 0.03));
  box-shadow: 0 1px 2px rgba(20, 184, 166, 0.1);
}

:root.dark .settings-tab-active {
  background: linear-gradient(135deg, rgba(45, 212, 191, 0.12), rgba(45, 212, 191, 0.05));
  box-shadow: 0 1px 3px rgb(0 0 0 / 0.25);
}

.settings-tab-icon {
  @apply flex h-7 w-7 items-center justify-center rounded-lg
         transition-all duration-200;
}

.settings-tab-active .settings-tab-icon {
  @apply bg-primary-500/15 text-primary-600
         dark:bg-primary-400/15 dark:text-primary-400;
}
</style>
