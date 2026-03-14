import { describe, expect, it } from 'vitest'
import {
  createDefaultAccountCustomErrorCodesState,
  createDefaultAccountPoolModeState
} from '../accountApiKeyAdvancedSettings'

describe('accountApiKeyAdvancedSettings', () => {
  it('creates default pool mode state', () => {
    expect(createDefaultAccountPoolModeState(3)).toEqual({
      enabled: false,
      retryCount: 3
    })
  })

  it('creates default custom error code state', () => {
    expect(createDefaultAccountCustomErrorCodesState()).toEqual({
      enabled: false,
      selectedCodes: [],
      input: null
    })
  })
})
