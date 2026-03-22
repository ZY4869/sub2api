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
          :dailyResetMode="editQuotaDailyResetMode"
          :dailyResetHour="editQuotaDailyResetHour"
          :weeklyResetMode="editQuotaWeeklyResetMode"
          :weeklyResetDay="editQuotaWeeklyResetDay"
          :weeklyResetHour="editQuotaWeeklyResetHour"
          :resetTimezone="editQuotaResetTimezone"
          @update:totalLimit="editQuotaLimit = $event"
          @update:dailyLimit="editQuotaDailyLimit = $event"
          @update:weeklyLimit="editQuotaWeeklyLimit = $event"
          @update:dailyResetMode="editQuotaDailyResetMode = $event"
          @update:dailyResetHour="editQuotaDailyResetHour = $event"
          @update:weeklyResetMode="editQuotaWeeklyResetMode = $event"
          @update:weeklyResetDay="editQuotaWeeklyResetDay = $event"
          @update:weeklyResetHour="editQuotaWeeklyResetHour = $event"
          @update:resetTimezone="editQuotaResetTimezone = $event"
        />
      </div>
      <AccountModelScopeEditor
        v-if="accountCategory === 'oauth-based' && form.platform !== 'antigravity'"
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

    <AccountCopilotDeviceFlowPanel
      v-else-if="form.platform === 'copilot'"
      ref="copilotDeviceFlowRef"
      :proxy-id="form.proxy_id"
      :submit-label="t('common.create')"
      :submit-loading="copilotSubmitting"
      @submit="handleCreateCopilotAccount"
    />

    <AccountKiroTokenImportPanel
      v-else-if="form.platform === 'kiro'"
      ref="kiroImportRef"
      :submit-label="t('common.create')"
      :submitting="submitting"
      @submit="handleCreateKiroAccount"
    />

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
import { ref, reactive, computed, toRef, watch } from 'vue'
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
import { useCreateAccountAnthropicCookieAuth } from '@/composables/useCreateAccountAnthropicCookieAuth'
import { useCreateAccountAnthropicExchange } from '@/composables/useCreateAccountAnthropicExchange'
import { useCreateAccountAntigravityHandlers } from '@/composables/useCreateAccountAntigravityHandlers'
import { useCreateAccountOpenAIExchange } from '@/composables/useCreateAccountOpenAIExchange'
import { useCreateAccountOpenAIRefreshTokenValidation } from '@/composables/useCreateAccountOpenAIRefreshTokenValidation'
import { useCreateAccountReset } from '@/composables/useCreateAccountReset'
import { useCreateAccountSoraAccessTokenImport } from '@/composables/useCreateAccountSoraAccessTokenImport'
import { useCreateAccountSoraSessionTokenValidation } from '@/composables/useCreateAccountSoraSessionTokenValidation'
import { useCreateAccountSubmit } from '@/composables/useCreateAccountSubmit'
import type {
  Proxy,
  AdminGroup,
  AccountPlatform,
  AccountType,
  Account
} from '@/types'
import BaseDialog from '@/components/common/BaseDialog.vue'
import AccountApiKeyBasicSettingsEditor from '@/components/account/AccountApiKeyBasicSettingsEditor.vue'
import AccountAntigravityModelMappingEditor from '@/components/account/AccountAntigravityModelMappingEditor.vue'
import AccountAutoPauseToggle from '@/components/account/AccountAutoPauseToggle.vue'
import AccountCopilotDeviceFlowPanel from '@/components/account/AccountCopilotDeviceFlowPanel.vue'
import AccountCreateFooterActions from '@/components/account/AccountCreateFooterActions.vue'
import AccountCreateOAuthStep from '@/components/account/AccountCreateOAuthStep.vue'
import AccountCreatePlatformSelector from '@/components/account/AccountCreatePlatformSelector.vue'
import AccountCreatePlatformTypeEditor from '@/components/account/AccountCreatePlatformTypeEditor.vue'
import AccountCustomErrorCodesEditor from '@/components/account/AccountCustomErrorCodesEditor.vue'
import AccountGatewaySettingsEditor from '@/components/account/AccountGatewaySettingsEditor.vue'
import AccountGeminiHelpDialog from '@/components/account/AccountGeminiHelpDialog.vue'
import AccountGroupSettingsEditor from '@/components/account/AccountGroupSettingsEditor.vue'
import AccountKiroTokenImportPanel from '@/components/account/AccountKiroTokenImportPanel.vue'
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
  createDefaultAccountCustomErrorCodesState,
  createDefaultAccountPoolModeState,
  type ModelMapping
} from '@/utils/accountFormShared'
import {
  applyAccountCustomErrorCodesStateToCredentials,
  applyAccountPoolModeStateToCredentials
} from '@/utils/accountApiKeyAdvancedSettingsForm'
import { resolveAccountApiKeyDefaultBaseUrl } from '@/utils/accountApiKeyBasicSettings'
import { buildAnthropicExtra, buildOpenAIExtra, buildSoraExtra } from '@/utils/accountCreateExtras'
import type { ParsedKiroTokenImport } from '@/utils/kiroTokenImport'
import {
  OPENAI_WS_MODE_OFF,
  OPENAI_WS_MODE_PASSTHROUGH,
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
  if (form.platform === 'copilot') return t('admin.accounts.copilotDeviceFlow.title')
  if (form.platform === 'kiro') return t('admin.accounts.kiroImport.title')
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
  if (form.platform === 'copilot' || form.platform === 'kiro') return ''
  if (form.platform === 'openai' || form.platform === 'sora') return activeOpenAIOAuth.value.error.value
  if (form.platform === 'gemini') return geminiOAuth.error.value
  if (form.platform === 'antigravity') return antigravityOAuth.error.value
  return oauth.error.value
})

