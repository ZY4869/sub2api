import type { AccountPlatform, UpdateAccountRequest } from '@/types'
import type { ModelMapping } from '@/utils/accountFormShared'
import type { AnthropicQuotaRPMStrategy } from '@/utils/accountQuotaControl'

export type BulkEditModelRestrictionMode = 'whitelist' | 'mapping'
export type BulkEditAccountStatus = 'active' | 'inactive'

export interface BulkEditAccountFormState {
  enableBaseUrl: boolean
  enableModelRestriction: boolean
  enableCustomErrorCodes: boolean
  enableInterceptWarmup: boolean
  enableProxy: boolean
  enableConcurrency: boolean
  enableLoadFactor: boolean
  enablePriority: boolean
  enableRateMultiplier: boolean
  enableStatus: boolean
  enableGroups: boolean
  enableRpmLimit: boolean
  baseUrl: string
  modelRestrictionMode: BulkEditModelRestrictionMode
  allowedModels: string[]
  modelMappings: ModelMapping[]
  selectedErrorCodes: number[]
  customErrorCodeInput: number | null
  interceptWarmupRequests: boolean
  proxyId: number | null
  concurrency: number
  loadFactor: number | null
  priority: number
  rateMultiplier: number
  status: BulkEditAccountStatus
  groupIds: number[]
  rpmLimitEnabled: boolean
  bulkBaseRpm: number | null
  bulkRpmStrategy: AnthropicQuotaRPMStrategy
  bulkRpmStickyBuffer: number | null
  userMsgQueueMode: string | null
}

export const createDefaultBulkEditAccountFormState = (): BulkEditAccountFormState => ({
  enableBaseUrl: false,
  enableModelRestriction: false,
  enableCustomErrorCodes: false,
  enableInterceptWarmup: false,
  enableProxy: false,
  enableConcurrency: false,
  enableLoadFactor: false,
  enablePriority: false,
  enableRateMultiplier: false,
  enableStatus: false,
  enableGroups: false,
  enableRpmLimit: false,
  baseUrl: '',
  modelRestrictionMode: 'whitelist',
  allowedModels: [],
  modelMappings: [],
  selectedErrorCodes: [],
  customErrorCodeInput: null,
  interceptWarmupRequests: false,
  proxyId: null,
  concurrency: 1,
  loadFactor: null,
  priority: 1,
  rateMultiplier: 1,
  status: 'active',
  groupIds: [],
  rpmLimitEnabled: false,
  bulkBaseRpm: null,
  bulkRpmStrategy: 'tiered',
  bulkRpmStickyBuffer: null,
  userMsgQueueMode: null
})

export const hasBulkEditAccountFieldEnabled = (state: BulkEditAccountFormState): boolean => {
  return (
    state.enableBaseUrl ||
    state.enableModelRestriction ||
    state.enableCustomErrorCodes ||
    state.enableInterceptWarmup ||
    state.enableProxy ||
    state.enableConcurrency ||
    state.enableLoadFactor ||
    state.enablePriority ||
    state.enableRateMultiplier ||
    state.enableStatus ||
    state.enableGroups ||
    state.enableRpmLimit ||
    state.userMsgQueueMode !== null
  )
}

export const canBulkEditAccountPreCheck = (
  state: Pick<BulkEditAccountFormState, 'enableGroups' | 'groupIds'>,
  selectedPlatforms: AccountPlatform[]
): boolean => {
  return (
    state.enableGroups &&
    state.groupIds.length > 0 &&
    selectedPlatforms.length === 1 &&
    (selectedPlatforms[0] === 'antigravity' || selectedPlatforms[0] === 'anthropic')
  )
}

export const buildBulkEditAccountPayload = (
  state: BulkEditAccountFormState,
  resolveModelMapping: () => Record<string, string> | null
): Partial<UpdateAccountRequest> | null => {
  const updates: Partial<UpdateAccountRequest> = {}
  const credentials: Record<string, unknown> = {}
  let credentialsChanged = false

  if (state.enableProxy) {
    updates.proxy_id = state.proxyId === null ? 0 : state.proxyId
  }

  if (state.enableConcurrency) {
    updates.concurrency = state.concurrency
  }

  if (state.enableLoadFactor) {
    const loadFactor = state.loadFactor
    updates.load_factor =
      loadFactor != null && !Number.isNaN(loadFactor) && loadFactor > 0 ? loadFactor : 0
  }

  if (state.enablePriority) {
    updates.priority = state.priority
  }

  if (state.enableRateMultiplier) {
    updates.rate_multiplier = state.rateMultiplier
  }

  if (state.enableStatus) {
    updates.status = state.status
  }

  if (state.enableGroups) {
    updates.group_ids = state.groupIds
  }

  if (state.enableBaseUrl) {
    const baseUrlValue = state.baseUrl.trim()
    if (baseUrlValue) {
      credentials.base_url = baseUrlValue
      credentialsChanged = true
    }
  }

  if (state.enableModelRestriction) {
    const modelMapping = resolveModelMapping()
    if (modelMapping) {
      credentials.model_mapping = modelMapping
      credentialsChanged = true
    }
  }

  if (state.enableCustomErrorCodes) {
    credentials.custom_error_codes_enabled = true
    credentials.custom_error_codes = [...state.selectedErrorCodes]
    credentialsChanged = true
  }

  if (state.enableInterceptWarmup) {
    credentials.intercept_warmup_requests = state.interceptWarmupRequests
    credentialsChanged = true
  }

  if (credentialsChanged) {
    updates.credentials = credentials
  }

  const extra = buildBulkEditExtra(state)
  if (extra) {
    updates.extra = extra
  }

  return Object.keys(updates).length > 0 ? updates : null
}

function buildBulkEditExtra(
  state: Pick<
    BulkEditAccountFormState,
    | 'enableRpmLimit'
    | 'rpmLimitEnabled'
    | 'bulkBaseRpm'
    | 'bulkRpmStrategy'
    | 'bulkRpmStickyBuffer'
    | 'userMsgQueueMode'
  >
): Record<string, unknown> | undefined {
  const extra: Record<string, unknown> = {}

  if (state.enableRpmLimit) {
    if (state.rpmLimitEnabled && state.bulkBaseRpm != null && state.bulkBaseRpm > 0) {
      extra.base_rpm = state.bulkBaseRpm
      extra.rpm_strategy = state.bulkRpmStrategy
      if (state.bulkRpmStickyBuffer != null && state.bulkRpmStickyBuffer > 0) {
        extra.rpm_sticky_buffer = state.bulkRpmStickyBuffer
      }
    } else {
      extra.base_rpm = 0
      extra.rpm_strategy = ''
      extra.rpm_sticky_buffer = 0
    }
  }

  if (state.userMsgQueueMode !== null) {
    extra.user_msg_queue_mode = state.userMsgQueueMode
  }

  return Object.keys(extra).length > 0 ? extra : undefined
}
