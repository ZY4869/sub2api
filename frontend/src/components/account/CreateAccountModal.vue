<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.createAccount')"
    width="wide"
    @close="handleClose"
  >
    <!-- Step Indicator for OAuth accounts -->
    <div v-if="isOAuthFlow" class="mb-6 flex items-center justify-center">
      <div class="flex items-center space-x-4">
        <div class="flex items-center">
          <div
            :class="[
              'flex h-8 w-8 items-center justify-center rounded-full text-sm font-semibold',
              step >= 1 ? 'bg-primary-500 text-white' : 'bg-gray-200 text-gray-500 dark:bg-dark-600'
            ]"
          >
            1
          </div>
          <span class="ml-2 text-sm font-medium text-gray-700 dark:text-gray-300">{{
            t('admin.accounts.oauth.authMethod')
          }}</span>
        </div>
        <div class="h-0.5 w-8 bg-gray-300 dark:bg-dark-600" />
        <div class="flex items-center">
          <div
            :class="[
              'flex h-8 w-8 items-center justify-center rounded-full text-sm font-semibold',
              step >= 2 ? 'bg-primary-500 text-white' : 'bg-gray-200 text-gray-500 dark:bg-dark-600'
            ]"
          >
            2
          </div>
          <span class="ml-2 text-sm font-medium text-gray-700 dark:text-gray-300">{{
            oauthStepTitle
          }}</span>
        </div>
      </div>
    </div>

    <!-- Step 1: Basic Info -->
    <form
      v-if="step === 1"
      id="create-account-form"
      @submit.prevent="handleSubmit"
      class="space-y-5"
    >
      <div>
        <label class="input-label">{{ t('admin.accounts.accountName') }}</label>
        <input
          v-model="form.name"
          type="text"
          required
          class="input"
          :placeholder="t('admin.accounts.enterAccountName')"
          data-tour="account-form-name"
        />
      </div>
      <div>
        <label class="input-label">{{ t('admin.accounts.notes') }}</label>
        <textarea
          v-model="form.notes"
          rows="3"
          class="input"
          :placeholder="t('admin.accounts.notesPlaceholder')"
        ></textarea>
        <p class="input-hint">{{ t('admin.accounts.notesHint') }}</p>
      </div>

      <AccountCreatePlatformSelector v-model:platform="form.platform" />

      <AccountCreatePlatformTypeEditor
        v-model:platform="form.platform"
        v-model:account-category="accountCategory"
        v-model:add-method="addMethod"
        v-model:sora-account-type="soraAccountType"
        v-model:antigravity-account-type="antigravityAccountType"
        v-model:gemini-o-auth-type="geminiOAuthType"
        v-model:show-advanced="showAdvancedOAuth"
        v-model:gemini-tier-google-one="geminiTierGoogleOne"
        v-model:gemini-tier-gcp="geminiTierGcp"
        v-model:gemini-tier-ai-studio="geminiTierAIStudio"
        v-model:upstream-base-url="upstreamBaseUrl"
        v-model:upstream-api-key="upstreamApiKey"
        :ai-studio-o-auth-enabled="geminiAIStudioOAuthEnabled"
        :api-key-help-link="geminiHelpLinks.apiKey"
        :gcp-project-help-link="geminiHelpLinks.gcpProject"
        @open-gemini-help="showGeminiHelpDialog = true"
      />

      <!-- Antigravity model restriction (applies to OAuth + Upstream) -->
      <AccountAntigravityModelMappingEditor
        v-if="form.platform === 'antigravity'"
        :model-mappings="antigravityModelMappings"
        :preset-mappings="antigravityPresetMappings"
        :get-mapping-key="getAntigravityModelMappingKey"
        @add-mapping="addAntigravityModelMapping"
        @remove-mapping="removeAntigravityModelMapping"
        @add-preset="addAntigravityPresetMapping($event.from, $event.to)"
      />

      <!-- API Key input (only for apikey type, excluding Antigravity which has its own fields) -->
      <div v-if="form.type === 'apikey' && form.platform !== 'antigravity'" class="space-y-4">
        <AccountApiKeyBasicSettingsEditor
          v-model:base-url="apiKeyBaseUrl"
          v-model:api-key="apiKeyValue"
          v-model:model-scope-mode="modelRestrictionMode"
          v-model:allowed-models="allowedModels"
          v-model:gemini-tier-ai-studio="geminiTierAIStudio"
          :platform="form.platform"
          mode="create"
          :model-scope-disabled="isOpenAIModelRestrictionDisabled"
          :model-mappings="modelMappings"
          :preset-mappings="presetMappings"
          :get-mapping-key="getModelMappingKey"
          :show-gemini-tier="form.platform === 'gemini'"
          @add-mapping="addModelMapping"
          @remove-mapping="removeModelMapping"
          @add-preset="addPresetMapping($event.from, $event.to)"
        />

        <AccountPoolModeEditor
          v-model:state="poolModeState"
          :default-retry-count="DEFAULT_POOL_MODE_RETRY_COUNT"
          :max-retry-count="MAX_POOL_MODE_RETRY_COUNT"
        />

        <AccountCustomErrorCodesEditor
          v-model:state="customErrorCodesState"
          :error-code-options="commonErrorCodes"
          :show-error="showFormError"
          :show-info="showFormInfo"
        />

      </div>

      <div v-if="form.type === 'apikey'" class="border-t border-gray-200 pt-4 dark:border-dark-600 space-y-4">
        <div class="mb-3">
          <h3 class="input-label mb-0 text-base font-semibold">{{ t('admin.accounts.quotaLimit') }}</h3>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.accounts.quotaLimitHint') }}
          </p>
        </div>
        <QuotaLimitCard
          :totalLimit="editQuotaLimit"
          :dailyLimit="editQuotaDailyLimit"
          :weeklyLimit="editQuotaWeeklyLimit"
          @update:totalLimit="editQuotaLimit = $event"
          @update:dailyLimit="editQuotaDailyLimit = $event"
          @update:weeklyLimit="editQuotaWeeklyLimit = $event"
        />
      </div>
      <AccountModelScopeEditor
        v-if="form.platform === 'openai' && accountCategory === 'oauth-based'"
        :disabled="isOpenAIModelRestrictionDisabled"
        :platform="form.platform"
        :mode="modelRestrictionMode"
        :allowed-models="allowedModels"
        :model-mappings="modelMappings"
        :preset-mappings="presetMappings"
        :get-mapping-key="getModelMappingKey"
        @update:mode="modelRestrictionMode = $event"
        @update:allowedModels="allowedModels = $event"
        @add-mapping="addModelMapping"
        @remove-mapping="removeModelMapping"
        @add-preset="addPresetMapping($event.from, $event.to)"
      />

      <AccountTempUnschedRulesEditor
        :enabled="tempUnschedEnabled"
        :rules="tempUnschedRules"
        :presets="tempUnschedPresets"
        :get-rule-key="getTempUnschedRuleKey"
        @update:enabled="tempUnschedEnabled = $event"
        @add-rule="addTempUnschedRule"
        @remove-rule="removeTempUnschedRule"
        @move-rule="moveTempUnschedRule($event.index, $event.direction)"
      />

      <!-- Intercept Warmup Requests (Anthropic/Antigravity) -->
      <div
        v-if="form.platform === 'anthropic' || form.platform === 'antigravity'"
        class="border-t border-gray-200 pt-4 dark:border-dark-600"
      >
        <div class="flex items-center justify-between">
          <div>
            <label class="input-label mb-0">{{
              t('admin.accounts.interceptWarmupRequests')
            }}</label>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.accounts.interceptWarmupRequestsDesc') }}
            </p>
          </div>
          <button
            type="button"
            @click="interceptWarmupRequests = !interceptWarmupRequests"
            :class="[
              'relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2',
              interceptWarmupRequests ? 'bg-primary-600' : 'bg-gray-200 dark:bg-dark-600'
            ]"
          >
            <span
              :class="[
                'pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
                interceptWarmupRequests ? 'translate-x-5' : 'translate-x-0'
              ]"
            />
          </button>
        </div>
      </div>

      <AccountQuotaControlEditor
        v-if="form.platform === 'anthropic' && accountCategory === 'oauth-based'"
        v-model:state="quotaControlState"
        :umq-mode-options="umqModeOptions"
      />

      <AccountRuntimeSettingsEditor
        v-model:proxy-id="form.proxy_id"
        v-model:concurrency="form.concurrency"
        v-model:load-factor="form.load_factor"
        v-model:priority="form.priority"
        v-model:rate-multiplier="form.rate_multiplier"
        v-model:expires-at-input="expiresAtInput"
        :proxies="proxies"
      />

      <AccountGatewaySettingsEditor
        :show-open-ai-passthrough="form.platform === 'openai'"
        :open-ai-passthrough-enabled="openaiPassthroughEnabled"
        :show-open-ai-ws-mode="form.platform === 'openai' && (accountCategory === 'oauth-based' || accountCategory === 'apikey')"
        :open-ai-ws-mode="openaiResponsesWebSocketV2Mode"
        :open-ai-ws-mode-options="openAIWSModeOptions"
        :open-ai-ws-mode-concurrency-hint-key="openAIWSModeConcurrencyHintKey"
        :show-anthropic-passthrough="form.platform === 'anthropic' && accountCategory === 'apikey'"
        :anthropic-passthrough-enabled="anthropicPassthroughEnabled"
        :show-codex-cli-only="form.platform === 'openai' && accountCategory === 'oauth-based'"
        :codex-cli-only-enabled="codexCLIOnlyEnabled"
        @update:open-ai-passthrough-enabled="openaiPassthroughEnabled = $event"
        @update:open-ai-ws-mode="openaiResponsesWebSocketV2Mode = $event"
        @update:anthropic-passthrough-enabled="anthropicPassthroughEnabled = $event"
        @update:codex-cli-only-enabled="codexCLIOnlyEnabled = $event"
      />

      <AccountAutoPauseToggle v-model:enabled="autoPauseOnExpired" />

      <AccountGroupSettingsEditor
        v-model:group-ids="form.group_ids"
        v-model:mixed-scheduling="mixedScheduling"
        :groups="groups"
        :platform="form.platform"
        :simple-mode="authStore.isSimpleMode"
        :show-mixed-scheduling="form.platform === 'antigravity'"
      />

    </form>

    <AccountCreateOAuthStep
      v-else
      ref="oauthFlowRef"
      :add-method="form.platform === 'anthropic' ? addMethod : 'oauth'"
      :auth-url="currentAuthUrl"
      :session-id="currentSessionId"
      :loading="currentOAuthLoading"
      :error="currentOAuthError"
      :show-help="form.platform === 'anthropic'"
      :show-proxy-warning="form.platform !== 'openai' && form.platform !== 'sora' && !!form.proxy_id"
      :allow-multiple="form.platform === 'anthropic'"
      :show-cookie-option="form.platform === 'anthropic'"
      :show-refresh-token-option="form.platform === 'openai' || form.platform === 'sora' || form.platform === 'antigravity'"
      :show-session-token-option="form.platform === 'sora'"
      :show-access-token-option="form.platform === 'sora'"
      :platform="form.platform"
      :show-project-id="geminiOAuthType === 'code_assist'"
      @generate-url="handleGenerateUrl"
      @cookie-auth="handleCookieAuth"
      @validate-refresh-token="handleValidateRefreshToken"
      @validate-session-token="handleValidateSessionToken"
      @import-access-token="handleImportAccessToken"
    />

    <template #footer>
      <AccountCreateFooterActions
        v-model:auto-import-models="autoImportModels"
        :step="step"
        :submitting="submitting"
        :is-o-auth-flow="isOAuthFlow"
        :is-manual-input-method="isManualInputMethod"
        :current-o-auth-loading="currentOAuthLoading"
        :can-exchange-code="canExchangeCode"
        @close="handleClose"
        @back="goBackToBasicInfo"
        @exchange-code="handleExchangeCode"
      />
    </template>
  </BaseDialog>

  <AccountGeminiHelpDialog :show="showGeminiHelpDialog" @close="showGeminiHelpDialog = false" />

  <AccountMixedChannelWarningDialog
    :show="showMixedChannelWarning"
    :message="mixedChannelWarningMessageText"
    @confirm="handleMixedChannelConfirm"
    @cancel="handleMixedChannelCancel"
  />
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { useModelInventoryStore } from '@/stores'
import { invalidateModelRegistry } from '@/stores/modelRegistry'
import {
  getPresetMappingsByPlatform,
  getModelsByPlatform,
  commonErrorCodes,
  buildModelMappingObject,
  fetchAntigravityDefaultMappings
} from '@/composables/useModelWhitelist'
import { buildAccountModelScopeExtra } from '@/utils/accountModelScope'
import { useAuthStore } from '@/stores/auth'
import { adminAPI } from '@/api/admin'
import type { AccountModelImportResult } from '@/api/admin/accounts'
import {
  useAccountOAuth,
  type AddMethod
} from '@/composables/useAccountOAuth'
import { useOpenAIOAuth } from '@/composables/useOpenAIOAuth'
import { useGeminiOAuth } from '@/composables/useGeminiOAuth'
import { useAntigravityOAuth } from '@/composables/useAntigravityOAuth'
import { useAnthropicQuotaControl } from '@/composables/useAnthropicQuotaControl'
import { useAccountMixedChannelRisk } from '@/composables/useAccountMixedChannelRisk'
import { useAccountTempUnschedRules } from '@/composables/useAccountTempUnschedRules'
import type {
  Proxy,
  AdminGroup,
  AccountPlatform,
  AccountType,
  CreateAccountRequest,
  Account
} from '@/types'
import BaseDialog from '@/components/common/BaseDialog.vue'
import AccountApiKeyBasicSettingsEditor from '@/components/account/AccountApiKeyBasicSettingsEditor.vue'
import AccountAntigravityModelMappingEditor from '@/components/account/AccountAntigravityModelMappingEditor.vue'
import AccountAutoPauseToggle from '@/components/account/AccountAutoPauseToggle.vue'
import AccountCreateFooterActions from '@/components/account/AccountCreateFooterActions.vue'
import AccountCreateOAuthStep from '@/components/account/AccountCreateOAuthStep.vue'
import AccountCreatePlatformSelector from '@/components/account/AccountCreatePlatformSelector.vue'
import AccountCreatePlatformTypeEditor from '@/components/account/AccountCreatePlatformTypeEditor.vue'
import AccountCustomErrorCodesEditor from '@/components/account/AccountCustomErrorCodesEditor.vue'
import AccountGatewaySettingsEditor from '@/components/account/AccountGatewaySettingsEditor.vue'
import AccountGeminiHelpDialog from '@/components/account/AccountGeminiHelpDialog.vue'
import AccountGroupSettingsEditor from '@/components/account/AccountGroupSettingsEditor.vue'
import AccountMixedChannelWarningDialog from '@/components/account/AccountMixedChannelWarningDialog.vue'
import AccountModelScopeEditor from '@/components/account/AccountModelScopeEditor.vue'
import AccountPoolModeEditor from '@/components/account/AccountPoolModeEditor.vue'
import AccountQuotaControlEditor from '@/components/account/AccountQuotaControlEditor.vue'
import AccountRuntimeSettingsEditor from '@/components/account/AccountRuntimeSettingsEditor.vue'
import AccountTempUnschedRulesEditor from '@/components/account/AccountTempUnschedRulesEditor.vue'
import QuotaLimitCard from '@/components/account/QuotaLimitCard.vue'
import { applyInterceptWarmup } from '@/components/account/credentialsBuilder'
import {
  buildAccountModelImportToastPayload,
  extractSyncableRegistryModels,
  mergeAccountModelImportResults,
  resolveAccountModelImportErrorMessage,
  shouldInvalidateModelInventory
} from '@/utils/accountModelImport'
import { formatDateTimeLocalInput, parseDateTimeLocalInput } from '@/utils/format'
import { createStableObjectKeyResolver } from '@/utils/stableObjectKey'
import {
  DEFAULT_POOL_MODE_RETRY_COUNT,
  MAX_POOL_MODE_RETRY_COUNT,
  normalizePoolModeRetryCount,
  type ModelMapping
} from '@/utils/accountFormShared'
import {
  createDefaultAccountCustomErrorCodesState,
  createDefaultAccountPoolModeState
} from '@/utils/accountApiKeyAdvancedSettings'
import { resolveAccountApiKeyDefaultBaseUrl } from '@/utils/accountApiKeyBasicSettings'
import {
  OPENAI_WS_MODE_OFF,
  OPENAI_WS_MODE_PASSTHROUGH,
  isOpenAIWSModeEnabled,
  resolveOpenAIWSModeConcurrencyHintKey,
  type OpenAIWSMode
} from '@/utils/openaiWsMode'
import type { OAuthFlowExposed } from './oauthFlow.types'

