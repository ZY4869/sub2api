<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.editAccount')"
    width="normal"
    @close="handleClose"
  >
    <form
      v-if="account"
      id="edit-account-form"
      @submit.prevent="handleSubmit"
      class="space-y-5"
    >
      <div>
        <label class="input-label">{{ t('common.name') }}</label>
        <input v-model="form.name" type="text" required class="input" data-tour="edit-account-form-name" />
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

      <!-- API Key fields (only for apikey type) -->
      <div v-if="account.type === 'apikey'" class="space-y-4">
        <AccountApiKeyBasicSettingsEditor
          v-model:base-url="editBaseUrl"
          v-model:api-key="editApiKey"
          v-model:model-scope-mode="modelRestrictionMode"
          v-model:allowed-models="allowedModels"
          :platform="account.platform"
          mode="edit"
          :model-scope-disabled="isOpenAIModelRestrictionDisabled"
          :model-mappings="modelMappings"
          :preset-mappings="presetMappings"
          :get-mapping-key="getModelMappingKey"
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

      <AccountModelScopeEditor
        v-if="account.platform === 'openai' && account.type === 'oauth'"
        :disabled="isOpenAIModelRestrictionDisabled"
        :platform="account?.platform || 'anthropic'"
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

      <!-- Upstream fields (only for upstream type) -->
      <div v-if="account.type === 'upstream'" class="space-y-4">
        <AccountUpstreamSettingsEditor
          v-model:base-url="editBaseUrl"
          v-model:api-key="editApiKey"
          mode="edit"
        />
      </div>

      <!-- Antigravity model restriction (applies to all antigravity types) -->
      <AccountAntigravityModelMappingEditor
        v-if="account.platform === 'antigravity'"
        :model-mappings="antigravityModelMappings"
        :preset-mappings="antigravityPresetMappings"
        :get-mapping-key="getAntigravityModelMappingKey"
        @add-mapping="addAntigravityModelMapping"
        @remove-mapping="removeAntigravityModelMapping"
        @add-preset="addAntigravityPresetMapping($event.from, $event.to)"
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
        v-if="account?.platform === 'anthropic' || account?.platform === 'antigravity'"
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
        :show-open-ai-passthrough="account?.platform === 'openai' && (account?.type === 'oauth' || account?.type === 'apikey')"
        :open-ai-passthrough-enabled="openaiPassthroughEnabled"
        :show-open-ai-ws-mode="account?.platform === 'openai' && (account?.type === 'oauth' || account?.type === 'apikey')"
        :open-ai-ws-mode="openaiResponsesWebSocketV2Mode"
        :open-ai-ws-mode-options="openAIWSModeOptions"
        :open-ai-ws-mode-concurrency-hint-key="openAIWSModeConcurrencyHintKey"
        :show-anthropic-passthrough="account?.platform === 'anthropic' && account?.type === 'apikey'"
        :anthropic-passthrough-enabled="anthropicPassthroughEnabled"
        :show-codex-cli-only="account?.platform === 'openai' && account?.type === 'oauth'"
        :codex-cli-only-enabled="codexCLIOnlyEnabled"
        @update:open-ai-passthrough-enabled="openaiPassthroughEnabled = $event"
        @update:open-ai-ws-mode="openaiResponsesWebSocketV2Mode = $event"
        @update:anthropic-passthrough-enabled="anthropicPassthroughEnabled = $event"
        @update:codex-cli-only-enabled="codexCLIOnlyEnabled = $event"
      />

      <div v-if="account?.type === 'apikey'" class="border-t border-gray-200 pt-4 dark:border-dark-600 space-y-4">
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

      <AccountAutoPauseToggle v-model:enabled="autoPauseOnExpired" />

      <AccountQuotaControlEditor
        v-if="account?.platform === 'anthropic' && (account?.type === 'oauth' || account?.type === 'setup-token')"
        v-model:state="quotaControlState"
        :umq-mode-options="umqModeOptions"
      />

      <div class="border-t border-gray-200 pt-4 dark:border-dark-600">
        <div>
          <label class="input-label">{{ t('common.status') }}</label>
          <Select v-model="form.status" :options="statusOptions" />
        </div>
      </div>

      <AccountGroupSettingsEditor
        v-model:group-ids="form.group_ids"
        v-model:mixed-scheduling="mixedScheduling"
        :groups="groups"
        :platform="account?.platform || null"
        :simple-mode="authStore.isSimpleMode"
        :show-mixed-scheduling="account?.platform === 'antigravity'"
        mixed-scheduling-readonly
      />

    </form>

    <template #footer>
      <div v-if="account" class="flex justify-end gap-3">
        <button @click="handleClose" type="button" class="btn btn-secondary">
          {{ t('common.cancel') }}
        </button>
        <button
          type="submit"
          form="edit-account-form"
          :disabled="submitting"
          class="btn btn-primary"
          data-tour="account-form-submit"
        >
          <svg
            v-if="submitting"
            class="-ml-1 mr-2 h-4 w-4 animate-spin"
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
          {{ submitting ? t('admin.accounts.updating') : t('common.update') }}
        </button>
      </div>
    </template>
  </BaseDialog>

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
import { useAuthStore } from '@/stores/auth'
import { adminAPI } from '@/api/admin'
import { useAnthropicQuotaControl } from '@/composables/useAnthropicQuotaControl'
import { useAccountMixedChannelRisk } from '@/composables/useAccountMixedChannelRisk'
import { useAccountTempUnschedRules } from '@/composables/useAccountTempUnschedRules'
import type { Account, Proxy, AdminGroup } from '@/types'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import AccountApiKeyBasicSettingsEditor from '@/components/account/AccountApiKeyBasicSettingsEditor.vue'
import AccountAntigravityModelMappingEditor from '@/components/account/AccountAntigravityModelMappingEditor.vue'
import AccountAutoPauseToggle from '@/components/account/AccountAutoPauseToggle.vue'
import AccountCustomErrorCodesEditor from '@/components/account/AccountCustomErrorCodesEditor.vue'
import AccountGatewaySettingsEditor from '@/components/account/AccountGatewaySettingsEditor.vue'
import AccountGroupSettingsEditor from '@/components/account/AccountGroupSettingsEditor.vue'
import AccountMixedChannelWarningDialog from '@/components/account/AccountMixedChannelWarningDialog.vue'
import AccountModelScopeEditor from '@/components/account/AccountModelScopeEditor.vue'
import AccountPoolModeEditor from '@/components/account/AccountPoolModeEditor.vue'
import AccountQuotaControlEditor from '@/components/account/AccountQuotaControlEditor.vue'
import AccountRuntimeSettingsEditor from '@/components/account/AccountRuntimeSettingsEditor.vue'
import AccountTempUnschedRulesEditor from '@/components/account/AccountTempUnschedRulesEditor.vue'
import AccountUpstreamSettingsEditor from '@/components/account/AccountUpstreamSettingsEditor.vue'
import QuotaLimitCard from '@/components/account/QuotaLimitCard.vue'
import { applyInterceptWarmup } from '@/components/account/credentialsBuilder'
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
  applyAccountPoolModeStateToCredentials,
  loadAccountCustomErrorCodesStateFromCredentials,
  loadAccountPoolModeStateFromCredentials,
  resetAccountCustomErrorCodesState,
  resetAccountPoolModeState
} from '@/utils/accountApiKeyAdvancedSettingsForm'
import { resolveAccountApiKeyDefaultBaseUrl } from '@/utils/accountApiKeyBasicSettings'
import {
  OPENAI_WS_MODE_OFF,
  OPENAI_WS_MODE_PASSTHROUGH,
  isOpenAIWSModeEnabled,
  resolveOpenAIWSModeConcurrencyHintKey,
  type OpenAIWSMode,
  resolveOpenAIWSModeFromExtra
} from '@/utils/openaiWsMode'
import {
  getPresetMappingsByPlatform,
  commonErrorCodes,
  buildModelMappingObject
} from '@/composables/useModelWhitelist'
import { buildAccountModelScopeExtra } from '@/utils/accountModelScope'

