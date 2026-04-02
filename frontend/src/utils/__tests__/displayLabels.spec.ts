import { describe, expect, it } from 'vitest'
import {
  buildTimezoneOptions,
  formatAwsRegionLabel,
  formatCountryLabel,
  formatProxyLocationLabel,
  formatVertexLocationLabel,
  resolveUsageWindowLabel,
} from '../displayLabels'

describe('displayLabels', () => {
  it('localizes usage window labels', () => {
    const t = (key: string) => ({
      'ui.usageWindow.fiveHour': '5H',
      'ui.usageWindow.daily': '日',
      'ui.usageWindow.weekly': '周',
      'ui.usageWindow.total': '总',
      'ui.usageWindow.pro': 'Pro',
      'ui.usageWindow.flash': 'Flash',
    }[key] || key)

    expect(resolveUsageWindowLabel('1d', t)).toBe('日')
    expect(resolveUsageWindowLabel('7d', t)).toBe('周')
    expect(resolveUsageWindowLabel('total', t)).toBe('总')
  })

  it('formats timezone and region labels as zh plus original code', () => {
    expect(buildTimezoneOptions('zh').find((option) => option.value === 'Asia/Shanghai')?.label).toBe('中国上海（Asia/Shanghai）')
    expect(formatAwsRegionLabel('us-east-1', 'zh')).toBe('美国东部 1（us-east-1）')
    expect(formatVertexLocationLabel('global', 'zh')).toBe('全球（global）')
  })

  it('formats country and proxy location labels with localized country names', () => {
    expect(formatCountryLabel('US', 'United States', 'zh')).toContain('美国')
    expect(formatProxyLocationLabel({ country_code: 'US', city: 'Seattle' }, 'zh')).toContain('Seattle')
  })
})