const { t } = useI18n()
const authStore = useAuthStore()
const modelInventoryStore = useModelInventoryStore()

const oauthStepTitle = computed(() => {
  if (form.platform === 'openai' || form.platform === 'sora') return t('admin.accounts.oauth.openai.title')
  if (form.platform === 'gemini') return t('admin.accounts.oauth.gemini.title')
  if (form.platform === 'antigravity') return t('admin.accounts.oauth.antigravity.title')
  return t('admin.accounts.oauth.title')
})

interface Props {
  show: boolean
  proxies: Proxy[]
  groups: AdminGroup[]
}

const props = defineProps<Props>()
const emit = defineEmits<{
  close: []
  created: []
  'models-imported': [result: AccountModelImportResult]
}>()

const appStore = useAppStore()
const pendingImportedModelsResult = ref<AccountModelImportResult | null>(null)
const showFormError = (message: string) => appStore.showError(message)
const showFormInfo = (message: string) => appStore.showInfo(message)

// OAuth composables
const oauth = useAccountOAuth() // For Anthropic OAuth
const openaiOAuth = useOpenAIOAuth({ platform: 'openai' }) // For OpenAI OAuth
const soraOAuth = useOpenAIOAuth({ platform: 'sora' }) // For Sora OAuth
const geminiOAuth = useGeminiOAuth() // For Gemini OAuth
const antigravityOAuth = useAntigravityOAuth() // For Antigravity OAuth
const activeOpenAIOAuth = computed(() => (form.platform === 'sora' ? soraOAuth : openaiOAuth))

// Computed: current OAuth state for template binding
const currentAuthUrl = computed(() => {
  if (form.platform === 'openai' || form.platform === 'sora') return activeOpenAIOAuth.value.authUrl.value
  if (form.platform === 'gemini') return geminiOAuth.authUrl.value
  if (form.platform === 'antigravity') return antigravityOAuth.authUrl.value
  return oauth.authUrl.value
})

