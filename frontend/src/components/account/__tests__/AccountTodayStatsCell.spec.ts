import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import AccountTodayStatsCell from '../AccountTodayStatsCell.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => ({
        'admin.accounts.stats.requests': 'Req',
        'usage.accountBilled': 'Bill',
        'admin.accounts.status.active': 'Active',
        'admin.accounts.status.window7d': '7d',
        'common.total': 'Total',
        'dates.today': 'Today',
      }[key] ?? key)
    })
  }
})

vi.mock('@/composables/useTokenDisplayMode', () => ({
  useTokenDisplayMode: () => ({
    formatTokenDisplay: (value: number) => `${value}T`
  })
}))

vi.mock('@/utils/format', () => ({
  formatNumber: (value: number) => String(value),
  formatCurrency: (value: number) => `$${value.toFixed(2)}`,
}))

describe('AccountTodayStatsCell', () => {
  it('renders day, weekly and total stats with latency and success health', () => {
    const wrapper = mount(AccountTodayStatsCell, {
      props: {
        stats: {
          requests: 12,
          tokens: 345,
          cost: 1.2,
          success_rate: 91.4,
          average_duration_ms: 1450,
          weekly: {
            requests: 78,
            tokens: 2000,
            cost: 8.9,
          },
          total: {
            requests: 999,
            tokens: 5000,
            cost: 42,
          },
        } as any,
      },
    })

    const compactText = wrapper.get('[data-testid="account-today-stats-cell"]').text().replace(/\s/g, '')
    expect(compactText).toContain('Today12$1.20')
    expect(compactText).toContain('7d78$8.90')
    expect(compactText).toContain('Total999$42.00')
    expect(wrapper.text()).toContain('345T')
    expect(wrapper.text()).toContain('1.4s')
    expect(wrapper.text()).toContain('91.4%')
    expect(wrapper.find('.text-rose-600').exists()).toBe(true)
  })

  it('keeps loading, error and empty states', () => {
    expect(mount(AccountTodayStatsCell, { props: { loading: true } }).find('.animate-pulse').exists()).toBe(true)
    expect(mount(AccountTodayStatsCell, { props: { error: 'failed' } }).text()).toContain('failed')
    expect(mount(AccountTodayStatsCell).text()).toContain('-')
  })
})