interface Props {
  show: boolean
  account: Account | null
  proxies: Proxy[]
  groups: AdminGroup[]
}

const props = defineProps<Props>()
const emit = defineEmits<{
  close: []
  updated: [account: Account]
}>()

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const showFormError = (message: string) => appStore.showError(message)
const showFormInfo = (message: string) => appStore.showInfo(message)

const antigravityPresetMappings = computed(() => getPresetMappingsByPlatform('antigravity'))

// State
const submitting = ref(false)
const editBaseUrl = ref(resolveAccountApiKeyDefaultBaseUrl('anthropic'))
const editApiKey = ref('')
const modelMappings = ref<ModelMapping[]>([])
const modelRestrictionMode = ref<'whitelist' | 'mapping'>('whitelist')
const allowedModels = ref<string[]>([])
const poolModeState = reactive(createDefaultAccountPoolModeState(DEFAULT_POOL_MODE_RETRY_COUNT))
const customErrorCodesState = reactive(createDefaultAccountCustomErrorCodesState())
const interceptWarmupRequests = ref(false)
const autoPauseOnExpired = ref(false)
const mixedScheduling = ref(false) // For antigravity accounts: enable mixed scheduling
const antigravityModelRestrictionMode = ref<'whitelist' | 'mapping'>('whitelist')
const antigravityWhitelistModels = ref<string[]>([])
const antigravityModelMappings = ref<ModelMapping[]>([])
const getModelMappingKey = createStableObjectKeyResolver<ModelMapping>('edit-model-mapping')
const getAntigravityModelMappingKey = createStableObjectKeyResolver<ModelMapping>('edit-antigravity-model-mapping')
const quotaControl = useAnthropicQuotaControl()
const quotaControlState = quotaControl.state
const umqModeOptions = quotaControl.umqModeOptions