const currentSessionId = computed(() => {
  if (form.platform === 'openai' || form.platform === 'sora') return activeOpenAIOAuth.value.sessionId.value
  if (form.platform === 'gemini') return geminiOAuth.sessionId.value
  if (form.platform === 'antigravity') return antigravityOAuth.sessionId.value
  return oauth.sessionId.value
})

const currentOAuthLoading = computed(() => {
  if (form.platform === 'openai' || form.platform === 'sora') return activeOpenAIOAuth.value.loading.value
  if (form.platform === 'gemini') return geminiOAuth.loading.value
  if (form.platform === 'antigravity') return antigravityOAuth.loading.value
  return oauth.loading.value
})

const currentOAuthError = computed(() => {
  if (form.platform === 'openai' || form.platform === 'sora') return activeOpenAIOAuth.value.error.value
  if (form.platform === 'gemini') return geminiOAuth.error.value
  if (form.platform === 'antigravity') return antigravityOAuth.error.value
  return oauth.error.value
})

// Refs
const oauthFlowRef = ref<OAuthFlowExposed | null>(null)

// State
const step = ref(1)
const submitting = ref(false)
const autoImportModels = ref(false)
const accountCategory = ref<'oauth-based' | 'apikey'>('oauth-based') // UI selection for account category
const addMethod = ref<AddMethod>('oauth') // For oauth-based: 'oauth' or 'setup-token'
const apiKeyBaseUrl = ref(resolveAccountApiKeyDefaultBaseUrl('anthropic'))
const apiKeyValue = ref('')
const editQuotaLimit = ref<number | null>(null)
const editQuotaDailyLimit = ref<number | null>(null)
const editQuotaWeeklyLimit = ref<number | null>(null)
const modelMappings = ref<ModelMapping[]>([])
const modelRestrictionMode = ref<'whitelist' | 'mapping'>('whitelist')
const allowedModels = ref<string[]>([])
const poolModeState = reactive(createDefaultAccountPoolModeState(DEFAULT_POOL_MODE_RETRY_COUNT))
const customErrorCodesState = reactive(createDefaultAccountCustomErrorCodesState())
const interceptWarmupRequests = ref(false)
const autoPauseOnExpired = ref(true)
const openaiPassthroughEnabled = ref(false)
const openaiOAuthResponsesWebSocketV2Mode = ref<OpenAIWSMode>(OPENAI_WS_MODE_OFF)
const openaiAPIKeyResponsesWebSocketV2Mode = ref<OpenAIWSMode>(OPENAI_WS_MODE_OFF)
const codexCLIOnlyEnabled = ref(false)
const anthropicPassthroughEnabled = ref(false)
const mixedScheduling = ref(false) // For antigravity accounts: enable mixed scheduling
const antigravityAccountType = ref<'oauth' | 'upstream'>('oauth') // For antigravity: oauth or upstream
const soraAccountType = ref<'oauth' | 'apikey'>('oauth') // For sora: oauth or apikey (upstream)
const upstreamBaseUrl = ref('') // For upstream type: base URL
const upstreamApiKey = ref('') // For upstream type: API key
const antigravityModelMappings = ref<ModelMapping[]>([])
const antigravityPresetMappings = computed(() => getPresetMappingsByPlatform('antigravity'))
const getModelMappingKey = createStableObjectKeyResolver<ModelMapping>('create-model-mapping')
const getAntigravityModelMappingKey = createStableObjectKeyResolver<ModelMapping>('create-antigravity-model-mapping')
const geminiOAuthType = ref<'code_assist' | 'google_one' | 'ai_studio'>('google_one')
const geminiAIStudioOAuthEnabled = ref(false)

const showAdvancedOAuth = ref(false)
const showGeminiHelpDialog = ref(false)
const quotaControl = useAnthropicQuotaControl()
const quotaControlState = quotaControl.state
const umqModeOptions = quotaControl.umqModeOptions

// Gemini tier selection (used as fallback when auto-detection is unavailable/fails)
const geminiTierGoogleOne = ref<'google_one_free' | 'google_ai_pro' | 'google_ai_ultra'>('google_one_free')
const geminiTierGcp = ref<'gcp_standard' | 'gcp_enterprise'>('gcp_standard')
const geminiTierAIStudio = ref<'aistudio_free' | 'aistudio_paid'>('aistudio_free')

const geminiSelectedTier = computed(() => {
  if (form.platform !== 'gemini') return ''
  if (accountCategory.value === 'apikey') return geminiTierAIStudio.value
  switch (geminiOAuthType.value) {
    case 'google_one':
      return geminiTierGoogleOne.value
    case 'code_assist':
      return geminiTierGcp.value
    default:
      return geminiTierAIStudio.value
  }
})

const openAIWSModeOptions = computed(() => [
  { value: OPENAI_WS_MODE_OFF, label: t('admin.accounts.openai.wsModeOff') },
  { value: OPENAI_WS_MODE_PASSTHROUGH, label: t('admin.accounts.openai.wsModePassthrough') }
])

const openaiResponsesWebSocketV2Mode = computed({
  get: () => {
    if (form.platform === 'openai' && accountCategory.value === 'apikey') {
      return openaiAPIKeyResponsesWebSocketV2Mode.value
    }
    return openaiOAuthResponsesWebSocketV2Mode.value
  },
  set: (mode: OpenAIWSMode) => {
    if (form.platform === 'openai' && accountCategory.value === 'apikey') {
      openaiAPIKeyResponsesWebSocketV2Mode.value = mode
      return
    }
    openaiOAuthResponsesWebSocketV2Mode.value = mode
  }
})

const openAIWSModeConcurrencyHintKey = computed(() =>
  resolveOpenAIWSModeConcurrencyHintKey(openaiResponsesWebSocketV2Mode.value)
)

const isOpenAIModelRestrictionDisabled = computed(() =>
  form.platform === 'openai' && openaiPassthroughEnabled.value
)

const geminiHelpLinks = {
  apiKey: 'https://aistudio.google.com/app/apikey',
  gcpProject: 'https://console.cloud.google.com/welcome/new'
}

// Computed: current preset mappings based on platform
const presetMappings = computed(() => getPresetMappingsByPlatform(form.platform))

const form = reactive({
  name: '',
  notes: '',
  platform: 'anthropic' as AccountPlatform,
  type: 'oauth' as AccountType, // Will be 'oauth', 'setup-token', or 'apikey'
  credentials: {} as Record<string, unknown>,
  proxy_id: null as number | null,
  concurrency: 10,
  load_factor: null as number | null,
  priority: 1,
  rate_multiplier: 1,
  group_ids: [] as number[],
  expires_at: null as number | null
})

const {
  enabled: tempUnschedEnabled,
  rules: tempUnschedRules,
  presets: tempUnschedPresets,
  getRuleKey: getTempUnschedRuleKey,
  addRule: addTempUnschedRule,
  removeRule: removeTempUnschedRule,
  moveRule: moveTempUnschedRule,
  buildRulesPayload: buildTempUnschedPayload,
  applyToCredentials: applyTempUnschedConfig,
  reset: resetTempUnschedRules
} = useAccountTempUnschedRules({
  keyPrefix: 'create',
  invalidMessage: () => t('admin.accounts.tempUnschedulable.rulesInvalid'),
  showError: showFormError,
  t: (key) => t(key)
})

const {
  showWarning: showMixedChannelWarning,
  warningMessageText: mixedChannelWarningMessageText,
  openDialog: openMixedChannelDialog,
  withConfirmFlag,
  ensureConfirmed: ensureMixedChannelConfirmed,
  handleConfirm: handleMixedChannelConfirm,
  handleCancel: handleMixedChannelCancel,
  reset: resetMixedChannelRisk,
  requiresCheck: requiresMixedChannelCheck
} = useAccountMixedChannelRisk({
  currentPlatform: () => form.platform,
  buildCheckPayload: () => ({
    platform: form.platform,
    group_ids: form.group_ids
  }),
  buildWarningText: (details) => t('admin.accounts.mixedChannelWarning', { ...details }),
  fallbackMessage: () => t('admin.accounts.failedToCreate'),
  showError: showFormError
})

// Helper to check if current type needs OAuth flow
const isOAuthFlow = computed(() => {
  if (form.platform === 'antigravity' && antigravityAccountType.value === 'upstream') {
    return false
  }
  return accountCategory.value === 'oauth-based'
})

const isManualInputMethod = computed(() => {
  return oauthFlowRef.value?.inputMethod === 'manual'
})

const expiresAtInput = computed({
  get: () => formatDateTimeLocal(form.expires_at),
  set: (value: string) => {
    form.expires_at = parseDateTimeLocal(value)
  }
})

const canExchangeCode = computed(() => {
  const authCode = oauthFlowRef.value?.authCode || ''
  if (form.platform === 'openai' || form.platform === 'sora') {
    return Boolean(authCode.trim() && activeOpenAIOAuth.value.sessionId.value && !activeOpenAIOAuth.value.loading.value)
  }
  if (form.platform === 'gemini') {
    return Boolean(authCode.trim() && geminiOAuth.sessionId.value && !geminiOAuth.loading.value)
  }
  if (form.platform === 'antigravity') {
    return Boolean(authCode.trim() && antigravityOAuth.sessionId.value && !antigravityOAuth.loading.value)
  }
  return Boolean(authCode.trim() && oauth.sessionId.value && !oauth.loading.value)
})

