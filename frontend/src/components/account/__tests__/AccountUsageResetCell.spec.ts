import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'
import AccountUsageCell from '../AccountUsageCell.vue'
import AccountUsageResetCell from '../AccountUsageResetCell.vue'
import { resetAccountUsagePresentationCache } from '@/composables/useAccountUsagePresentation'
import { resetUiNowForTests } from '@/composables/useUiNow'

const { getUsage } = vi.hoisted(() => ({
  getUsage: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      getUsage,
    },
  },
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => {
        const dict: Record<string, string> = {
          'dates.today': 'Today',
          'dates.tomorrow': 'Tomorrow',
          'common.error': 'Error',
          'admin.accounts.usageWindow.snapshotUpdatedAt': 'Snapshot updated {time}',
          'admin.accounts.usageWindow.now': 'Now',
          'admin.accounts.gemini.rateLimit.unlimited': 'Unlimited',
        }
        return dict[key] ?? key
      },
    }),
  }
})

const usageBarStub = {
  props: ['label', 'utilization', 'resetsAt', 'remainingSeconds', 'windowStats', 'inlineReset', 'color'],
  template: '<div>{{ label }}|{{ utilization }}</div>',
}

enableAutoUnmount(afterEach)

describe('AccountUsageResetCell', () => {
  beforeEach(() => {
    getUsage.mockReset()
    resetAccountUsagePresentationCache()
    resetUiNowForTests()
  })

  afterEach(() => {
    resetUiNowForTests()
    vi.useRealTimers()
  })

  it('renders separate reset rows for 5h and 7d windows', async () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-03-13T12:29:00'))

    const wrapper = mount(AccountUsageResetCell, {
      props: {
        account: {
          id: 3001,
          platform: 'openai',
          type: 'oauth',
          extra: {
            codex_usage_updated_at: '2099-03-07T10:00:00Z',
            codex_5h_used_percent: 78,
            codex_5h_reset_at: '2026-03-13T15:22:00',
            codex_7d_used_percent: 24,
            codex_7d_reset_at: '2026-03-20T01:09:00',
          },
        } as any,
      },
    })

    await flushPromises()

    expect(wrapper.text()).toContain('5h')
    expect(wrapper.text()).toContain('2h 53m')
    expect(wrapper.text()).toContain('·')
    expect(wrapper.text()).toContain('Today 15:22')
    expect(wrapper.text()).toContain('7d')
    expect(wrapper.text()).toContain('6d 13h')
    expect(wrapper.text()).toContain('03-20 01:09')

  })

  it('updates day labels when the shared clock crosses midnight', async () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-03-13T23:59:00'))
    getUsage.mockResolvedValue({})

    const wrapper = mount(AccountUsageResetCell, {
      props: {
        account: {
          id: 3004,
          platform: 'openai',
          type: 'oauth',
          extra: {
            codex_usage_updated_at: '2099-03-13T10:00:00Z',
            codex_5h_used_percent: 10,
            codex_5h_reset_at: '2026-03-14T00:01:00',
          },
        } as any,
      },
    })

    await flushPromises()
    expect(wrapper.text()).toContain('2m')
    expect(wrapper.text()).toContain('Tomorrow 00:01')

    vi.advanceTimersByTime(2 * 60 * 1000)
    await flushPromises()

    expect(wrapper.text()).toContain('Now')
    expect(wrapper.text()).toContain('Today 00:01')
  })

  it('falls back to a dash when no reset rows are available', async () => {
    getUsage.mockResolvedValue({})

    const wrapper = mount(AccountUsageResetCell, {
      props: {
        account: {
          id: 3002,
          platform: 'openai',
          type: 'oauth',
          extra: {},
        } as any,
      },
    })

    await flushPromises()

    expect(wrapper.text()).toBe('-')
  })

  it('shares the same usage fetch with the usage cell', async () => {
    getUsage.mockResolvedValue({
      five_hour: {
        utilization: 78,
        resets_at: '2026-03-13T15:22:00Z',
        remaining_seconds: 10380,
        window_stats: {
          requests: 3,
          tokens: 1200,
          cost: 0.03,
          standard_cost: 0.03,
          user_cost: 0.03,
        },
      },
      seven_day: {
        utilization: 24,
        resets_at: '2026-03-20T01:09:00Z',
        remaining_seconds: 565740,
        window_stats: {
          requests: 5,
          tokens: 2400,
          cost: 0.08,
          standard_cost: 0.08,
          user_cost: 0.08,
        },
      },
    })

    const account = {
      id: 3003,
      platform: 'openai',
      type: 'oauth',
      extra: {},
    } as any

    mount(AccountUsageCell, {
      props: { account },
      global: {
        stubs: {
          UsageProgressBar: usageBarStub,
        },
      },
    })

    mount(AccountUsageResetCell, {
      props: { account },
    })

    await flushPromises()

    expect(getUsage).toHaveBeenCalledTimes(1)
    expect(getUsage).toHaveBeenCalledWith(3003, { force: undefined, source: undefined })
  })
})
