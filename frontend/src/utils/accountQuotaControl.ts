import type { Account } from '@/types'

export type AnthropicQuotaRPMStrategy = 'tiered' | 'sticky_exempt'

export interface AnthropicQuotaControlState {
  windowCostEnabled: boolean
  windowCostLimit: number | null
  windowCostStickyReserve: number | null
  sessionLimitEnabled: boolean
  maxSessions: number | null
  sessionIdleTimeout: number | null
  rpmLimitEnabled: boolean
  baseRpm: number | null
  rpmStrategy: AnthropicQuotaRPMStrategy
  rpmStickyBuffer: number | null
  userMsgQueueMode: string
  tlsFingerprintEnabled: boolean
  tlsFingerprintProfileId: number | null
  sessionIdMaskingEnabled: boolean
  cacheTTLOverrideEnabled: boolean
  cacheTTLOverrideTarget: string
  customBaseUrlEnabled: boolean
  customBaseUrl: string
}

export const DEFAULT_WINDOW_COST_STICKY_RESERVE = 10
export const DEFAULT_SESSION_IDLE_TIMEOUT_MINUTES = 5
export const DEFAULT_BASE_RPM = 15
export const DEFAULT_CACHE_TTL_OVERRIDE_TARGET = '5m'

const hasPositiveNumber = (value: number | null | undefined): value is number =>
  typeof value === 'number' && Number.isFinite(value) && value > 0

const parseProfileId = (value: unknown): number | null => {
  if (
    typeof value === 'number' &&
    Number.isFinite(value) &&
    Number.isInteger(value) &&
    (value === -1 || value > 0)
  ) {
    return value
  }
  if (typeof value === 'string' && value.trim() !== '') {
    const parsed = Number.parseInt(value.trim(), 10)
    return Number.isFinite(parsed) && (parsed === -1 || parsed > 0) ? parsed : null
  }
  return null
}

export const createDefaultAnthropicQuotaControlState = (): AnthropicQuotaControlState => ({
  windowCostEnabled: false,
  windowCostLimit: null,
  windowCostStickyReserve: null,
  sessionLimitEnabled: false,
  maxSessions: null,
  sessionIdleTimeout: null,
  rpmLimitEnabled: false,
  baseRpm: null,
  rpmStrategy: 'tiered',
  rpmStickyBuffer: null,
  userMsgQueueMode: '',
  tlsFingerprintEnabled: false,
  tlsFingerprintProfileId: null,
  sessionIdMaskingEnabled: false,
  cacheTTLOverrideEnabled: false,
  cacheTTLOverrideTarget: DEFAULT_CACHE_TTL_OVERRIDE_TARGET,
  customBaseUrlEnabled: false,
  customBaseUrl: ''
})

export const readAnthropicQuotaControlState = (
  account: Account | null | undefined
): AnthropicQuotaControlState => {
  const state = createDefaultAnthropicQuotaControlState()
  if (!account || account.platform !== 'anthropic' || (account.type !== 'oauth' && account.type !== 'setup-token')) {
    return state
  }

  if (hasPositiveNumber(account.window_cost_limit)) {
    state.windowCostEnabled = true
    state.windowCostLimit = account.window_cost_limit
    state.windowCostStickyReserve = hasPositiveNumber(account.window_cost_sticky_reserve)
      ? account.window_cost_sticky_reserve
      : DEFAULT_WINDOW_COST_STICKY_RESERVE
  }

  if (hasPositiveNumber(account.max_sessions)) {
    state.sessionLimitEnabled = true
    state.maxSessions = account.max_sessions
    state.sessionIdleTimeout = hasPositiveNumber(account.session_idle_timeout_minutes)
      ? account.session_idle_timeout_minutes
      : DEFAULT_SESSION_IDLE_TIMEOUT_MINUTES
  }

  if (hasPositiveNumber(account.base_rpm)) {
    state.rpmLimitEnabled = true
    state.baseRpm = account.base_rpm
    state.rpmStrategy = account.rpm_strategy === 'sticky_exempt' ? 'sticky_exempt' : 'tiered'
    state.rpmStickyBuffer = hasPositiveNumber(account.rpm_sticky_buffer)
      ? account.rpm_sticky_buffer
      : null
  }

  state.userMsgQueueMode = account.user_msg_queue_mode ?? ''
  state.tlsFingerprintEnabled = account.enable_tls_fingerprint === true
  state.tlsFingerprintProfileId =
    parseProfileId(account.tls_fingerprint_profile_id) ??
    parseProfileId(account.extra?.tls_fingerprint_profile_id)
  state.sessionIdMaskingEnabled = account.session_id_masking_enabled === true

  if (account.cache_ttl_override_enabled === true) {
    state.cacheTTLOverrideEnabled = true
    state.cacheTTLOverrideTarget =
      account.cache_ttl_override_target || DEFAULT_CACHE_TTL_OVERRIDE_TARGET
  }

  if (account.custom_base_url_enabled === true) {
    state.customBaseUrlEnabled = true
    state.customBaseUrl = String(account.custom_base_url ?? account.extra?.custom_base_url ?? '').trim()
  }

  return state
}