const loadAntigravityDefaultMappings = async () => {
  antigravityModelMappings.value = [...await fetchAntigravityDefaultMappings()]
}

// Watchers
watch(
  () => props.show,
  (newVal) => {
    if (newVal) {
      // Modal opened - fill related models
      allowedModels.value = [...getModelsByPlatform(form.platform, 'whitelist')]
      if (form.platform === 'antigravity') {
        loadAntigravityDefaultMappings()
      } else {
        antigravityModelMappings.value = []
      }
    } else {
      resetForm()
    }
  }
)

// Sync form.type based on accountCategory, addMethod, and platform-specific type
watch(
  [accountCategory, addMethod, antigravityAccountType, soraAccountType],
  ([category, method, agType, soraType]) => {
    if (form.platform === 'antigravity' && agType === 'upstream') {
      form.type = 'apikey'
      return
    }
    if (form.platform === 'sora' && soraType === 'apikey') {
      form.type = 'apikey'
      return
    }
    if (category === 'oauth-based') {
      form.type = method as AccountType // 'oauth' or 'setup-token'
    } else {
      form.type = 'apikey'
    }
  },
  { immediate: true }
)

// Reset platform-specific settings when platform changes
watch(
  () => form.platform,
  (newPlatform) => {
    apiKeyBaseUrl.value = resolveAccountApiKeyDefaultBaseUrl(newPlatform)
    allowedModels.value = []
    modelMappings.value = []
    if (newPlatform === 'antigravity') {
      loadAntigravityDefaultMappings()
      accountCategory.value = 'oauth-based'
      antigravityAccountType.value = 'oauth'
    } else {
      antigravityModelMappings.value = []
    }
    // Reset Anthropic/Antigravity-specific settings when switching to other platforms
    if (newPlatform !== 'anthropic' && newPlatform !== 'antigravity') {
      interceptWarmupRequests.value = false
    }
    if (newPlatform === 'sora') {
      accountCategory.value = 'oauth-based'
      addMethod.value = 'oauth'
      form.type = 'oauth'
      soraAccountType.value = 'oauth'
    }
    if (newPlatform !== 'openai') {
      openaiPassthroughEnabled.value = false
      openaiOAuthResponsesWebSocketV2Mode.value = OPENAI_WS_MODE_OFF
      openaiAPIKeyResponsesWebSocketV2Mode.value = OPENAI_WS_MODE_OFF
      codexCLIOnlyEnabled.value = false
    }
    if (newPlatform !== 'anthropic') {
      anthropicPassthroughEnabled.value = false
    }
    // Reset OAuth states
    oauth.resetState()
    openaiOAuth.resetState()
    soraOAuth.resetState()
    geminiOAuth.resetState()
    antigravityOAuth.resetState()
  }
)

// Gemini AI Studio OAuth availability (requires operator-configured OAuth client)
watch(
  [accountCategory, () => form.platform],
  ([category, platform]) => {
    if (platform === 'openai' && category !== 'oauth-based') {
      codexCLIOnlyEnabled.value = false
    }
    if (platform !== 'anthropic' || category !== 'apikey') {
      anthropicPassthroughEnabled.value = false
    }
  }
)

watch(
  [() => props.show, () => form.platform, accountCategory],
  async ([show, platform, category]) => {
    if (!show || platform !== 'gemini' || category !== 'oauth-based') {
      geminiAIStudioOAuthEnabled.value = false
      return
    }
    const caps = await geminiOAuth.getCapabilities()
    geminiAIStudioOAuthEnabled.value = !!caps?.ai_studio_oauth_enabled
    if (!geminiAIStudioOAuthEnabled.value && geminiOAuthType.value === 'ai_studio') {
      geminiOAuthType.value = 'code_assist'
    }
  },
  { immediate: true }
)

// Auto-fill related models when switching to whitelist mode or changing platform
watch(
  [modelRestrictionMode, () => form.platform],
  ([newMode]) => {
    if (newMode === 'whitelist') {
      allowedModels.value = [...getModelsByPlatform(form.platform, 'whitelist')]
    }
  }
)

// Model mapping helpers
const addModelMapping = () => {
  modelMappings.value.push({ from: '', to: '' })
}

const removeModelMapping = (index: number) => {
  modelMappings.value.splice(index, 1)
}

const addPresetMapping = (from: string, to: string) => {
  if (modelMappings.value.some((m) => m.from === from)) {
    appStore.showInfo(t('admin.accounts.mappingExists', { model: from }))
    return
  }
  modelMappings.value.push({ from, to })
}

const addAntigravityModelMapping = () => {
  antigravityModelMappings.value.push({ from: '', to: '' })
}

const removeAntigravityModelMapping = (index: number) => {
  antigravityModelMappings.value.splice(index, 1)
}

const addAntigravityPresetMapping = (from: string, to: string) => {
  if (antigravityModelMappings.value.some((m) => m.from === from)) {
    appStore.showInfo(t('admin.accounts.mappingExists', { model: from }))
    return
  }
  antigravityModelMappings.value.push({ from, to })
}

const maybeImportCreatedAccounts = async (createdAccounts: Account[]) => {
  pendingImportedModelsResult.value = null
  if (!autoImportModels.value || createdAccounts.length === 0) {
    return
  }
  appStore.showInfo(t('admin.accounts.probingModels'))
  const results: Parameters<typeof mergeAccountModelImportResults>[0] = []
  let firstFailureMessage = ''
  for (const account of createdAccounts) {
    try {
      const result = await adminAPI.accounts.importModels(account.id, { trigger: 'create' })
      results.push(result)
    } catch (error) {
      console.error('Failed to auto import models after account creation:', error)
      if (!firstFailureMessage) {
        firstFailureMessage = resolveAccountModelImportErrorMessage(t, error)
      }
    }
  }

  const mergedResult = mergeAccountModelImportResults(results)
  if (!mergedResult) {
    if (firstFailureMessage) {
      appStore.showError(firstFailureMessage)
    }
    return
  }

  const toastPayload = buildAccountModelImportToastPayload(t, mergedResult)
  const toastOptions = {
    ...toastPayload.options,
    details: toastPayload.options.details ? [...toastPayload.options.details] : undefined,
    copyText: toastPayload.options.copyText
  }
  let toastType = toastPayload.type
  let toastMessage = toastPayload.message

  if (firstFailureMessage) {
    toastType = mergedResult.imported_count > 0 ? 'warning' : 'error'
    toastMessage = `${toastMessage} - ${firstFailureMessage}`
    toastOptions.details = [...(toastOptions.details || []), firstFailureMessage]
    toastOptions.copyText = toastOptions.copyText
      ? `${toastOptions.copyText}
${firstFailureMessage}`
      : firstFailureMessage
    toastOptions.persistent = true
  }

  if (toastType === 'error') {
    appStore.showError(toastMessage, toastOptions)
  } else if (toastType === 'warning') {
    appStore.showWarning(toastMessage, toastOptions)
  } else {
    appStore.showSuccess(toastMessage, toastOptions)
  }

  if (shouldInvalidateModelInventory(mergedResult)) {
    invalidateModelRegistry()
    modelInventoryStore.invalidate()
  }
  if (extractSyncableRegistryModels(mergedResult).length > 0) {
    pendingImportedModelsResult.value = mergedResult
  }
}

const submitCreateAccount = async (payload: CreateAccountRequest): Promise<Account | null> => {
  submitting.value = true
  try {
    const payloadWithScope: CreateAccountRequest = {
      ...payload,
      extra: buildAccountModelScopeExtra(payload.extra as Record<string, unknown> | undefined, {
        platform: payload.platform,
        enabled: payload.platform === 'antigravity'
          ? true
          : !(payload.platform === 'openai' && isOpenAIModelRestrictionDisabled.value),
        mode: payload.platform === 'antigravity' ? 'mapping' : modelRestrictionMode.value,
        allowedModels: allowedModels.value,
        modelMappings: payload.platform === 'antigravity' ? antigravityModelMappings.value : modelMappings.value
      })
    }
    const createdAccount = await adminAPI.accounts.create(withConfirmFlag(payloadWithScope))
    appStore.showSuccess(t('admin.accounts.accountCreated'))
    await maybeImportCreatedAccounts([createdAccount])
    emit('created')
    handleClose()
    return createdAccount
  } catch (error: any) {
    if (
      error.response?.status === 409 &&
      error.response?.data?.error === 'mixed_channel_warning' &&
      requiresMixedChannelCheck.value
    ) {
      openMixedChannelDialog({
        message: error.response?.data?.message,
        onConfirm: async () => submitCreateAccount(payload)
      })
      return null
    }
    appStore.showError(error.response?.data?.message || error.response?.data?.detail || t('admin.accounts.failedToCreate'))
    return null
  } finally {
    submitting.value = false
  }
}

