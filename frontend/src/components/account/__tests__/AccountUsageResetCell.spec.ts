import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, flushPromises, mount, type MountingOptions } from '@vue/test-utils'
import { createPinia } from 'pinia'
import AccountUsageCell from '../AccountUsageCell.vue'
import AccountUsageResetCell from '../AccountUsageResetCell.vue'
import { resetAccountUsagePresentationCache } from '@/composables/useAccountUsagePresentation'
import { resetUiNowForTests } from '@/composables/useUiNow'

let confirmSpy: ReturnType<typeof vi.spyOn>

const { getUsage, resetAccountQuota, showSuccess, showError } = vi.hoisted(() => ({
  getUsage: vi.fn(),
  resetAccountQuota: vi.fn(),
  showSuccess: vi.fn(),
  showError: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      getUsage,
      resetAccountQuota,
    },
  },
}))

vi.mock('@/api', () => ({
  adminAPI: {
    accounts: {
      getUsage,
      resetAccountQuota,
    },
  },
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    showSuccess,
    showError,
  }),
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showSuccess,
    showError,
  }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => {
        const dict: Record<string, string> = {
          'dates.today': 'Today',
          'dates.tomorrow': 'Tomorrow',
          'common.error': 'Error',
          'admin.accounts.usageWindow.snapshotUpdatedAt': 'Snapshot updated {time}',
          'admin.accounts.usageWindow.now': 'Now',
          'admin.accounts.usageWindow.spark5h': 'Spark 5h',
          'admin.accounts.usageWindow.spark7d': 'Spark 7d',
          'admin.accounts.usageWindow.refreshResetCredits': 'Refresh count',
          'admin.accounts.usageWindow.refreshingResetCredits': 'Refreshing',
          'admin.accounts.usageWindow.refreshResetCreditsTitle': 'Refresh OpenAI reset credits',
          'admin.accounts.usageWindow.refreshResetCreditsSuccess': 'Reset credits refreshed',
          'admin.accounts.usageWindow.refreshResetCreditsFailed': 'Reset credits refresh failed',
          'admin.accounts.usageWindow.resetQuota': 'Reset quota',
          'admin.accounts.usageWindow.resettingQuota': 'Resetting',
          'admin.accounts.usageWindow.resetQuotaRemaining': '{count} resets left',
          'admin.accounts.usageWindow.resetQuotaUnsupported': 'Real reset unsupported',
          'admin.accounts.usageWindow.resetQuotaConfirm': 'Use real reset?',
          'admin.accounts.usageWindow.resetQuotaSuccess': 'Quota reset',
          'admin.accounts.usageWindow.resetQuotaFailed': 'Quota reset failed',
          'admin.accounts.usageWindow.resetQuotaNoCredit': 'No reset credits available',
          'admin.accounts.usageWindow.resetQuotaNothingToReset': 'Nothing to reset',
          'admin.accounts.gemini.rateLimit.unlimited': 'Unlimited',
        }
        let message = dict[key] ?? key
        for (const [name, value] of Object.entries(params || {})) {
          message = message.replace(`{${name}}`, String(value))
        }
        return message
      },
    }),
  }
})

const usageBarStub = {
  props: ['label', 'utilization', 'resetsAt', 'remainingSeconds', 'windowStats', 'inlineReset', 'color'],
  template: '<div>{{ label }}|{{ utilization }}</div>',
}

function mountWithPinia(component: any, options: MountingOptions<any>) {
  return mount(component, {
    ...options,
    global: {
      ...(options.global ?? {}),
      plugins: [createPinia(), ...((options.global?.plugins ?? []) as any[])],
    },
  })
}

enableAutoUnmount(afterEach)

