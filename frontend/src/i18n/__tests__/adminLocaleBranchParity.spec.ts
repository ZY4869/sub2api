import { describe, expect, it } from 'vitest'

import en from '@/i18n/locales/en'
import zh from '@/i18n/locales/zh'

function flattenKeys(value: unknown, prefix = ''): string[] {
  if (value === null || value === undefined || Array.isArray(value) || typeof value !== 'object') {
    return prefix ? [prefix] : []
  }

  return Object.entries(value as Record<string, unknown>).flatMap(([key, child]) =>
    flattenKeys(child, prefix ? `${prefix}.${key}` : key)
  )
}

describe('admin locale branch parity', () => {
  const branches = ['models', 'accounts', 'settings', 'ops', 'requestDetails'] as const

  for (const branch of branches) {
    it(`keeps zh/en keys aligned for admin.${branch}`, () => {
      const zhBranch = (zh as any).admin?.[branch]
      const enBranch = (en as any).admin?.[branch]

      expect(zhBranch).toBeTruthy()
      expect(enBranch).toBeTruthy()

      const zhKeys = flattenKeys(zhBranch).sort()
      const enKeys = flattenKeys(enBranch).sort()

      expect(zhKeys).toEqual(enKeys)
    })
  }
})