// Methods
const resetForm = () => {
  step.value = 1
  form.name = ''
  form.notes = ''
  form.platform = 'anthropic'
  form.type = 'oauth'
  form.credentials = {}
  autoImportModels.value = false
  form.proxy_id = null
  form.concurrency = 10
  form.load_factor = null
  form.priority = 1
  form.rate_multiplier = 1
  form.group_ids = []
  form.expires_at = null
  accountCategory.value = 'oauth-based'
  addMethod.value = 'oauth'
  apiKeyBaseUrl.value = resolveAccountApiKeyDefaultBaseUrl('anthropic')
  apiKeyValue.value = ''
  editQuotaLimit.value = null
  editQuotaDailyLimit.value = null
  editQuotaWeeklyLimit.value = null
  modelMappings.value = []
  modelRestrictionMode.value = 'whitelist'
  allowedModels.value = [...getModelsByPlatform('anthropic', 'whitelist')] // Default fill related models

  loadAntigravityDefaultMappings()
  poolModeState.enabled = false
  poolModeState.retryCount = DEFAULT_POOL_MODE_RETRY_COUNT
  customErrorCodesState.enabled = false
  customErrorCodesState.selectedCodes = []
  customErrorCodesState.input = null
  interceptWarmupRequests.value = false
  autoPauseOnExpired.value = true
  openaiPassthroughEnabled.value = false
  openaiOAuthResponsesWebSocketV2Mode.value = OPENAI_WS_MODE_OFF
  openaiAPIKeyResponsesWebSocketV2Mode.value = OPENAI_WS_MODE_OFF
  codexCLIOnlyEnabled.value = false
  anthropicPassthroughEnabled.value = false
  quotaControl.reset()
  antigravityAccountType.value = 'oauth'
  upstreamBaseUrl.value = ''
  upstreamApiKey.value = ''
  resetTempUnschedRules()
  geminiOAuthType.value = 'code_assist'
  geminiTierGoogleOne.value = 'google_one_free'
  geminiTierGcp.value = 'gcp_standard'
  geminiTierAIStudio.value = 'aistudio_free'
  oauth.resetState()
  openaiOAuth.resetState()
  soraOAuth.resetState()
  geminiOAuth.resetState()
  antigravityOAuth.resetState()
  oauthFlowRef.value?.reset()
  resetMixedChannelRisk()
}

const handleClose = () => {
  resetMixedChannelRisk()
  const importedResult = pendingImportedModelsResult.value
  pendingImportedModelsResult.value = null
  emit('close')
  if (importedResult) {
    queueMicrotask(() => emit('models-imported', importedResult))
  }
}

const buildOpenAIExtra = (base?: Record<string, unknown>): Record<string, unknown> | undefined => {
  if (form.platform !== 'openai') {
    return base
  }

  const extra: Record<string, unknown> = { ...(base || {}) }
  if (accountCategory.value === 'oauth-based') {
    extra.openai_oauth_responses_websockets_v2_mode = openaiOAuthResponsesWebSocketV2Mode.value
    extra.openai_oauth_responses_websockets_v2_enabled = isOpenAIWSModeEnabled(openaiOAuthResponsesWebSocketV2Mode.value)
  } else if (accountCategory.value === 'apikey') {
    extra.openai_apikey_responses_websockets_v2_mode = openaiAPIKeyResponsesWebSocketV2Mode.value
    extra.openai_apikey_responses_websockets_v2_enabled = isOpenAIWSModeEnabled(openaiAPIKeyResponsesWebSocketV2Mode.value)
  }
  delete extra.responses_websockets_v2_enabled
  delete extra.openai_ws_enabled
  if (openaiPassthroughEnabled.value) {
    extra.openai_passthrough = true
  } else {
    delete extra.openai_passthrough
    delete extra.openai_oauth_passthrough
  }

  if (accountCategory.value === 'oauth-based' && codexCLIOnlyEnabled.value) {
    extra.codex_cli_only = true
  } else {
    delete extra.codex_cli_only
  }

  return Object.keys(extra).length > 0 ? extra : undefined
}

const buildAnthropicExtra = (base?: Record<string, unknown>): Record<string, unknown> | undefined => {
  if (form.platform !== 'anthropic' || accountCategory.value !== 'apikey') {
    return base
  }

  const extra: Record<string, unknown> = { ...(base || {}) }
  if (anthropicPassthroughEnabled.value) {
    extra.anthropic_passthrough = true
  } else {
    delete extra.anthropic_passthrough
  }

  return Object.keys(extra).length > 0 ? extra : undefined
}

const buildSoraExtra = (
  base?: Record<string, unknown>,
  linkedOpenAIAccountId?: string | number
): Record<string, unknown> | undefined => {
  const extra: Record<string, unknown> = { ...(base || {}) }
  if (linkedOpenAIAccountId !== undefined && linkedOpenAIAccountId !== null) {
    const id = String(linkedOpenAIAccountId).trim()
    if (id) {
      extra.linked_openai_account_id = id
    }
  }
  delete extra.openai_passthrough
  delete extra.openai_oauth_passthrough
  delete extra.codex_cli_only
  delete extra.openai_oauth_responses_websockets_v2_mode
  delete extra.openai_apikey_responses_websockets_v2_mode
  delete extra.openai_oauth_responses_websockets_v2_enabled
  delete extra.openai_apikey_responses_websockets_v2_enabled
  delete extra.responses_websockets_v2_enabled
  delete extra.openai_ws_enabled
  return Object.keys(extra).length > 0 ? extra : undefined
}

// Helper function to create account with mixed channel warning handling
const doCreateAccount = async (payload: CreateAccountRequest) => {
  const canContinue = await ensureMixedChannelConfirmed(async () => {
    await submitCreateAccount(payload)
  })
  if (!canContinue) {
    return
  }
  await submitCreateAccount(payload)
}

const handleSubmit = async () => {
  // For OAuth-based type, handle OAuth flow (goes to step 2)
  if (isOAuthFlow.value) {
    if (!form.name.trim()) {
      appStore.showError(t('admin.accounts.pleaseEnterAccountName'))
      return
    }
    const canContinue = await ensureMixedChannelConfirmed(async () => {
      step.value = 2
    })
    if (!canContinue) {
      return
    }
    step.value = 2
    return
  }

  // For Antigravity upstream type, create directly
  if (form.platform === 'antigravity' && antigravityAccountType.value === 'upstream') {
    if (!form.name.trim()) {
      appStore.showError(t('admin.accounts.pleaseEnterAccountName'))
      return
    }
    if (!upstreamBaseUrl.value.trim()) {
      appStore.showError(t('admin.accounts.upstream.pleaseEnterBaseUrl'))
      return
    }
    if (!upstreamApiKey.value.trim()) {
      appStore.showError(t('admin.accounts.upstream.pleaseEnterApiKey'))
      return
    }

    // Build upstream credentials (and optional model restriction)
    const credentials: Record<string, unknown> = {
      base_url: upstreamBaseUrl.value.trim(),
      api_key: upstreamApiKey.value.trim()
    }

    const antigravityModelMapping = buildModelMappingObject(
      'mapping',
      [],
      antigravityModelMappings.value
    )
    if (antigravityModelMapping) {
      credentials.model_mapping = antigravityModelMapping
    }

    applyInterceptWarmup(credentials, interceptWarmupRequests.value, 'create')

    const extra = mixedScheduling.value ? { mixed_scheduling: true } : undefined
    await createAccountAndFinish(form.platform, 'apikey', credentials, extra)
    return
  }

  // For apikey type, create directly
  if (!apiKeyValue.value.trim()) {
    appStore.showError(t('admin.accounts.pleaseEnterApiKey'))
    return
  }

  if (form.platform === 'sora') {
    const soraBaseUrl = apiKeyBaseUrl.value.trim()
    if (!soraBaseUrl) {
      appStore.showError(t('admin.accounts.soraBaseUrlRequired'))
      return
    }
    if (!soraBaseUrl.startsWith('http://') && !soraBaseUrl.startsWith('https://')) {
      appStore.showError(t('admin.accounts.soraBaseUrlInvalidScheme'))
      return
    }
  }

  // Determine default base URL based on platform
  const defaultBaseUrl = resolveAccountApiKeyDefaultBaseUrl(form.platform)

  // Build credentials with optional model mapping
  const credentials: Record<string, unknown> = {
    base_url: apiKeyBaseUrl.value.trim() || defaultBaseUrl,
    api_key: apiKeyValue.value.trim()
  }
  if (form.platform === 'gemini') {
    credentials.tier_id = geminiTierAIStudio.value
  }

  if (!isOpenAIModelRestrictionDisabled.value) {
    const modelMapping = buildModelMappingObject(modelRestrictionMode.value, allowedModels.value, modelMappings.value)
    if (modelMapping) {
      credentials.model_mapping = modelMapping
    }
  }

  // Add pool mode if enabled
  if (poolModeState.enabled) {
    credentials.pool_mode = true
    credentials.pool_mode_retry_count = normalizePoolModeRetryCount(poolModeState.retryCount)
  }

  if (customErrorCodesState.enabled) {
    credentials.custom_error_codes_enabled = true
    credentials.custom_error_codes = [...customErrorCodesState.selectedCodes]
  }

  applyInterceptWarmup(credentials, interceptWarmupRequests.value, 'create')
  if (!applyTempUnschedConfig(credentials)) {
    return
  }

  form.credentials = credentials
  const extra = buildAnthropicExtra(buildOpenAIExtra())

  await doCreateAccount({
    ...form,
    group_ids: form.group_ids,
    extra,
    auto_pause_on_expired: autoPauseOnExpired.value
  })
}

