import { describe, expect, it } from 'vitest'

import admin from '@/i18n/locales/zh/admin'
import adminModelOverrides from '@/i18n/locales/zh/adminModelOverrides'
import composeZhAdmin from '@/i18n/locales/zh/admin/compose'

describe('zh admin compose', () => {
  it('keeps targeted models overrides merged after modularization', () => {
    const composed = composeZhAdmin(admin as any, adminModelOverrides as any) as any

    expect(composed.admin).toBeTruthy()

    // Ensure pages.debug / pages.billing remain present after merge (not part of overrides).
    expect(composed.admin.models.pages.debug.nav).toBe('模型调试')
    expect(composed.admin.models.pages.billing.nav).toBeTruthy()

    // Ensure pages.all merge keeps base keys and applies targeted overrides.
    expect(composed.admin.models.pages.all.bulk.moveProvider).toBe('迁移到目标厂商')
    expect(composed.admin.models.pages.all.nav).toBe('总模型')
    expect(composed.admin.models.pages.all.viewModes.grid).toBe('九宫格')

    // Ensure available.activateDialog is merged deeply.
    expect(composed.admin.models.available.activateDialog.searchPlaceholder).toBe('搜索未启用模型')
  })
})

