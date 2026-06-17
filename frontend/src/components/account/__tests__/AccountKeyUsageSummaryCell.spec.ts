import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AccountKeyUsageSummaryCell from '../AccountKeyUsageSummaryCell.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => ({
        'admin.accounts.keyUsage.requests': 'Req',
        'admin.accounts.keyUsage.tokens': 'Tok',
        'admin.accounts.keyUsage.discountedCost': 'Discount',
        'admin.accounts.keyUsage.standardCost': 'Standard',
        'admin.accounts.keyUsage.saved': 'Saved',
        'admin.accounts.keyUsage.unlimited': 'Unlimited',
        'admin.accounts.keyUsage.callQuota': 'Quota',
        'admin.accounts.usageWindow.now': 'Now',
        'dates.today': 'Today',
        'dates.tomorrow': 'Tomorrow',
        'ui.usageWindow.total': 'Total',
      }[key] ?? key)
    })
  }
})

vi.mock('@/composables/useTokenDisplayMode', () => ({
  useTokenDisplayMode: () => ({
    formatTokenDisplay: (value: number) => `${value}T`
  })
}))

vi.mock('@/composables/useRealtimeCountdownNow', () => ({
  useRealtimeCountdownNow: () => ({
    nowDate: { value: new Date('2026-03-13T12:00:00Z') }
  })
}))

describe('AccountKeyUsageSummaryCell', () => {
  it('flattens usage costs and savings without US currency prefixes', () => {
    const wrapper = mount(AccountKeyUsageSummaryCell, {
      props: {
        account: {
          id: 1,
          type: 'apikey',
          platform: 'openai',
          extra: {},
        } as any,
        stats: {
          requests: 12,
          tokens: 345,
          cost: 0.4,
          standard_cost: 5,
          user_cost: 0.4,
        } as any,
      }
    })

    expect(wrapper.text()).toContain('Req')
    expect(wrapper.text()).toContain('12')
    expect(wrapper.text()).toContain('345T')
    const todayRow = wrapper.get('[data-testid="account-key-usage-today-row"]')
    const quotaRow = wrapper.get('[data-testid="account-key-usage-quota-row"]')
    expect(todayRow.text()).toContain('Req')
    expect(todayRow.text()).toContain('Tok')
    expect(todayRow.text()).toContain('Discount')
    expect(todayRow.text()).toContain('Standard')
    expect(todayRow.text()).toContain('Saved')
    expect(todayRow.text()).not.toContain('Unlimited')
    expect(todayRow.classes()).toEqual(expect.arrayContaining(['overflow-x-auto', 'whitespace-nowrap']))
    expect(quotaRow.text()).toContain('Unlimited')
    expect(quotaRow.classes()).toEqual(expect.arrayContaining(['overflow-x-auto', 'whitespace-nowrap']))
    expect(wrapper.get('[data-testid="account-key-usage-discounted-cost"]').text()).toContain('$0.40')
    expect(wrapper.get('[data-testid="account-key-usage-standard-cost"]').text()).toContain('$5.00')
    expect(wrapper.get('[data-testid="account-key-usage-saved"]').text()).toContain('$4.60 / 92%')
    expect(wrapper.text()).not.toContain('US')
  })

  it('shows all available quota windows and unlimited when no limits exist', () => {
    const quotaWrapper = mount(AccountKeyUsageSummaryCell, {
      props: {
        account: {
          id: 2,
          type: 'apikey',
          platform: 'openai',
          quota_daily_used: 1,
          quota_daily_limit: 10,
          quota_weekly_used: 2,
          quota_weekly_limit: 20,
          quota_monthly_used: 3,
          quota_monthly_limit: 30,
          quota_used: 4,
          quota_limit: 40,
          quota_daily_reset_at: '2026-03-14T12:00:00Z',
          quota_weekly_reset_at: '2026-03-20T12:00:00Z',
          quota_monthly_reset_at: '2026-04-13T12:00:00Z',
          extra: {},
        } as any,
        stats: {
          requests: 0,
          tokens: 0,
          cost: 0,
        } as any,
      }
    })

    expect(quotaWrapper.find('[data-testid="account-key-usage-unlimited"]').exists()).toBe(false)
    const todayRow = quotaWrapper.get('[data-testid="account-key-usage-today-row"]')
    const quotaRow = quotaWrapper.get('[data-testid="account-key-usage-quota-row"]')
    expect(todayRow.text()).toContain('Req')
    expect(todayRow.text()).not.toContain('1D')
    expect(quotaRow.text()).toContain('1D')
    expect(quotaWrapper.get('[data-testid="account-key-quota-daily"]').text()).toContain('1D')
    expect(quotaWrapper.get('[data-testid="account-key-quota-weekly"]').text()).toContain('7D')
    expect(quotaWrapper.get('[data-testid="account-key-quota-monthly"]').text()).toContain('30D')
    expect(quotaWrapper.get('[data-testid="account-key-quota-total"]').text()).toContain('Total')

    const unlimitedWrapper = mount(AccountKeyUsageSummaryCell, {
      props: {
        account: {
          id: 3,
          type: 'apikey',
          platform: 'openai',
          extra: {},
        } as any,
        stats: {
          requests: 0,
          tokens: 0,
          cost: 0,
        } as any,
      }
    })

    expect(unlimitedWrapper.get('[data-testid="account-key-usage-unlimited"]').text()).toContain('Unlimited')
  })
})