const goBackToBasicInfo = () => {
  step.value = 1
  oauth.resetState()
  openaiOAuth.resetState()
  soraOAuth.resetState()
  geminiOAuth.resetState()
  antigravityOAuth.resetState()
  oauthFlowRef.value?.reset()
}

const handleGenerateUrl = async () => {
  if (form.platform === 'openai' || form.platform === 'sora') {
    await activeOpenAIOAuth.value.generateAuthUrl(form.proxy_id)
  } else if (form.platform === 'gemini') {
    await geminiOAuth.generateAuthUrl(
      form.proxy_id,
      oauthFlowRef.value?.projectId,
      geminiOAuthType.value,
      geminiSelectedTier.value
    )
  } else if (form.platform === 'antigravity') {
    await antigravityOAuth.generateAuthUrl(form.proxy_id)
  } else {
    await oauth.generateAuthUrl(addMethod.value, form.proxy_id)
  }
}

const handleValidateRefreshToken = (rt: string) => {
  if (form.platform === 'openai' || form.platform === 'sora') {
    handleOpenAIValidateRT(rt)
  } else if (form.platform === 'antigravity') {
    handleAntigravityValidateRT(rt)
  }
}

const handleValidateSessionToken = (sessionToken: string) => {
  if (form.platform === 'sora') {
    handleSoraValidateST(sessionToken)
  }
}

const handleImportAccessToken = async (accessTokenInput: string) => {
  const oauthClient = activeOpenAIOAuth.value
  if (!accessTokenInput.trim()) return

  const accessTokens = accessTokenInput
    .split('\n')
    .map((at) => at.trim())
    .filter((at) => at)

  if (accessTokens.length === 0) {
    oauthClient.error.value = 'Please enter at least one Access Token'
    return
  }

  oauthClient.loading.value = true
  oauthClient.error.value = ''

  let successCount = 0
  let failedCount = 0
  const errors: string[] = []
  const createdAccounts: Account[] = []

  try {
    for (let i = 0; i < accessTokens.length; i++) {
      try {
        const credentials: Record<string, unknown> = {
          access_token: accessTokens[i],
        }
        const soraExtra = buildSoraExtra()

        const accountName = accessTokens.length > 1 ? `${form.name} #${i + 1}` : form.name
        const createdAccount = await adminAPI.accounts.create({
          name: accountName,
          notes: form.notes,
          platform: 'sora',
          type: 'oauth',
          credentials,
          extra: soraExtra,
          proxy_id: form.proxy_id,
          concurrency: form.concurrency,
          load_factor: form.load_factor ?? undefined,
          priority: form.priority,
          rate_multiplier: form.rate_multiplier,
          group_ids: form.group_ids,
          expires_at: form.expires_at,
          auto_pause_on_expired: autoPauseOnExpired.value
        })
        createdAccounts.push(createdAccount)
        successCount++
      } catch (error: any) {
        failedCount++
        const errMsg = error.response?.data?.detail || error.message || 'Unknown error'
        errors.push(`#${i + 1}: ${errMsg}`)
      }
    }

    if (successCount > 0 && failedCount === 0) {
      appStore.showSuccess(
        accessTokens.length > 1
          ? t('admin.accounts.oauth.batchSuccess', { count: successCount })
          : t('admin.accounts.accountCreated')
      )
      await maybeImportCreatedAccounts(createdAccounts)
      emit('created')
      handleClose()
    } else if (successCount > 0 && failedCount > 0) {
      appStore.showWarning(
        t('admin.accounts.oauth.batchPartialSuccess', { success: successCount, failed: failedCount })
      )
      await maybeImportCreatedAccounts(createdAccounts)
      oauthClient.error.value = errors.join('\n')
      emit('created')
    } else {
      oauthClient.error.value = errors.join('\n')
      appStore.showError(t('admin.accounts.oauth.batchFailed'))
    }
  } finally {
    oauthClient.loading.value = false
  }
}

const formatDateTimeLocal = formatDateTimeLocalInput
const parseDateTimeLocal = parseDateTimeLocalInput

// Create account and handle success/failure
const createAccountAndFinish = async (
  platform: AccountPlatform,
  type: AccountType,
  credentials: Record<string, unknown>,
  extra?: Record<string, unknown>
) => {
  if (!applyTempUnschedConfig(credentials)) {
    return
  }
  // Inject quota limits for apikey accounts
  let finalExtra = extra
  if (type === 'apikey') {
    const quotaExtra: Record<string, unknown> = { ...(extra || {}) }
    if (editQuotaLimit.value != null && editQuotaLimit.value > 0) {
      quotaExtra.quota_limit = editQuotaLimit.value
    }
    if (editQuotaDailyLimit.value != null && editQuotaDailyLimit.value > 0) {
      quotaExtra.quota_daily_limit = editQuotaDailyLimit.value
    }
    if (editQuotaWeeklyLimit.value != null && editQuotaWeeklyLimit.value > 0) {
      quotaExtra.quota_weekly_limit = editQuotaWeeklyLimit.value
    }
    if (Object.keys(quotaExtra).length > 0) {
      finalExtra = quotaExtra
    }
  }
  await doCreateAccount({
    name: form.name,
    notes: form.notes,
    platform,
    type,
    credentials,
    extra: finalExtra,
    proxy_id: form.proxy_id,
    concurrency: form.concurrency,
    load_factor: form.load_factor ?? undefined,
    priority: form.priority,
    rate_multiplier: form.rate_multiplier,
    group_ids: form.group_ids,
    expires_at: form.expires_at,
    auto_pause_on_expired: autoPauseOnExpired.value
  })
}

const handleOpenAIExchange = async (authCode: string) => {
  const oauthClient = activeOpenAIOAuth.value
  if (!authCode.trim() || !oauthClient.sessionId.value) return

  oauthClient.loading.value = true
  oauthClient.error.value = ''

  try {
    const stateToUse = (oauthFlowRef.value?.oauthState || oauthClient.oauthState.value || '').trim()
    if (!stateToUse) {
      oauthClient.error.value = t('admin.accounts.oauth.authFailed')
      appStore.showError(oauthClient.error.value)
      return
    }

    const tokenInfo = await oauthClient.exchangeAuthCode(
      authCode.trim(),
      oauthClient.sessionId.value,
      stateToUse,
      form.proxy_id
    )
    if (!tokenInfo) return

    const credentials = oauthClient.buildCredentials(tokenInfo)
    const oauthExtra = oauthClient.buildExtraInfo(tokenInfo) as Record<string, unknown> | undefined
    const extra = buildOpenAIExtra(oauthExtra)
    const shouldCreateOpenAI = form.platform === 'openai'
    const shouldCreateSora = form.platform === 'sora'

    if (shouldCreateOpenAI && !isOpenAIModelRestrictionDisabled.value) {
      const modelMapping = buildModelMappingObject(modelRestrictionMode.value, allowedModels.value, modelMappings.value)
      if (modelMapping) {
        credentials.model_mapping = modelMapping
      }
    }

    if (!applyTempUnschedConfig(credentials)) {
      return
    }

    let openaiAccountId: string | number | undefined
    const createdAccounts: Account[] = []

    if (shouldCreateOpenAI) {
      const openaiAccount = await adminAPI.accounts.create({
        name: form.name,
        notes: form.notes,
        platform: 'openai',
        type: 'oauth',
        credentials,
        extra,
        proxy_id: form.proxy_id,
        concurrency: form.concurrency,
        load_factor: form.load_factor ?? undefined,
        priority: form.priority,
        rate_multiplier: form.rate_multiplier,
        group_ids: form.group_ids,
        expires_at: form.expires_at,
        auto_pause_on_expired: autoPauseOnExpired.value
      })
      openaiAccountId = openaiAccount.id
      createdAccounts.push(openaiAccount)
      appStore.showSuccess(t('admin.accounts.accountCreated'))
    }

    if (shouldCreateSora) {
      const soraCredentials = {
        access_token: credentials.access_token,
        refresh_token: credentials.refresh_token,
        client_id: credentials.client_id,
        expires_at: credentials.expires_at
      }

      const soraName = shouldCreateOpenAI ? `${form.name} (Sora)` : form.name
      const soraExtra = buildSoraExtra(shouldCreateOpenAI ? extra : oauthExtra, openaiAccountId)
      const soraAccount = await adminAPI.accounts.create({
        name: soraName,
        notes: form.notes,
        platform: 'sora',
        type: 'oauth',
        credentials: soraCredentials,
        extra: soraExtra,
        proxy_id: form.proxy_id,
        concurrency: form.concurrency,
        load_factor: form.load_factor ?? undefined,
        priority: form.priority,
        rate_multiplier: form.rate_multiplier,
        group_ids: form.group_ids,
        expires_at: form.expires_at,
        auto_pause_on_expired: autoPauseOnExpired.value
      })
      createdAccounts.push(soraAccount)
      appStore.showSuccess(t('admin.accounts.accountCreated'))
    }

    await maybeImportCreatedAccounts(createdAccounts)
    emit('created')
    handleClose()
  } catch (error: any) {
    oauthClient.error.value = error.response?.data?.detail || t('admin.accounts.oauth.authFailed')
    appStore.showError(oauthClient.error.value)
  } finally {
    oauthClient.loading.value = false
  }
}

