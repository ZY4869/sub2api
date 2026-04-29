import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import UsageStatsCards from '../UsageStatsCards.vue'

const messages: Record<string, string> = {
  'usage.totalRequests': 'Total Requests',
  'usage.totalTokens': 'Total Tokens',
  'usage.totalCost': 'Total Cost',
  'usage.inSelectedRange': 'in selected range',
  'usage.in': 'In',
  'usage.out': 'Out',
  'usage.avgDuration': 'Avg Duration',
  'usage.actualCost': 'Actual',
  'usage.standardCost': 'Standard',
  'usage.todaySoFar': "From today's 00:00 to now",
  'usage.todayAvgDuration': "Today's Avg Duration",
  'usage.cacheTokens': 'Cache',
  'admin.usage.todayStats': 'Today Stats',
  'admin.usage.todayRequests': 'Today Requests',
  'admin.usage.todayTokens': 'Today Tokens',
  'admin.usage.todayCost': 'Today Cost',
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
    expect(text).toContain('Today Stats')
    expect(text).toContain("From today's 00:00 to now")
    expect(text).toContain('Today Requests')
    expect(text).toContain('Today Tokens')
    expect(text).toContain('Today Cost')
    expect(text).toContain("Today's Avg Duration")
  })
})
