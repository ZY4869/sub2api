import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, ref } from 'vue'
import AccountCard from '../AccountCard.vue'

const countdownHookSpy = vi.hoisted(() =>
  vi.fn(() => ({
    nowMs: { value: 0 },
    nowDate: { value: new Date(0) }
  }))
)

vi.mock('@/composables/useRealtimeCountdownNow', () => ({
  useRealtimeCountdownNow: countdownHookSpy
}))

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

function mountCard(autoRecoveryProbe: Record<string, unknown>, accountOverrides: Record<string, unknown> = {}) {
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
        last_used_at: '2026-04-09T00:00:00Z',
        ...accountOverrides
      },
      selected: false,
      togglingSchedulable: null,
      todayStatsByAccountId: {},
      todayStatsLoading: false,
      usageManualRefreshToken: 0,
      visualStyle: 'airy'
    } as any,
    global: {
      stubs: {
        AccountCapacityCell: defineComponent({
          props: ['visualVariant', 'whiteSurfaceEnabled'],
          template: '<div class="capacity-stub" :data-visual-variant="visualVariant" :data-white-surface-enabled="String(whiteSurfaceEnabled)" />'
        }),
        AccountGroupsCell: true,
        AccountsViewAiryRowActions: defineComponent({
          props: ['account', 'togglingSchedulable'],
          emits: ['toggle-schedulable', 'edit', 'delete', 'more'],
          template: `
            <div class="airy-row-actions" :data-account-id="account.id" :data-toggling="String(togglingSchedulable)">
              <button class="airy-row-toggle" @click="$emit('toggle-schedulable')" />
              <button class="airy-row-edit" @click="$emit('edit')" />
              <button class="airy-row-delete" @click="$emit('delete')" />
              <button class="airy-row-more" @click="$emit('more', $event)" />
            </div>
          `
        }),
        PlatformIcon: {
          props: ['platform', 'size'],
          template: '<span class="platform-icon-stub" :data-platform="platform" :data-size="size" />'
        },
        AccountStatusIndicator: {
          template: '<div class="status-classic-stub" />'
        },
        AccountStatusVisualCell: defineComponent({
          props: ['visualStyle', 'whiteSurfaceEnabled'],
          template: '<div class="status-visual-stub" :data-visual-style="visualStyle" :data-white-surface-enabled="String(whiteSurfaceEnabled)" />'
        }),
        AccountUsageCell: {
          template: '<div class="usage-classic-stub" />'
        },
        AccountUsageVisualCell: {
          props: ['whiteSurfaceEnabled'],
          template: '<div class="usage-visual-stub" :data-white-surface-enabled="String(whiteSurfaceEnabled)" />'
        },
        AccountsViewRowActions: {
          template: '<div class="classic-row-actions" />'
        }
      }
    }
  })
}

describe('AccountCard', () => {
  beforeEach(() => {
    countdownHookSpy.mockClear()
  })

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

  it('hides stale blacklisted recovery notices after the account is restored', () => {
    const wrapper = mountCard(
      {
        status: 'blacklisted',
        blacklisted: true,
        summary: 'API returned 502',
        error_code: 'auto_recovery_probe_failed',
      },
      {
        lifecycle_state: 'normal',
      },
    )

    expect(wrapper.text()).not.toContain('API returned 502')
    expect(wrapper.text()).not.toContain('admin.accounts.autoRecoveryProbe.headline')
  })

  it('applies airy visual tone and preserves service/capacity content blocks', () => {
    const wrapper = mountCard(
      {
        status: 'retry_scheduled',
        summary: 'Temporary gateway error',
        checked_at: '2026-04-09T00:00:00Z'
      },
      {
        current_concurrency: 1,
        concurrency: 4,
        credentials: {
          plan_type: 'plus'
        }
      }
    )

    expect(wrapper.classes()).toContain('account-visual-row')
    expect(wrapper.attributes('style')).toContain('--account-row-bg')
    expect(wrapper.text()).toContain('admin.accounts.columns.capacity')
    expect(wrapper.text()).toContain('admin.accounts.platforms.openai')
    expect(wrapper.text()).toContain('ui.platformType.key')
    expect(wrapper.get('.platform-icon-stub').attributes('data-platform')).toBe('openai')
    expect(wrapper.get('.capacity-stub').attributes('data-visual-variant')).toBe('glass')
    expect(wrapper.get('.capacity-stub').attributes('data-white-surface-enabled')).toBe('false')
    expect(wrapper.find('.status-visual-stub').exists()).toBe(true)
    expect(wrapper.get('.status-visual-stub').attributes('data-visual-style')).toBe('airy')
    expect(wrapper.get('.status-visual-stub').attributes('data-white-surface-enabled')).toBe('false')
    expect(wrapper.find('.usage-visual-stub').exists()).toBe(true)
    expect(wrapper.get('.usage-visual-stub').attributes('data-white-surface-enabled')).toBe('false')
    expect(wrapper.find('.airy-row-actions').exists()).toBe(true)
    expect(wrapper.find('.classic-row-actions').exists()).toBe(false)
    expect(countdownHookSpy).not.toHaveBeenCalled()
  })

  it('switches airy card surfaces to white when the site setting is enabled', async () => {
    const wrapper = mountCard(
      {
        status: 'retry_scheduled',
        summary: 'Temporary gateway error',
        checked_at: '2026-04-09T00:00:00Z'
      },
      {
        current_concurrency: 1,
        concurrency: 4,
        credentials: {
          plan_type: 'plus'
        }
      }
    )

    await wrapper.setProps({ whiteSurfaceEnabled: true })

    expect(wrapper.classes()).toContain('bg-white')
    expect(wrapper.get('.capacity-stub').attributes('data-white-surface-enabled')).toBe('true')
    expect(wrapper.get('.status-visual-stub').attributes('data-white-surface-enabled')).toBe('true')
    expect(wrapper.get('.usage-visual-stub').attributes('data-white-surface-enabled')).toBe('true')
  })

  it('uses classic card styling without airy row tone background', async () => {
    const wrapper = mountCard(
      {
        status: 'retry_scheduled',
        summary: 'Temporary gateway error',
        checked_at: '2026-04-09T00:00:00Z'
      },
      {
        current_concurrency: 1,
        concurrency: 4,
        credentials: {
          plan_type: 'plus'
        }
      }
    )

    await wrapper.setProps({ visualStyle: 'classic' })

    expect(wrapper.classes()).not.toContain('account-visual-row')
    expect(wrapper.attributes('style') || '').not.toContain('--account-row-bg')
    expect(wrapper.get('.capacity-stub').attributes('data-visual-variant')).toBe('default')
    expect(wrapper.get('.capacity-stub').attributes('data-white-surface-enabled')).toBe('false')
    expect(wrapper.find('.status-classic-stub').exists()).toBe(true)
    expect(wrapper.find('.usage-classic-stub').exists()).toBe(true)
    expect(wrapper.find('.classic-row-actions').exists()).toBe(true)
    expect(wrapper.find('.airy-row-actions').exists()).toBe(false)
    expect(wrapper.find('.status-visual-stub').exists()).toBe(false)
    expect(wrapper.find('.usage-visual-stub').exists()).toBe(false)
  })
})