const handleOpenAIValidateRT = async (refreshTokenInput: string) => {
  const oauthClient = activeOpenAIOAuth.value
  if (!refreshTokenInput.trim()) return

  // Parse multiple refresh tokens (one per line)
  const refreshTokens = refreshTokenInput
    .split('\n')
    .map((rt) => rt.trim())
    .filter((rt) => rt)

  if (refreshTokens.length === 0) {
    oauthClient.error.value = t('admin.accounts.oauth.openai.pleaseEnterRefreshToken')
    return
  }

  oauthClient.loading.value = true
  oauthClient.error.value = ''

  let successCount = 0
  let failedCount = 0
  const errors: string[] = []
  const createdAccounts: Account[] = []
  const shouldCreateOpenAI = form.platform === 'openai'
  const shouldCreateSora = form.platform === 'sora'

  try {
    for (let i = 0; i < refreshTokens.length; i++) {
      try {
        const tokenInfo = await oauthClient.validateRefreshToken(
          refreshTokens[i],
          form.proxy_id
        )
        if (!tokenInfo) {
          failedCount++
          errors.push(`#${i + 1}: ${oauthClient.error.value || 'Validation failed'}`)
          oauthClient.error.value = ''
          continue
        }

        const credentials = oauthClient.buildCredentials(tokenInfo)
        const oauthExtra = oauthClient.buildExtraInfo(tokenInfo) as Record<string, unknown> | undefined
        const extra = buildOpenAIExtra(oauthExtra)

        if (shouldCreateOpenAI && !isOpenAIModelRestrictionDisabled.value) {
          const modelMapping = buildModelMappingObject(modelRestrictionMode.value, allowedModels.value, modelMappings.value)
          if (modelMapping) {
            credentials.model_mapping = modelMapping
          }
        }

        // Generate account name with index for batch
        const accountName = refreshTokens.length > 1 ? `${form.name} #${i + 1}` : form.name

        let openaiAccountId: string | number | undefined

        if (shouldCreateOpenAI) {
          const openaiAccount = await adminAPI.accounts.create({
            name: accountName,
            notes: form.notes,
            platform: 'openai',
            type: 'oauth',
            credentials,
            extra,
            proxy_id: form.proxy_id,
            concurrency: form.concurrency,
            load_factor: form.load_factor ?? undefined,
            priority: form.priority,
            rate_multiplier: form.rate_multiplier,
            group_ids: form.group_ids,
            expires_at: form.expires_at,
            auto_pause_on_expired: autoPauseOnExpired.value
          })
          openaiAccountId = openaiAccount.id
          createdAccounts.push(openaiAccount)
        }

        if (shouldCreateSora) {
          const soraCredentials = {
            access_token: credentials.access_token,
            refresh_token: credentials.refresh_token,
            client_id: credentials.client_id,
            expires_at: credentials.expires_at
          }
          const soraName = shouldCreateOpenAI ? `${accountName} (Sora)` : accountName
          const soraExtra = buildSoraExtra(shouldCreateOpenAI ? extra : oauthExtra, openaiAccountId)
          const soraAccount = await adminAPI.accounts.create({
            name: soraName,
            notes: form.notes,
            platform: 'sora',
            type: 'oauth',
            credentials: soraCredentials,
            extra: soraExtra,
            proxy_id: form.proxy_id,
            concurrency: form.concurrency,
            load_factor: form.load_factor ?? undefined,
            priority: form.priority,
            rate_multiplier: form.rate_multiplier,
            group_ids: form.group_ids,
            expires_at: form.expires_at,
            auto_pause_on_expired: autoPauseOnExpired.value
          })
          createdAccounts.push(soraAccount)
        }

        successCount++
      } catch (error: any) {
        failedCount++
        const errMsg = error.response?.data?.detail || error.message || 'Unknown error'
        errors.push(`#${i + 1}: ${errMsg}`)
      }
    }

    // Show results
    if (successCount > 0 && failedCount === 0) {
      appStore.showSuccess(
        refreshTokens.length > 1
          ? t('admin.accounts.oauth.batchSuccess', { count: successCount })
          : t('admin.accounts.accountCreated')
      )
      await maybeImportCreatedAccounts(createdAccounts)
      emit('created')
      handleClose()
    } else if (successCount > 0 && failedCount > 0) {
      appStore.showWarning(
        t('admin.accounts.oauth.batchPartialSuccess', { success: successCount, failed: failedCount })
      )
      await maybeImportCreatedAccounts(createdAccounts)
      oauthClient.error.value = errors.join('\n')
      emit('created')
    } else {
      oauthClient.error.value = errors.join('\n')
      appStore.showError(t('admin.accounts.oauth.batchFailed'))
    }
  } finally {
    oauthClient.loading.value = false
  }
}

const handleSoraValidateST = async (sessionTokenInput: string) => {
  const oauthClient = activeOpenAIOAuth.value
  if (!sessionTokenInput.trim()) return

  const sessionTokens = sessionTokenInput
    .split('\n')
    .map((st) => st.trim())
    .filter((st) => st)

  if (sessionTokens.length === 0) {
    oauthClient.error.value = t('admin.accounts.oauth.openai.pleaseEnterSessionToken')
    return
  }

  oauthClient.loading.value = true
  oauthClient.error.value = ''

  let successCount = 0
  let failedCount = 0
  const errors: string[] = []
  const createdAccounts: Account[] = []

  try {
    for (let i = 0; i < sessionTokens.length; i++) {
      try {
        const tokenInfo = await oauthClient.validateSessionToken(sessionTokens[i], form.proxy_id)
        if (!tokenInfo) {
          failedCount++
          errors.push(`#${i + 1}: ${oauthClient.error.value || 'Validation failed'}`)
          oauthClient.error.value = ''
          continue
        }

        const credentials = oauthClient.buildCredentials(tokenInfo)
        credentials.session_token = sessionTokens[i]
        const oauthExtra = oauthClient.buildExtraInfo(tokenInfo) as Record<string, unknown> | undefined
        const soraExtra = buildSoraExtra(oauthExtra)

        const accountName = sessionTokens.length > 1 ? `${form.name} #${i + 1}` : form.name
        const createdAccount = await adminAPI.accounts.create({
          name: accountName,
          notes: form.notes,
          platform: 'sora',
          type: 'oauth',
          credentials,
          extra: soraExtra,
          proxy_id: form.proxy_id,
          concurrency: form.concurrency,
          load_factor: form.load_factor ?? undefined,
          priority: form.priority,
          rate_multiplier: form.rate_multiplier,
          group_ids: form.group_ids,
          expires_at: form.expires_at,
          auto_pause_on_expired: autoPauseOnExpired.value
        })
        createdAccounts.push(createdAccount)
        successCount++
      } catch (error: any) {
        failedCount++
        const errMsg = error.response?.data?.detail || error.message || 'Unknown error'
        errors.push(`#${i + 1}: ${errMsg}`)
      }
    }

    if (successCount > 0 && failedCount === 0) {
      appStore.showSuccess(
        sessionTokens.length > 1
          ? t('admin.accounts.oauth.batchSuccess', { count: successCount })
          : t('admin.accounts.accountCreated')
      )
      await maybeImportCreatedAccounts(createdAccounts)
      emit('created')
      handleClose()
    } else if (successCount > 0 && failedCount > 0) {
      appStore.showWarning(
        t('admin.accounts.oauth.batchPartialSuccess', { success: successCount, failed: failedCount })
      )
      await maybeImportCreatedAccounts(createdAccounts)
      oauthClient.error.value = errors.join('\n')
      emit('created')
    } else {
      oauthClient.error.value = errors.join('\n')
      appStore.showError(t('admin.accounts.oauth.batchFailed'))
    }
  } finally {
    oauthClient.loading.value = false
  }
}

const handleAntigravityValidateRT = async (refreshTokenInput: string) => {
  if (!refreshTokenInput.trim()) return

  // Parse multiple refresh tokens (one per line)
  const refreshTokens = refreshTokenInput
    .split('\n')
    .map((rt) => rt.trim())
    .filter((rt) => rt)

  if (refreshTokens.length === 0) {
    antigravityOAuth.error.value = t('admin.accounts.oauth.antigravity.pleaseEnterRefreshToken')
    return
  }

  antigravityOAuth.loading.value = true
  antigravityOAuth.error.value = ''

  let successCount = 0
  let failedCount = 0
  const errors: string[] = []
  const createdAccounts: Account[] = []

  try {
    for (let i = 0; i < refreshTokens.length; i++) {
      try {
        const tokenInfo = await antigravityOAuth.validateRefreshToken(
          refreshTokens[i],
          form.proxy_id
        )
        if (!tokenInfo) {
          failedCount++
          errors.push(`#${i + 1}: ${antigravityOAuth.error.value || 'Validation failed'}`)
          antigravityOAuth.error.value = ''
          continue
        }

        const credentials = antigravityOAuth.buildCredentials(tokenInfo)
        
        // Generate account name with index for batch
        const accountName = refreshTokens.length > 1 ? `${form.name} #${i + 1}` : form.name

        // Note: Antigravity doesn't have buildExtraInfo, so we pass empty extra or rely on credentials
        const createPayload: CreateAccountRequest = withConfirmFlag({
          name: accountName,
          notes: form.notes,
          platform: 'antigravity' as const,
          type: 'oauth' as const,
          credentials,
          extra: {},
          proxy_id: form.proxy_id,
          concurrency: form.concurrency,
          load_factor: form.load_factor ?? undefined,
          priority: form.priority,
          rate_multiplier: form.rate_multiplier,
          group_ids: form.group_ids,
          expires_at: form.expires_at,
          auto_pause_on_expired: autoPauseOnExpired.value
        })
        const createdAccount = await adminAPI.accounts.create(createPayload)
        createdAccounts.push(createdAccount)
        successCount++
      } catch (error: any) {
        failedCount++
        const errMsg = error.response?.data?.detail || error.message || 'Unknown error'
        errors.push(`#${i + 1}: ${errMsg}`)
      }
    }

    // Show results
    if (successCount > 0 && failedCount === 0) {
      appStore.showSuccess(
        refreshTokens.length > 1
          ? t('admin.accounts.oauth.batchSuccess', { count: successCount })
          : t('admin.accounts.accountCreated')
      )
      await maybeImportCreatedAccounts(createdAccounts)
      emit('created')
      handleClose()
    } else if (successCount > 0 && failedCount > 0) {
      appStore.showWarning(
        t('admin.accounts.oauth.batchPartialSuccess', { success: successCount, failed: failedCount })
      )
      await maybeImportCreatedAccounts(createdAccounts)
      antigravityOAuth.error.value = errors.join('\n')
      emit('created')
    } else {
      antigravityOAuth.error.value = errors.join('\n')
      appStore.showError(t('admin.accounts.oauth.batchFailed'))
    }
  } finally {
    antigravityOAuth.loading.value = false
  }
}

