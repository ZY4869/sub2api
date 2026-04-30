import { describe, expect, it } from 'vitest'

import enAdmin from '@/i18n/locales/en/admin'
import zhAdmin from '@/i18n/locales/zh/admin'

describe('admin locale wrappers', () => {
  it('keeps zh/en admin wrapper exports stable', () => {
    const zh = zhAdmin as any
    const en = enAdmin as any

    expect(zh.admin).toBeTruthy()
    expect(en.admin).toBeTruthy()

    expect(zh.admin.models?.pages?.all?.bulk).toBeTruthy()
    expect(en.admin.models?.pages?.all?.bulk).toBeTruthy()

    expect(zh.admin.accounts?.protocolGateway).toBeTruthy()
    expect(en.admin.accounts?.protocolGateway).toBeTruthy()

    expect(zh.admin.settings?.tabs).toBeTruthy()
    expect(en.admin.settings?.tabs).toBeTruthy()

    expect(zh.admin.ops?.timeRange).toBeTruthy()
    expect(en.admin.ops?.timeRange).toBeTruthy()

    expect(zh.admin.requestDetails?.filters).toBeTruthy()
    expect(en.admin.requestDetails?.filters).toBeTruthy()
  })
})

