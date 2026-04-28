import { describe, expect, it } from 'vitest'
import { resolveSettingsTab } from '../settingsTabs'

describe('settings tab query resolution', () => {
  it('accepts valid settings tab query values', () => {
    expect(resolveSettingsTab('security')).toBe('security')
    expect(resolveSettingsTab('gateway')).toBe('gateway')
    expect(resolveSettingsTab(['email'])).toBe('email')
  })

  it('falls back to the general tab for missing or invalid values', () => {
    expect(resolveSettingsTab(undefined)).toBe('general')
    expect(resolveSettingsTab('unknown')).toBe('general')
    expect(resolveSettingsTab(['unknown'])).toBe('general')
  })
})