const openaiPassthroughEnabled = ref(false)
const openaiOAuthResponsesWebSocketV2Mode = ref<OpenAIWSMode>(OPENAI_WS_MODE_OFF)
const openaiAPIKeyResponsesWebSocketV2Mode = ref<OpenAIWSMode>(OPENAI_WS_MODE_OFF)
const codexCLIOnlyEnabled = ref(false)
const anthropicPassthroughEnabled = ref(false)
const editQuotaLimit = ref<number | null>(null)
const editQuotaDailyLimit = ref<number | null>(null)
const editQuotaWeeklyLimit = ref<number | null>(null)
const openAIWSModeOptions = computed(() => [
  { value: OPENAI_WS_MODE_OFF, label: t('admin.accounts.openai.wsModeOff') },
  { value: OPENAI_WS_MODE_PASSTHROUGH, label: t('admin.accounts.openai.wsModePassthrough') }
])
const openaiResponsesWebSocketV2Mode = computed({
  get: () => {
    if (props.account?.type === 'apikey') {
      return openaiAPIKeyResponsesWebSocketV2Mode.value
    }
    return openaiOAuthResponsesWebSocketV2Mode.value
  },
  set: (mode: OpenAIWSMode) => {
    if (props.account?.type === 'apikey') {
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
  props.account?.platform === 'openai' && openaiPassthroughEnabled.value
)

// Computed: current preset mappings based on platform
const presetMappings = computed(() => getPresetMappingsByPlatform(props.account?.platform || 'anthropic'))

// Computed: default base URL based on platform
const defaultBaseUrl = computed(() => {
  return resolveAccountApiKeyDefaultBaseUrl(props.account?.platform || 'anthropic')
})


const form = reactive({
  name: '',
  notes: '',
  proxy_id: null as number | null,
  concurrency: 1,
  load_factor: null as number | null,
  priority: 1,
  rate_multiplier: 1,
  status: 'active' as 'active' | 'inactive' | 'error',
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
  applyToCredentials: applyTempUnschedConfig,
  loadFromCredentials: loadTempUnschedRules,
  reset: resetTempUnschedRules
} = useAccountTempUnschedRules({
  keyPrefix: 'edit',
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
  currentPlatform: () => props.account?.platform ?? null,
  buildCheckPayload: () => {
    if (!props.account) {
      return null
    }
    return {
      platform: props.account.platform,
      group_ids: form.group_ids,
      account_id: props.account.id
    }
  },
  buildWarningText: (details) => t('admin.accounts.mixedChannelWarning', { ...details }),
  fallbackMessage: () => t('admin.accounts.failedToUpdate'),
  showError: showFormError
})

const statusOptions = computed(() => {
  const options = [
    { value: 'active', label: t('common.active') },
    { value: 'inactive', label: t('common.inactive') }
  ]
  if (form.status === 'error') {
    options.push({ value: 'error', label: t('admin.accounts.status.error') })
  }
  return options
})

const expiresAtInput = computed({
  get: () => formatDateTimeLocal(form.expires_at),
  set: (value: string) => {
    form.expires_at = parseDateTimeLocal(value)
  }
})

// Watchers
watch(
  () => props.account,
  (newAccount) => {
    if (newAccount) {
      resetMixedChannelRisk()
      form.name = newAccount.name
      form.notes = newAccount.notes || ''
      form.proxy_id = newAccount.proxy_id
      form.concurrency = newAccount.concurrency
      form.load_factor = newAccount.load_factor ?? null
      form.priority = newAccount.priority
      form.rate_multiplier = newAccount.rate_multiplier ?? 1
      form.status = (newAccount.status === 'active' || newAccount.status === 'inactive' || newAccount.status === 'error')
        ? newAccount.status
        : 'active'
      form.group_ids = newAccount.group_ids || []
      form.expires_at = newAccount.expires_at ?? null

      // Load intercept warmup requests setting (applies to all account types)
      const credentials = newAccount.credentials as Record<string, unknown> | undefined
      interceptWarmupRequests.value = credentials?.intercept_warmup_requests === true
      autoPauseOnExpired.value = newAccount.auto_pause_on_expired === true

      // Load mixed scheduling setting (only for antigravity accounts)
      const extra = newAccount.extra as Record<string, unknown> | undefined
      mixedScheduling.value = extra?.mixed_scheduling === true

      // Load OpenAI passthrough toggle (OpenAI OAuth/API Key)
      openaiPassthroughEnabled.value = false
      openaiOAuthResponsesWebSocketV2Mode.value = OPENAI_WS_MODE_OFF
      openaiAPIKeyResponsesWebSocketV2Mode.value = OPENAI_WS_MODE_OFF
      codexCLIOnlyEnabled.value = false
      anthropicPassthroughEnabled.value = false
      if (newAccount.platform === 'openai' && (newAccount.type === 'oauth' || newAccount.type === 'apikey')) {
        openaiPassthroughEnabled.value = extra?.openai_passthrough === true || extra?.openai_oauth_passthrough === true
        openaiOAuthResponsesWebSocketV2Mode.value = resolveOpenAIWSModeFromExtra(extra, {
          modeKey: 'openai_oauth_responses_websockets_v2_mode',
          enabledKey: 'openai_oauth_responses_websockets_v2_enabled',
          fallbackEnabledKeys: ['responses_websockets_v2_enabled', 'openai_ws_enabled'],
          defaultMode: OPENAI_WS_MODE_OFF
        })
        openaiAPIKeyResponsesWebSocketV2Mode.value = resolveOpenAIWSModeFromExtra(extra, {
          modeKey: 'openai_apikey_responses_websockets_v2_mode',
          enabledKey: 'openai_apikey_responses_websockets_v2_enabled',
          fallbackEnabledKeys: ['responses_websockets_v2_enabled', 'openai_ws_enabled'],
          defaultMode: OPENAI_WS_MODE_OFF
        })
        if (newAccount.type === 'oauth') {
          codexCLIOnlyEnabled.value = extra?.codex_cli_only === true
        }
      }
      if (newAccount.platform === 'anthropic' && newAccount.type === 'apikey') {
        anthropicPassthroughEnabled.value = extra?.anthropic_passthrough === true
      }

      // Load quota limit for apikey accounts
      if (newAccount.type === 'apikey') {
        const quotaVal = extra?.quota_limit as number | undefined
        editQuotaLimit.value = (quotaVal && quotaVal > 0) ? quotaVal : null
        const dailyVal = extra?.quota_daily_limit as number | undefined
        editQuotaDailyLimit.value = (dailyVal && dailyVal > 0) ? dailyVal : null
        const weeklyVal = extra?.quota_weekly_limit as number | undefined
        editQuotaWeeklyLimit.value = (weeklyVal && weeklyVal > 0) ? weeklyVal : null
      } else {
        editQuotaLimit.value = null
        editQuotaDailyLimit.value = null
        editQuotaWeeklyLimit.value = null
      }

      if (newAccount.platform === 'antigravity') {
        const credentials = newAccount.credentials as Record<string, unknown> | undefined

        antigravityModelRestrictionMode.value = 'mapping'
        antigravityWhitelistModels.value = []

        const rawAgMapping = credentials?.model_mapping as Record<string, string> | undefined
        if (rawAgMapping && typeof rawAgMapping === 'object') {
          const entries = Object.entries(rawAgMapping)
          antigravityModelMappings.value = entries.map(([from, to]) => ({ from, to }))
        } else {
          const rawWhitelist = credentials?.model_whitelist
          if (Array.isArray(rawWhitelist) && rawWhitelist.length > 0) {
            antigravityModelMappings.value = rawWhitelist
              .map((v) => String(v).trim())
              .filter((v) => v.length > 0)
              .map((m) => ({ from: m, to: m }))
          } else {
            antigravityModelMappings.value = []
          }
        }
      } else {
        antigravityModelRestrictionMode.value = 'mapping'
        antigravityWhitelistModels.value = []
        antigravityModelMappings.value = []
      }

      quotaControl.loadFromAccount(newAccount)

      loadTempUnschedRules(credentials)

      // Initialize API Key fields for apikey type
      if (newAccount.type === 'apikey' && newAccount.credentials) {
        const credentials = newAccount.credentials as Record<string, unknown>
        const platformDefaultUrl = resolveAccountApiKeyDefaultBaseUrl(newAccount.platform)
        editBaseUrl.value = (credentials.base_url as string) || platformDefaultUrl

        // Load model mappings and detect mode
        const existingMappings = credentials.model_mapping as Record<string, string> | undefined
        if (existingMappings && typeof existingMappings === 'object') {
          const entries = Object.entries(existingMappings)

          // Detect if this is whitelist mode (all from === to) or mapping mode
          const isWhitelistMode = entries.length > 0 && entries.every(([from, to]) => from === to)

          if (isWhitelistMode) {
            // Whitelist mode: populate allowedModels
            modelRestrictionMode.value = 'whitelist'
            allowedModels.value = entries.map(([from]) => from)
            modelMappings.value = []
          } else {
            // Mapping mode: populate modelMappings
            modelRestrictionMode.value = 'mapping'
            modelMappings.value = entries.map(([from, to]) => ({ from, to }))
            allowedModels.value = []
          }
        } else {
          // No mappings: default to whitelist mode with empty selection (allow all)
          modelRestrictionMode.value = 'whitelist'
          modelMappings.value = []
          allowedModels.value = []
        }

        loadAccountPoolModeStateFromCredentials(poolModeState, credentials, DEFAULT_POOL_MODE_RETRY_COUNT)
        loadAccountCustomErrorCodesStateFromCredentials(customErrorCodesState, credentials)
      } else if (newAccount.type === 'upstream' && newAccount.credentials) {
        const credentials = newAccount.credentials as Record<string, unknown>
        editBaseUrl.value = (credentials.base_url as string) || ''
      } else {
        const platformDefaultUrl = resolveAccountApiKeyDefaultBaseUrl(newAccount.platform)
        editBaseUrl.value = platformDefaultUrl

        // Load model mappings for OpenAI OAuth accounts
        if (newAccount.platform === 'openai' && newAccount.credentials) {
          const oauthCredentials = newAccount.credentials as Record<string, unknown>
          const existingMappings = oauthCredentials.model_mapping as Record<string, string> | undefined
          if (existingMappings && typeof existingMappings === 'object') {
            const entries = Object.entries(existingMappings)
            const isWhitelistMode = entries.length > 0 && entries.every(([from, to]) => from === to)
            if (isWhitelistMode) {
              modelRestrictionMode.value = 'whitelist'
              allowedModels.value = entries.map(([from]) => from)
              modelMappings.value = []
            } else {
              modelRestrictionMode.value = 'mapping'
              modelMappings.value = entries.map(([from, to]) => ({ from, to }))
              allowedModels.value = []
            }
          } else {
            modelRestrictionMode.value = 'whitelist'
            modelMappings.value = []
            allowedModels.value = []
          }
        } else {
          modelRestrictionMode.value = 'whitelist'
          modelMappings.value = []
          allowedModels.value = []
        }
        resetAccountPoolModeState(poolModeState, DEFAULT_POOL_MODE_RETRY_COUNT)
        resetAccountCustomErrorCodesState(customErrorCodesState)
      }
      editApiKey.value = ''
    } else {
      resetMixedChannelRisk()
      resetTempUnschedRules()
      quotaControl.reset()
    }
  },
  { immediate: true }
)

// Model mapping helpers
const addModelMapping = () => {
  modelMappings.value.push({ from: '', to: '' })
}

const removeModelMapping = (index: number) => {
  modelMappings.value.splice(index, 1)
}

const addPresetMapping = (from: string, to: string) => {
  const exists = modelMappings.value.some((m) => m.from === from)
  if (exists) {
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
  const exists = antigravityModelMappings.value.some((m) => m.from === from)
  if (exists) {
    appStore.showInfo(t('admin.accounts.mappingExists', { model: from }))
    return
  }
  antigravityModelMappings.value.push({ from, to })
}

const formatDateTimeLocal = formatDateTimeLocalInput
const parseDateTimeLocal = parseDateTimeLocalInput

// Methods
const handleClose = () => {
  resetMixedChannelRisk()
  emit('close')
}

const submitUpdateAccount = async (accountID: number, updatePayload: Record<string, unknown>) => {
  submitting.value = true
  try {
    const updatedAccount = await adminAPI.accounts.update(accountID, withConfirmFlag(updatePayload))
    appStore.showSuccess(t('admin.accounts.accountUpdated'))
    emit('updated', updatedAccount)
    handleClose()
  } catch (error: any) {
    if (
      error.status === 409 &&
      error.error === 'mixed_channel_warning' &&
      requiresMixedChannelCheck.value
    ) {
      openMixedChannelDialog({
        message: error.message,
        onConfirm: async () => submitUpdateAccount(accountID, updatePayload)
      })
      return
    }
    appStore.showError(error.message || t('admin.accounts.failedToUpdate'))
  } finally {
    submitting.value = false
  }
}

const handleSubmit = async () => {
  if (!props.account) return
  const accountID = props.account.id

  if (form.status !== 'active' && form.status !== 'inactive' && form.status !== 'error') {
    appStore.showError(t('admin.accounts.pleaseSelectStatus'))
    return
  }

  const updatePayload: Record<string, unknown> = { ...form }
  try {
    if (updatePayload.proxy_id === null) {
      updatePayload.proxy_id = 0
    }
    if (form.expires_at === null) {
      updatePayload.expires_at = 0
    }
    const lf = form.load_factor
    if (lf == null || Number.isNaN(lf) || lf <= 0) {
      updatePayload.load_factor = 0
    }
    updatePayload.auto_pause_on_expired = autoPauseOnExpired.value

    if (props.account.type === 'apikey') {
      const currentCredentials = (props.account.credentials as Record<string, unknown>) || {}
      const newBaseUrl = editBaseUrl.value.trim() || defaultBaseUrl.value
      const shouldApplyModelMapping = !(props.account.platform === 'openai' && openaiPassthroughEnabled.value)

      const newCredentials: Record<string, unknown> = {
        ...currentCredentials,
        base_url: newBaseUrl
      }

      if (editApiKey.value.trim()) {
        newCredentials.api_key = editApiKey.value.trim()
      } else if (currentCredentials.api_key) {
        newCredentials.api_key = currentCredentials.api_key
      } else {
        appStore.showError(t('admin.accounts.apiKeyIsRequired'))
        return
      }

      if (shouldApplyModelMapping) {
        const modelMapping = buildModelMappingObject(modelRestrictionMode.value, allowedModels.value, modelMappings.value)
        if (modelMapping) {
          newCredentials.model_mapping = modelMapping
        } else {
          delete newCredentials.model_mapping
        }
      } else if (currentCredentials.model_mapping) {
        newCredentials.model_mapping = currentCredentials.model_mapping
      }

      applyAccountPoolModeStateToCredentials(newCredentials, poolModeState)
      applyAccountCustomErrorCodesStateToCredentials(newCredentials, customErrorCodesState)

      applyInterceptWarmup(newCredentials, interceptWarmupRequests.value, 'edit')
      if (!applyTempUnschedConfig(newCredentials)) {
        return
      }

      updatePayload.credentials = newCredentials
    } else if (props.account.type === 'upstream') {
      const currentCredentials = (props.account.credentials as Record<string, unknown>) || {}
      const newCredentials: Record<string, unknown> = { ...currentCredentials }

      newCredentials.base_url = editBaseUrl.value.trim()

      if (editApiKey.value.trim()) {
        newCredentials.api_key = editApiKey.value.trim()
      }

      applyInterceptWarmup(newCredentials, interceptWarmupRequests.value, 'edit')
      if (!applyTempUnschedConfig(newCredentials)) {
        return
      }

      updatePayload.credentials = newCredentials
    } else {
      const currentCredentials = (props.account.credentials as Record<string, unknown>) || {}
      const newCredentials: Record<string, unknown> = { ...currentCredentials }

      applyInterceptWarmup(newCredentials, interceptWarmupRequests.value, 'edit')
      if (!applyTempUnschedConfig(newCredentials)) {
        return
      }

      updatePayload.credentials = newCredentials
    }

    if (props.account.platform === 'openai' && props.account.type === 'oauth') {
      const currentCredentials = (updatePayload.credentials as Record<string, unknown>) ||
        ((props.account.credentials as Record<string, unknown>) || {})
      const newCredentials: Record<string, unknown> = { ...currentCredentials }
      const shouldApplyModelMapping = !openaiPassthroughEnabled.value

      if (shouldApplyModelMapping) {
        const modelMapping = buildModelMappingObject(modelRestrictionMode.value, allowedModels.value, modelMappings.value)
        if (modelMapping) {
          newCredentials.model_mapping = modelMapping
        } else {
          delete newCredentials.model_mapping
        }
      } else if (currentCredentials.model_mapping) {
        newCredentials.model_mapping = currentCredentials.model_mapping
      }

      updatePayload.credentials = newCredentials
    }

    if (props.account.platform === 'antigravity') {
      const currentCredentials = (updatePayload.credentials as Record<string, unknown>) ||
        ((props.account.credentials as Record<string, unknown>) || {})
      const newCredentials: Record<string, unknown> = { ...currentCredentials }

      delete newCredentials.model_whitelist
      delete newCredentials.model_mapping

      const antigravityModelMapping = buildModelMappingObject(
        'mapping',
        [],
        antigravityModelMappings.value
      )
      if (antigravityModelMapping) {
        newCredentials.model_mapping = antigravityModelMapping
      }

      updatePayload.credentials = newCredentials
    }

    if (props.account.platform === 'antigravity') {
      const currentExtra = (props.account.extra as Record<string, unknown>) || {}
      const newExtra: Record<string, unknown> = { ...currentExtra }
      if (mixedScheduling.value) {
        newExtra.mixed_scheduling = true
      } else {
        delete newExtra.mixed_scheduling
      }
      updatePayload.extra = newExtra
    }

    if (props.account.platform === 'anthropic' && (props.account.type === 'oauth' || props.account.type === 'setup-token')) {
      const currentExtra = (props.account.extra as Record<string, unknown>) || {}
      updatePayload.extra = quotaControl.buildExtra(currentExtra)
    }

    if (props.account.platform === 'anthropic' && props.account.type === 'apikey') {
      const currentExtra = (props.account.extra as Record<string, unknown>) || {}
      const newExtra: Record<string, unknown> = { ...currentExtra }
      if (anthropicPassthroughEnabled.value) {
        newExtra.anthropic_passthrough = true
      } else {
        delete newExtra.anthropic_passthrough
      }
      updatePayload.extra = newExtra
    }

    if (props.account.platform === 'openai' && (props.account.type === 'oauth' || props.account.type === 'apikey')) {
      const currentExtra = (props.account.extra as Record<string, unknown>) || {}
      const newExtra: Record<string, unknown> = { ...currentExtra }
      const hadCodexCLIOnlyEnabled = currentExtra.codex_cli_only === true
      if (props.account.type === 'oauth') {
        newExtra.openai_oauth_responses_websockets_v2_mode = openaiOAuthResponsesWebSocketV2Mode.value
        newExtra.openai_oauth_responses_websockets_v2_enabled = isOpenAIWSModeEnabled(openaiOAuthResponsesWebSocketV2Mode.value)
      } else if (props.account.type === 'apikey') {
        newExtra.openai_apikey_responses_websockets_v2_mode = openaiAPIKeyResponsesWebSocketV2Mode.value
        newExtra.openai_apikey_responses_websockets_v2_enabled = isOpenAIWSModeEnabled(openaiAPIKeyResponsesWebSocketV2Mode.value)
      }
      delete newExtra.responses_websockets_v2_enabled
      delete newExtra.openai_ws_enabled
      if (openaiPassthroughEnabled.value) {
        newExtra.openai_passthrough = true
      } else {
        delete newExtra.openai_passthrough
        delete newExtra.openai_oauth_passthrough
      }

      if (props.account.type === 'oauth') {
        if (codexCLIOnlyEnabled.value) {
          newExtra.codex_cli_only = true
        } else if (hadCodexCLIOnlyEnabled) {
          newExtra.codex_cli_only = false
        } else {
          delete newExtra.codex_cli_only
        }
      }

      updatePayload.extra = newExtra
    }

    if (props.account.type === 'apikey') {
      const currentExtra = (updatePayload.extra as Record<string, unknown>) ||
        (props.account.extra as Record<string, unknown>) || {}
      const newExtra: Record<string, unknown> = { ...currentExtra }
      if (editQuotaLimit.value != null && editQuotaLimit.value > 0) {
        newExtra.quota_limit = editQuotaLimit.value
      } else {
        delete newExtra.quota_limit
      }
      if (editQuotaDailyLimit.value != null && editQuotaDailyLimit.value > 0) {
        newExtra.quota_daily_limit = editQuotaDailyLimit.value
      } else {
        delete newExtra.quota_daily_limit
      }
      if (editQuotaWeeklyLimit.value != null && editQuotaWeeklyLimit.value > 0) {
        newExtra.quota_weekly_limit = editQuotaWeeklyLimit.value
      } else {
        delete newExtra.quota_weekly_limit
      }
      updatePayload.extra = newExtra
    }

    updatePayload.extra = buildAccountModelScopeExtra(
      ((updatePayload.extra as Record<string, unknown>) ||
        (props.account.extra as Record<string, unknown>) ||
        undefined),
      {
        platform: props.account.platform,
        enabled: props.account.platform === 'antigravity'
          ? true
          : !(props.account.platform === 'openai' && openaiPassthroughEnabled.value),
        mode: props.account.platform === 'antigravity' ? 'mapping' : modelRestrictionMode.value,
        allowedModels: allowedModels.value,
        modelMappings: props.account.platform === 'antigravity'
          ? antigravityModelMappings.value
          : modelMappings.value
      }
    )

    const canContinue = await ensureMixedChannelConfirmed(async () => {
      await submitUpdateAccount(accountID, updatePayload)
    })
    if (!canContinue) {
      return
    }

    await submitUpdateAccount(accountID, updatePayload)
  } catch (error: any) {
    appStore.showError(error.message || t('admin.accounts.failedToUpdate'))
  }
}
</script>
