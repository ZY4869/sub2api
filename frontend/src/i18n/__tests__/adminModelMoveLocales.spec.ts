import { describe, expect, it } from 'vitest'

import en from '../locales/en'
import zh from '../locales/zh'

describe('admin model move locale messages', () => {
  it('keeps zh bulk model action labels after locale overrides are merged', () => {
    expect(zh.admin.models.pages.all.bulk.addToTest).toBe('选中加入测试')
    expect(zh.admin.models.pages.all.bulk.removeFromTest).toBe('选中移出测试')
    expect(zh.admin.models.pages.all.bulk.moveProvider).toBe('迁移到目标厂商')
    expect(zh.admin.models.pages.all.bulk.moveProviderPlaceholder).toBe('请选择目标厂商')
    expect(zh.admin.models.pages.all.bulk.moveProviderHint).toBe('已选模型后，请先选择目标厂商，再执行迁移。')
    expect(zh.admin.models.pages.all.bulk.moveProviderSelectRequired).toBe('请先选择目标厂商')
  })

  it('keeps en bulk model move labels aligned with the new interaction copy', () => {
    expect(en.admin.models.pages.all.bulk.moveProvider).toBe('Move to Target Provider')
    expect(en.admin.models.pages.all.bulk.moveProviderPlaceholder).toBe('Please select a target provider')
    expect(en.admin.models.pages.all.bulk.moveProviderHint).toBe(
      'After selecting models, choose a target provider before moving them.'
    )
    expect(en.admin.models.pages.all.bulk.moveProviderSelectRequired).toBe(
      'Please select a target provider first'
    )
  })
})
