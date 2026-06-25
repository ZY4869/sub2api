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
  'usage.cacheWrite': '写入',
  'usage.cacheRead': '读取',
  'usage.cacheHitRate': '命中率',
  'usage.inputTokens': '输入 Token',
  'usage.outputTokens': '输出 Token',
  'usage.cacheCreationTokens': '缓存写入 Token',
  'usage.cacheReadTokens': '缓存读取 Token',
  'common.total': '总计',
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
      t: (key: string, params?: Record<string, unknown>) => {
        let message = messages[key] ?? key
        Object.entries(params || {}).forEach(([name, value]) => {
          message = message.replace(`{${name}}`, String(value))
        })
        return message
      },
    }),
  }
})

const stats = {
  total_requests: 8,
  total_input_tokens: 120,
  total_output_tokens: 240,
  total_cache_creation_tokens: 1_250_000,
  total_cache_read_tokens: 3_400_000,
  total_cache_tokens: 4_650_000,
  total_tokens: 1_663_471,
  cache_hit_rate: 0.896,
  total_cost: 1.2,
  total_actual_cost: 1.1,
  admin_free_requests: 0,
  admin_free_standard_cost: 0,
  average_duration_ms: 150,
  today_requests: 3,
  today_input_tokens: 30,
  today_output_tokens: 60,
  today_cache_creation_tokens: 200_000,
  today_cache_read_tokens: 700_000,
  today_cache_tokens: 900_000,
  today_tokens: 990_000,
  today_cache_hit_rate: 75,
  today_cost: 0.45,
  today_actual_cost: 0.4,
  today_average_duration_ms: 120,
}

describe('admin UsageStatsCards', () => {
  it('renders today usage metrics alongside selected-range totals', () => {
    const wrapper = mount(UsageStatsCards, {
      props: {
        stats,
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
    expect(text).toContain('75.0%')
    expect(text).toContain('输入 Token')
    expect(text).toContain('缓存写入 Token')
    expect(text).toContain('缓存读取 Token')
    expect(text).toContain('输出 Token')
  })

  it('renders cache hit rate as a standalone selected-range card', () => {
    const wrapper = mount(UsageStatsCards, {
      props: {
        stats,
      },
      global: {
        stubs: {
          Icon: true,
        },
      },
    })

    const cacheCard = wrapper.get('[data-testid="usage-cache-stats-card"]')
    expect(cacheCard.text()).toContain('命中率')
    expect(cacheCard.text()).toContain('89.6%')
    expect(cacheCard.text()).toContain('写入')
    expect(cacheCard.text()).toContain('1.3M')
    expect(cacheCard.text()).toContain('读取')
    expect(cacheCard.text()).toContain('3.4M')
    expect(cacheCard.text()).toContain('4.7M')
    expect(wrapper.text()).toContain('1.7M')
  })

  it('keeps cache and today token zero values visible', () => {
    const zeroStats = {
      ...stats,
      total_cache_creation_tokens: 0,
      total_cache_read_tokens: 0,
      total_cache_tokens: 0,
      cache_hit_rate: 0,
      today_input_tokens: 0,
      today_output_tokens: 0,
      today_cache_creation_tokens: 0,
      today_cache_read_tokens: 0,
      today_cache_tokens: 0,
      today_cache_hit_rate: 0,
    }

    const wrapper = mount(UsageStatsCards, {
      props: {
        stats: zeroStats,
      },
      global: {
        stubs: {
          Icon: true,
        },
      },
    })

    const text = wrapper.text()
    expect(text).toContain('输入 Token')
    expect(text).toContain('缓存写入 Token')
    expect(text).toContain('缓存读取 Token')
    expect(text).toContain('输出 Token')
    expect(text).toContain('0.0%')

    const cacheCard = wrapper.get('[data-testid="usage-cache-stats-card"]')
    expect(cacheCard.text()).toContain('写入')
    expect(cacheCard.text()).toContain('读取')
    expect(cacheCard.text()).toContain('总计')
    expect(cacheCard.text()).toContain('0.0%')
  })
})
