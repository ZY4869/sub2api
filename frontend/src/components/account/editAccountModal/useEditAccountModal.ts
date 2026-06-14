import { ref, reactive, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { adminAPI } from '@/api/admin'
import { useAnthropicQuotaControl } from '@/composables/useAnthropicQuotaControl'
import { useAccountMixedChannelRisk } from '@/composables/useAccountMixedChannelRisk'
import { useAccountTempUnschedRules } from '@/composables/useAccountTempUnschedRules'
import type { AccountManualModel } from '@/api/admin/accounts'
import type { AccountPlatform, AccountTier, GatewayProtocol, GroupPlatform } from '@/types'
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
  applyDeepSeekModelConcurrencyLimitsExtra,
  createDefaultDeepSeekModelConcurrencyLimitDraft,
  readDeepSeekModelConcurrencyLimitDraft
} from '@/utils/deepseekAccount'
import {
  OPENAI_WS_MODE_CTX_POOL,
  OPENAI_WS_MODE_OFF,
  OPENAI_WS_MODE_PASSTHROUGH,
  isOpenAIWSModeEnabled,
  resolveOpenAIWSModeConcurrencyHintKey,
  type OpenAIWSMode,
  resolveOpenAIWSModeFromExtra
} from '@/utils/openaiWsMode'
import {
  getPresetMappingsByPlatform,
  createCommonErrorCodeOptions,
  buildModelMappingObject
} from '@/composables/useModelWhitelist'
import { ensureModelRegistryFresh } from '@/stores/modelRegistry'
import { buildAccountModelScopeExtra, loadAccountModelScopeDraft } from '@/utils/accountModelScope'
import type { ProtocolGatewayProbeModel } from '@/api/admin/accounts'
import {
  GEMINI_API_KEY_VARIANT_VERTEX_EXPRESS,
  resolveVertexAuthBaseUrl,
  resolveVertexBaseUrl,
  type VertexAuthMode
} from '@/utils/vertexAi'
import {
  grokDefaultModelIdsForTier,
  grokDefaultModelMappingForTier,
  mappingRecordToRows,
  normalizeGrokTier,
  type GrokTier
} from '@/utils/grokAccount'
import {
  DEFAULT_GATEWAY_OPENAI_IMAGE_PROTOCOL_MODE,
  DEFAULT_GATEWAY_OPENAI_REQUEST_FORMAT,
  applyProtocolGatewayOpenAIImageProtocolModeExtra,
  applyProtocolGatewayOpenAIRequestFormatExtra,
  applyProtocolGatewayGeminiBatchExtra,
  applyProtocolGatewayClaudeClientMimicExtra,
  PROTOCOL_GATEWAY_PROTOCOLS,
  isProtocolGatewayPlatform,
  normalizeGatewayBatchEnabled,
  normalizeGatewayAcceptedProtocols,
  normalizeGatewayClientProfile,
  normalizeGatewayClientRoutes,
  resolveAccountGatewayOpenAIRequestFormat,
  resolveAccountGatewayOpenAIImageProtocolMode,
  resolveProtocolGatewayBatchRequestFormats,
  resolveAccountGatewayProtocol,
  resolveEffectiveAccountPlatform,
  resolveEffectiveAccountPlatforms,
  resolveGatewayProtocolDescriptor,
  supportsProtocolGatewayClaudeClientMimic,
  supportsProtocolGatewayGeminiBatch,
  supportsProtocolGatewayOpenAIRequestFormat
} from '@/utils/accountProtocolGateway'
import {
  normalizeGeminiAIStudioTier,
  normalizeGeminiOAuthType,
  isGeminiVertexAI,
  type GeminiAIStudioTier,
  type GeminiOAuthType
} from '@/utils/geminiAccount'
import { resolveOpenAIImageProtocolState } from '@/utils/openaiAccountDefaults'
import {
  applyAccountTierToExtra,
  defaultAccountTierForPlatform,
  normalizeAccountTier,
  resolveAccountTierCapacity
} from '@/utils/accountTier'
import {
  applyGoogleBatchArchiveExtra,
  createDefaultGoogleBatchArchiveFormState,
  readGoogleBatchArchiveFormState,
  resolveGoogleBatchArchiveTargetKind,
  type GoogleBatchArchiveBillingMode
} from '@/utils/accountGoogleBatchArchive'
import {
  BAIDU_DOCUMENT_AI_DEFAULT_ASYNC_BASE_URL,
  isBaiduDocumentAIPlatform,
  parseBaiduDocumentAIDirectApiUrlsInput,
  stringifyBaiduDocumentAIDirectApiUrls
} from '@/utils/baiduDocumentAI'
import type {
  AccountAutoRenewPeriod,
  GatewayAcceptedProtocol,
  GatewayClientProfile,
  GatewayClientRoute,
  OpenAIImageProtocolMode,
  GatewayOpenAIRequestFormat
} from '@/types'
import { formatModelDisplayName } from '@/utils/modelDisplayName'
import {
  buildLocalAccountModelProbeSnapshot,
  deriveConfiguredAccountModelIds,
  mergeAccountModelProbeSnapshotIntoExtra,
  mergeAccountManualModelsIntoExtra,
  mergeResolvedUpstreamDraftIntoExtra,
  readAccountModelProbeSnapshot,
  readAccountManualModelsFromExtra,
  readAccountResolvedUpstreamDraft,
  type AccountModelProbeSnapshotDraft,
  type AccountResolvedUpstreamDraft
} from '@/utils/accountProbeDraft'
import { useEditAccountModalWatchers } from './watchers'
import { createEditAccountSubmit } from './submit'
import type { EditAccountModalEmit, EditAccountModalProps } from './types'

