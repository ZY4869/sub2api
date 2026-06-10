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
        'admin.accounts.stats.monthlyUsage': 'Month',
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

const statsFixture = {
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
  monthly: {
    requests: 150,
    tokens: 3200,
    cost: 16.5,
  },
  total: {
    requests: 999,
    tokens: 5000,
    cost: 42,
  },
} as any

describe('AccountTodayStatsCell', () => {
  it('keeps the default card stack layout for classic usage', () => {
    const wrapper = mount(AccountTodayStatsCell, {
      props: {
        stats: statsFixture,
      },
    })

    const compactText = wrapper.get('[data-testid="account-today-stats-cell"]').text().replace(/\s/g, '')
    expect(compactText).toContain('Today12$1.20')
    expect(compactText).toContain('7d78$8.90')
    expect(compactText).toContain('Month150$16.50')
    expect(compactText).toContain('Total999$42.00')
    expect(wrapper.text()).toContain('345T')
    expect(wrapper.text()).toContain('1.4s')
    expect(wrapper.text()).toContain('91.4%')
    expect(wrapper.find('.text-rose-600').exists()).toBe(true)
    expect(wrapper.find('[data-testid="account-today-stats-airy-panel"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="account-today-stats-cell"] .grid').classes()).toContain('grid-cols-1')
    expect(wrapper.get('[data-testid="account-today-stats-cell"]').classes()).toContain('w-[120px]')
    expect(wrapper.get('[data-testid="account-today-stats-cell"]').classes()).toContain('max-w-[132px]')
  })

  it('renders the airy stats as one divided compact panel', () => {
    const wrapper = mount(AccountTodayStatsCell, {
      props: {
        stats: statsFixture,
        visualVariant: 'airy',
      },
    })

    const panel = wrapper.get('[data-testid="account-today-stats-airy-panel"]')
    const compactText = wrapper.get('[data-testid="account-today-stats-cell"]').text().replace(/\s/g, '')

    expect(compactText).toContain('Today12$1.20')
    expect(compactText).toContain('7d78$8.90')
    expect(compactText).toContain('Month150$16.50')
    expect(compactText).toContain('Total999$42.00')
    expect(wrapper.text()).toContain('345T')
    expect(wrapper.text()).toContain('1.4s')
    expect(wrapper.text()).toContain('91.4%')
    expect(wrapper.find('.text-rose-600').exists()).toBe(true)
    expect(panel.classes()).toContain('divide-y')
    expect(wrapper.findAll('[data-testid="account-today-stats-row"]')).toHaveLength(4)
    expect(panel.find('[data-testid="account-today-stats-footer"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="account-today-stats-cell"] .grid').exists()).toBe(false)
    expect(wrapper.get('[data-testid="account-today-stats-cell"]').classes()).toContain('w-[136px]')
    expect(wrapper.get('[data-testid="account-today-stats-cell"]').classes()).toContain('max-w-[152px]')
  })

  it('renders selected stats windows and only shows quality footer when today is visible', () => {
    const stats = { ...statsFixture, success_rate: 99, average_duration_ms: 500 }

    const weeklyOnly = mount(AccountTodayStatsCell, {
      props: {
        stats,
        visibleWindows: ['weekly'],
      },
    })
    expect(weeklyOnly.text()).not.toContain('Today')
    expect(weeklyOnly.text()).toContain('7d')
    expect(weeklyOnly.text()).not.toContain('Total')
    expect(weeklyOnly.text()).not.toContain('345T')
    expect(weeklyOnly.get('[data-testid="account-today-stats-cell"]').classes()).toContain('w-[104px]')
    expect(weeklyOnly.get('[data-testid="account-today-stats-cell"]').classes()).toContain('max-w-[120px]')
    expect(weeklyOnly.get('[data-testid="account-today-stats-cell"] .grid').classes()).toContain('grid-cols-1')

    const todayAndTotal = mount(AccountTodayStatsCell, {
      props: {
        stats,
        visibleWindows: ['today', 'total'],
      },
    })
    expect(todayAndTotal.text()).toContain('Today')
    expect(todayAndTotal.text()).not.toContain('7d')
    expect(todayAndTotal.text()).toContain('Total')
    expect(todayAndTotal.text()).toContain('345T')
    expect(todayAndTotal.get('[data-testid="account-today-stats-cell"]').classes()).toContain('w-[120px]')
    expect(todayAndTotal.get('[data-testid="account-today-stats-cell"]').classes()).toContain('max-w-[132px]')
    expect(todayAndTotal.get('[data-testid="account-today-stats-cell"] .grid').classes()).toContain('grid-cols-1')
  })

  it('renders selected airy windows and keeps footer tied to today visibility', () => {
    const stats = { ...statsFixture, success_rate: 99, average_duration_ms: 500 }
    const weeklyOnly = mount(AccountTodayStatsCell, {
      props: {
        stats,
        visibleWindows: ['weekly'],
        visualVariant: 'airy',
      },
    })

    expect(weeklyOnly.text()).not.toContain('Today')
    expect(weeklyOnly.text()).toContain('7d')
    expect(weeklyOnly.text()).not.toContain('Total')
    expect(weeklyOnly.text()).not.toContain('345T')
    expect(weeklyOnly.findAll('[data-testid="account-today-stats-row"]')).toHaveLength(1)
    expect(weeklyOnly.find('[data-testid="account-today-stats-footer"]').exists()).toBe(false)

    const todayAndTotal = mount(AccountTodayStatsCell, {
      props: {
        stats,
        visibleWindows: ['today', 'total'],
        visualVariant: 'airy',
      },
    })

    expect(todayAndTotal.text()).toContain('Today')
    expect(todayAndTotal.text()).not.toContain('7d')
    expect(todayAndTotal.text()).toContain('Total')
    expect(todayAndTotal.text()).toContain('345T')
    expect(todayAndTotal.findAll('[data-testid="account-today-stats-row"]')).toHaveLength(2)
    expect(todayAndTotal.get('[data-testid="account-today-stats-footer"]').exists()).toBe(true)
  })

  it('keeps loading, error and empty states', () => {
    expect(mount(AccountTodayStatsCell, { props: { loading: true } }).find('.animate-pulse').exists()).toBe(true)
    expect(mount(AccountTodayStatsCell, { props: { error: 'failed' } }).text()).toContain('failed')
    expect(mount(AccountTodayStatsCell).text()).toContain('-')
  })
})
