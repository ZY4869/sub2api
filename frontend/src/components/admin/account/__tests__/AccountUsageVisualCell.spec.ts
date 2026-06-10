import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import AccountUsageVisualCell from '../AccountUsageVisualCell.vue'
import { useAccountUsageDisplayMode } from '@/composables/useAccountUsageDisplayMode'
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
          'admin.accounts.usageWindow.snapshotUpdatedAt': 'Snapshot {time}',
          'admin.accounts.usageWindow.displayMode.used': 'Used',
          'admin.accounts.usageWindow.displayMode.remaining': 'Remaining',
          'admin.accounts.usageWindow.sampledBadge': 'Sampled',
          'admin.accounts.gemini.rateLimit.unlimited': 'Unlimited',
          'common.error': 'Error',
        }
        let value = dict[key] ?? key
        Object.entries(params || {}).forEach(([name, replacement]) => {
          value = value.replace(`{${name}}`, String(replacement))
        })
        return value
      }
    })
  }
})

const account = {
  id: 88,
  platform: 'anthropic',
  type: 'oauth',
  extra: {},
}

enableAutoUnmount(afterEach)

describe('AccountUsageVisualCell', () => {
  beforeEach(() => {
    getUsage.mockReset()
    resetAccountUsagePresentationCache()
    localStorage.clear()
    useAccountUsageDisplayMode().setAccountUsageDisplayMode('used')
  })

  it('renders true 5h/7d dual tracks in used mode', async () => {
    getUsage.mockResolvedValue({
      five_hour: {
        utilization: 35,
        resets_at: null,
        remaining_seconds: 0,
      },
      seven_day: {
        utilization: 82,
        resets_at: null,
        remaining_seconds: 0,
      },
    })

    const wrapper = mount(AccountUsageVisualCell, {
      props: {
        account: account as any,
      },
      global: {
        plugins: [createPinia()],
      },
    })

    await flushPromises()

    expect(wrapper.text()).toContain('5h')
    expect(wrapper.text()).toContain('7d')
    expect(wrapper.text()).toContain('35%')
    expect(wrapper.text()).toContain('82%')
    expect(wrapper.find('[data-testid="account-usage-visual-cell"]').exists()).toBe(true)
  })

  it('shows only the dynamic long-window track for OpenAI Free accounts', async () => {
    const wrapper = mount(AccountUsageVisualCell, {
      props: {
        account: {
          id: 90,
          platform: 'openai',
          type: 'oauth',
          credentials: {
            plan_type: 'free',
          },
          extra: {
            codex_5h_used_percent: 44,
            codex_5h_reset_at: '2099-05-22T17:00:00Z',
            codex_7d_used_percent: 12,
            codex_7d_window_minutes: 43200,
            codex_7d_reset_at: '2099-05-29T12:00:00Z',
            codex_usage_updated_at: '2099-05-22T12:00:00Z',
          },
        } as any,
      },
      global: {
        plugins: [createPinia()],
      },
    })

    await flushPromises()

    expect(wrapper.text()).not.toContain('5h')
    expect(wrapper.text()).toContain('30D')
    expect(wrapper.text()).not.toContain('7d')
    expect(wrapper.text()).toContain('12%')
    const rowLabel = wrapper.get('span.w-7')
    expect(rowLabel.classes()).toContain('bg-green-100')
  })

  it('uses the orange local tag for 7d rows', async () => {
    const wrapper = mount(AccountUsageVisualCell, {
      props: {
        account: {
          id: 91,
          platform: 'openai',
          type: 'oauth',
          credentials: {
            plan_type: 'free',
          },
          extra: {
            codex_7d_used_percent: 33,
            codex_7d_window_minutes: 10080,
            codex_7d_reset_at: '2099-05-29T12:00:00Z',
            codex_usage_updated_at: '2099-05-22T12:00:00Z',
          },
        } as any,
      },
      global: {
        plugins: [createPinia()],
      },
    })

    await flushPromises()

    expect(wrapper.text()).toContain('7D')
    expect(wrapper.get('span.w-7').classes()).toContain('bg-orange-100')
  })

  it('follows the shared remaining display mode', async () => {
    useAccountUsageDisplayMode().setAccountUsageDisplayMode('remaining')
    getUsage.mockResolvedValue({
      five_hour: {
        utilization: 25,
        resets_at: null,
        remaining_seconds: 0,
      },
      seven_day: {
        utilization: 90,
        resets_at: null,
        remaining_seconds: 0,
      },
    })

    const wrapper = mount(AccountUsageVisualCell, {
      props: {
        account: { ...account, id: 89 } as any,
      },
      global: {
        plugins: [createPinia()],
      },
    })

    await flushPromises()

    expect(wrapper.text()).toContain('75%')
    expect(wrapper.text()).toContain('10%')
  })
})
