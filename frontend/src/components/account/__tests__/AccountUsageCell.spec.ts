import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'
import AccountUsageCell from '../AccountUsageCell.vue'
import {
  invalidateAccountUsagePresentationCache,
  refreshAccountUsagePresentation,
  resolveActualUsageRefreshLoadOptions,
  resetAccountUsagePresentationCache
} from '@/composables/useAccountUsagePresentation'
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
      t: (key: string, params?: Record<string, unknown>) => {
        const dict: Record<string, string> = {
          'admin.accounts.usageWindow.snapshotUpdatedAt': 'Snapshot updated {time}',
          'admin.accounts.usageWindow.passiveSampled': 'Passive snapshot note',
          'admin.accounts.usageWindow.sampledBadge': 'Sampled',
          'admin.accounts.usageWindow.gemini3Image': 'Gemini Image',
          'admin.accounts.usageWindow.gemini3Pro': 'G3P',
          'admin.accounts.usageWindow.gemini3Flash': 'G3F',
          'admin.accounts.usageWindow.claude': 'Claude',
          'admin.accounts.gemini.rateLimit.unlimited': 'Unlimited',
          'admin.accounts.protocolGateway.usageWindow.badge': 'Protocol Gateway · {protocol}',
          'admin.accounts.protocolGateway.usageWindow.tightestWindowNote': 'Showing the tightest upstream daily window.',
          'admin.accounts.ineligibleWarning': 'Ineligible warning',
          'admin.accounts.gemini.quotaPolicy.title': 'Quota policy',
          'admin.accounts.gemini.quotaPolicy.note': 'Quota note',
          'admin.accounts.gemini.quotaPolicy.columns.docs': 'Docs',
          'ui.usageWindow.daily': '日',
          'ui.usageWindow.weekly': '周',
          'ui.usageWindow.total': '总',
          'ui.usageWindow.fiveHour': '5H',
          'ui.usageWindow.pro': 'Pro',
          'ui.usageWindow.flash': 'Flash',
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

const passiveUsageResponse = {
  source: 'passive',
  five_hour: {
    utilization: 21,
    resets_at: '2026-03-08T12:00:00Z',
    remaining_seconds: 3600,
    window_stats: {
      requests: 2,
      tokens: 200,
      cost: 0.02,
      standard_cost: 0.02,
      user_cost: 0.02,
    },
  },
  seven_day: {
    utilization: 61,
    resets_at: '2026-03-13T12:00:00Z',
    remaining_seconds: 7200,
    window_stats: {
      requests: 6,
      tokens: 610,
      cost: 0.06,
      standard_cost: 0.06,
      user_cost: 0.06,
    },
  },
}

const createMatchMediaMock = (matches: boolean) => {
  return vi.fn().mockImplementation((query: string) => ({
    matches,
    media: query,
    onchange: null,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    addListener: vi.fn(),
    removeListener: vi.fn(),
    dispatchEvent: vi.fn(),
  }))
}

enableAutoUnmount(afterEach)

describe('AccountUsageCell', () => {
  beforeEach(() => {
    getUsage.mockReset()
    getUsage.mockResolvedValue({})
    resetAccountUsagePresentationCache()
    resetUiNowForTests()
  })

  afterEach(() => {
    resetUiNowForTests()
    vi.useRealTimers()
  })

  it('defers mobile usage auto loads until the cell enters the viewport', async () => {
    const originalMatchMedia = window.matchMedia
    const originalIntersectionObserver = globalThis.IntersectionObserver

    const observerRecords: Array<{
      callback: IntersectionObserverCallback
      observe: ReturnType<typeof vi.fn>
      disconnect: ReturnType<typeof vi.fn>
    }> = []

    window.matchMedia = createMatchMediaMock(false) as typeof window.matchMedia
    globalThis.IntersectionObserver = class {
      observe = vi.fn()
      disconnect = vi.fn()
      readonly callback: IntersectionObserverCallback

      constructor(callback: IntersectionObserverCallback) {
        this.callback = callback
        observerRecords.push({
          callback,
          observe: this.observe,
          disconnect: this.disconnect,
        })
      }
    } as unknown as typeof IntersectionObserver

    getUsage.mockResolvedValue(passiveUsageResponse)

    try {
      const wrapper = mount(AccountUsageCell, {
        props: {
          account: {
            id: 1050,
            platform: 'anthropic',
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

      expect(getUsage).not.toHaveBeenCalled()
      expect(observerRecords.length).toBeGreaterThan(0)
      const activeObserver = observerRecords.at(-1)
      expect(activeObserver?.observe).toHaveBeenCalledTimes(1)

      const target = activeObserver?.observe.mock.calls[0]?.[0]
      activeObserver?.callback(
        [{ isIntersecting: true, target } as IntersectionObserverEntry],
        {} as IntersectionObserver,
      )
      await flushPromises()

      expect(getUsage).toHaveBeenCalledTimes(1)
      expect(getUsage).toHaveBeenCalledWith(1050, { force: undefined, source: 'passive' })
      expect(wrapper.text()).toContain('5h|21|2026-03-08T12:00:00Z|3600|false|200')
    } finally {
      window.matchMedia = originalMatchMedia
      globalThis.IntersectionObserver = originalIntersectionObserver
    }
  })

  it('keeps desktop usage auto loads immediate', async () => {
    const originalMatchMedia = window.matchMedia
    const originalIntersectionObserver = globalThis.IntersectionObserver
    const intersectionObserverSpy = vi.fn()

    window.matchMedia = createMatchMediaMock(true) as typeof window.matchMedia
    globalThis.IntersectionObserver = class {
      observe = vi.fn()
      disconnect = vi.fn()

      constructor() {
        intersectionObserverSpy()
      }
    } as unknown as typeof IntersectionObserver

    getUsage.mockResolvedValue(passiveUsageResponse)

    try {
      mount(AccountUsageCell, {
        props: {
          account: {
            id: 1051,
            platform: 'anthropic',
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

      expect(getUsage).toHaveBeenCalledTimes(1)
      expect(getUsage).toHaveBeenCalledWith(1051, { force: undefined, source: 'passive' })
      expect(intersectionObserverSpy).not.toHaveBeenCalled()
    } finally {
      window.matchMedia = originalMatchMedia
      globalThis.IntersectionObserver = originalIntersectionObserver
    }
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

  it('renders protocol gateway gemini accounts without fetching or showing account-level quota bars', async () => {
    getUsage.mockResolvedValue({
      gemini_pro_daily: {
        utilization: 25,
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
      gemini_flash_daily: {
        utilization: 70,
        resets_at: '2026-03-08T14:00:00Z',
        remaining_seconds: 7200,
        window_stats: {
          requests: 8,
          tokens: 800,
          cost: 0.08,
          standard_cost: 0.08,
          user_cost: 0.08,
        },
      },
    })

    const wrapper = mount(AccountUsageCell, {
      props: {
        account: {
          id: 1003,
          platform: 'protocol_gateway',
          gateway_protocol: 'gemini',
          type: 'apikey',
          credentials: {
            tier_id: 'aistudio_tier_1',
          },
          extra: {
            gateway_protocol: 'gemini',
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

    expect(wrapper.text()).toContain('Protocol Gateway · Gemini')
    expect(getUsage).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('Unlimited')
    expect(wrapper.text()).not.toContain('|70|2026-03-08T14:00:00Z|7200|false|800')
    expect(wrapper.text()).not.toContain('Showing the tightest upstream daily window.')
    expect(wrapper.text()).not.toContain('AI Studio')
    expect(wrapper.text()).not.toContain('Free')
    expect(wrapper.text()).not.toContain('Tier 1')
    expect(wrapper.text()).not.toContain('Flash')
  })

  it('renders total account quota with a localized label instead of the raw total value', async () => {
    const wrapper = mount(AccountUsageCell, {
      props: {
        account: {
          id: 1002,
          platform: 'openai',
          type: 'apikey',
          quota_limit: 100,
          quota_used: 25,
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

    expect(wrapper.text()).toContain('总|25')
    expect(wrapper.text()).not.toContain('total|25')
  })

  it('falls back to active anthropic usage when passive claudecloud data misses 7d', async () => {
    getUsage
      .mockResolvedValueOnce({
        source: 'passive',
        updated_at: '2026-03-07T10:00:00Z',
        five_hour: {
          utilization: 21,
          resets_at: '2026-03-08T12:00:00Z',
          remaining_seconds: 3600,
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
      .mockResolvedValueOnce({
        source: 'active',
        updated_at: '2026-03-07T10:01:00Z',
        five_hour: {
          utilization: 22,
          resets_at: '2026-03-08T12:00:00Z',
          remaining_seconds: 3600,
          window_stats: {
            requests: 2,
            tokens: 220,
            cost: 0.02,
            standard_cost: 0.02,
            user_cost: 0.02,
          },
        },
        seven_day: {
          utilization: 63,
          resets_at: '2026-03-13T12:00:00Z',
          remaining_seconds: 7200,
          window_stats: {
            requests: 6,
            tokens: 630,
            cost: 0.06,
            standard_cost: 0.06,
            user_cost: 0.06,
          },
        },
      })

    const wrapper = mount(AccountUsageCell, {
      props: {
        account: {
          id: 1100,
          platform: 'anthropic',
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

    expect(getUsage).toHaveBeenNthCalledWith(1, 1100, { force: undefined, source: 'passive' })
    expect(getUsage).toHaveBeenNthCalledWith(2, 1100, { force: undefined, source: 'active' })
    expect(wrapper.text()).toContain('5h|22|2026-03-08T12:00:00Z|3600|false|220')
    expect(wrapper.text()).toContain('7d|63|2026-03-13T12:00:00Z|7200|false|630')
  })

  it('renders a sampled badge instead of the passive snapshot sentence for passive claudecloud data', async () => {
    getUsage.mockResolvedValue({
      ...passiveUsageResponse,
      updated_at: '2026-03-07T10:00:00Z',
    })

    const wrapper = mount(AccountUsageCell, {
      props: {
        account: {
          id: 1101,
          platform: 'anthropic',
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

    expect(wrapper.text()).toContain('Sampled')
    expect(wrapper.text()).not.toContain('Passive snapshot note')

    await wrapper.get('button').trigger('mouseenter')
    await flushPromises()

    expect(document.body.textContent).toContain('Passive snapshot note')
  })

  it('limits concurrent auto usage loads across mounted cells', async () => {
    let inFlight = 0
    let maxInFlight = 0
    const deferreds: Array<{ resolve: (value: unknown) => void }> = []

    getUsage.mockImplementation(() => {
      inFlight += 1
      maxInFlight = Math.max(maxInFlight, inFlight)

      return new Promise((resolve) => {
        deferreds.push({
          resolve: (value) => {
            inFlight -= 1
            resolve(value)
          },
        })
      })
    })

    const wrappers = Array.from({ length: 5 }, (_, index) =>
      mount(AccountUsageCell, {
        props: {
          account: {
            id: 1200 + index,
            platform: 'anthropic',
            type: 'oauth',
            extra: {},
          } as any,
        },
        global: {
          stubs: {
            UsageProgressBar: usageBarStub,
          },
        },
      }),
    )

    await flushPromises()

    expect(getUsage).toHaveBeenCalledTimes(3)
    expect(maxInFlight).toBeLessThanOrEqual(3)

    deferreds.shift()?.resolve(passiveUsageResponse)
    await flushPromises()

    expect(getUsage).toHaveBeenCalledTimes(4)
    expect(maxInFlight).toBeLessThanOrEqual(3)

    while (deferreds.length > 0) {
      deferreds.shift()?.resolve(passiveUsageResponse)
      await flushPromises()
    }

    expect(getUsage).toHaveBeenCalledTimes(5)
    expect(maxInFlight).toBeLessThanOrEqual(3)

    wrappers.forEach((wrapper) => wrapper.unmount())
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

    expect(getUsage).toHaveBeenCalledWith(2000, { force: undefined, source: undefined })
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

  it('supplements missing openai 7d snapshots with fetched usage', async () => {
    getUsage.mockResolvedValue({
      updated_at: '2026-03-07T11:00:00Z',
      five_hour: {
        utilization: 88,
        resets_at: '2026-03-08T12:00:00Z',
        remaining_seconds: 3600,
        window_stats: {
          requests: 8,
          tokens: 800,
          cost: 0.08,
          standard_cost: 0.08,
          user_cost: 0.08,
        },
      },
      seven_day: {
        utilization: 66,
        resets_at: '2026-03-13T12:00:00Z',
        remaining_seconds: 7200,
        window_stats: {
          requests: 9,
          tokens: 900,
          cost: 0.09,
          standard_cost: 0.09,
          user_cost: 0.09,
        },
      },
    })

    const wrapper = mount(AccountUsageCell, {
      props: {
        account: {
          id: 2007,
          platform: 'openai',
          type: 'oauth',
          extra: {
            codex_usage_updated_at: '2099-03-07T10:00:00Z',
            codex_5h_used_percent: 12,
            codex_5h_reset_at: '2099-03-07T12:00:00Z',
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

    expect(getUsage).toHaveBeenCalledWith(2007, { force: undefined, source: undefined })
    expect(wrapper.text()).toContain('5h|12|2099-03-07T12:00:00.000Z||true')
    expect(wrapper.text()).toContain('7d|66|2026-03-13T12:00:00Z|7200|true|900')
  })

  it('uses forced fetched openai usage after a manual real refresh', async () => {
    const account = {
      id: 2010,
      platform: 'openai',
      type: 'oauth',
      extra: {
        codex_usage_updated_at: '2099-03-07T10:00:00Z',
        codex_5h_used_percent: 12,
        codex_5h_reset_at: '2099-03-07T12:00:00Z',
        codex_7d_used_percent: 34,
        codex_7d_reset_at: '2099-03-13T12:00:00Z',
      },
    } as any

    const wrapper = mount(AccountUsageCell, {
      props: { account },
      global: {
        stubs: {
          UsageProgressBar: usageBarStub,
        },
      },
    })

    await flushPromises()

    expect(getUsage).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('5h|12|2099-03-07T12:00:00.000Z||true')

    getUsage.mockResolvedValueOnce({
      five_hour: {
        utilization: 88,
        resets_at: '2026-03-08T12:00:00Z',
        remaining_seconds: 3600,
        window_stats: {
          requests: 8,
          tokens: 800,
          cost: 0.08,
          standard_cost: 0.08,
          user_cost: 0.08,
        },
      },
      seven_day: {
        utilization: 66,
        resets_at: '2026-03-13T12:00:00Z',
        remaining_seconds: 7200,
        window_stats: {
          requests: 9,
          tokens: 900,
          cost: 0.09,
          standard_cost: 0.09,
          user_cost: 0.09,
        },
      },
    })

    invalidateAccountUsagePresentationCache([account.id])
    const result = await refreshAccountUsagePresentation([account], { force: true, concurrency: 1 })
    await flushPromises()

    expect(result).toEqual({ total: 1, success: 1, failed: 0 })
    expect(getUsage).toHaveBeenCalledWith(2010, { force: true })
    expect(wrapper.text()).toContain('5h|88|2026-03-08T12:00:00Z|3600|true|800')
    expect(wrapper.text()).toContain('7d|66|2026-03-13T12:00:00Z|7200|true|900')
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

    expect(getUsage).toHaveBeenCalledWith(2002, { force: undefined, source: undefined })
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

  it('reloads openai usage after a local codex window reaches its reset time', async () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-03-13T12:00:00Z'))

    getUsage.mockResolvedValueOnce({
      five_hour: {
        utilization: 44,
        resets_at: '2026-03-13T17:00:00Z',
        remaining_seconds: 18000,
        window_stats: {
          requests: 5,
          tokens: 500,
          cost: 0.05,
          standard_cost: 0.05,
          user_cost: 0.05,
        },
      },
      seven_day: {
        utilization: 12,
        resets_at: '2026-03-20T12:00:00Z',
        remaining_seconds: 604800,
        window_stats: {
          requests: 9,
          tokens: 900,
          cost: 0.09,
          standard_cost: 0.09,
          user_cost: 0.09,
        },
      },
    })

    const wrapper = mount(AccountUsageCell, {
      props: {
        account: {
          id: 2006,
          platform: 'openai',
          type: 'oauth',
          extra: {
            codex_usage_updated_at: '2026-03-13T11:59:30Z',
            codex_5h_used_percent: 12,
            codex_5h_reset_at: '2026-03-13T12:01:00Z',
            codex_7d_used_percent: 34,
            codex_7d_reset_at: '2026-03-20T12:00:00Z',
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
    expect(wrapper.text()).toContain('5h|12|2026-03-13T12:01:00.000Z||true')

    vi.advanceTimersByTime(65_000)
    await flushPromises()

    expect(getUsage).toHaveBeenCalledTimes(1)
    expect(getUsage).toHaveBeenCalledWith(2006, { force: undefined, source: undefined })
    expect(wrapper.text()).toContain('5h|44|2026-03-13T17:00:00Z|18000|true|500')
    expect(wrapper.text()).toContain('7d|12|2026-03-20T12:00:00Z|604800|true|900')
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

    expect(getUsage).toHaveBeenCalledWith(2004, { force: undefined, source: undefined })
    expect(wrapper.text()).toContain('5h|100|2026-03-07T12:00:00Z|3600|true|106540000')
    expect(wrapper.text()).toContain('7d|100|2026-03-13T12:00:00Z|3600|true|106540000')
    expect(wrapper.text()).not.toContain('5h|0|')
  })

  it('forces active source only for manual claudecloud oauth refresh', async () => {
    const anthropicOauthAccount = {
      id: 3100,
      platform: 'anthropic',
      type: 'oauth',
      extra: {},
    } as any
    const anthropicSetupTokenAccount = {
      id: 3101,
      platform: 'anthropic',
      type: 'setup-token',
      extra: {},
    } as any
    const openaiOauthAccount = {
      id: 3102,
      platform: 'openai',
      type: 'oauth',
      extra: {},
    } as any

    getUsage.mockResolvedValue({})

    invalidateAccountUsagePresentationCache([
      anthropicOauthAccount.id,
      anthropicSetupTokenAccount.id,
      openaiOauthAccount.id,
    ])

    const result = await refreshAccountUsagePresentation(
      [anthropicOauthAccount, anthropicSetupTokenAccount, openaiOauthAccount],
      {
        force: true,
        concurrency: 1,
        resolveLoadOptions: resolveActualUsageRefreshLoadOptions,
      },
    )

    expect(result).toEqual({ total: 3, success: 3, failed: 0 })
    expect(getUsage).toHaveBeenNthCalledWith(1, 3100, { force: true, source: 'active' })
    expect(getUsage).toHaveBeenNthCalledWith(2, 3101, { force: true, source: 'passive' })
    expect(getUsage).toHaveBeenNthCalledWith(3, 3102, { force: true, source: undefined })
  })
})
