import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import UsageStatsCards from '../UsageStatsCards.vue'

const messages: Record<string, string> = {
  'usage.totalRequests': '总请求数',
  'usage.totalTokens': '总 Token',
  'usage.totalCost': '总费用',
  'usage.inSelectedRange': '所选范围内',
  'usage.in': '输入',
  'usage.out': '输出',
  'usage.avgDuration': '平均耗时',
  'usage.actualCost': '实际费用',
  'usage.standardCost': '标准费用',
  'usage.todaySoFar': '从今日 00:00 到当前',
  'usage.todayAvgDuration': '今日平均耗时',
  'usage.cacheTokens': '缓存',
  'admin.usage.todayStats': '今日统计',
  'admin.usage.todayRequests': '今日请求',
  'admin.usage.todayTokens': '今日 Token',
  'admin.usage.todayCost': '今日费用',
}

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => messages[key] ?? key,
    }),
  }
})

describe('admin UsageStatsCards', () => {
  it('renders today usage metrics alongside selected-range totals', () => {
    const wrapper = mount(UsageStatsCards, {
      props: {
        stats: {
          total_requests: 8,
          total_input_tokens: 120,
          total_output_tokens: 240,
          total_cache_tokens: 36,
          total_tokens: 396,
          total_cost: 1.2,
          total_actual_cost: 1.1,
          admin_free_requests: 0,
          admin_free_standard_cost: 0,
          average_duration_ms: 150,
          today_requests: 3,
          today_input_tokens: 30,
          today_output_tokens: 60,
          today_cache_tokens: 9,
          today_tokens: 99,
          today_cost: 0.45,
          today_actual_cost: 0.4,
          today_average_duration_ms: 120,
        },
      },
      global: {
        stubs: {
          Icon: true,
        },
      },
    })

    const text = wrapper.text()
    expect(text).toContain('今日统计')
    expect(text).toContain('从今日 00:00 到当前')
    expect(text).toContain('今日请求')
    expect(text).toContain('今日 Token')
    expect(text).toContain('今日费用')
    expect(text).toContain('今日平均耗时')
  })
})
