import { ref, reactive, computed, toRef, watch, nextTick, type Ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { useModelInventoryStore } from '@/stores'
import { ensureModelRegistryFresh } from '@/stores/modelRegistry'
import {
  getPresetMappingsByPlatform,
  createCommonErrorCodeOptions,
  buildModelMappingObject,
  fetchAntigravityDefaultMappings
} from '@/composables/useModelWhitelist'
import { useAuthStore } from '@/stores/auth'
import type {
  AccountManualModel,
  ProtocolGatewayProbeModel
} from '@/api/admin/accounts'
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
import { useCreateAccountSubmit } from '@/composables/useCreateAccountSubmit'
import type { GrokImportResult } from '@/api/admin/accounts'
import type {
  AccountPlatform,
  AccountType,
  Account,
  AccountAutoRenewPeriod,
  GatewayAcceptedProtocol,
  GatewayClientProfile,
  GatewayClientRoute,
  OpenAIImageProtocolMode,
  GatewayOpenAIRequestFormat,
  GatewayProtocol,
  GroupPlatform
} from '@/types'
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
  buildLocalAccountModelProbeSnapshot,
  createAccountModelProbeSnapshotDraft,
  mergeAccountModelProbeSnapshotIntoExtra,
  mergeAccountManualModelsIntoExtra,
  mergeResolvedUpstreamDraftIntoExtra,
  type AccountModelProbeSnapshotDraft,
  type AccountResolvedUpstreamDraft
} from '@/utils/accountProbeDraft'
import {
  applyAccountCustomErrorCodesStateToCredentials,
  applyAccountPoolModeStateToCredentials
} from '@/utils/accountApiKeyAdvancedSettingsForm'
import { resolveAccountApiKeyDefaultBaseUrl } from '@/utils/accountApiKeyBasicSettings'
import { buildAnthropicExtra, buildOpenAIExtra } from '@/utils/accountCreateExtras'
import {
  applyDeepSeekModelConcurrencyLimitsExtra,
  createDefaultDeepSeekModelConcurrencyLimitDraft
} from '@/utils/deepseekAccount'
import {
  resolveOpenAIImageProtocolState
} from '@/utils/openaiAccountDefaults'
import {
  resolveOpenAIOAuthDefaultAllowedModels
} from '@/utils/openaiOAuthDefaults'
import {
  DEFAULT_GATEWAY_OPENAI_IMAGE_PROTOCOL_MODE,
  DEFAULT_GATEWAY_OPENAI_REQUEST_FORMAT,
  applyProtocolGatewayOpenAIImageProtocolModeExtra,
  applyProtocolGatewayOpenAIRequestFormatExtra,
  applyProtocolGatewayGeminiBatchExtra,
  applyProtocolGatewayClaudeClientMimicExtra,
  isProtocolGatewayPlatform,
  resolveProtocolGatewayBatchRequestFormats,
  resolveEffectiveAccountPlatforms,
  resolveEffectiveAccountPlatform,
  supportsProtocolGatewayClaudeClientMimic,
  supportsProtocolGatewayGeminiBatch,
  supportsProtocolGatewayOpenAIRequestFormat
} from '@/utils/accountProtocolGateway'
import {
  normalizeGeminiAIStudioTier,
  isGeminiVertexAI,
  type GeminiBrowserOAuthType,
  type GeminiAIStudioTier,
  type GeminiOAuthType
} from '@/utils/geminiAccount'
import {
  GEMINI_API_KEY_VARIANT_VERTEX_EXPRESS,
  resolveVertexAuthBaseUrl,
  resolveVertexBaseUrl,
  type VertexAuthMode
} from '@/utils/vertexAi'
import {
  createDefaultGoogleBatchArchiveFormState,
  type GoogleBatchArchiveBillingMode
} from '@/utils/accountGoogleBatchArchive'
import {
  OPENAI_WS_MODE_CTX_POOL,
  OPENAI_WS_MODE_OFF,
  OPENAI_WS_MODE_PASSTHROUGH,
  resolveOpenAIWSModeConcurrencyHintKey,
  type OpenAIWSMode
} from '@/utils/openaiWsMode'
import {
  BAIDU_DOCUMENT_AI_DEFAULT_ASYNC_BASE_URL,
  isBaiduDocumentAIPlatform,
  parseBaiduDocumentAIDirectApiUrlsInput
} from '@/utils/baiduDocumentAI'
import type { OAuthFlowExposed } from '../oauthFlow.types'
import { useCreateAccountModalWatchers } from './watchers'
import { createCreateAccountSubmit } from './submit'
import { syncProtocolGatewaySelectedModels } from './protocolGatewayImport'
import type { CreateAccountModalEmit, CreateAccountModalProps } from './types'

