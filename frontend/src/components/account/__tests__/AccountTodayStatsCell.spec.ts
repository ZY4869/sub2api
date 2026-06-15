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
        'admin.accounts.keyUsage.discountedCost': 'Discount',
        'admin.accounts.keyUsage.standardCost': 'Standard',
        'admin.accounts.keyUsage.saved': 'Saved',
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
}))

const statsFixture = {
  requests: 12,
  tokens: 345,
  cost: 1.2,
  standard_cost: 2.4,
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
  it('renders classic usage as compact icon-prefixed rows without currency locale prefixes', () => {
    const wrapper = mount(AccountTodayStatsCell, {
      props: {
        stats: statsFixture,
      },
    })

    const compactText = wrapper.get('[data-testid="account-today-stats-cell"]').text().replace(/\s/g, '')
    expect(compactText).toContain('12$1.20')
    expect(compactText).toContain('78$8.90')
    expect(compactText).toContain('150$16.50')
    expect(compactText).toContain('999$42.00')
    expect(wrapper.text()).not.toContain('US')
    expect(wrapper.text()).not.toContain('Today')
    expect(wrapper.findAll('[data-testid="account-today-stats-window-icon"]')).toHaveLength(4)
    expect(wrapper.text()).toContain('345T')
    expect(wrapper.text()).toContain('1.4s')
    expect(wrapper.text()).toContain('91.4%')
    expect(wrapper.find('.text-rose-600').exists()).toBe(true)
    expect(wrapper.find('[data-testid="account-today-stats-airy-panel"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="account-today-stats-cell"] .grid').exists()).toBe(false)
    expect(wrapper.get('[data-testid="account-today-stats-cell"]').classes()).toContain('w-[156px]')
    expect(wrapper.get('[data-testid="account-today-stats-cell"]').classes()).toContain('max-w-[168px]')

    const firstRow = wrapper.findAll('[data-testid="account-today-stats-row"]')[0]
    expect(firstRow.attributes('title')).toContain('Discount: $1.20')
    expect(firstRow.attributes('title')).toContain('Standard: $2.40')
    expect(firstRow.attributes('title')).toContain('Saved: $1.20 (50%)')
    expect(firstRow.findAll('span').at(-1)?.classes()).toContain('text-left')
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

    expect(compactText).toContain('12$1.20')
    expect(compactText).toContain('78$8.90')
    expect(compactText).toContain('150$16.50')
    expect(compactText).toContain('999$42.00')
    expect(wrapper.text()).not.toContain('Today')
    expect(wrapper.text()).not.toContain('US')
    expect(wrapper.text()).toContain('345T')
    expect(wrapper.text()).toContain('1.4s')
    expect(wrapper.text()).toContain('91.4%')
    expect(wrapper.find('.text-rose-600').exists()).toBe(true)
    expect(panel.classes()).toContain('divide-y')
    expect(wrapper.findAll('[data-testid="account-today-stats-row"]')).toHaveLength(4)
    expect(wrapper.findAll('[data-testid="account-today-stats-window-icon"]')).toHaveLength(4)
    expect(panel.find('[data-testid="account-today-stats-footer"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="account-today-stats-cell"] .grid').exists()).toBe(false)
    expect(wrapper.get('[data-testid="account-today-stats-cell"]').classes()).toContain('w-[168px]')
    expect(wrapper.get('[data-testid="account-today-stats-cell"]').classes()).toContain('max-w-[184px]')
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
    expect(weeklyOnly.find('[data-testid="account-today-stats-window-icon"]').attributes('title')).toBe('7d')
    expect(weeklyOnly.text()).not.toContain('Total')
    expect(weeklyOnly.text()).not.toContain('345T')
    expect(weeklyOnly.get('[data-testid="account-today-stats-cell"]').classes()).toContain('w-[128px]')
    expect(weeklyOnly.get('[data-testid="account-today-stats-cell"]').classes()).toContain('max-w-[144px]')
    expect(weeklyOnly.find('[data-testid="account-today-stats-cell"] .grid').exists()).toBe(false)

    const todayAndTotal = mount(AccountTodayStatsCell, {
      props: {
        stats,
        visibleWindows: ['today', 'total'],
      },
    })
    expect(todayAndTotal.text()).not.toContain('Today')
    expect(todayAndTotal.text()).not.toContain('7d')
    const iconTitles = todayAndTotal
      .findAll('[data-testid="account-today-stats-window-icon"]')
      .map((icon) => icon.attributes('title'))
    expect(iconTitles).toEqual(['Today', 'Total'])
    expect(todayAndTotal.text()).toContain('345T')
    expect(todayAndTotal.get('[data-testid="account-today-stats-cell"]').classes()).toContain('w-[156px]')
    expect(todayAndTotal.get('[data-testid="account-today-stats-cell"]').classes()).toContain('max-w-[168px]')
    expect(todayAndTotal.find('[data-testid="account-today-stats-cell"] .grid').exists()).toBe(false)
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
    expect(weeklyOnly.find('[data-testid="account-today-stats-window-icon"]').attributes('title')).toBe('7d')
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

    expect(todayAndTotal.text()).not.toContain('Today')
    expect(todayAndTotal.text()).not.toContain('7d')
    const iconTitles = todayAndTotal
      .findAll('[data-testid="account-today-stats-window-icon"]')
      .map((icon) => icon.attributes('title'))
    expect(iconTitles).toEqual(['Today', 'Total'])
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