interface GatewayProtocolOption extends Record<string, unknown> {
  value: GatewayProtocol
  label: string
  requestFormatsText: string
  iconPlatform: AccountPlatform
}

export function useEditAccountModal(props: EditAccountModalProps, emit: EditAccountModalEmit) {
const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const showFormError = (message: string) => appStore.showError(message)
const showFormInfo = (message: string) => appStore.showInfo(message)

const antigravityPresetMappings = computed(() => getPresetMappingsByPlatform('antigravity'))

// State
const submitting = ref(false)
const gatewayProtocol = ref<GatewayProtocol>('openai')
const isInitializingGatewayProtocol = ref(false)
const editBaseUrl = ref(resolveAccountApiKeyDefaultBaseUrl('anthropic'))
const editApiKey = ref('')
const editOpenRouterHTTPReferer = ref('')
const editOpenRouterTitle = ref('')
const deepSeekModelConcurrencyLimits = ref(createDefaultDeepSeekModelConcurrencyLimitDraft())
const editGrokSSOToken = ref('')
const editGrokTier = ref<GrokTier>('basic')
const modelMappings = ref<ModelMapping[]>([])
const actualModelLocked = ref(true)
const modelRestrictionEnabled = ref(true)
const modelRestrictionMode = ref<'whitelist' | 'mapping'>('whitelist')
const allowedModels = ref<string[]>([])
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
const autoPauseOnExpired = ref(false)
const autoRenewEnabled = ref(false)
const autoRenewPeriod = ref<AccountAutoRenewPeriod>('month')
const expiryProbeExtensionDays = ref(1)
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
const openAIImageProtocolMode = ref<OpenAIImageProtocolMode>('native')
const openAIImageCompatAllowed = ref(true)
const openaiOAuthResponsesWebSocketV2Mode = ref<OpenAIWSMode>(OPENAI_WS_MODE_OFF)
const openaiAPIKeyResponsesWebSocketV2Mode = ref<OpenAIWSMode>(OPENAI_WS_MODE_OFF)
const codexCLIOnlyEnabled = ref(false)
const anthropicPassthroughEnabled = ref(false)
const gatewayOpenAIImageProtocolMode = ref<OpenAIImageProtocolMode>(DEFAULT_GATEWAY_OPENAI_IMAGE_PROTOCOL_MODE)
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
const geminiOAuthType = ref<GeminiOAuthType>('code_assist')
const geminiTierAIStudio = ref<GeminiAIStudioTier>('aistudio_free')
const accountTier = ref<AccountTier | ''>('')
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
const currentAccountCredentials = computed<Record<string, unknown>>(
  () => ((props.account?.credentials as Record<string, unknown> | undefined) || {})
)
const currentAccountExtra = computed<Record<string, unknown>>(
  () => ((props.account?.extra as Record<string, unknown> | undefined) || {})
)
const apiKeyProbeCredentials = computed<Record<string, unknown>>(() => {
  const credentials: Record<string, unknown> = {
    api_key: editApiKey.value.trim() || String(currentAccountCredentials.value.api_key || '').trim(),
    base_url: editBaseUrl.value.trim() || resolveAccountApiKeyDefaultBaseUrl(props.account?.platform || 'anthropic', gatewayProtocol.value)
  }
  if (shouldPersistGeminiTierId.value) {
    credentials.tier_id =
      normalizeGeminiAIStudioTier(geminiTierAIStudio.value || currentAccountCredentials.value.tier_id) ||
      'aistudio_free'
  }
  if (props.account?.platform === 'openrouter') {
    const httpReferer = editOpenRouterHTTPReferer.value.trim() || String(currentAccountCredentials.value.http_referer || '').trim()
    const openrouterTitle = editOpenRouterTitle.value.trim() || String(currentAccountCredentials.value.openrouter_title || '').trim()
    if (httpReferer) {
      credentials.http_referer = httpReferer
    }
    if (openrouterTitle) {
      credentials.openrouter_title = openrouterTitle
    }
  }
  return credentials
})
const upstreamProbeCredentials = computed<Record<string, unknown>>(() => ({
  api_key: editApiKey.value.trim() || String(currentAccountCredentials.value.api_key || '').trim(),
  base_url: editBaseUrl.value.trim() || String(currentAccountCredentials.value.base_url || '').trim()
}))
const vertexProbeCredentials = computed<Record<string, unknown>>(() => {
  const baseUrl = geminiVertexBaseUrl.value.trim() || resolveVertexAuthBaseUrl(
    geminiVertexAuthMode.value,
    geminiVertexLocation.value
  )
  if (geminiVertexAuthMode.value === 'express_api_key') {
    return {
      gemini_api_variant: GEMINI_API_KEY_VARIANT_VERTEX_EXPRESS,
      api_key: geminiVertexApiKey.value.trim() || String(currentAccountCredentials.value.api_key || '').trim(),
      base_url: baseUrl
    }
  }
  const credentials: Record<string, unknown> = {
    oauth_type: 'vertex_ai',
    vertex_project_id: geminiVertexProjectId.value.trim(),
    vertex_location: geminiVertexLocation.value.trim(),
    base_url: baseUrl
  }
  const serviceAccountJson =
    geminiVertexServiceAccountJson.value.trim() ||
    String(currentAccountCredentials.value.vertex_service_account_json || '').trim()
  if (serviceAccountJson) {
    credentials.vertex_service_account_json = serviceAccountJson
  }
  const legacyToken = geminiVertexAccessToken.value.trim() || String(currentAccountCredentials.value.access_token || '').trim()
  if (legacyToken) {
    credentials.access_token = legacyToken
  }
  return credentials
})
const isApiKeyProbeReady = computed(() => Boolean(apiKeyProbeCredentials.value.api_key))
const isUpstreamProbeReady = computed(() => Boolean(upstreamProbeCredentials.value.api_key))
const isVertexProbeReady = computed(() => {
  if (geminiVertexAuthMode.value === 'express_api_key') {
    return Boolean(vertexProbeCredentials.value.api_key)
  }
  return Boolean(
    geminiVertexProjectId.value.trim() &&
      geminiVertexLocation.value.trim() &&
      ((vertexProbeCredentials.value.vertex_service_account_json as string | undefined) ||
        (vertexProbeCredentials.value.access_token as string | undefined))
  )
})
const oauthProbeCredentials = computed<Record<string, unknown>>(() => {
  const credentials: Record<string, unknown> = {
    ...currentAccountCredentials.value
  }
  if (isGrokSSOAccount.value) {
    const ssoToken = editGrokSSOToken.value.trim() || String(currentAccountCredentials.value.sso_token || '').trim()
    if (ssoToken) {
      credentials.sso_token = ssoToken
    }
  }
  return credentials
})
const oauthProbeReady = computed(() => {
  if (isGrokSSOAccount.value) {
    return Boolean(String(oauthProbeCredentials.value.sso_token || '').trim())
  }
  return Object.keys(oauthProbeCredentials.value).length > 0
})
const effectivePlatform = computed<GroupPlatform>(() => {
  const platform = resolveEffectiveAccountPlatform(props.account?.platform || 'anthropic', gatewayProtocol.value)
  return platform === 'protocol_gateway' ? 'openai' : platform
})
const effectiveGroupPlatforms = computed<GroupPlatform[] | undefined>(() => {
  if (!isProtocolGatewayPlatform(props.account?.platform)) {
    return undefined
  }
  return resolveEffectiveAccountPlatforms(
    props.account?.platform || 'protocol_gateway',
    gatewayProtocol.value,
    gatewayAcceptedProtocols.value
  ) as GroupPlatform[]
})
const isProtocolGatewayAccount = computed(() =>
  isProtocolGatewayPlatform(props.account?.platform)
)
const isBaiduDocumentAIAccount = computed(() => isBaiduDocumentAIPlatform(props.account?.platform))
const isGrokSSOAccount = computed(() => props.account?.platform === 'grok' && props.account?.type === 'sso')
const isGeminiVertexAccount = computed(() =>
  effectivePlatform.value === 'gemini' &&
  (
    (props.account?.type === 'oauth' && isGeminiVertexAI(geminiOAuthType.value)) ||
    (props.account?.type === 'apikey' && String(currentAccountCredentials.value.gemini_api_variant || '').trim().toLowerCase() === GEMINI_API_KEY_VARIANT_VERTEX_EXPRESS)
  )
)
const isGeminiVertexLegacyMode = computed(() => {
  const credentials = (props.account?.credentials as Record<string, unknown> | undefined) || {}
  return (
    isGeminiVertexAccount.value &&
    geminiVertexAuthMode.value === 'service_account' &&
    !String(credentials.vertex_service_account_json || '').trim() &&
    Boolean(String(credentials.access_token || '').trim())
  )
})
const showAccountTierSelector = computed(() => {
  if (!props.account) {
    return false
  }
  if (props.account.platform === 'openai') {
    return props.account.type === 'oauth'
  }
  return props.account.platform === 'anthropic' &&
    (props.account.type === 'oauth' || props.account.type === 'setup-token')
})
const showCommonApiKeySection = computed(() =>
  props.account?.type === 'apikey' &&
  !isGeminiVertexAccount.value &&
  !isBaiduDocumentAIAccount.value
)
const showDeepSeekConcurrencyEditor = computed(() =>
  showCommonApiKeySection.value && effectivePlatform.value === 'deepseek'
)
const supportsUnifiedModelEditor = computed(() => {
  if (!props.account) {
    return false
  }
  if (props.account.platform === 'antigravity') {
    return false
  }
  if (isProtocolGatewayAccount.value || isGeminiVertexAccount.value || props.account.type === 'upstream') {
    return true
  }
  if (props.account.type === 'apikey') {
    return true
  }
  if (props.account.platform === 'grok' && props.account.type === 'sso') {
    return true
  }
  if (props.account.type === 'oauth') {
    return ['openai', 'anthropic', 'gemini', 'kiro'].includes(props.account.platform)
  }
  return props.account.type === 'setup-token' && props.account.platform === 'anthropic'
})
const showUnifiedProtocolGatewayProbeEditor = computed(() =>
  supportsUnifiedModelEditor.value && isProtocolGatewayAccount.value
)
const showUnifiedAPIModelProbeEditor = computed(() =>
  supportsUnifiedModelEditor.value && !isProtocolGatewayAccount.value
)
const showStandaloneModelScopeEditor = computed(() => {
  if (!supportsUnifiedModelEditor.value || effectivePlatform.value === 'antigravity') {
    return false
  }
  if (isProtocolGatewayAccount.value || isGeminiVertexAccount.value || props.account?.type === 'upstream') {
    return true
  }
  if (isBaiduDocumentAIAccount.value) {
    return true
  }
  return props.account?.type === 'oauth' || props.account?.type === 'setup-token' || props.account?.type === 'sso'
})
const unifiedProbeAccountType = computed(() => {
  if (isGeminiVertexAccount.value) {
    return geminiVertexAuthMode.value === 'express_api_key' ? 'apikey' : 'oauth'
  }
  return String(props.account?.type || 'apikey')
})
const unifiedProbeCredentials = computed<Record<string, unknown>>(() => {
  if (isGeminiVertexAccount.value) {
    return vertexProbeCredentials.value
  }
  if (props.account?.type === 'upstream') {
    return upstreamProbeCredentials.value
  }
  if (props.account?.type === 'apikey') {
    return apiKeyProbeCredentials.value
  }
  return oauthProbeCredentials.value
})
const unifiedProbeReady = computed(() => {
  if (isGeminiVertexAccount.value) {
    return isVertexProbeReady.value
  }
  if (props.account?.type === 'upstream') {
    return isUpstreamProbeReady.value
  }
  if (props.account?.type === 'apikey') {
    return isApiKeyProbeReady.value
  }
  return oauthProbeReady.value
})
const showQuotaLimitSection = computed(() => Boolean(props.account))
const shouldPersistGeminiTierId = computed(() =>
  props.account?.platform === 'gemini' &&
  props.account?.type === 'apikey' &&
  !isGeminiVertexAccount.value
)
const showGeminiAIStudioBatchArchiveEditor = computed(() =>
  resolveGoogleBatchArchiveTargetKind(
    props.account?.platform,
    props.account?.type,
    currentAccountCredentials.value,
  ) === 'ai_studio'
)
const showGeminiVertexBatchArchiveEditor = computed(() =>
  resolveGoogleBatchArchiveTargetKind(
    props.account?.platform,
    props.account?.type,
    currentAccountCredentials.value,
  ) === 'vertex'
)
const grokCapabilityModels = computed(() => grokDefaultModelIdsForTier(editGrokTier.value))
const showProtocolGatewayClaudeMimicEditor = computed(() =>
  supportsProtocolGatewayClaudeClientMimic({
    platform: props.account?.platform,
    type: props.account?.type,
    gatewayProtocol: gatewayProtocol.value,
    acceptedProtocols: gatewayAcceptedProtocols.value
  })
)
const showProtocolGatewayBatchEditor = computed(() =>
  supportsProtocolGatewayGeminiBatch({
    platform: props.account?.platform,
    type: props.account?.type,
    gatewayProtocol: gatewayProtocol.value,
    acceptedProtocols: gatewayAcceptedProtocols.value
  })
)
const showProtocolGatewayOpenAIRequestFormatEditor = computed(() =>
  supportsProtocolGatewayOpenAIRequestFormat({
    platform: props.account?.platform,
    type: props.account?.type,
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
const resolvedProtocolGatewayApiKey = computed(() => {
  if (editApiKey.value.trim()) {
    return editApiKey.value.trim()
  }
  const currentCredentials = (props.account?.credentials as Record<string, unknown>) || {}
  return typeof currentCredentials.api_key === 'string' ? currentCredentials.api_key : ''
})
const gatewayProtocolOptions = computed<GatewayProtocolOption[]>(() =>
  PROTOCOL_GATEWAY_PROTOCOLS.map((id) => {
    const descriptor = resolveGatewayProtocolDescriptor(id)
    return {
      value: id,
      label: descriptor?.displayName || id,
      requestFormatsText: (descriptor?.requestFormats || []).join(', '),
      iconPlatform: descriptor?.targetGroupPlatform || 'protocol_gateway'
    }
  })
)

function isGatewayProtocolOption(option: unknown): option is GatewayProtocolOption {
  return typeof option === 'object' && option !== null && 'value' in option && 'label' in option
}
const openAIWSModeOptions = computed(() => [
  { value: OPENAI_WS_MODE_OFF, label: t('admin.accounts.openai.wsModeOff') },
  { value: OPENAI_WS_MODE_CTX_POOL, label: t('admin.accounts.openai.wsModeCtxPool') },
  { value: OPENAI_WS_MODE_PASSTHROUGH, label: t('admin.accounts.openai.wsModePassthrough') }
])
const openaiResponsesWebSocketV2Mode = computed({
  get: () => {
    if (effectivePlatform.value === 'openai' && props.account?.type === 'apikey') {
      return openaiAPIKeyResponsesWebSocketV2Mode.value
    }
    return openaiOAuthResponsesWebSocketV2Mode.value
  },
  set: (mode: OpenAIWSMode) => {
    if (effectivePlatform.value === 'openai' && props.account?.type === 'apikey') {
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
  effectivePlatform.value === 'openai' && openaiPassthroughEnabled.value
)

// Computed: current preset mappings based on platform
const presetMappings = computed(() => getPresetMappingsByPlatform(effectivePlatform.value))

// Computed: default base URL based on platform
const defaultBaseUrl = computed(() => {
  return resolveAccountApiKeyDefaultBaseUrl(props.account?.platform || 'anthropic', gatewayProtocol.value)
})
const commonErrorCodeOptions = computed(() => createCommonErrorCodeOptions(t))

function buildBaiduDocumentAICredentialsForUpdate(): Record<string, unknown> | null {
  const currentCredentials = (props.account?.credentials as Record<string, unknown> | undefined) || {}
  let directAPIURLs: Record<string, string> = {}
  try {
    directAPIURLs = parseBaiduDocumentAIDirectApiUrlsInput(
      baiduDocumentAIDirectApiUrlsText.value
    )
  } catch {
    appStore.showError(t('admin.accounts.baiduDocumentAI.directApiUrlsInvalid'))
    return null
  }

  const newCredentials: Record<string, unknown> = {
    ...currentCredentials,
    async_base_url:
      baiduDocumentAIAsyncBaseUrl.value.trim() ||
      String(currentCredentials.async_base_url || '').trim() ||
      BAIDU_DOCUMENT_AI_DEFAULT_ASYNC_BASE_URL
  }
  delete newCredentials.base_url
  delete newCredentials.api_key
  delete newCredentials.tier_id
  delete newCredentials.model_mapping
  delete newCredentials.model_whitelist

  const accessToken = baiduDocumentAIAccessToken.value.trim()
  if (accessToken) {
    newCredentials.async_bearer_token = accessToken
    newCredentials.direct_token = accessToken
  }
  if (Object.keys(directAPIURLs).length > 0) {
    newCredentials.direct_api_urls = directAPIURLs
  } else {
    delete newCredentials.direct_api_urls
  }

  const asyncBearerToken = String(newCredentials.async_bearer_token || '').trim()
  const directToken = String(newCredentials.direct_token || '').trim()
  if (!asyncBearerToken && !directToken) {
    appStore.showError(t('admin.accounts.baiduDocumentAI.tokenRequired'))
    return null
  }

  return newCredentials
}

function loadModelScopeFromExtra(extra?: Record<string, unknown>): boolean {
  const draft = loadAccountModelScopeDraft(extra)
  if (!draft) {
    return false
  }

  modelRestrictionEnabled.value = draft.enabled
  modelRestrictionMode.value = isProtocolGatewayAccount.value ? 'mapping' : draft.mode
  allowedModels.value = [...draft.allowedModels]
  modelMappings.value = draft.modelMappings.map((item) => ({ ...item }))
  if (isProtocolGatewayAccount.value) {
    protocolGatewayProbeModels.value = draft.enabled
      ? createStaticProbeModels(draft.allowedModels)
      : []
  }
  return true
}

function applyModelRestrictionFromRecord(value: unknown) {
  const entries = Object.entries(value && typeof value === 'object' ? value as Record<string, unknown> : {})
    .map(([from, to]) => ({ from: String(from || '').trim(), to: String(to || '').trim() }))
    .filter((row) => row.from.length > 0 && row.to.length > 0)

  if (entries.length === 0) {
    modelRestrictionEnabled.value = true
    modelRestrictionMode.value = isProtocolGatewayAccount.value ? 'mapping' : 'whitelist'
    allowedModels.value = []
    modelMappings.value = []
    if (isProtocolGatewayAccount.value) {
      protocolGatewayProbeModels.value = []
    }
    return
  }

  modelRestrictionEnabled.value = true
  if (isProtocolGatewayAccount.value) {
    modelRestrictionMode.value = 'mapping'
    const selectedModels = [...new Set(entries.map(({ to }) => to))]
    modelMappings.value = entries.filter(({ from, to }) => from !== to)
    allowedModels.value = selectedModels
    protocolGatewayProbeModels.value = createStaticProbeModels(selectedModels)
    return
  }

  const isWhitelistMode = entries.every(({ from, to }) => from === to)
  if (isWhitelistMode) {
    modelRestrictionMode.value = 'whitelist'
    allowedModels.value = entries.map(({ from }) => from)
    modelMappings.value = []
    return
  }

  modelRestrictionMode.value = 'mapping'
  allowedModels.value = [...new Set(entries.map(({ to }) => to))]
  modelMappings.value = entries.filter(({ from, to }) => from !== to)
}

function buildScopedModelMapping(
  mode: 'whitelist' | 'mapping' = modelRestrictionMode.value,
  allowed: string[] = allowedModels.value,
  mappings: ModelMapping[] = modelMappings.value
) {
  if (!modelRestrictionEnabled.value) {
    return null
  }
  return buildModelMappingObject(mode, allowed, mappings)
}

function applyDefaultGrokCapabilityMapping() {
  modelRestrictionMode.value = 'mapping'
  modelMappings.value = mappingRecordToRows(grokDefaultModelMappingForTier(editGrokTier.value))
  allowedModels.value = []
}

function createStaticProbeModels(modelIds: string[]): ProtocolGatewayProbeModel[] {
  return modelIds.map((modelId) => ({
    id: modelId,
    display_name: formatModelDisplayName(modelId) || modelId,
    registry_state: 'existing',
    registry_model_id: modelId
  }))
}


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
  currentPlatform: () => effectivePlatform.value,
  buildCheckPayload: () => {
    if (!props.account) {
      return null
    }
    return {
      platform: props.account.platform,
      gateway_protocol: isProtocolGatewayPlatform(props.account.platform) ? gatewayProtocol.value : undefined,
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

const resetProtocolGatewayClaudeMimicState = () => {
  claudeCodeMimicEnabled.value = false
  claudeTLSFingerprintEnabled.value = false
  claudeSessionIDMaskingEnabled.value = false
}

const handleOpenAIImageProtocolModeChange = (value: OpenAIImageProtocolMode) => {
  if (!openAIImageCompatAllowed.value && value === 'compat') {
    openAIImageProtocolMode.value = 'native'
    return
  }
  openAIImageProtocolMode.value = value
}

const applyAccountTierCapacity = (capacity?: number) => {
  const nextCapacity = capacity || resolveAccountTierCapacity(props.account?.platform, accountTier.value)
  if (nextCapacity > 0) {
    form.concurrency = nextCapacity
  }
}

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
    if (error?.reason === 'ACCOUNT_INVALID_BASE_URL') {
      appStore.showError(t('admin.accounts.invalidBaseUrl'))
      return
    }
    appStore.showError(error.message || t('admin.accounts.failedToUpdate'))
  } finally {
    submitting.value = false
  }
}

const probeExtraForEditor = computed(() => buildProbeExtra())

const resolveConfiguredModelProbeSnapshot = () =>
  buildLocalAccountModelProbeSnapshot({
    current: modelProbeSnapshot.value,
    enabled: effectivePlatform.value === 'antigravity'
      ? true
      : modelRestrictionEnabled.value &&
        !(effectivePlatform.value === 'openai' && openaiPassthroughEnabled.value),
    modelRestrictionMode: effectivePlatform.value === 'antigravity'
      ? antigravityModelRestrictionMode.value
      : modelRestrictionMode.value,
    allowedModels: effectivePlatform.value === 'antigravity'
      ? antigravityWhitelistModels.value
      : allowedModels.value,
    modelMappings: effectivePlatform.value === 'antigravity'
      ? antigravityModelMappings.value
      : modelMappings.value,
    source: 'model_scope_preview'
  })

function buildProbeExtra(base?: Record<string, unknown>) {
  return mergeResolvedUpstreamDraftIntoExtra(
    mergeAccountModelProbeSnapshotIntoExtra(
      mergeAccountManualModelsIntoExtra(
        base || currentAccountExtra.value,
        manualModels.value,
        isProtocolGatewayAccount.value
      ),
      resolveConfiguredModelProbeSnapshot()
    ),
    resolvedUpstream.value
  )
}

const modalContext = {
  props, GEMINI_API_KEY_VARIANT_VERTEX_EXPRESS, acceptAIStudioBatchOverflow, allowVertexBatchOverflow, allowedModels, anthropicPassthroughEnabled, antigravityModelMappings, appStore,
  applyAccountCustomErrorCodesStateToCredentials, applyAccountPoolModeStateToCredentials, applyDeepSeekModelConcurrencyLimitsExtra, applyGoogleBatchArchiveExtra, applyInterceptWarmup, applyProtocolGatewayClaudeClientMimicExtra, applyProtocolGatewayGeminiBatchExtra, applyProtocolGatewayOpenAIImageProtocolModeExtra,
  applyProtocolGatewayOpenAIRequestFormatExtra, applyTempUnschedConfig, applyAccountTierToExtra, autoPauseOnExpired, autoRenewEnabled, autoRenewPeriod, batchArchiveAutoPrefetchEnabled, batchArchiveBillingMode, batchArchiveDownloadPriceUSD, batchArchiveEnabled, batchArchiveRetentionDays,
  buildAccountModelScopeExtra, buildBaiduDocumentAICredentialsForUpdate, buildModelMappingObject, buildProbeExtra, buildScopedModelMapping, claudeCodeMimicEnabled, claudeSessionIDMaskingEnabled, claudeTLSFingerprintEnabled,
  codexCLIOnlyEnabled, currentAccountCredentials, customErrorCodesState, deepSeekModelConcurrencyLimits, defaultBaseUrl, editApiKey, editBaseUrl, editGrokSSOToken,
  editGrokTier, editOpenRouterHTTPReferer, editOpenRouterTitle, editQuotaDailyLimit, editQuotaDailyResetHour, editQuotaDailyResetMode, editQuotaLimit, editQuotaResetTimezone,
  editQuotaWeeklyLimit, editQuotaWeeklyResetDay, editQuotaWeeklyResetHour, editQuotaWeeklyResetMode, effectivePlatform, ensureMixedChannelConfirmed, expiryProbeExtensionDays, form,
  gatewayAcceptedProtocols, gatewayBatchEnabled, gatewayClientProfiles, gatewayClientRoutes, gatewayOpenAIImageProtocolMode, gatewayOpenAIRequestFormat, gatewayProtocol, gatewayTestModelId,
  gatewayTestProvider, geminiTierAIStudio, geminiVertexAccessToken, geminiVertexApiKey, geminiVertexAuthMode, geminiVertexBaseUrl, geminiVertexExpiresAtInput, geminiVertexLocation,
  geminiVertexProjectId, geminiVertexServiceAccountJson, interceptWarmupRequests, isBaiduDocumentAIAccount, isGeminiVertexAccount, isOpenAIWSModeEnabled, isProtocolGatewayAccount, mixedScheduling,
  modelMappings, modelRestrictionEnabled, modelRestrictionMode, normalizeGeminiAIStudioTier, openAIImageCompatAllowed, openAIImageProtocolMode, openaiAPIKeyResponsesWebSocketV2Mode, openaiOAuthResponsesWebSocketV2Mode,
  openaiPassthroughEnabled, parseDateTimeLocal, poolModeState, quotaControl, resolveGoogleBatchArchiveTargetKind, resolveVertexAuthBaseUrl, resolveVertexBaseUrl, shouldPersistGeminiTierId,
  submitUpdateAccount, t, BAIDU_DOCUMENT_AI_DEFAULT_ASYNC_BASE_URL, DEFAULT_GATEWAY_OPENAI_IMAGE_PROTOCOL_MODE, DEFAULT_GATEWAY_OPENAI_REQUEST_FORMAT, DEFAULT_POOL_MODE_RETRY_COUNT, OPENAI_WS_MODE_OFF, actualModelLocked,
  antigravityModelRestrictionMode, antigravityWhitelistModels, applyModelRestrictionFromRecord, baiduDocumentAIAccessToken, baiduDocumentAIAsyncBaseUrl, baiduDocumentAIDirectApiUrlsText, createDefaultDeepSeekModelConcurrencyLimitDraft, createStaticProbeModels,
  defaultGoogleBatchArchiveState, deriveConfiguredAccountModelIds, ensureModelRegistryFresh, formatDateTimeLocal, geminiOAuthType, grokDefaultModelMappingForTier, isBaiduDocumentAIPlatform, isGeminiVertexAI,
  isInitializingGatewayProtocol, loadAccountCustomErrorCodesStateFromCredentials, loadAccountPoolModeStateFromCredentials, loadModelScopeFromExtra, loadTempUnschedRules, manualModels, modelProbeSnapshot, normalizeGatewayAcceptedProtocols,
  normalizeGatewayBatchEnabled, normalizeGatewayClientProfile, normalizeGatewayClientRoutes, normalizeGeminiOAuthType, normalizeGrokTier, protocolGatewayProbeModels, readAccountManualModelsFromExtra, readAccountModelProbeSnapshot,
  readAccountResolvedUpstreamDraft, readDeepSeekModelConcurrencyLimitDraft, readGoogleBatchArchiveFormState, resetAccountCustomErrorCodesState, resetAccountPoolModeState, resetMixedChannelRisk, resetProtocolGatewayClaudeMimicState, resetTempUnschedRules,
  resolveAccountApiKeyDefaultBaseUrl, resolveAccountGatewayOpenAIImageProtocolMode, resolveAccountGatewayOpenAIRequestFormat, resolveAccountGatewayProtocol, resolveEffectiveAccountPlatform, resolveOpenAIImageProtocolState, resolveOpenAIWSModeFromExtra, resolvedUpstream,
  showProtocolGatewayBatchEditor, showProtocolGatewayClaudeMimicEditor, showProtocolGatewayOpenAIRequestFormatEditor, stringifyBaiduDocumentAIDirectApiUrls, watch, authStore, showFormError, showFormInfo,
  antigravityPresetMappings, submitting, getModelMappingKey, getAntigravityModelMappingKey, quotaControlState, umqModeOptions, effectiveGroupPlatforms, isGrokSSOAccount,
  isGeminiVertexLegacyMode, showCommonApiKeySection, showDeepSeekConcurrencyEditor, showUnifiedProtocolGatewayProbeEditor, showUnifiedAPIModelProbeEditor, showStandaloneModelScopeEditor, unifiedProbeAccountType, unifiedProbeCredentials,
  unifiedProbeReady, showQuotaLimitSection, showGeminiAIStudioBatchArchiveEditor, showGeminiVertexBatchArchiveEditor, grokCapabilityModels, protocolGatewayBatchRequestFormats, resolvedProtocolGatewayApiKey, gatewayProtocolOptions,
  isGatewayProtocolOption, openAIWSModeOptions, openaiResponsesWebSocketV2Mode, openAIWSModeConcurrencyHintKey, isOpenAIModelRestrictionDisabled, presetMappings, commonErrorCodeOptions, applyDefaultGrokCapabilityMapping,
  tempUnschedEnabled, tempUnschedRules, tempUnschedPresets, getTempUnschedRuleKey, addTempUnschedRule, removeTempUnschedRule, moveTempUnschedRule, showMixedChannelWarning,
  mixedChannelWarningMessageText, handleMixedChannelConfirm, handleMixedChannelCancel, statusOptions, expiresAtInput, handleOpenAIImageProtocolModeChange, addModelMapping, removeModelMapping,
  addPresetMapping, addAntigravityModelMapping, removeAntigravityModelMapping, addAntigravityPresetMapping, handleClose, probeExtraForEditor, MAX_POOL_MODE_RETRY_COUNT,
  accountTier, showAccountTierSelector, applyAccountTierCapacity, defaultAccountTierForPlatform, normalizeAccountTier, resolveAccountTierCapacity
}
const handleSubmit = createEditAccountSubmit(modalContext)

useEditAccountModalWatchers(modalContext)

return {
  ...modalContext,
  handleSubmit
}}
