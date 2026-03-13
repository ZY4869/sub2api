import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import AccountUsageCell from '../AccountUsageCell.vue'
import { resetAccountUsagePresentationCache } from '@/composables/useAccountUsagePresentation'

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
      t: (key: string, params?: Record<string, unknown>) => {
        const dict: Record<string, string> = {
          'admin.accounts.usageWindow.snapshotUpdatedAt': 'Snapshot updated {time}',
          'admin.accounts.usageWindow.gemini3Image': 'Gemini Image',
          'admin.accounts.usageWindow.gemini3Pro': 'G3P',
          'admin.accounts.usageWindow.gemini3Flash': 'G3F',
          'admin.accounts.usageWindow.claude': 'Claude',
          'admin.accounts.gemini.rateLimit.unlimited': 'Unlimited',
          'admin.accounts.ineligibleWarning': 'Ineligible warning',
          'admin.accounts.gemini.quotaPolicy.title': 'Quota policy',
          'admin.accounts.gemini.quotaPolicy.note': 'Quota note',
          'admin.accounts.gemini.quotaPolicy.columns.docs': 'Docs',
          'dates.today': 'Today',
          'dates.tomorrow': 'Tomorrow',
          'common.error': 'Error',
        }
        let value = dict[key] ?? key
        if (params) {
          Object.entries(params).forEach(([paramKey, paramValue]) => {
            value = value.replace(`{${paramKey}}`, String(paramValue))
          })
        }
        return value
      },
    }),
  }
})

const usageBarStub = {
  props: ['label', 'utilization', 'resetsAt', 'remainingSeconds', 'windowStats', 'inlineReset', 'color'],
  template:
    '<div class="usage-bar">{{ label }}|{{ utilization }}|{{ resetsAt }}|{{ remainingSeconds }}|{{ inlineReset }}|{{ windowStats?.tokens }}</div>',
}