describe('AccountUsageResetCell', () => {
  beforeEach(() => {
    getUsage.mockReset()
    resetAccountQuota.mockReset()
    showSuccess.mockReset()
    showError.mockReset()
    confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(true)
    resetAccountUsagePresentationCache()
    resetUiNowForTests()
  })

  afterEach(() => {
    resetUiNowForTests()
    vi.useRealTimers()
    confirmSpy?.mockRestore()
  })

  it('renders separate reset rows with dynamic 5H and 30D window labels', async () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-03-13T12:29:00'))

    const wrapper = mountWithPinia(AccountUsageResetCell, {
      props: {
        account: {
          id: 3001,
          platform: 'openai',
          type: 'oauth',
          extra: {
            codex_usage_updated_at: '2099-03-07T10:00:00Z',
            codex_5h_used_percent: 78,
            codex_5h_window_minutes: 300,
            codex_5h_reset_at: '2026-03-13T15:22:00',
            codex_7d_used_percent: 24,
            codex_7d_window_minutes: 43200,
            codex_7d_reset_at: '2026-03-20T01:09:00',
          },
        } as any,
      },
    })

    await flushPromises()

    expect(wrapper.text()).toContain('5H')
    expect(wrapper.text()).toContain('2h 53m')
    expect(wrapper.text()).toContain('·')
    expect(wrapper.text()).toContain('Today 15:22:00')
    expect(wrapper.text()).toContain('30D')
    expect(wrapper.text()).toContain('6d 13h')
    expect(wrapper.text()).toContain('03-20 01:09:00')
    const labels = wrapper.findAll('[data-testid="account-usage-reset-window-label"]')
    expect(labels[0].classes()).toContain('bg-indigo-50')
    expect(labels[1].classes()).toContain('bg-green-50')
    expect(wrapper.text()).not.toContain('30D · 03-20')

  })

  it('keeps pro openai normal and spark reset rows aligned with their labels', async () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-03-13T12:00:00'))

    const wrapper = mountWithPinia(AccountUsageResetCell, {
      props: {
        account: {
          id: 3005,
          platform: 'openai',
          type: 'oauth',
          credentials: {
            plan_type: 'pro',
          },
          extra: {
            codex_usage_updated_at: '2099-03-07T10:00:00Z',
            codex_5h_used_percent: 11,
            codex_5h_window_minutes: 300,
            codex_5h_reset_at: '2026-03-13T13:00:00',
            codex_7d_used_percent: 22,
            codex_7d_window_minutes: 43200,
            codex_7d_reset_at: '2026-03-13T14:00:00',
            codex_spark_5h_used_percent: 33,
            codex_spark_5h_window_minutes: 300,
            codex_spark_5h_reset_at: '2026-03-13T15:00:00',
            codex_spark_7d_used_percent: 44,
            codex_spark_7d_window_minutes: 43200,
            codex_spark_7d_reset_at: '2026-03-13T16:00:00',
          },
        } as any,
      },
    })

    await flushPromises()

    const text = wrapper.text()
    expect(text).toContain('5H')
    expect(text).toContain('1h')
    expect(text).toContain('Today 13:00:00')
    expect(text).toContain('30D')
    expect(text).toContain('2h')
    expect(text).toContain('Today 14:00:00')
    expect(text).toContain('Spark 5H')
    expect(text).toContain('3h')
    expect(text).toContain('Today 15:00:00')
    expect(text).toContain('Spark 30D')
    expect(text).toContain('4h')
    expect(text).toContain('Today 16:00:00')
  })

  it('treats 31D reset labels as monthly green capsules', async () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-03-13T12:00:00'))

    const wrapper = mountWithPinia(AccountUsageResetCell, {
      props: {
        account: {
          id: 3006,
          platform: 'openai',
          type: 'oauth',
          extra: {
            codex_usage_updated_at: '2099-03-07T10:00:00Z',
            codex_7d_used_percent: 24,
            codex_7d_window_minutes: 44640,
            codex_7d_reset_at: '2026-03-20T01:09:00',
          },
        } as any,
      },
    })

    await flushPromises()

    expect(wrapper.text()).toContain('31D')
    const label = wrapper.get('[data-testid="account-usage-reset-window-label"]')
    expect(label.classes()).toContain('bg-green-50')
    expect(label.classes()).not.toContain('bg-orange-50')
  })

  it('updates day labels when the shared clock crosses midnight', async () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-03-13T23:59:00'))
    getUsage.mockResolvedValue({})

    const wrapper = mountWithPinia(AccountUsageResetCell, {
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
    expect(wrapper.text()).toContain('Tomorrow 00:01:00')

    vi.advanceTimersByTime(2 * 60 * 1000)
    await flushPromises()

    expect(wrapper.text()).toContain('Now')
    expect(wrapper.text()).toContain('Today 00:01:00')
  })

  it('falls back to a dash when no reset rows are available', async () => {
    getUsage.mockResolvedValue({})

    const wrapper = mountWithPinia(AccountUsageResetCell, {
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

    mountWithPinia(AccountUsageCell, {
      props: { account },
      global: {
        stubs: {
          UsageProgressBar: usageBarStub,
        },
      },
    })

    mountWithPinia(AccountUsageResetCell, {
      props: { account },
    })

    await flushPromises()

    expect(getUsage).toHaveBeenCalledTimes(1)
    expect(getUsage).toHaveBeenCalledWith(3003, { force: undefined, source: undefined })
  })

  it('resets openai quota from the usage reset cell and reloads active usage', async () => {
    resetAccountQuota.mockResolvedValue({})
    getUsage.mockResolvedValueOnce({
      five_hour: {
        utilization: 90,
        resets_at: '2026-03-13T15:22:00Z',
      },
    })
    getUsage.mockResolvedValueOnce({
      five_hour: {
        utilization: 0,
        resets_at: '2026-03-13T15:22:00Z',
      },
    })

    const wrapper = mountWithPinia(AccountUsageResetCell, {
      props: {
        account: {
          id: 3007,
          platform: 'openai',
          type: 'oauth',
          extra: {
            codex_usage_updated_at: '2099-03-07T10:00:00Z',
            codex_5h_used_percent: 90,
            codex_5h_reset_at: '2026-03-13T15:22:00',
            openai_rate_limit_reset_credits_available_count: 1,
          },
        } as any,
      },
    })

    await flushPromises()
    const button = wrapper.get('[data-testid="account-usage-reset-quota-button"]')
    expect(button.text()).toContain('Reset quota')

    await button.trigger('click')
    await flushPromises()

    expect(window.confirm).toHaveBeenCalledWith('Use real reset?')
    expect(resetAccountQuota).toHaveBeenCalledWith(3007)
    expect(getUsage).toHaveBeenCalledWith(3007, { force: true, source: 'active' })
    expect(showSuccess).toHaveBeenCalledWith('Quota reset')
  })

  it('keeps reset credit count and refresh controls out of the reset date column', async () => {
    const wrapper = mountWithPinia(AccountUsageResetCell, {
      props: {
        account: {
          id: 3014,
          platform: 'openai',
          type: 'oauth',
          extra: {
            codex_usage_updated_at: '2099-03-07T10:00:00Z',
            codex_5h_used_percent: 10,
            codex_5h_reset_at: '2026-03-13T15:22:00',
            openai_rate_limit_reset_credits_available_count: 2,
          },
        } as any,
      },
    })

    await flushPromises()

    expect(wrapper.get('[data-testid="account-usage-reset-quota-button"]').text()).toContain('Reset quota')
    expect(wrapper.find('[data-testid="account-usage-reset-credits-refresh"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="account-usage-reset-quota-remaining"]').exists()).toBe(false)
  })

  it('shows the openai quota reset remaining count from account extra', async () => {
    const wrapper = mountWithPinia(AccountUsageResetCell, {
      props: {
        account: {
          id: 3008,
          platform: 'openai',
          type: 'oauth',
          extra: {
            codex_usage_updated_at: '2099-03-07T10:00:00Z',
            codex_5h_used_percent: 10,
            codex_5h_reset_at: '2026-03-13T15:22:00',
            openai_rate_limit_reset_credits_available_count: '3',
          },
        } as any,
      },
    })

    await flushPromises()

    expect(wrapper.get('[data-testid="account-usage-reset-quota-button"]').text()).toContain('Reset quota')
    expect(wrapper.find('[data-testid="account-usage-reset-quota-remaining"]').exists()).toBe(false)
  })

  it('shows unknown openai quota reset remaining count when no real count exists', async () => {
    const wrapper = mountWithPinia(AccountUsageResetCell, {
      props: {
        account: {
          id: 3009,
          platform: 'openai',
          type: 'oauth',
          extra: {
            codex_usage_updated_at: '2099-03-07T10:00:00Z',
            codex_5h_used_percent: 10,
            codex_5h_reset_at: '2026-03-13T15:22:00',
          },
        } as any,
      },
    })

    await flushPromises()

    expect(wrapper.get('[data-testid="account-usage-reset-quota-button"]').text()).toContain('Reset quota')
    expect(wrapper.find('[data-testid="account-usage-reset-quota-remaining"]').exists()).toBe(false)
  })

  it('keeps usage unknown authoritative over stale account extra reset credits', async () => {
    getUsage.mockResolvedValue({
      openai_reset_credits: {
        available_count: null,
        status: 'unknown_or_unsupported',
        source: 'codex_app_server',
      },
      five_hour: {
        utilization: 10,
        resets_at: '2026-03-13T15:22:00Z',
      },
    })

    const wrapper = mountWithPinia(AccountUsageResetCell, {
      props: {
        account: {
          id: 3011,
          platform: 'openai',
          type: 'oauth',
          extra: {
            codex_usage_updated_at: '2020-03-07T10:00:00Z',
            codex_5h_used_percent: 10,
            codex_5h_reset_at: '2026-03-13T15:22:00',
            openai_rate_limit_reset_credits_available_count: 3,
          },
        } as any,
      },
    })

    await flushPromises()

    expect(getUsage).toHaveBeenCalledWith(3011, { force: undefined, source: undefined })
    expect(wrapper.find('[data-testid="account-usage-reset-quota-remaining"]').exists()).toBe(false)
  })

  it('shows unknown when active usage refresh fails with stale reset credit extra', async () => {
    getUsage.mockRejectedValue(new Error('Codex app-server unavailable'))

    const wrapper = mountWithPinia(AccountUsageResetCell, {
      props: {
        account: {
          id: 3012,
          platform: 'openai',
          type: 'oauth',
          extra: {
            codex_usage_updated_at: '2020-03-07T10:00:00Z',
            codex_5h_used_percent: 10,
            codex_5h_reset_at: '2026-03-13T15:22:00',
            openai_rate_limit_reset_credits_available_count: 3,
            openai_rate_limit_reset_credits_status: 'unknown_or_unsupported',
            openai_rate_limit_reset_credits_updated_at: null,
            openai_rate_limits_app_server_updated_at: '2026-03-13T15:00:00Z',
          },
        } as any,
      },
    })

    await flushPromises()

    expect(getUsage).toHaveBeenCalledWith(3012, { force: undefined, source: undefined })
    expect(wrapper.find('[data-testid="account-usage-reset-quota-remaining"]').exists()).toBe(false)
  })

  it('shows friendly no-credit reset error and refreshes usage', async () => {
    resetAccountQuota.mockRejectedValue({
      response: {
        data: {
          reason: 'OPENAI_RESET_CREDITS_NO_CREDIT',
          message: 'raw backend message',
        },
      },
    })
    getUsage.mockResolvedValue({
      five_hour: {
        utilization: 10,
        resets_at: '2026-03-13T15:22:00Z',
      },
    })

    const wrapper = mountWithPinia(AccountUsageResetCell, {
      props: {
        account: {
          id: 3012,
          platform: 'openai',
          type: 'oauth',
          extra: {
            codex_usage_updated_at: '2099-03-07T10:00:00Z',
            codex_5h_used_percent: 10,
            codex_5h_reset_at: '2026-03-13T15:22:00',
            openai_rate_limit_reset_credits_available_count: 1,
          },
        } as any,
      },
    })

    await flushPromises()
    await wrapper.get('[data-testid="account-usage-reset-quota-button"]').trigger('click')
    await flushPromises()

    expect(resetAccountQuota).toHaveBeenCalledWith(3012)
    expect(getUsage).toHaveBeenCalledWith(3012, { force: true, source: 'active' })
    expect(showError).toHaveBeenCalledWith('No reset credits available')
  })

  it('shows friendly nothing-to-reset error and refreshes usage', async () => {
    resetAccountQuota.mockRejectedValue({
      response: {
        data: {
          error: 'OPENAI_RESET_CREDITS_NOTHING_TO_RESET',
          message: 'raw backend message',
        },
      },
    })
    getUsage.mockResolvedValue({
      five_hour: {
        utilization: 10,
        resets_at: '2026-03-13T15:22:00Z',
      },
    })

    const wrapper = mountWithPinia(AccountUsageResetCell, {
      props: {
        account: {
          id: 3013,
          platform: 'openai',
          type: 'oauth',
          extra: {
            codex_usage_updated_at: '2099-03-07T10:00:00Z',
            codex_5h_used_percent: 10,
            codex_5h_reset_at: '2026-03-13T15:22:00',
            openai_rate_limit_reset_credits_available_count: 1,
          },
        } as any,
      },
    })

    await flushPromises()
    await wrapper.get('[data-testid="account-usage-reset-quota-button"]').trigger('click')
    await flushPromises()

    expect(resetAccountQuota).toHaveBeenCalledWith(3013)
    expect(getUsage).toHaveBeenCalledWith(3013, { force: true, source: 'active' })
    expect(showError).toHaveBeenCalledWith('Nothing to reset')
  })

  it('disables openai quota reset when app-server reports unsupported reset credits', async () => {
    const wrapper = mountWithPinia(AccountUsageResetCell, {
      props: {
        account: {
          id: 3010,
          platform: 'openai',
          type: 'oauth',
          extra: {
            codex_usage_updated_at: '2099-03-07T10:00:00Z',
            codex_5h_used_percent: 10,
            codex_5h_reset_at: '2026-03-13T15:22:00',
            openai_rate_limit_reset_credits_status: 'unsupported',
            openai_rate_limit_reset_credits_unsupported_reason: 'This Codex app-server cannot reset',
          },
        } as any,
      },
    })

    await flushPromises()

    expect(wrapper.get('[data-testid="account-usage-reset-quota-button"]').attributes('disabled')).toBeDefined()
    expect(wrapper.find('[data-testid="account-usage-reset-quota-remaining"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="account-usage-reset-credits-refresh"]').exists()).toBe(false)

    await wrapper.get('[data-testid="account-usage-reset-quota-button"]').trigger('click')

    expect(resetAccountQuota).not.toHaveBeenCalled()
    expect(window.confirm).not.toHaveBeenCalled()
  })
})
