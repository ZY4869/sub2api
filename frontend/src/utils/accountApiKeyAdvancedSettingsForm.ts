import {
  DEFAULT_POOL_MODE_RETRY_COUNT,
  normalizePoolModeRetryCount,
  normalizePoolModeRetryStatusCodes,
  type AccountCustomErrorCodesState,
  type AccountPoolModeState
} from '@/utils/accountFormShared'

export function loadAccountPoolModeStateFromCredentials(
  state: AccountPoolModeState,
  credentials: Record<string, unknown> | null | undefined,
  defaultRetryCount = DEFAULT_POOL_MODE_RETRY_COUNT
): void {
  const payload = credentials ?? {}
  state.enabled = payload.pool_mode === true
  state.retryCount = normalizePoolModeRetryCount(
    Number(payload.pool_mode_retry_count ?? defaultRetryCount)
  )
  state.retryStatusCodes = normalizePoolModeRetryStatusCodes(
    payload.pool_mode_retry_status_codes
  )
}

export function resetAccountPoolModeState(
  state: AccountPoolModeState,
  defaultRetryCount = DEFAULT_POOL_MODE_RETRY_COUNT
): void {
  state.enabled = false
  state.retryCount = defaultRetryCount
  state.retryStatusCodes = normalizePoolModeRetryStatusCodes(null)
}

export function applyAccountPoolModeStateToCredentials(
  credentials: Record<string, unknown>,
  state: AccountPoolModeState
): void {
  if (state.enabled) {
    credentials.pool_mode = true
    credentials.pool_mode_retry_count = normalizePoolModeRetryCount(state.retryCount)
    credentials.pool_mode_retry_status_codes = normalizePoolModeRetryStatusCodes(
      state.retryStatusCodes
    )
    return
  }

  delete credentials.pool_mode
  delete credentials.pool_mode_retry_count
  delete credentials.pool_mode_retry_status_codes
}

export function loadAccountCustomErrorCodesStateFromCredentials(
  state: AccountCustomErrorCodesState,
  credentials: Record<string, unknown> | null | undefined
): void {
  const payload = credentials ?? {}
  state.enabled = payload.custom_error_codes_enabled === true

  const existing = payload.custom_error_codes
  if (Array.isArray(existing)) {
    state.selectedCodes = existing.map((code) => Number(code)).filter((code) => Number.isFinite(code))
  } else {
    state.selectedCodes = []
  }

  state.input = null
}

export function resetAccountCustomErrorCodesState(state: AccountCustomErrorCodesState): void {
  state.enabled = false
  state.selectedCodes = []
  state.input = null
}

export function applyAccountCustomErrorCodesStateToCredentials(
  credentials: Record<string, unknown>,
  state: AccountCustomErrorCodesState
): void {
  if (state.enabled) {
    credentials.custom_error_codes_enabled = true
    credentials.custom_error_codes = [...state.selectedCodes]
    return
  }

  delete credentials.custom_error_codes_enabled
  delete credentials.custom_error_codes
}