export function useCreateAccountModal(props: CreateAccountModalProps, emit: CreateAccountModalEmit) {
const { t } = useI18n()
const authStore = useAuthStore()
const modelInventoryStore = useModelInventoryStore()

const oauthStepTitle = computed(() => {
  if (form.platform === 'openai') return t('admin.accounts.oauth.openai.title')
  if (form.platform === 'gemini') return t('admin.accounts.oauth.gemini.title')
  if (form.platform === 'antigravity') return t('admin.accounts.oauth.antigravity.title')
  if (form.platform === 'kiro') return t('admin.accounts.kiroAuth.title')
  return t('admin.accounts.oauth.title')
})

const appStore = useAppStore()
const showFormError = (message: string) => appStore.showError(message)
const showFormInfo = (message: string) => appStore.showInfo(message)

// OAuth composables
const oauth = useAccountOAuth() // For Anthropic OAuth
const openaiOAuth = useOpenAIOAuth({ platform: 'openai' }) // For OpenAI OAuth
const geminiOAuth = useGeminiOAuth() // For Gemini OAuth
const antigravityOAuth = useAntigravityOAuth() // For Antigravity OAuth

// Computed: current OAuth state for template binding
const currentAuthUrl = computed(() => {
  if (form.platform === 'openai') return openaiOAuth.authUrl.value
  if (form.platform === 'gemini') return geminiOAuth.authUrl.value
  if (form.platform === 'antigravity') return antigravityOAuth.authUrl.value
  return oauth.authUrl.value
})

const currentSessionId = computed(() => {
  if (form.platform === 'openai') return openaiOAuth.sessionId.value
  if (form.platform === 'gemini') return geminiOAuth.sessionId.value
  if (form.platform === 'antigravity') return antigravityOAuth.sessionId.value
  return oauth.sessionId.value
})

const currentOAuthLoading = computed(() => {
  if (form.platform === 'openai') return openaiOAuth.loading.value
  if (form.platform === 'gemini') return geminiOAuth.loading.value
  if (form.platform === 'antigravity') return antigravityOAuth.loading.value
  return oauth.loading.value
})

const currentOAuthError = computed(() => {
  if (form.platform === 'kiro') return ''
  if (form.platform === 'openai') return openaiOAuth.error.value
  if (form.platform === 'gemini') return geminiOAuth.error.value
  if (form.platform === 'antigravity') return antigravityOAuth.error.value
  return oauth.error.value
})

// Refs
const oauthFlowRef = ref<OAuthFlowExposed | null>(null)
const kiroAuthRef = ref<{ reset: () => void } | null>(null)

// State
const step = ref(1)
const autoImportModels = ref(false)
const accountCategory = ref<'oauth-based' | 'apikey' | 'vertex_ai'>('oauth-based') // UI selection for account category
const addMethod = ref<AddMethod>('oauth') // For oauth-based: 'oauth' or 'setup-token'
const gatewayProtocol = ref<GatewayProtocol>('openai')
const apiKeyBaseUrl = ref(resolveAccountApiKeyDefaultBaseUrl('anthropic'))
const apiKeyValue = ref('')
const openRouterHTTPReferer = ref('')
const openRouterTitle = ref('')
const deepSeekModelConcurrencyLimits = ref(createDefaultDeepSeekModelConcurrencyLimitDraft())
const grokSSOToken = ref('')
const grokTier = ref<'basic' | 'super' | 'heavy'>('basic')
const editQuotaLimit = ref<number | null>(null)
const editQuotaDailyLimit = ref<number | null>(null)
const editQuotaWeeklyLimit = ref<number | null>(null)
const editQuotaDailyResetMode = ref<'rolling' | 'fixed' | null>(null)
const editQuotaDailyResetHour = ref<number | null>(null)
const editQuotaWeeklyResetMode = ref<'rolling' | 'fixed' | null>(null)
const editQuotaWeeklyResetDay = ref<number | null>(null)
const editQuotaWeeklyResetHour = ref<number | null>(null)
const editQuotaResetTimezone = ref<string | null>(null)
const defaultGoogleBatchArchiveState = createDefaultGoogleBatchArchiveFormState()
const batchArchiveEnabled = ref(defaultGoogleBatchArchiveState.enabled)
const batchArchiveAutoPrefetchEnabled = ref(defaultGoogleBatchArchiveState.autoPrefetchEnabled)
const batchArchiveRetentionDays = ref(defaultGoogleBatchArchiveState.retentionDays)
const batchArchiveBillingMode = ref<GoogleBatchArchiveBillingMode>(defaultGoogleBatchArchiveState.billingMode)
const batchArchiveDownloadPriceUSD = ref(defaultGoogleBatchArchiveState.downloadPriceUSD)
const allowVertexBatchOverflow = ref(defaultGoogleBatchArchiveState.allowVertexBatchOverflow)
const acceptAIStudioBatchOverflow = ref(defaultGoogleBatchArchiveState.acceptAIStudioBatchOverflow)
const actualModelLocked = ref(true)
const modelRestrictionEnabled = ref(true)
const modelMappings = ref<ModelMapping[]>([])
const modelRestrictionMode = ref<'whitelist' | 'mapping'>('whitelist')
const allowedModels = ref<string[]>([])
const hasCustomizedOpenAIOAuthDefaults = ref(false)
const applyingOpenAIOAuthDefaults = ref(false)
const manualModels = ref<AccountManualModel[]>([])
const modelProbeSnapshot = ref<AccountModelProbeSnapshotDraft | null>(null)
const resolvedUpstream = ref<AccountResolvedUpstreamDraft | null>(null)
const protocolGatewayProbeModels = ref<ProtocolGatewayProbeModel[]>([])
const gatewayAcceptedProtocols = ref<GatewayAcceptedProtocol[]>(['openai'])
const gatewayClientProfiles = ref<GatewayClientProfile[]>([])
const gatewayClientRoutes = ref<GatewayClientRoute[]>([])
const gatewayTestProvider = ref('')
const gatewayTestModelId = ref('')
const gatewayOpenAIRequestFormat = ref<GatewayOpenAIRequestFormat>(DEFAULT_GATEWAY_OPENAI_REQUEST_FORMAT)
const gatewayBatchEnabled = ref(false)
const claudeCodeMimicEnabled = ref(false)
const claudeTLSFingerprintEnabled = ref(false)
const claudeSessionIDMaskingEnabled = ref(false)
const poolModeState = reactive(createDefaultAccountPoolModeState(DEFAULT_POOL_MODE_RETRY_COUNT))
const customErrorCodesState = reactive(createDefaultAccountCustomErrorCodesState())
const interceptWarmupRequests = ref(false)
const autoPauseOnExpired = ref(true)
const autoRenewEnabled = ref(false)
const autoRenewPeriod = ref<AccountAutoRenewPeriod>('month')
const expiryProbeExtensionDays = ref(1)
const openaiPassthroughEnabled = ref(false)
const openAIImageProtocolMode = ref<OpenAIImageProtocolMode>('native')
const openAIImageCompatAllowed = ref(true)
const openAIImageProtocolTouched = ref(false)
const openaiOAuthResponsesWebSocketV2Mode = ref<OpenAIWSMode>(OPENAI_WS_MODE_OFF)
const openaiAPIKeyResponsesWebSocketV2Mode = ref<OpenAIWSMode>(OPENAI_WS_MODE_OFF)
const codexCLIOnlyEnabled = ref(false)
const anthropicPassthroughEnabled = ref(false)
const gatewayOpenAIImageProtocolMode = ref<OpenAIImageProtocolMode>(DEFAULT_GATEWAY_OPENAI_IMAGE_PROTOCOL_MODE)
const mixedScheduling = ref(false) // For antigravity accounts: enable mixed scheduling
const antigravityModelRestrictionMode = ref<'whitelist' | 'mapping'>('whitelist')
const antigravityWhitelistModels = ref<string[]>([])
const antigravityAccountType = ref<'oauth' | 'upstream'>('oauth') // For antigravity: oauth or upstream
const upstreamBaseUrl = ref('') // For upstream type: base URL
const upstreamApiKey = ref('') // For upstream type: API key
const geminiVertexAuthMode = ref<VertexAuthMode>('service_account')
const geminiVertexProjectId = ref('')
const geminiVertexLocation = ref('')
const geminiVertexServiceAccountJson = ref('')
const geminiVertexApiKey = ref('')
const geminiVertexAccessToken = ref('')
const geminiVertexExpiresAtInput = ref('')
const geminiVertexBaseUrl = ref('')
const baiduDocumentAIAccessToken = ref('')
const baiduDocumentAIAsyncBaseUrl = ref(BAIDU_DOCUMENT_AI_DEFAULT_ASYNC_BASE_URL)
const baiduDocumentAIDirectApiUrlsText = ref('')
const oauthDraftCredentials = ref<Record<string, unknown>>({})
const oauthDraftExtra = ref<Record<string, unknown>>({})
const apiKeyProbeCredentials = computed<Record<string, unknown>>(() => {
  const credentials: Record<string, unknown> = {
    api_key: apiKeyValue.value.trim(),
    base_url: apiKeyBaseUrl.value.trim() || resolveAccountApiKeyDefaultBaseUrl(form.platform, gatewayProtocol.value)
  }
  if (form.platform === 'openrouter') {
    if (openRouterHTTPReferer.value.trim()) {
      credentials.http_referer = openRouterHTTPReferer.value.trim()
    }
    if (openRouterTitle.value.trim()) {
      credentials.openrouter_title = openRouterTitle.value.trim()
    }
  }
  if (shouldPersistGeminiTierId.value) {
    credentials.tier_id = normalizeGeminiAIStudioTier(geminiTierAIStudio.value)
  }
  return credentials
})
const upstreamProbeCredentials = computed<Record<string, unknown>>(() => ({
  api_key: upstreamApiKey.value.trim(),
  base_url: upstreamBaseUrl.value.trim()
}))
const vertexProbeCredentials = computed<Record<string, unknown>>(() => {
  const baseUrl = geminiVertexBaseUrl.value.trim() || resolveVertexAuthBaseUrl(
    geminiVertexAuthMode.value,
    geminiVertexLocation.value
  )
  if (geminiVertexAuthMode.value === 'express_api_key') {
    return {
      gemini_api_variant: GEMINI_API_KEY_VARIANT_VERTEX_EXPRESS,
      api_key: geminiVertexApiKey.value.trim(),
      base_url: baseUrl
    }
  }
  const credentials: Record<string, unknown> = {
    oauth_type: 'vertex_ai',
    vertex_project_id: geminiVertexProjectId.value.trim(),
    vertex_location: geminiVertexLocation.value.trim(),
    base_url: baseUrl
  }
  if (geminiVertexServiceAccountJson.value.trim()) {
    credentials.vertex_service_account_json = geminiVertexServiceAccountJson.value.trim()
  }
  if (geminiVertexAccessToken.value.trim()) {
    credentials.access_token = geminiVertexAccessToken.value.trim()
  }
  return credentials
})
const isApiKeyProbeReady = computed(() => Boolean(apiKeyValue.value.trim()))
const isUpstreamProbeReady = computed(() => Boolean(upstreamApiKey.value.trim()))
const oauthDraftProbeReady = computed(() => Object.keys(oauthDraftCredentials.value).length > 0)
const isVertexProbeReady = computed(() => {
  if (geminiVertexAuthMode.value === 'express_api_key') {
    return Boolean(geminiVertexApiKey.value.trim())
  }
  return Boolean(
    geminiVertexProjectId.value.trim() &&
      geminiVertexLocation.value.trim() &&
      (geminiVertexServiceAccountJson.value.trim() || geminiVertexAccessToken.value.trim())
  )
})
const showCommonApiKeySection = computed(() =>
  form.type === 'apikey' &&
  form.platform !== 'antigravity' &&
  !isBaiduDocumentAISelected.value &&
  !(form.platform === 'gemini' && accountCategory.value === 'vertex_ai')
)
const showApiKeyModelScopeEditor = computed(() =>
  showCommonApiKeySection.value &&
  form.platform !== 'protocol_gateway' &&
  effectivePlatform.value !== 'antigravity'
)
const showDeepSeekConcurrencyEditor = computed(() =>
  showCommonApiKeySection.value && effectivePlatform.value === 'deepseek'
)
const showStandaloneModelScopeEditor = computed(() => {
  if (form.platform === 'antigravity') {
    return false
  }
  if (form.platform === 'protocol_gateway') {
    return true
  }
  if (isBaiduDocumentAISelected.value) {
    return true
  }
  if (showApiKeyModelScopeEditor.value) {
    return false
  }
  if (form.platform === 'grok') {
    return form.type === 'sso'
  }
  return accountCategory.value === 'oauth-based' || accountCategory.value === 'vertex_ai'
})
const showQuotaLimitSection = computed(() => true)
const showGeminiAIStudioBatchArchiveEditor = computed(() =>
  form.platform === 'gemini' && accountCategory.value === 'apikey'
)
const showGeminiVertexBatchArchiveEditor = computed(() =>
  form.platform === 'gemini' &&
  accountCategory.value === 'vertex_ai' &&
  geminiVertexAuthMode.value !== 'express_api_key'
)
const showOAuthFinalizeStep = computed(() =>
  isOAuthFlow.value && form.platform === 'kiro'
)
const showOAuthFinalizeProbeEditor = computed(() =>
  showOAuthFinalizeStep.value && step.value === 3 && oauthDraftProbeReady.value
)
const antigravityModelMappings = ref<ModelMapping[]>([])
const antigravityPresetMappings = computed(() => getPresetMappingsByPlatform('antigravity'))
const getModelMappingKey = createStableObjectKeyResolver<ModelMapping>('create-model-mapping')
const getAntigravityModelMappingKey = createStableObjectKeyResolver<ModelMapping>('create-antigravity-model-mapping')
const geminiOAuthType = ref<GeminiOAuthType>('google_one')
const geminiAIStudioOAuthEnabled = ref(false)

const showAdvancedOAuth = ref(false)
const showGeminiHelpDialog = ref(false)
const quotaControl = useAnthropicQuotaControl()
const quotaControlState = quotaControl.state
const umqModeOptions = quotaControl.umqModeOptions

// Gemini tier selection (used as fallback when auto-detection is unavailable/fails)
const geminiTierGoogleOne = ref<'google_one_free' | 'google_ai_pro' | 'google_ai_ultra'>('google_one_free')
const geminiTierGcp = ref<'gcp_standard' | 'gcp_enterprise'>('gcp_standard')
const geminiTierAIStudio = ref<GeminiAIStudioTier>('aistudio_free')
const effectivePlatform = computed<GroupPlatform>(() => {
  const platform = resolveEffectiveAccountPlatform(form.platform, gatewayProtocol.value)
  return platform === 'protocol_gateway' ? 'openai' : platform
})
const effectiveGroupPlatforms = computed<GroupPlatform[] | undefined>(() => {
  if (!isProtocolGatewayPlatform(form.platform)) {
    return undefined
  }
  return resolveEffectiveAccountPlatforms(
    form.platform,
    gatewayProtocol.value,
    gatewayAcceptedProtocols.value
  ) as GroupPlatform[]
})
const showProtocolGatewayClaudeMimicEditor = computed(() =>
  supportsProtocolGatewayClaudeClientMimic({
    platform: form.platform,
    type: form.type,
    gatewayProtocol: gatewayProtocol.value,
    acceptedProtocols: gatewayAcceptedProtocols.value
  })
)
const showProtocolGatewayBatchEditor = computed(() =>
  supportsProtocolGatewayGeminiBatch({
    platform: form.platform,
    type: form.type,
    gatewayProtocol: gatewayProtocol.value,
    acceptedProtocols: gatewayAcceptedProtocols.value
  })
)
const showProtocolGatewayOpenAIRequestFormatEditor = computed(() =>
  supportsProtocolGatewayOpenAIRequestFormat({
    platform: form.platform,
    type: form.type,
    gatewayProtocol: gatewayProtocol.value,
    acceptedProtocols: gatewayAcceptedProtocols.value
  })
)
const protocolGatewayBatchRequestFormats = computed(() =>
  resolveProtocolGatewayBatchRequestFormats({
    gatewayProtocol: gatewayProtocol.value,
    acceptedProtocols: gatewayAcceptedProtocols.value
  })
)

const geminiSelectedTier = computed(() => {
  if (effectivePlatform.value !== 'gemini') return ''
  if (accountCategory.value === 'apikey') return geminiTierAIStudio.value
  if (accountCategory.value === 'vertex_ai') return ''
  switch (geminiOAuthType.value) {
    case 'google_one':
      return geminiTierGoogleOne.value
    case 'code_assist':
      return geminiTierGcp.value
    default:
      return geminiTierAIStudio.value
  }
})
const shouldPersistGeminiTierId = computed(() =>
  form.platform === 'gemini' && accountCategory.value === 'apikey'
)

const openAIWSModeOptions = computed(() => [
  { value: OPENAI_WS_MODE_OFF, label: t('admin.accounts.openai.wsModeOff') },
  { value: OPENAI_WS_MODE_CTX_POOL, label: t('admin.accounts.openai.wsModeCtxPool') },
  { value: OPENAI_WS_MODE_PASSTHROUGH, label: t('admin.accounts.openai.wsModePassthrough') }
])

const openaiResponsesWebSocketV2Mode = computed({
  get: () => {
    if (effectivePlatform.value === 'openai' && accountCategory.value === 'apikey') {
      return openaiAPIKeyResponsesWebSocketV2Mode.value
    }
    return openaiOAuthResponsesWebSocketV2Mode.value
  },
  set: (mode: OpenAIWSMode) => {
    if (effectivePlatform.value === 'openai' && accountCategory.value === 'apikey') {
      openaiAPIKeyResponsesWebSocketV2Mode.value = mode
      return
    }
    openaiOAuthResponsesWebSocketV2Mode.value = mode
  }
})

const openAIWSModeConcurrencyHintKey = computed(() =>
  resolveOpenAIWSModeConcurrencyHintKey(openaiResponsesWebSocketV2Mode.value)
)
const commonErrorCodeOptions = computed(() => createCommonErrorCodeOptions(t))

const isOpenAIModelRestrictionDisabled = computed(() =>
  effectivePlatform.value === 'openai' && openaiPassthroughEnabled.value
)

const geminiHelpLinks = {
  apiKey: 'https://aistudio.google.com/app/apikey',
  gcpProject: 'https://console.cloud.google.com/welcome/new'
}

// Computed: current preset mappings based on platform
const presetMappings = computed(() => getPresetMappingsByPlatform(effectivePlatform.value))

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

const isBaiduDocumentAISelected = computed(() => isBaiduDocumentAIPlatform(form.platform))

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
  currentPlatform: () => effectivePlatform.value,
  buildCheckPayload: () => ({
    platform: form.platform,
    gateway_protocol: isProtocolGatewayPlatform(form.platform) ? gatewayProtocol.value : undefined,
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
  if (form.platform === 'gemini' && (accountCategory.value === 'vertex_ai' || isGeminiVertexAI(geminiOAuthType.value))) {
    return false
  }
  if (form.platform === 'grok') {
    return false
  }
  return accountCategory.value === 'oauth-based'
})

const isManualInputMethod = computed(() => {
  if (form.platform === 'kiro') {
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
  if (form.platform === 'kiro') {
    return false
  }
  if (form.platform === 'openai') {
    return Boolean(authCode.trim() && openaiOAuth.sessionId.value && !openaiOAuth.loading.value)
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

const selectedProtocolGatewayMissingModels = computed(() => {
  if (!isProtocolGatewayPlatform(form.platform)) {
    return []
  }
  const selected = new Set(allowedModels.value)
  return protocolGatewayProbeModels.value.filter(
    (model) => model.registry_state === 'missing' && selected.has(model.id)
  )
})

const resetProtocolGatewayClaudeMimicState = () => {
  claudeCodeMimicEnabled.value = false
  claudeTLSFingerprintEnabled.value = false
  claudeSessionIDMaskingEnabled.value = false
}

const applyOpenAIImageProtocolDefaults = (planType?: string | null, force = false) => {
  if (form.platform !== 'openai') {
    openAIImageProtocolTouched.value = false
    openAIImageProtocolMode.value = 'native'
    openAIImageCompatAllowed.value = true
    return
  }

  const nextState = resolveOpenAIImageProtocolState({
    accountCategory: accountCategory.value,
    planType
  })
  openAIImageCompatAllowed.value = nextState.compatAllowed
  if (!nextState.compatAllowed) {
    openAIImageProtocolMode.value = 'native'
    return
  }
  if (force || !openAIImageProtocolTouched.value) {
    openAIImageProtocolMode.value = nextState.mode
  }
}

const handleOpenAIImageProtocolModeChange = (value: OpenAIImageProtocolMode) => {
  openAIImageProtocolTouched.value = true
  if (!openAIImageCompatAllowed.value && value === 'compat') {
    openAIImageProtocolMode.value = 'native'
    return
  }
  openAIImageProtocolMode.value = value
}

const resetOpenAIOAuthDefaultSelection = () => {
  hasCustomizedOpenAIOAuthDefaults.value = false
}

const markOpenAIOAuthDefaultsCustomized = () => {
  if (applyingOpenAIOAuthDefaults.value) {
    return
  }
  if (form.platform !== 'openai' || accountCategory.value !== 'oauth-based') {
    return
  }
  hasCustomizedOpenAIOAuthDefaults.value = true
}

const applyOpenAIOAuthPresetModels = (
  planType?: string | null,
  proMultiplier?: number | null,
  force = false
) => {
  if (form.platform !== 'openai' || accountCategory.value !== 'oauth-based') {
    return
  }
  if (!force && hasCustomizedOpenAIOAuthDefaults.value) {
    return
  }
  applyingOpenAIOAuthDefaults.value = true
  modelRestrictionEnabled.value = true
  modelRestrictionMode.value = 'whitelist'
  allowedModels.value = resolveOpenAIOAuthDefaultAllowedModels({
    planType,
    proMultiplier
  })
  modelMappings.value = []
  openAIImageProtocolTouched.value = false
  applyOpenAIImageProtocolDefaults(planType, true)
  void nextTick(() => {
    applyingOpenAIOAuthDefaults.value = false
  })
}

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
  if (createdAccounts.length > 0 && isProtocolGatewayPlatform(form.platform)) {
    await syncProtocolGatewaySelectedModels({
      createdAccount: createdAccounts[0],
      selectedModels: selectedProtocolGatewayMissingModels.value,
      emitModelsImported: (result) => emit('models-imported', result),
      invalidateModelInventory: () => modelInventoryStore.invalidate(),
      showPartialWarning: (failed) =>
        appStore.showWarning(t('admin.accounts.protocolGateway.probeImportPartial', { failed })),
      showFailedWarning: (message) =>
        appStore.showWarning(message || t('admin.accounts.protocolGateway.probeImportFailed'))
    })
    return
  }
  if (createdAccounts.length === 0 || !autoImportModels.value) {
    return
  }
  appStore.showInfo(t('admin.accounts.accountCreated'))
}

const handleClose = () => {
  resetMixedChannelRisk()
  emit('close')
}

const { resetForm } = useCreateAccountReset({
  step,
  form,
  autoImportModels,
  accountCategory,
  addMethod,
  gatewayProtocol,
  apiKeyBaseUrl,
  apiKeyValue,
  openRouterHTTPReferer,
  openRouterTitle,
  grokSSOToken,
  grokTier,
  editQuotaLimit,
  editQuotaDailyLimit,
  editQuotaWeeklyLimit,
  editQuotaDailyResetMode,
  editQuotaDailyResetHour,
  editQuotaWeeklyResetMode,
  editQuotaWeeklyResetDay,
  editQuotaWeeklyResetHour,
  editQuotaResetTimezone,
  batchArchiveEnabled,
  batchArchiveAutoPrefetchEnabled,
  batchArchiveRetentionDays,
  batchArchiveBillingMode,
  batchArchiveDownloadPriceUSD,
  allowVertexBatchOverflow,
  acceptAIStudioBatchOverflow,
  actualModelLocked,
  modelMappings,
  modelRestrictionMode,
  allowedModels,
  manualModels,
  modelProbeSnapshot,
  resolvedUpstream,
  oauthDraftCredentials,
  oauthDraftExtra,
  protocolGatewayProbedModels: protocolGatewayProbeModels as unknown as Ref<Array<Record<string, unknown>>>,
  gatewayAcceptedProtocols,
  gatewayClientProfiles,
  gatewayClientRoutes,
  gatewayOpenAIRequestFormat,
  gatewayBatchEnabled,
  claudeCodeMimicEnabled,
  claudeTLSFingerprintEnabled,
  claudeSessionIDMaskingEnabled,
  loadAntigravityDefaultMappings,
  poolModeState,
  customErrorCodesState,
  interceptWarmupRequests,
  autoPauseOnExpired,
  autoRenewEnabled,
  autoRenewPeriod,
  expiryProbeExtensionDays,
  openaiPassthroughEnabled,
  openAIImageProtocolMode,
  openAIImageCompatAllowed,
  gatewayOpenAIImageProtocolMode,
  openaiOAuthResponsesWebSocketV2Mode,
  openaiAPIKeyResponsesWebSocketV2Mode,
  codexCLIOnlyEnabled,
  anthropicPassthroughEnabled,
  quotaControlReset: () => quotaControl.reset(),
  antigravityAccountType,
  upstreamBaseUrl,
  upstreamApiKey,
  geminiVertexAuthMode,
  geminiVertexProjectId,
  geminiVertexLocation,
  geminiVertexServiceAccountJson,
  geminiVertexApiKey,
  geminiVertexAccessToken,
  geminiVertexExpiresAtInput,
  geminiVertexBaseUrl,
  baiduDocumentAIAccessToken,
  baiduDocumentAIAsyncBaseUrl,
  baiduDocumentAIDirectApiUrlsText,
  resetTempUnschedRules,
  geminiOAuthType,
  geminiTierGoogleOne,
  geminiTierGcp,
  geminiTierAIStudio,
  oauthReset: () => oauth.resetState(),
  openaiOAuthReset: () => openaiOAuth.resetState(),
  geminiOAuthReset: () => geminiOAuth.resetState(),
  antigravityOAuthReset: () => antigravityOAuth.resetState(),
  oauthFlowReset: () => oauthFlowRef.value?.reset(),
  kiroImportReset: () => kiroAuthRef.value?.reset(),
  resetMixedChannelRisk
})

const handleGrokImportCompleted = (result: GrokImportResult) => {
  if (result.created > 0) {
    emit('created')
  }
}

const goBackToBasicInfo = () => {
  if (showOAuthFinalizeStep.value && step.value === 3) {
    step.value = 2
    return
  }
  step.value = 1
  oauth.resetState()
  openaiOAuth.resetState()
  geminiOAuth.resetState()
  antigravityOAuth.resetState()
  oauthFlowRef.value?.reset?.()
  kiroAuthRef.value?.reset?.()
}

watch(isOAuthFlow, (enabled) => {
  if (enabled) {
    return
  }
  if (step.value !== 2) {
    return
  }
  goBackToBasicInfo()
})

const handleGenerateUrl = async () => {
  if (form.platform === 'openai') {
    await openaiOAuth.generateAuthUrl(form.proxy_id)
  } else if (form.platform === 'gemini') {
    await geminiOAuth.generateAuthUrl(
      form.proxy_id,
      oauthFlowRef.value?.projectId,
      geminiOAuthType.value as GeminiBrowserOAuthType,
      geminiSelectedTier.value
    )
  } else if (form.platform === 'antigravity') {
    await antigravityOAuth.generateAuthUrl(form.proxy_id)
  } else {
    await oauth.generateAuthUrl(addMethod.value, form.proxy_id)
  }
}

const handleValidateRefreshToken = (rt: string) => {
  if (form.platform === 'openai') {
    handleOpenAIValidateRT(rt)
  } else if (form.platform === 'antigravity') {
    handleAntigravityValidateRT(rt)
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
      oauthType: geminiOAuthType.value as GeminiBrowserOAuthType,
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
      return handleOpenAIExchange(authCode)
    case 'gemini':
      return handleGeminiExchange(authCode)
    case 'antigravity':
      return handleAntigravityExchange(authCode)
    default:
      return handleAnthropicExchange(authCode)
  }
}

const probeExtraForEditor = computed(() => buildProbeExtra())

const buildProbeExtra = (base?: Record<string, unknown>) =>
  mergeResolvedUpstreamDraftIntoExtra(
    mergeAccountModelProbeSnapshotIntoExtra(
      mergeAccountManualModelsIntoExtra(
        base,
        manualModels.value,
        isProtocolGatewayPlatform(form.platform)
      ),
      modelProbeSnapshot.value
    ),
    resolvedUpstream.value
  )

const modalContext = {
  BAIDU_DOCUMENT_AI_DEFAULT_ASYNC_BASE_URL, GEMINI_API_KEY_VARIANT_VERTEX_EXPRESS, acceptAIStudioBatchOverflow, accountCategory, addMethod, allowVertexBatchOverflow, allowedModels, anthropicPassthroughEnabled, antigravityAccountType, antigravityModelMappings,
  antigravityModelRestrictionMode, antigravityOAuth, antigravityWhitelistModels, apiKeyBaseUrl, apiKeyValue, appStore, applyAccountCustomErrorCodesStateToCredentials, applyAccountPoolModeStateToCredentials, applyDeepSeekModelConcurrencyLimitsExtra, applyInterceptWarmup,
  applyOpenAIImageProtocolDefaults, applyProtocolGatewayClaudeClientMimicExtra, applyProtocolGatewayGeminiBatchExtra, applyProtocolGatewayOpenAIImageProtocolModeExtra, applyProtocolGatewayOpenAIRequestFormatExtra, applyTempUnschedConfig, autoPauseOnExpired, autoRenewEnabled, autoRenewPeriod, baiduDocumentAIAccessToken, baiduDocumentAIAsyncBaseUrl, baiduDocumentAIDirectApiUrlsText,
  batchArchiveAutoPrefetchEnabled, batchArchiveBillingMode, batchArchiveDownloadPriceUSD, batchArchiveEnabled, batchArchiveRetentionDays, buildAnthropicExtra, buildLocalAccountModelProbeSnapshot, buildModelMappingObject, buildOpenAIExtra, buildTempUnschedPayload,
  claudeCodeMimicEnabled, claudeSessionIDMaskingEnabled, claudeTLSFingerprintEnabled, codexCLIOnlyEnabled, computed, createAccountModelProbeSnapshotDraft, customErrorCodesState, deepSeekModelConcurrencyLimits, editQuotaDailyLimit, editQuotaDailyResetHour,
  editQuotaDailyResetMode, editQuotaLimit, editQuotaResetTimezone, editQuotaWeeklyLimit, editQuotaWeeklyResetDay, editQuotaWeeklyResetHour, editQuotaWeeklyResetMode, effectivePlatform, emit, ensureMixedChannelConfirmed,
  expiryProbeExtensionDays, form, gatewayAcceptedProtocols, gatewayBatchEnabled, gatewayClientProfiles, gatewayClientRoutes, gatewayOpenAIImageProtocolMode, gatewayOpenAIRequestFormat, gatewayProtocol, gatewayTestModelId,
  gatewayTestProvider, geminiTierAIStudio, geminiVertexApiKey, geminiVertexAuthMode, geminiVertexBaseUrl, geminiVertexLocation, geminiVertexProjectId, geminiVertexServiceAccountJson, grokSSOToken, grokTier,
  handleClose, hasCustomizedOpenAIOAuthDefaults, interceptWarmupRequests, isBaiduDocumentAISelected, isOAuthFlow, isOpenAIModelRestrictionDisabled, isProtocolGatewayPlatform, manualModels, maybeImportCreatedAccounts, mergeAccountManualModelsIntoExtra,
  mergeAccountModelProbeSnapshotIntoExtra, mergeResolvedUpstreamDraftIntoExtra, mixedScheduling, modelMappings, modelProbeSnapshot, modelRestrictionEnabled, modelRestrictionMode, normalizeGeminiAIStudioTier, oauth, oauthDraftCredentials,
  oauthDraftExtra, oauthDraftProbeReady, oauthFlowRef, openAIImageCompatAllowed, openAIImageProtocolMode, openMixedChannelDialog, openRouterHTTPReferer, openRouterTitle, openaiAPIKeyResponsesWebSocketV2Mode, openaiOAuth,
  openaiOAuthResponsesWebSocketV2Mode, openaiPassthroughEnabled, parseBaiduDocumentAIDirectApiUrlsInput, poolModeState, quotaControl, requiresMixedChannelCheck, resolveAccountApiKeyDefaultBaseUrl, resolveVertexAuthBaseUrl, resolveVertexBaseUrl, resolvedUpstream,
  shouldPersistGeminiTierId, showOAuthFinalizeStep, step, t, tempUnschedEnabled, toRef, upstreamApiKey, upstreamBaseUrl, useCreateAccountAnthropicCookieAuth, useCreateAccountAnthropicExchange,
  useCreateAccountAntigravityHandlers, useCreateAccountOpenAIExchange, useCreateAccountOpenAIRefreshTokenValidation, useCreateAccountSubmit, withConfirmFlag, DEFAULT_GATEWAY_OPENAI_IMAGE_PROTOCOL_MODE, DEFAULT_GATEWAY_OPENAI_REQUEST_FORMAT, OPENAI_WS_MODE_OFF, actualModelLocked, applyOpenAIOAuthPresetModels,
  autoImportModels, createDefaultDeepSeekModelConcurrencyLimitDraft, ensureModelRegistryFresh, geminiAIStudioOAuthEnabled, geminiOAuth, geminiOAuthType, geminiVertexAccessToken, geminiVertexExpiresAtInput, isBaiduDocumentAIPlatform, kiroAuthRef,
  loadAntigravityDefaultMappings, markOpenAIOAuthDefaultsCustomized, nextTick, openAIImageProtocolTouched, props, protocolGatewayProbeModels, resetForm, resetOpenAIOAuthDefaultSelection, resetProtocolGatewayClaudeMimicState, showProtocolGatewayBatchEditor,
  showProtocolGatewayClaudeMimicEditor, showProtocolGatewayOpenAIRequestFormatEditor, watch, authStore, oauthStepTitle, showFormError, showFormInfo, currentAuthUrl, currentSessionId, currentOAuthLoading,
  currentOAuthError, apiKeyProbeCredentials, upstreamProbeCredentials, vertexProbeCredentials, isApiKeyProbeReady, isUpstreamProbeReady, isVertexProbeReady, showCommonApiKeySection, showApiKeyModelScopeEditor, showDeepSeekConcurrencyEditor,
  showStandaloneModelScopeEditor, showQuotaLimitSection, showGeminiAIStudioBatchArchiveEditor, showGeminiVertexBatchArchiveEditor, showOAuthFinalizeProbeEditor, antigravityPresetMappings, getModelMappingKey, getAntigravityModelMappingKey, showAdvancedOAuth, showGeminiHelpDialog,
  quotaControlState, umqModeOptions, geminiTierGoogleOne, geminiTierGcp, effectiveGroupPlatforms, protocolGatewayBatchRequestFormats, openAIWSModeOptions, openaiResponsesWebSocketV2Mode, openAIWSModeConcurrencyHintKey, commonErrorCodeOptions,
  geminiHelpLinks, presetMappings, tempUnschedRules, tempUnschedPresets, getTempUnschedRuleKey, addTempUnschedRule, removeTempUnschedRule, moveTempUnschedRule, showMixedChannelWarning, mixedChannelWarningMessageText,
  handleMixedChannelConfirm, handleMixedChannelCancel, isManualInputMethod, expiresAtInput, canExchangeCode, handleOpenAIImageProtocolModeChange, addModelMapping, removeModelMapping, addPresetMapping, addAntigravityModelMapping,
  removeAntigravityModelMapping, addAntigravityPresetMapping, handleGrokImportCompleted, goBackToBasicInfo, handleGenerateUrl, handleValidateRefreshToken, handleExchangeCode, probeExtraForEditor, buildProbeExtra, DEFAULT_POOL_MODE_RETRY_COUNT,
  MAX_POOL_MODE_RETRY_COUNT
}
const submitBindings = createCreateAccountSubmit(modalContext)
const { createAccountAndFinish, handleAnthropicExchange, handleOpenAIExchange, handleOpenAIValidateRT, handleAntigravityValidateRT, handleAntigravityExchange } = submitBindings

useCreateAccountModalWatchers(modalContext)

return {
  ...modalContext,
  ...submitBindings
}}
