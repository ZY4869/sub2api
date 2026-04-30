import { describe, expect, it } from 'vitest'
import en from '@/i18n/locales/en'
import zh from '@/i18n/locales/zh'

describe('zh locale audit', () => {
  it('keeps targeted region-related labels in Chinese', () => {
    expect(zh.admin.dataManagement.form.s3.region).toBe('区域')
    expect(zh.admin.accounts.kiroAuth.startUrl).toBe('起始 URL')
    expect(zh.admin.accounts.kiroAuth.region).toBe('区域')
    expect(zh.admin.dataManagement.form.s3.accessKeyID).toBe('访问密钥 ID')
    expect(zh.admin.dataManagement.form.s3.secretAccessKey).toBe('秘密访问密钥')
  })

  it('does not leak raw english keywords in targeted zh labels', () => {
    expect(zh.admin.dataManagement.form.s3.region).not.toContain('Region')
    expect(zh.admin.accounts.kiroAuth.startUrl).not.toContain('Start URL')
    expect(zh.admin.accounts.kiroAuth.region).not.toContain('Region')
  })

  it('exposes merged billing and channels locale branches in the final zh tree', () => {
    expect(zh.admin.channels.nav).toBeTruthy()
    expect(zh.admin.channels.title).toBeTruthy()
    expect(zh.admin.channels.description).toBeTruthy()
    expect(zh.admin.channels.nav).not.toBe('admin.channels.nav')
    expect(zh.admin.models.pages.billing.nav).toBeTruthy()
    expect(zh.admin.models.pages.billing.title).toBeTruthy()
    expect(zh.admin.models.pages.billing.description).toBeTruthy()
    expect(zh.admin.models.pages.billing.nav).not.toBe('admin.models.pages.billing.nav')
  })

  it('keeps debug page and today stats labels localized in the merged zh tree', () => {
    expect(zh.admin.models.pages.debug.nav).toBe('模型调试')
    expect(zh.admin.models.pages.debug.title).toBe('模型调试')
    expect(zh.admin.models.pages.debug.description).toContain('管理员代理链路')
    expect(zh.admin.usage.todayStats).toBe('今日统计')
    expect(zh.admin.usage.todayRequests).toBe('今日请求')
    expect(zh.admin.usage.todayTokens).toBe('今日 Token')
    expect(zh.admin.usage.todayCost).toBe('今日费用')
  })

  it('keeps the english today stats keys aligned with zh additions', () => {
    expect(en.admin.usage.todayStats).toBe('Today Stats')
    expect(en.admin.usage.todayRequests).toBe('Today Requests')
    expect(en.admin.usage.todayTokens).toBe('Today Tokens')
    expect(en.admin.usage.todayCost).toBe('Today Cost')
  })
})
