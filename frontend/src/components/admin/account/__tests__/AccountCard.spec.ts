import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import AccountCard from '../AccountCard.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      locale: ref('zh'),
      t: (key: string) => key
    })
  }
})

vi.mock('@/utils/format', () => ({
  formatRelativeTime: () => '1 day ago',
  formatDateTime: () => '2026/04/09 00:00:00'
}))

function mountCard(autoRecoveryProbe: Record<string, unknown>) {
  return mount(AccountCard, {
    props: {
      account: {
        id: 1,
        name: 'Primary Account',
        platform: 'openai',
        type: 'apikey',
        status: 'active',
        schedulable: true,
        extra: {},
        auto_recovery_probe: autoRecoveryProbe,
        last_used_at: '2026-04-09T00:00:00Z'
      },
      selected: false,
      togglingSchedulable: null,
      todayStatsByAccountId: {},
      todayStatsLoading: false,
      usageManualRefreshToken: 0
    } as any,
    global: {
      stubs: {
        PlatformTypeBadge: true,
        AccountCapacityCell: true,
        AccountGroupsCell: true,
        AccountStatusIndicator: true,
        AccountUsageCell: true,
        AccountsViewRowActions: true
      }
    }
  })
}

describe('AccountCard', () => {
  it('shows the recovery success icon and hides the success notice block', () => {
    const wrapper = mountCard({
      status: 'success',
      summary: 'Recovered',
      checked_at: '2026-04-09T00:00:00Z'
    })

    const successIndicator = wrapper.find(
      '[title="admin.accounts.autoRecoveryProbe.successIndicator"]'
    )

    expect(successIndicator.exists()).toBe(true)
    expect(successIndicator.attributes('aria-label')).toBe(
      'admin.accounts.autoRecoveryProbe.successIndicator'
    )
    expect(wrapper.text()).not.toContain('Recovered')
    expect(wrapper.text()).not.toContain('admin.accounts.autoRecoveryProbe.headline')
  })

  it('keeps non-success recovery notices visible', () => {
    const wrapper = mountCard({
      status: 'retry_scheduled',
      summary: 'Temporary gateway error',
      checked_at: '2026-04-09T00:00:00Z'
    })

    expect(wrapper.text()).toContain('Temporary gateway error')
    expect(wrapper.text()).toContain('admin.accounts.autoRecoveryProbe.headline')
  })
})
