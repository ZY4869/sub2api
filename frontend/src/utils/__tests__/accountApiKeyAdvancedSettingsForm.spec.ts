import { describe, expect, it } from 'vitest'
import {
  applyAccountCustomErrorCodesStateToCredentials,
  applyAccountPoolModeStateToCredentials,
  loadAccountCustomErrorCodesStateFromCredentials,
  loadAccountPoolModeStateFromCredentials,
  resetAccountCustomErrorCodesState,
  resetAccountPoolModeState
} from '../accountApiKeyAdvancedSettingsForm'
import {
  DEFAULT_POOL_MODE_RETRY_COUNT,
  createDefaultAccountCustomErrorCodesState,
  createDefaultAccountPoolModeState
} from '../accountFormShared'

describe('accountApiKeyAdvancedSettingsForm', () => {
  it('loads/resets/applies pool mode state', () => {
    const state = createDefaultAccountPoolModeState(DEFAULT_POOL_MODE_RETRY_COUNT)

    loadAccountPoolModeStateFromCredentials(state, {
      pool_mode: true,
      pool_mode_retry_count: 99
    })
    expect(state.enabled).toBe(true)
    expect(state.retryCount).toBe(10)

    const credentials: Record<string, unknown> = {}
    applyAccountPoolModeStateToCredentials(credentials, state)
    expect(credentials).toEqual({
      pool_mode: true,
      pool_mode_retry_count: 10
    })

    resetAccountPoolModeState(state, DEFAULT_POOL_MODE_RETRY_COUNT)
    expect(state).toEqual({
      enabled: false,
      retryCount: DEFAULT_POOL_MODE_RETRY_COUNT
    })

    applyAccountPoolModeStateToCredentials(credentials, state)
    expect(credentials).toEqual({})
  })

  it('loads/resets/applies custom error codes state', () => {
    const state = createDefaultAccountCustomErrorCodesState()

    loadAccountCustomErrorCodesStateFromCredentials(state, {
      custom_error_codes_enabled: true,
      custom_error_codes: [429, '503', 'bad']
    })
    expect(state.enabled).toBe(true)
    expect(state.selectedCodes).toEqual([429, 503])
    expect(state.input).toBeNull()

    const credentials: Record<string, unknown> = {}
    applyAccountCustomErrorCodesStateToCredentials(credentials, state)
    expect(credentials).toEqual({
      custom_error_codes_enabled: true,
      custom_error_codes: [429, 503]
    })

    resetAccountCustomErrorCodesState(state)
    expect(state).toEqual({
      enabled: false,
      selectedCodes: [],
      input: null
    })

    applyAccountCustomErrorCodesStateToCredentials(credentials, state)
    expect(credentials).toEqual({})
  })
})