describe('AccountUsageCell', () => {
  beforeEach(() => {
    getUsage.mockReset()
    resetAccountUsagePresentationCache()
  })

  it('aggregates antigravity image usage from multiple models', async () => {
    getUsage.mockResolvedValue({
      antigravity_quota: {
        'gemini-2.5-flash-image': {
          utilization: 45,
          reset_time: '2026-03-01T11:00:00Z',
        },
        'gemini-3.1-flash-image': {
          utilization: 20,
          reset_time: '2026-03-01T10:00:00Z',
        },
        'gemini-3-pro-image': {
          utilization: 70,
          reset_time: '2026-03-01T09:00:00Z',
        },
      },
    })

    const wrapper = mount(AccountUsageCell, {
      props: {
        account: {
          id: 1001,
          platform: 'antigravity',
          type: 'oauth',
          extra: {},
        } as any,
      },
      global: {
        stubs: {
          UsageProgressBar: usageBarStub,
        },
      },
    })

    await flushPromises()

    expect(wrapper.text()).toContain('Gemini Image|70|2026-03-01T09:00:00Z')
  })

  it('refreshes stale openai codex snapshots from the usage endpoint', async () => {
    getUsage.mockResolvedValue({
      five_hour: {
        utilization: 15,
        resets_at: '2026-03-08T12:00:00Z',
        remaining_seconds: 3600,
        window_stats: {
          requests: 3,
          tokens: 300,
          cost: 0.03,
          standard_cost: 0.03,
          user_cost: 0.03,
        },
      },
      seven_day: {
        utilization: 77,
        resets_at: '2026-03-13T12:00:00Z',
        remaining_seconds: 3600,
        window_stats: {
          requests: 3,
          tokens: 300,
          cost: 0.03,
          standard_cost: 0.03,
          user_cost: 0.03,
        },
      },
    })

    const wrapper = mount(AccountUsageCell, {
      props: {
        account: {
          id: 2000,
          platform: 'openai',
          type: 'oauth',
          extra: {
            codex_usage_updated_at: '2026-03-07T00:00:00Z',
            codex_5h_used_percent: 12,
            codex_5h_reset_at: '2026-03-08T12:00:00Z',
            codex_7d_used_percent: 34,
            codex_7d_reset_at: '2026-03-13T12:00:00Z',
          },
        } as any,
      },
      global: {
        stubs: {
          UsageProgressBar: usageBarStub,
        },
      },
    })

    await flushPromises()

    expect(getUsage).toHaveBeenCalledWith(2000)
    expect(wrapper.text()).toContain('5h|15|2026-03-08T12:00:00Z|3600|true|300')
    expect(wrapper.text()).toContain('7d|77|2026-03-13T12:00:00Z|3600|true|300')
  })

  it('keeps using local openai snapshots when they are still fresh', async () => {
    const wrapper = mount(AccountUsageCell, {
      props: {
        account: {
          id: 2001,
          platform: 'openai',
          type: 'oauth',
          extra: {
            codex_usage_updated_at: '2099-03-07T10:00:00Z',
            codex_5h_used_percent: 12,
            codex_5h_reset_at: '2099-03-07T12:00:00Z',
            codex_7d_used_percent: 34,
            codex_7d_reset_at: '2099-03-13T12:00:00Z',
          },
        } as any,
      },
      global: {
        stubs: {
          UsageProgressBar: usageBarStub,
        },
      },
    })

    await flushPromises()

    expect(getUsage).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('5h|12|2099-03-07T12:00:00.000Z||true')
    expect(wrapper.text()).toContain('7d|34|2099-03-13T12:00:00.000Z||true')
  })

  it('hides openai identity and model summaries but keeps snapshot update text', async () => {
    const wrapper = mount(AccountUsageCell, {
      props: {
        account: {
          id: 2005,
          platform: 'openai',
          type: 'oauth',
          credentials: {
            plan_type: 'plus',
            chatgpt_account_id: 'acc_1234567890',
          },
          extra: {
            codex_usage_updated_at: '2099-03-07T10:00:00Z',
            codex_5h_used_percent: 12,
            codex_5h_reset_at: '2099-03-07T12:00:00Z',
            codex_7d_used_percent: 34,
            codex_7d_reset_at: '2099-03-13T12:00:00Z',
            openai_known_models: ['gpt-5.4', 'gpt-4.1-mini', 'o4-mini', 'gpt-4o'],
          },
        } as any,
      },
      global: {
        stubs: {
          UsageProgressBar: usageBarStub,
        },
      },
    })

    await flushPromises()

    expect(wrapper.text()).not.toContain('acc_12...7890')
    expect(wrapper.text()).not.toContain('gpt-5.4')
    expect(wrapper.text()).toContain('Snapshot updated')
    expect(getUsage).not.toHaveBeenCalled()
  })

  it('falls back to fetched usage windows when no codex snapshot exists', async () => {
    getUsage.mockResolvedValue({
      five_hour: {
        utilization: 0,
        resets_at: null,
        remaining_seconds: 0,
        window_stats: {
          requests: 2,
          tokens: 27700,
          cost: 0.06,
          standard_cost: 0.06,
          user_cost: 0.06,
        },
      },
      seven_day: {
        utilization: 0,
        resets_at: null,
        remaining_seconds: 0,
        window_stats: {
          requests: 2,
          tokens: 27700,
          cost: 0.06,
          standard_cost: 0.06,
          user_cost: 0.06,
        },
      },
    })

    const wrapper = mount(AccountUsageCell, {
      props: {
        account: {
          id: 2002,
          platform: 'openai',
          type: 'oauth',
          extra: {},
        } as any,
      },
      global: {
        stubs: {
          UsageProgressBar: usageBarStub,
        },
      },
    })

    await flushPromises()

    expect(getUsage).toHaveBeenCalledWith(2002)
    expect(wrapper.text()).toContain('5h|0||0|true|27700')
    expect(wrapper.text()).toContain('7d|0||0|true|27700')
  })

  it('reloads openai usage when the row refresh key changes without a codex snapshot', async () => {
    getUsage
      .mockResolvedValueOnce({
        five_hour: {
          utilization: 0,
          resets_at: null,
          remaining_seconds: 0,
          window_stats: {
            requests: 1,
            tokens: 100,
            cost: 0.01,
            standard_cost: 0.01,
            user_cost: 0.01,
          },
        },
        seven_day: null,
      })
      .mockResolvedValueOnce({
        five_hour: {
          utilization: 0,
          resets_at: null,
          remaining_seconds: 0,
          window_stats: {
            requests: 2,
            tokens: 200,
            cost: 0.02,
            standard_cost: 0.02,
            user_cost: 0.02,
          },
        },
        seven_day: null,
      })

    const wrapper = mount(AccountUsageCell, {
      props: {
        account: {
          id: 2003,
          platform: 'openai',
          type: 'oauth',
          updated_at: '2026-03-07T10:00:00Z',
          extra: {},
        } as any,
      },
      global: {
        stubs: {
          UsageProgressBar: usageBarStub,
        },
      },
    })

    await flushPromises()
    expect(wrapper.text()).toContain('5h|0||0|true|100')
    expect(getUsage).toHaveBeenCalledTimes(1)

    await wrapper.setProps({
      account: {
        id: 2003,
        platform: 'openai',
        type: 'oauth',
        updated_at: '2026-03-07T10:01:00Z',
        extra: {},
      } as any,
    })

    await flushPromises()

    expect(getUsage).toHaveBeenCalledTimes(2)
    expect(wrapper.text()).toContain('5h|0||0|true|200')
  })

  it('prefers fetched openai usage when the account is actively rate limited', async () => {
    getUsage.mockResolvedValue({
      five_hour: {
        utilization: 100,
        resets_at: '2026-03-07T12:00:00Z',
        remaining_seconds: 3600,
        window_stats: {
          requests: 211,
          tokens: 106540000,
          cost: 38.13,
          standard_cost: 38.13,
          user_cost: 38.13,
        },
      },
      seven_day: {
        utilization: 100,
        resets_at: '2026-03-13T12:00:00Z',
        remaining_seconds: 3600,
        window_stats: {
          requests: 211,
          tokens: 106540000,
          cost: 38.13,
          standard_cost: 38.13,
          user_cost: 38.13,
        },
      },
    })

    const wrapper = mount(AccountUsageCell, {
      props: {
        account: {
          id: 2004,
          platform: 'openai',
          type: 'oauth',
          rate_limit_reset_at: '2099-03-07T12:00:00Z',
          extra: {
            codex_5h_used_percent: 0,
            codex_7d_used_percent: 0,
          },
        } as any,
      },
      global: {
        stubs: {
          UsageProgressBar: usageBarStub,
        },
      },
    })

    await flushPromises()

    expect(getUsage).toHaveBeenCalledWith(2004)
    expect(wrapper.text()).toContain('5h|100|2026-03-07T12:00:00Z|3600|true|106540000')
    expect(wrapper.text()).toContain('7d|100|2026-03-13T12:00:00Z|3600|true|106540000')
    expect(wrapper.text()).not.toContain('5h|0|')
  })
})