// Refs
const oauthFlowRef = ref<OAuthFlowExposed | null>(null)
const copilotDeviceFlowRef = ref<{ reset: () => void } | null>(null)
const kiroImportRef = ref<{ reset: () => void } | null>(null)

// State
const step = ref(1)
const autoImportModels = ref(false)
const copilotSubmitting = ref(false)
const accountCategory = ref<'oauth-based' | 'apikey'>('oauth-based') // UI selection for account category
const addMethod = ref<AddMethod>('oauth') // For oauth-based: 'oauth' or 'setup-token'
const apiKeyBaseUrl = ref(resolveAccountApiKeyDefaultBaseUrl('anthropic'))
const apiKeyValue = ref('')
const editQuotaLimit = ref<number | null>(null)
const editQuotaDailyLimit = ref<number | null>(null)
const editQuotaWeeklyLimit = ref<number | null>(null)
const editQuotaDailyResetMode = ref<'rolling' | 'fixed' | null>(null)
const editQuotaDailyResetHour = ref<number | null>(null)
const editQuotaWeeklyResetMode = ref<'rolling' | 'fixed' | null>(null)
const editQuotaWeeklyResetDay = ref<number | null>(null)
const editQuotaWeeklyResetHour = ref<number | null>(null)
const editQuotaResetTimezone = ref<string | null>(null)
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
  if (form.platform === 'copilot' || form.platform === 'kiro') {
    return false
  }
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
  if (form.platform === 'copilot' || form.platform === 'kiro') {
    return false
  }
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
      // Modal opened - default to unrestricted model scope unless user selects explicitly.
      allowedModels.value = accountCategory.value === 'apikey'
        ? [...getModelsByPlatform(form.platform, 'whitelist')]
        : []
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
      form.type = form.platform === 'anthropic' ? method as AccountType : 'oauth'
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
    if (newPlatform !== 'anthropic') {
      addMethod.value = 'oauth'
    }
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
    if (newPlatform === 'copilot' || newPlatform === 'kiro') {
      accountCategory.value = 'oauth-based'
      form.type = 'oauth'
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
    copilotDeviceFlowRef.value?.reset()
    kiroImportRef.value?.reset()
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

const handleClose = () => {
  resetMixedChannelRisk()
  const importedResult = pendingImportedModelsResult.value
  pendingImportedModelsResult.value = null
  emit('close')
  if (importedResult) {
    queueMicrotask(() => emit('models-imported', importedResult))
  }
}

const { resetForm } = useCreateAccountReset({
  step,
  form,
  autoImportModels,
  accountCategory,
  addMethod,
  apiKeyBaseUrl,
  apiKeyValue,
  editQuotaLimit,
  editQuotaDailyLimit,
  editQuotaWeeklyLimit,
  editQuotaDailyResetMode,
  editQuotaDailyResetHour,
  editQuotaWeeklyResetMode,
  editQuotaWeeklyResetDay,
  editQuotaWeeklyResetHour,
  editQuotaResetTimezone,
  modelMappings,
  modelRestrictionMode,
  allowedModels,
  loadAntigravityDefaultMappings,
  poolModeState,
  customErrorCodesState,
  interceptWarmupRequests,
  autoPauseOnExpired,
  openaiPassthroughEnabled,
  openaiOAuthResponsesWebSocketV2Mode,
  openaiAPIKeyResponsesWebSocketV2Mode,
  codexCLIOnlyEnabled,
  anthropicPassthroughEnabled,
  quotaControlReset: () => quotaControl.reset(),
  antigravityAccountType,
  upstreamBaseUrl,
  upstreamApiKey,
  resetTempUnschedRules,
  geminiOAuthType,
  geminiTierGoogleOne,
  geminiTierGcp,
  geminiTierAIStudio,
  oauthReset: () => oauth.resetState(),
  openaiOAuthReset: () => openaiOAuth.resetState(),
  soraOAuthReset: () => soraOAuth.resetState(),
  geminiOAuthReset: () => geminiOAuth.resetState(),
  antigravityOAuthReset: () => antigravityOAuth.resetState(),
  oauthFlowReset: () => oauthFlowRef.value?.reset(),
  copilotFlowReset: () => copilotDeviceFlowRef.value?.reset(),
  kiroImportReset: () => kiroImportRef.value?.reset(),
  resetMixedChannelRisk
})

const buildAccountExtra = (base?: Record<string, unknown>) => {
  const openaiExtra = buildOpenAIExtra({
    platform: form.platform,
    accountCategory: accountCategory.value,
    base,
    openaiOAuthResponsesWebSocketV2Mode: openaiOAuthResponsesWebSocketV2Mode.value,
    openaiAPIKeyResponsesWebSocketV2Mode: openaiAPIKeyResponsesWebSocketV2Mode.value,
    openaiPassthroughEnabled: openaiPassthroughEnabled.value,
    codexCLIOnlyEnabled: codexCLIOnlyEnabled.value
  })

  return buildAnthropicExtra({
    platform: form.platform,
    accountCategory: accountCategory.value,
    base: openaiExtra,
    anthropicPassthroughEnabled: anthropicPassthroughEnabled.value
  })
}

const buildSoraAccountExtra = (
  base?: Record<string, unknown>,
  linkedOpenAIAccountId?: string | number
) => buildSoraExtra({ base, linkedOpenAIAccountId })

const { submitting, createAccountAndFinish } = useCreateAccountSubmit({
  withConfirmFlag,
  ensureMixedChannelConfirmed,
  requiresMixedChannelCheck,
  openMixedChannelDialog,
  isOpenAIModelRestrictionDisabled,
  modelRestrictionMode,
  allowedModels,
  modelMappings,
  antigravityModelMappings,
  applyTempUnschedConfig,
  form,
  autoPauseOnExpired,
  editQuotaLimit,
  editQuotaDailyLimit,
  editQuotaWeeklyLimit,
  editQuotaDailyResetMode,
  editQuotaDailyResetHour,
  editQuotaWeeklyResetMode,
  editQuotaWeeklyResetDay,
  editQuotaWeeklyResetHour,
  editQuotaResetTimezone,
  afterCreateImportModels: maybeImportCreatedAccounts,
  emitCreated: () => emit('created'),
  onClose: handleClose
})

const { handleAnthropicExchange } = useCreateAccountAnthropicExchange({
  oauthClient: oauth,
  platform: toRef(form, 'platform'),
  addMethod,
  proxyId: toRef(form, 'proxy_id'),
  interceptWarmupRequests,
  quotaControl,
  createAccountAndFinish
})

const { handleCookieAuth } = useCreateAccountAnthropicCookieAuth({
  oauthClient: oauth,
  platform: toRef(form, 'platform'),
  addMethod,
  proxyId: toRef(form, 'proxy_id'),
  form,
  autoPauseOnExpired,
  interceptWarmupRequests,
  quotaControl,
  tempUnschedEnabled,
  buildTempUnschedPayload,
  afterCreateImportModels: maybeImportCreatedAccounts,
  emitCreated: () => emit('created'),
  onClose: handleClose
})

const getOpenAIOAuthState = () =>
  (oauthFlowRef.value?.oauthState || activeOpenAIOAuth.value.oauthState.value || '').trim()

const { handleOpenAIExchange } = useCreateAccountOpenAIExchange({
  oauthClient: activeOpenAIOAuth,
  getOAuthState: getOpenAIOAuthState,
  form,
  autoPauseOnExpired,
  applyTempUnschedConfig,
  isOpenAIModelRestrictionDisabled,
  modelRestrictionMode,
  allowedModels,
  modelMappings,
  buildAccountExtra,
  buildSoraAccountExtra,
  afterCreateImportModels: maybeImportCreatedAccounts,
  emitCreated: () => emit('created'),
  onClose: handleClose
})

const { handleImportAccessToken } = useCreateAccountSoraAccessTokenImport({
  oauthClient: activeOpenAIOAuth,
  form,
  autoPauseOnExpired,
  buildSoraAccountExtra: () => buildSoraAccountExtra(),
  afterCreateImportModels: maybeImportCreatedAccounts,
  emitCreated: () => emit('created'),
  onClose: handleClose
})

const { handleOpenAIValidateRT } = useCreateAccountOpenAIRefreshTokenValidation({
  oauthClient: activeOpenAIOAuth,
  form,
  autoPauseOnExpired,
  isOpenAIModelRestrictionDisabled,
  modelRestrictionMode,
  allowedModels,
  modelMappings,
  buildAccountExtra,
  buildSoraAccountExtra,
  afterCreateImportModels: maybeImportCreatedAccounts,
  emitCreated: () => emit('created'),
  onClose: handleClose
})

const { handleSoraValidateST } = useCreateAccountSoraSessionTokenValidation({
  oauthClient: activeOpenAIOAuth,
  form,
  autoPauseOnExpired,
  buildSoraAccountExtra,
  afterCreateImportModels: maybeImportCreatedAccounts,
  emitCreated: () => emit('created'),
  onClose: handleClose
})

const getAntigravityOAuthState = () =>
  (oauthFlowRef.value?.oauthState || antigravityOAuth.state.value || '').trim()

const { handleAntigravityValidateRT, handleAntigravityExchange } = useCreateAccountAntigravityHandlers({
  oauthClient: antigravityOAuth,
  getOAuthState: getAntigravityOAuthState,
  withConfirmFlag,
  form,
  autoPauseOnExpired,
  interceptWarmupRequests,
  antigravityModelMappings,
  mixedScheduling,
  createAccountAndFinish,
  afterCreateImportModels: maybeImportCreatedAccounts,
  emitCreated: () => emit('created'),
  onClose: handleClose
})

const handleCreateCopilotAccount = async (payload: { sessionId: string }) => {
  if (!form.name.trim()) {
    appStore.showError(t('admin.accounts.pleaseEnterAccountName'))
    return
  }

  copilotSubmitting.value = true
  try {
    const createdAccount = await adminAPI.accounts.createCopilotAccountFromDevice({
      session_id: payload.sessionId,
      proxy_id: form.proxy_id,
      name: form.name,
      notes: form.notes || undefined,
      concurrency: form.concurrency,
      load_factor: form.load_factor ?? undefined,
      priority: form.priority,
      rate_multiplier: form.rate_multiplier,
      group_ids: form.group_ids,
      expires_at: form.expires_at,
      auto_pause_on_expired: autoPauseOnExpired.value
    })

    appStore.showSuccess(t('admin.accounts.accountCreated'))
    await maybeImportCreatedAccounts([createdAccount])
    emit('created')
    handleClose()
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.accounts.failedToCreate'))
  } finally {
    copilotSubmitting.value = false
  }
}

const handleCreateKiroAccount = async (payload: ParsedKiroTokenImport) => {
  await createAccountAndFinish('kiro', 'oauth', payload.credentials, payload.extra)
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

  applyAccountPoolModeStateToCredentials(credentials, poolModeState)
  applyAccountCustomErrorCodesStateToCredentials(credentials, customErrorCodesState)

  applyInterceptWarmup(credentials, interceptWarmupRequests.value, 'create')
  const extra = buildAccountExtra()
  await createAccountAndFinish(form.platform, 'apikey', credentials, extra)
}

const goBackToBasicInfo = () => {
  step.value = 1
  copilotSubmitting.value = false
  oauth.resetState()
  openaiOAuth.resetState()
  soraOAuth.resetState()
  geminiOAuth.resetState()
  antigravityOAuth.resetState()
  oauthFlowRef.value?.reset()
  copilotDeviceFlowRef.value?.reset()
  kiroImportRef.value?.reset()
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

const formatDateTimeLocal = formatDateTimeLocalInput
const parseDateTimeLocal = parseDateTimeLocalInput

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
    geminiOAuth.error.value = error?.message || t('admin.accounts.oauth.authFailed')
    appStore.showError(geminiOAuth.error.value)
  } finally {
    geminiOAuth.loading.value = false
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
</script>

