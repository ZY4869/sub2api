import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'
import AccountUsageCell from '../AccountUsageCell.vue'
import {
  refreshAccountUsagePresentation,
  resolveActualUsageRefreshLoadOptions,
  resetAccountUsagePresentationCache,
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
      t: (key: string) => {
        const dict: Record<string, string> = {
          'ui.usageWindow.fiveHour': '5h',
          'ui.usageWindow.weekly': '7d',
          'common.error': 'Error',
        }
        return dict[key] ?? key
      },
    }),
  }
})

const usageBarStub = {
  props: ['label', 'utilization', 'resetsAt', 'remainingSeconds', 'windowStats', 'inlineReset', 'color'],
  template:
    '<div class="usage-bar">{{ label }}|{{ utilization }}|{{ resetsAt }}|{{ remainingSeconds }}|{{ inlineReset }}|{{ windowStats?.tokens }}</div>',
}

enableAutoUnmount(afterEach)

describe('AccountUsageCell Kiro usage handling', () => {
  beforeEach(() => {
    getUsage.mockReset()
    resetAccountUsagePresentationCache()
    resetUiNowForTests()
  })

  afterEach(() => {
    resetUiNowForTests()
  })

  it('keeps kiro oauth accounts on passive usage when the passive 7d snapshot is missing', async () => {
    getUsage.mockResolvedValue({
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

    mount(AccountUsageCell, {
      props: {
        account: {
          id: 4100,
          platform: 'kiro',
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
    expect(getUsage).toHaveBeenCalledWith(4100, { force: undefined, source: 'passive' })
  })

  it('does not force active usage during manual refresh for kiro oauth accounts', async () => {
    const kiroOauthAccount = {
      id: 4101,
      platform: 'kiro',
      type: 'oauth',
      extra: {},
    } as any

    getUsage.mockResolvedValue({})

    const result = await refreshAccountUsagePresentation([kiroOauthAccount], {
      force: true,
      concurrency: 1,
      resolveLoadOptions: resolveActualUsageRefreshLoadOptions,
    })

    expect(result).toEqual({ total: 1, success: 1, failed: 0 })
    expect(getUsage).toHaveBeenCalledTimes(1)
    expect(getUsage).toHaveBeenCalledWith(4101, { force: true, source: 'passive' })
  })
})
