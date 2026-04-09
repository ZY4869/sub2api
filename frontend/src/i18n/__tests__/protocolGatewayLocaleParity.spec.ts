import { describe, expect, it, vi } from 'vitest'

import enAdmin from '@/i18n/locales/en/admin'
import zhAdmin from '@/i18n/locales/zh/admin'

function flattenKeys(value: unknown, prefix = ''): string[] {
  if (value === null || value === undefined || Array.isArray(value) || typeof value !== 'object') {
    return prefix ? [prefix] : []
  }

  return Object.entries(value as Record<string, unknown>).flatMap(([key, child]) =>
    flattenKeys(child, prefix ? `${prefix}.${key}` : key)
  )
}

describe('protocol gateway locale parity', () => {
  it('keeps zh/en keys aligned for protocol gateway copy', () => {
    const zhBranch = (zhAdmin as any).admin.accounts.protocolGateway
    const enBranch = (enAdmin as any).admin.accounts.protocolGateway

    expect(zhBranch).toBeTruthy()
    expect(enBranch).toBeTruthy()

    const zhKeys = flattenKeys(zhBranch).sort()
    const enKeys = flattenKeys(enBranch).sort()

    expect(zhKeys).toEqual(enKeys)
  })

  it('defaults to zh when no saved locale exists', async () => {
    vi.resetModules()
    localStorage.removeItem('sub2api_locale')
    Object.defineProperty(window.navigator, 'language', {
      configurable: true,
      value: 'en-US'
    })

    const module = await import('@/i18n')
    expect(module.getLocale()).toBe('zh')
  })
})