export const buildAnthropicQuotaControlExtra = (
  state: AnthropicQuotaControlState,
  base?: Record<string, unknown>
): Record<string, unknown> | undefined => {
  const extra: Record<string, unknown> = { ...(base || {}) }

  if (state.windowCostEnabled && hasPositiveNumber(state.windowCostLimit)) {
    extra.window_cost_limit = state.windowCostLimit
    extra.window_cost_sticky_reserve = hasPositiveNumber(state.windowCostStickyReserve)
      ? state.windowCostStickyReserve
      : DEFAULT_WINDOW_COST_STICKY_RESERVE
  } else {
    delete extra.window_cost_limit
    delete extra.window_cost_sticky_reserve
  }

  if (state.sessionLimitEnabled && hasPositiveNumber(state.maxSessions)) {
    extra.max_sessions = state.maxSessions
    extra.session_idle_timeout_minutes = hasPositiveNumber(state.sessionIdleTimeout)
      ? state.sessionIdleTimeout
      : DEFAULT_SESSION_IDLE_TIMEOUT_MINUTES
  } else {
    delete extra.max_sessions
    delete extra.session_idle_timeout_minutes
  }

  if (state.rpmLimitEnabled) {
    extra.base_rpm = hasPositiveNumber(state.baseRpm) ? state.baseRpm : DEFAULT_BASE_RPM
    extra.rpm_strategy = state.rpmStrategy
    if (hasPositiveNumber(state.rpmStickyBuffer)) {
      extra.rpm_sticky_buffer = state.rpmStickyBuffer
    } else {
      delete extra.rpm_sticky_buffer
    }
  } else {
    delete extra.base_rpm
    delete extra.rpm_strategy
    delete extra.rpm_sticky_buffer
  }

  if (state.userMsgQueueMode) {
    extra.user_msg_queue_mode = state.userMsgQueueMode
  } else {
    delete extra.user_msg_queue_mode
  }
  delete extra.user_msg_queue_enabled

  if (state.tlsFingerprintEnabled) {
    extra.enable_tls_fingerprint = true
    if (state.tlsFingerprintProfileId !== null) {
      extra.tls_fingerprint_profile_id = state.tlsFingerprintProfileId
    } else {
      delete extra.tls_fingerprint_profile_id
    }
  } else {
    delete extra.enable_tls_fingerprint
    delete extra.tls_fingerprint_profile_id
  }

  if (state.sessionIdMaskingEnabled) {
    extra.session_id_masking_enabled = true
  } else {
    delete extra.session_id_masking_enabled
  }

  if (state.cacheTTLOverrideEnabled) {
    extra.cache_ttl_override_enabled = true
    extra.cache_ttl_override_target =
      state.cacheTTLOverrideTarget || DEFAULT_CACHE_TTL_OVERRIDE_TARGET
  } else {
    delete extra.cache_ttl_override_enabled
    delete extra.cache_ttl_override_target
  }

  if (state.customBaseUrlEnabled && state.customBaseUrl.trim()) {
    extra.custom_base_url_enabled = true
    extra.custom_base_url = state.customBaseUrl.trim()
  } else {
    delete extra.custom_base_url_enabled
    delete extra.custom_base_url
  }

  return Object.keys(extra).length > 0 ? extra : undefined
}
