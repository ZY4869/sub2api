import { describe, expect, it } from 'vitest'
import zh from '@/i18n/locales/zh'

describe('zh locale audit', () => {
  it('keeps targeted region-related labels in Chinese', () => {
    expect(zh.admin.dataManagement.form.s3.region).toBe('区域')
    expect(zh.admin.accounts.kiroAuth.startUrl).toBe('起始 URL')
    expect(zh.admin.accounts.kiroAuth.region).toBe('区域')
    expect(zh.admin.settings.soraS3.accessKeyId).toBe('访问密钥 ID')
    expect(zh.admin.settings.soraS3.secretAccessKey).toBe('秘密访问密钥')
  })

  it('does not leak raw english keywords in targeted zh labels', () => {
    expect(zh.admin.dataManagement.form.s3.region).not.toContain('Region')
    expect(zh.admin.accounts.kiroAuth.startUrl).not.toContain('Start URL')
    expect(zh.admin.accounts.kiroAuth.region).not.toContain('Region')
  })
})
