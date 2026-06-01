import { ref } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import {
  buildExpiryInput,
  useAccountExpirationShortcuts,
} from '../useAccountExpirationShortcuts'

describe('useAccountExpirationShortcuts', () => {
  it('builds local datetime input values for quick expiry presets', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date(2026, 0, 15, 8, 30, 45))

    expect(buildExpiryInput(7, 'day')).toBe('2026-01-22T08:30')
    expect(buildExpiryInput(1, 'month')).toBe('2026-02-15T08:30')
    expect(buildExpiryInput(1, 'year')).toBe('2027-01-15T08:30')

    vi.useRealTimers()
  })

  it('clears auto renew when expiration is disabled', () => {
    const expiresAtInput = ref('2026-06-01T12:30')
    const autoRenewEnabled = ref(true)
    const { expirationEnabled } = useAccountExpirationShortcuts({
      expiresAtInput,
      autoRenewEnabled
    })

    expirationEnabled.value = false

    expect(expiresAtInput.value).toBe('')
    expect(autoRenewEnabled.value).toBe(false)
  })

  it('enables expiration with a default future month value', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date(2026, 4, 1, 9, 5, 0))

    const expiresAtInput = ref('')
    const autoRenewEnabled = ref(false)
    const { expirationEnabled } = useAccountExpirationShortcuts({
      expiresAtInput,
      autoRenewEnabled
    })

    expirationEnabled.value = true

    expect(expiresAtInput.value).toBe('2026-06-01T09:05')
    expect(autoRenewEnabled.value).toBe(false)

    vi.useRealTimers()
  })
})