const handleGeminiExchange = async (authCode: string) => {
  if (!authCode.trim() || !geminiOAuth.sessionId.value) return

  geminiOAuth.loading.value = true
  geminiOAuth.error.value = ''

  try {
    const stateFromInput = oauthFlowRef.value?.oauthState || ''
    const stateToUse = stateFromInput || geminiOAuth.state.value
    if (!stateToUse) {
      geminiOAuth.error.value = t('admin.accounts.oauth.authFailed')
      appStore.showError(geminiOAuth.error.value)
      return
    }

    const tokenInfo = await geminiOAuth.exchangeAuthCode({
      code: authCode.trim(),
      sessionId: geminiOAuth.sessionId.value,
      state: stateToUse,
      proxyId: form.proxy_id,
      oauthType: geminiOAuthType.value,
      tierId: geminiSelectedTier.value
    })
    if (!tokenInfo) return

    const credentials = geminiOAuth.buildCredentials(tokenInfo)
    const extra = geminiOAuth.buildExtraInfo(tokenInfo)
    await createAccountAndFinish('gemini', 'oauth', credentials, extra)
  } catch (error: any) {
    geminiOAuth.error.value = error.response?.data?.detail || t('admin.accounts.oauth.authFailed')
    appStore.showError(geminiOAuth.error.value)
  } finally {
    geminiOAuth.loading.value = false
  }
}

const handleAntigravityExchange = async (authCode: string) => {
  if (!authCode.trim() || !antigravityOAuth.sessionId.value) return

  antigravityOAuth.loading.value = true
  antigravityOAuth.error.value = ''

  try {
    const stateFromInput = oauthFlowRef.value?.oauthState || ''
    const stateToUse = stateFromInput || antigravityOAuth.state.value
    if (!stateToUse) {
      antigravityOAuth.error.value = t('admin.accounts.oauth.authFailed')
      appStore.showError(antigravityOAuth.error.value)
      return
    }

    const tokenInfo = await antigravityOAuth.exchangeAuthCode({
      code: authCode.trim(),
      sessionId: antigravityOAuth.sessionId.value,
      state: stateToUse,
      proxyId: form.proxy_id
    })
		if (!tokenInfo) return

		const credentials = antigravityOAuth.buildCredentials(tokenInfo)
		applyInterceptWarmup(credentials, interceptWarmupRequests.value, 'create')
		const antigravityModelMapping = buildModelMappingObject(
			'mapping',
			[],
			antigravityModelMappings.value
		)
		if (antigravityModelMapping) {
			credentials.model_mapping = antigravityModelMapping
		}
		const extra = mixedScheduling.value ? { mixed_scheduling: true } : undefined
		await createAccountAndFinish('antigravity', 'oauth', credentials, extra)
  } catch (error: any) {
    antigravityOAuth.error.value = error.response?.data?.detail || t('admin.accounts.oauth.authFailed')
    appStore.showError(antigravityOAuth.error.value)
  } finally {
    antigravityOAuth.loading.value = false
  }
}

const handleAnthropicExchange = async (authCode: string) => {
  if (!authCode.trim() || !oauth.sessionId.value) return

  oauth.loading.value = true
  oauth.error.value = ''

  try {
    const proxyConfig = form.proxy_id ? { proxy_id: form.proxy_id } : {}
    const endpoint =
      addMethod.value === 'oauth'
        ? '/admin/accounts/exchange-code'
        : '/admin/accounts/exchange-setup-token-code'

    const tokenInfo = await adminAPI.accounts.exchangeCode(endpoint, {
      session_id: oauth.sessionId.value,
      code: authCode.trim(),
      ...proxyConfig
    })

    const baseExtra = oauth.buildExtraInfo(tokenInfo) || {}
    const extra = quotaControl.buildExtra(baseExtra)

    const credentials: Record<string, unknown> = { ...tokenInfo }
    applyInterceptWarmup(credentials, interceptWarmupRequests.value, 'create')
    await createAccountAndFinish(form.platform, addMethod.value as AccountType, credentials, extra)
  } catch (error: any) {
    oauth.error.value = error.response?.data?.detail || t('admin.accounts.oauth.authFailed')
    appStore.showError(oauth.error.value)
  } finally {
    oauth.loading.value = false
  }
}

const handleExchangeCode = async () => {
  const authCode = oauthFlowRef.value?.authCode || ''

  switch (form.platform) {
    case 'openai':
    case 'sora':
      return handleOpenAIExchange(authCode)
    case 'gemini':
      return handleGeminiExchange(authCode)
    case 'antigravity':
      return handleAntigravityExchange(authCode)
    default:
      return handleAnthropicExchange(authCode)
  }
}

const handleCookieAuth = async (sessionKey: string) => {
  oauth.loading.value = true
  oauth.error.value = ''

  try {
    const proxyConfig = form.proxy_id ? { proxy_id: form.proxy_id } : {}
    const keys = oauth.parseSessionKeys(sessionKey)

    if (keys.length === 0) {
      oauth.error.value = t('admin.accounts.oauth.pleaseEnterSessionKey')
      return
    }

    const tempUnschedPayload = tempUnschedEnabled.value
      ? buildTempUnschedPayload()
      : []
    if (tempUnschedEnabled.value && tempUnschedPayload.length === 0) {
      appStore.showError(t('admin.accounts.tempUnschedulable.rulesInvalid'))
      return
    }

    const endpoint =
      addMethod.value === 'oauth'
        ? '/admin/accounts/cookie-auth'
        : '/admin/accounts/setup-token-cookie-auth'

    let successCount = 0
    let failedCount = 0
    const errors: string[] = []
    const createdAccounts: Account[] = []

    for (let i = 0; i < keys.length; i++) {
      try {
        const tokenInfo = await adminAPI.accounts.exchangeCode(endpoint, {
          session_id: '',
          code: keys[i],
          ...proxyConfig
        })

        // Build extra with quota control settings
        const baseExtra = oauth.buildExtraInfo(tokenInfo) || {}
        const extra = quotaControl.buildExtra(baseExtra)

        const accountName = keys.length > 1 ? `${form.name} #${i + 1}` : form.name

        const credentials: Record<string, unknown> = { ...tokenInfo }
        applyInterceptWarmup(credentials, interceptWarmupRequests.value, 'create')
        if (tempUnschedEnabled.value) {
          credentials.temp_unschedulable_enabled = true
          credentials.temp_unschedulable_rules = tempUnschedPayload
        }

        const createdAccount = await adminAPI.accounts.create({
          name: accountName,
          notes: form.notes,
          platform: form.platform,
          type: addMethod.value, // Use addMethod as type: 'oauth' or 'setup-token'
          credentials,
          extra,
          proxy_id: form.proxy_id,
          concurrency: form.concurrency,
          load_factor: form.load_factor ?? undefined,
          priority: form.priority,
          rate_multiplier: form.rate_multiplier,
          group_ids: form.group_ids,
          expires_at: form.expires_at,
          auto_pause_on_expired: autoPauseOnExpired.value
        })

        createdAccounts.push(createdAccount)
        successCount++
      } catch (error: any) {
        failedCount++
        errors.push(
          t('admin.accounts.oauth.keyAuthFailed', {
            index: i + 1,
            error: error.response?.data?.detail || t('admin.accounts.oauth.authFailed')
          })
        )
      }
    }

    if (successCount > 0) {
      appStore.showSuccess(t('admin.accounts.oauth.successCreated', { count: successCount }))
      if (failedCount === 0) {
        await maybeImportCreatedAccounts(createdAccounts)
        emit('created')
        handleClose()
      } else {
        await maybeImportCreatedAccounts(createdAccounts)
        emit('created')
      }
    }

    if (failedCount > 0) {
      oauth.error.value = errors.join('\n')
    }
  } catch (error: any) {
    oauth.error.value = error.response?.data?.detail || t('admin.accounts.oauth.cookieAuthFailed')
  } finally {
    oauth.loading.value = false
  }
}
</script>

