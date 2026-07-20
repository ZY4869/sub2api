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
        'admin.accounts.keyUsage.inputTokens': 'In',
        'admin.accounts.keyUsage.outputTokens': 'Out',
        'admin.accounts.keyUsage.discountedCost': 'Discount',
        'admin.accounts.keyUsage.standardCost': 'Standard',
        'admin.accounts.keyUsage.saved': 'Saved',
        'admin.accounts.keyUsage.successRate': 'Success',
        'admin.accounts.keyUsage.avgLatency': 'Latency',
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
  it('renders 9 items in grid layout with input/output tokens, success rate, and latency', () => {
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
          input_tokens: 100,
          output_tokens: 245,
          cost: 0.4,
          standard_cost: 5,
          user_cost: 0.4,
          success_rate: 0.985,
          average_duration_ms: 450,
        } as any,
      }
    })

    const todayRow = wrapper.get('[data-testid="account-key-usage-today-row"]')
    const todayRow2 = wrapper.get('[data-testid="account-key-usage-today-row-2"]')

    // Row 1 uses grid-cols-5
    expect(todayRow.classes()).toContain('grid')
    expect(todayRow.classes()).toContain('grid-cols-5')

    // Row 2 uses grid-cols-4
    expect(todayRow2.classes()).toContain('grid')
    expect(todayRow2.classes()).toContain('grid-cols-4')

    // Row 1 items: requests, input-tokens, output-tokens, discounted-cost, success-rate
    expect(wrapper.get('[data-testid="account-key-usage-requests"]').text()).toBe('12')
    expect(wrapper.get('[data-testid="account-key-usage-requests"]').attributes('title')).toBe('Req: 12')
    expect(wrapper.get('[data-testid="account-key-usage-input-tokens"]').text()).toBe('100T')
    expect(wrapper.get('[data-testid="account-key-usage-input-tokens"]').attributes('title')).toBe('In: 100T')
    expect(wrapper.get('[data-testid="account-key-usage-output-tokens"]').text()).toBe('245T')
    expect(wrapper.get('[data-testid="account-key-usage-output-tokens"]').attributes('title')).toBe('Out: 245T')
    expect(wrapper.get('[data-testid="account-key-usage-discounted-cost"]').text()).toBe('$0.40')
    expect(wrapper.get('[data-testid="account-key-usage-discounted-cost"]').attributes('title')).toBe('Discount: $0.40')
    expect(wrapper.get('[data-testid="account-key-usage-success-rate"]').text()).toBe('98.5%')
    expect(wrapper.get('[data-testid="account-key-usage-success-rate"]').attributes('title')).toBe('Success: 98.5%')

    // Row 2 items: standard-cost, saved, avg-latency, quota
    expect(wrapper.get('[data-testid="account-key-usage-standard-cost"]').text()).toBe('$5.00')
    expect(wrapper.get('[data-testid="account-key-usage-standard-cost"]').attributes('title')).toBe('Standard: $5.00')
    expect(wrapper.get('[data-testid="account-key-usage-saved"]').text()).toBe('$4.60 / 92%')
    expect(wrapper.get('[data-testid="account-key-usage-saved"]').attributes('title')).toBe('Saved: $4.60 / 92%')
    expect(wrapper.get('[data-testid="account-key-usage-avg-latency"]').text()).toBe('450ms')
    expect(wrapper.get('[data-testid="account-key-usage-avg-latency"]').attributes('title')).toBe('Latency: 450ms')
    expect(wrapper.get('[data-testid="account-key-usage-quota"]').text()).toContain('Unlimited')

    // No US prefix or label text in pill values
    expect(wrapper.text()).not.toContain('US')
    expect(todayRow.text()).not.toContain('Req')
    expect(todayRow.text()).not.toContain('In')
    expect(todayRow.text()).not.toContain('Out')
  })

  it('shows quota summary with most restrictive window', () => {
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

    const quotaPill = quotaWrapper.get('[data-testid="account-key-usage-quota"]')
    expect(quotaPill.text()).toContain('1D')
    expect(quotaWrapper.find('[data-testid="account-key-usage-unlimited"]').exists()).toBe(false)

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

    expect(unlimitedWrapper.get('[data-testid="account-key-usage-quota"]').text()).toContain('Unlimited')
  })

  it('handles missing stats with dash fallback for success rate and latency', () => {
    const wrapper = mount(AccountKeyUsageSummaryCell, {
      props: {
        account: {
          id: 4,
          type: 'apikey',
          platform: 'openai',
          extra: {},
        } as any,
        stats: {
          requests: 5,
          tokens: 100,
          cost: 1,
        } as any,
      }
    })

    expect(wrapper.get('[data-testid="account-key-usage-success-rate"]').text()).toBe('—')
    expect(wrapper.get('[data-testid="account-key-usage-avg-latency"]').text()).toBe('—')
    expect(wrapper.get('[data-testid="account-key-usage-input-tokens"]').text()).toBe('0T')
    expect(wrapper.get('[data-testid="account-key-usage-output-tokens"]').text()).toBe('0T')
  })
})
