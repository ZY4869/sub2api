import { mount } from '@vue/test-utils'
import { defineComponent } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import AccountsViewTable from '../AccountsViewTable.vue'
import { useAccountUsageDisplayMode } from '@/composables/useAccountUsageDisplayMode'

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

const DataTableStub = defineComponent({
  props: ['data', 'columns', 'rowClass', 'rowStyle'],
  template: `
    <div>
      <div class="header-select"><slot name="header-select" /></div>
      <div class="column-capacity">{{ columns?.find((column) => column.key === 'capacity')?.label }}</div>
      <div class="header-usage"><slot name="header-usage" :column="{ key: 'usage', label: 'Usage Windows' }" /></div>
      <div class="row-class">{{ rowClass ? rowClass(data[0], 0) : '' }}</div>
      <div class="row-style">{{ rowStyle ? JSON.stringify(rowStyle(data[0], 0)) : '' }}</div>
      <div class="cell-name"><slot name="cell-name" :row="data[0]" :value="data[0].name" /></div>
      <div class="cell-platform"><slot name="cell-platform_type" :row="data[0]" /></div>
      <div class="cell-capacity"><slot name="cell-capacity" :row="data[0]" /></div>
      <div class="cell-select"><slot name="cell-select" :row="data[0]" /></div>
      <div class="cell-status"><slot name="cell-status" :row="data[0]" /></div>
      <div class="cell-groups"><slot name="cell-groups" :row="data[0]" /></div>
      <div class="cell-usage"><slot name="cell-usage" :row="data[0]" /></div>
      <div class="cell-actions"><slot name="cell-actions" :row="data[0]" /></div>
    </div>
  `
})

const PaginationStub = defineComponent({
  emits: ['update:page', 'update:page-size'],
  template: `
    <div>
      <button class="page-change" @click="$emit('update:page', 2)" />
      <button class="page-size-change" @click="$emit('update:page-size', 50)" />
    </div>
  `
})

const RowActionsStub = defineComponent({
  emits: ['edit', 'delete', 'more'],
  template: `
    <div>
      <button class="row-edit" @click="$emit('edit')" />
      <button class="row-delete" @click="$emit('delete')" />
      <button class="row-more" @click="$emit('more', $event)" />
    </div>
  `
})

const AiryRowActionsStub = defineComponent({
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
})

function mountTable(accountOverrides: Record<string, unknown> = {}) {
  return mount(AccountsViewTable, {
    props: {
      columns: [
        { key: 'select', label: '' },
        { key: 'name', label: '名称' },
        { key: 'platform_type', label: '平台/类型' },
        { key: 'capacity', label: '容量' },
        { key: 'status', label: 'Status' },
        { key: 'actions', label: 'Actions' }
      ],
      accounts: [
        {
          id: 1,
          name: 'Primary',
          platform: 'openai',
          type: 'apikey',
          status: 'active',
          schedulable: true,
          concurrency: 3,
          current_concurrency: 1,
          lifecycle_state: 'normal',
          extra: {},
          credentials: {
            plan_type: 'plus'
          },
          groups: [
            {
              id: 7,
              name: 'Admin',
              platform: 'openai',
              subscription_type: 'standard',
              rate_multiplier: 1
            }
          ],
          auto_recovery_probe: {
            status: 'retry_scheduled',
            summary: 'Temporary gateway error',
            checked_at: '2026-04-09T00:00:00Z'
          },
          ...accountOverrides
        }
      ],
      loading: false,
      allVisibleSelected: true,
      selectedIds: [1],
      togglingSchedulable: null,
      todayStatsByAccountId: {},
      todayStatsLoading: false,
      todayStatsError: null,
      usageManualRefreshToken: 0,
      visualStyle: 'airy',
      sortStorageKey: 'account-table-sort',
      pagination: {
        total: 1,
        page: 1,
        page_size: 20
      }
    } as any,
    global: {
      stubs: {
        DataTable: DataTableStub,
        Pagination: PaginationStub,
        AccountStatusIndicator: {
          emits: ['show-temp-unsched'],
          template: '<button class="show-temp-unsched status-classic-stub" @click="$emit(\'show-temp-unsched\')" />'
        },
        PlatformIcon: {
          props: ['platform', 'size'],
          template: '<span class="platform-icon-stub" :data-platform="platform" :data-size="size" />'
        },
        AccountStatusVisualCell: defineComponent({
          props: ['visualStyle', 'whiteSurfaceEnabled'],
          emits: ['show-temp-unsched'],
          template: '<button class="show-temp-unsched status-visual-stub" :data-visual-style="visualStyle" :data-white-surface-enabled="String(whiteSurfaceEnabled)" @click="$emit(\'show-temp-unsched\')" />'
        }),
        AccountsViewRowActions: RowActionsStub,
        AccountsViewAiryRowActions: AiryRowActionsStub,
        AccountCapacityCell: {
          props: ['account', 'visualVariant', 'whiteSurfaceEnabled', 'compact'],
          template: '<div class="capacity-stub" :data-visual-variant="visualVariant" :data-white-surface-enabled="String(whiteSurfaceEnabled)" :data-compact="String(compact)">{{ String(account.current_concurrency).padStart(2, "0") }}/{{ String(account.concurrency).padStart(2, "0") }}</div>'
        },
        AccountTodayStatsCell: true,
        AccountGroupsCell: {
          props: ['groups', 'maxDisplay', 'visualVariant'],
          template: '<div class="groups-stub" :data-visual-variant="visualVariant">{{ groups?.map((group) => group.name).join(",") }}</div>'
        },
        AccountUsageCell: {
          template: '<div class="usage-classic-stub" />'
        },
        AccountUsageVisualCell: {
          props: ['whiteSurfaceEnabled'],
          template: '<div class="usage-visual-stub" :data-white-surface-enabled="String(whiteSurfaceEnabled)" />'
        },
        AccountUsageResetCell: true
      }
    }
  })
}

describe('AccountsViewTable', () => {
  beforeEach(() => {
    localStorage.clear()
    countdownHookSpy.mockClear()
    useAccountUsageDisplayMode().setAccountUsageDisplayMode('used')
  })

  it('toggles and persists the shared usage display mode from the usage header', async () => {
    const wrapper = mountTable()

    expect(localStorage.getItem('account-usage-display-mode')).toBe('used')

    await wrapper.get('[data-testid="usage-display-mode-toggle"]').trigger('click')

    expect(useAccountUsageDisplayMode().accountUsageDisplayMode.value).toBe('remaining')
    expect(localStorage.getItem('account-usage-display-mode')).toBe('remaining')
  })

  it('emits selection, row action and pagination events', async () => {
    const wrapper = mountTable()

    await wrapper.get('.header-select input').setValue(false)
    await wrapper.get('.cell-select input').setValue(false)
    await wrapper.get('.show-temp-unsched').trigger('click')
    await wrapper.get('.airy-row-toggle').trigger('click')
    await wrapper.get('.airy-row-edit').trigger('click')
    await wrapper.get('.airy-row-delete').trigger('click')
    await wrapper.get('.airy-row-more').trigger('click')
    await wrapper.get('.page-change').trigger('click')
    await wrapper.get('.page-size-change').trigger('click')

    expect(wrapper.emitted('toggle-select-all-visible')).toEqual([[false]])
    expect(wrapper.emitted('toggle-selected')).toEqual([[1]])
    expect(wrapper.emitted('show-temp-unsched')).toEqual([[expect.objectContaining({ id: 1 })]])
    expect(wrapper.emitted('toggle-schedulable')).toEqual([[expect.objectContaining({ id: 1 })]])
    expect(wrapper.emitted('edit')).toEqual([[expect.objectContaining({ id: 1 })]])
    expect(wrapper.emitted('delete')).toEqual([[expect.objectContaining({ id: 1 })]])
    expect(wrapper.emitted('open-menu')?.[0]?.[0].account).toEqual(expect.objectContaining({ id: 1 }))
    expect(wrapper.emitted('page-change')).toEqual([[2]])
    expect(wrapper.emitted('page-size-change')).toEqual([[50]])
  })

  it('keeps the name cell aligned to the reference name-only visual', () => {
    const wrapper = mountTable()

    expect(wrapper.get('.cell-name').text()).toContain('Primary')
    expect(wrapper.get('.cell-name').text()).not.toContain('admin.accounts.autoRecoveryProbe.statuses.retry_scheduled')
    expect(wrapper.get('.cell-name').text()).not.toContain('Temporary gateway error')
    expect(wrapper.get('.cell-name').text()).not.toContain('admin.accounts.autoRecoveryProbe.headline')
    expect(wrapper.get('.cell-name [title="admin.accounts.autoRecoveryProbe.headline"]').exists()).toBe(true)
  })

  it('shows the recovery success indicator and hides the success notice block', () => {
    const wrapper = mountTable({
      auto_recovery_probe: {
        status: 'success',
        summary: 'Recovered',
        checked_at: '2026-04-09T00:00:00Z'
      }
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

  it('hides stale blacklisted recovery notices after the account is restored', () => {
    const wrapper = mountTable({
      lifecycle_state: 'normal',
      auto_recovery_probe: {
        status: 'blacklisted',
        blacklisted: true,
        summary: 'API returned 502',
        error_code: 'auto_recovery_probe_failed',
      }
    })

    expect(wrapper.text()).not.toContain('API returned 502')
    expect(wrapper.text()).not.toContain('admin.accounts.autoRecoveryProbe.headline')
  })

  it('keeps airy visual upgrades while preserving name, service and capacity content', () => {
    const wrapper = mountTable({
      extra: {
        email_address: 'owner@example.com'
      }
    })

    expect(wrapper.get('.column-capacity').text()).toBe('容量')
    expect(wrapper.get('.cell-name').text()).toContain('Primary')
    expect(wrapper.get('.cell-name').text()).toContain('owner@example.com')
    expect(wrapper.get('.cell-name .platform-icon-stub').attributes('data-platform')).toBe('openai')
    expect(wrapper.get('.cell-platform').text()).toContain('admin.accounts.platforms.openai')
    expect(wrapper.get('.cell-platform').text()).toContain('ui.platformType.key')
    expect(wrapper.get('.cell-capacity').text()).toContain('01/03')
    expect(wrapper.get('.cell-capacity .capacity-stub').attributes('data-visual-variant')).toBe('glass')
    expect(wrapper.get('.cell-capacity .capacity-stub').attributes('data-white-surface-enabled')).toBe('false')
    expect(wrapper.get('.cell-capacity .capacity-stub').attributes('data-compact')).toBe('true')
    expect(wrapper.find('.status-visual-stub').exists()).toBe(true)
    expect(wrapper.get('.status-visual-stub').attributes('data-visual-style')).toBe('airy')
    expect(wrapper.get('.status-visual-stub').attributes('data-white-surface-enabled')).toBe('false')
    expect(wrapper.find('.usage-visual-stub').exists()).toBe(true)
    expect(wrapper.get('.usage-visual-stub').attributes('data-white-surface-enabled')).toBe('false')
    expect(wrapper.get('.groups-stub').attributes('data-visual-variant')).toBe('airy')
    expect(wrapper.find('.airy-row-actions').exists()).toBe(true)
    expect(wrapper.find('.row-edit').exists()).toBe(false)
    expect(wrapper.get('.row-class').text()).toContain('account-visual-row')
    expect(wrapper.get('.row-style').text()).toContain('--account-row-bg')
    expect(countdownHookSpy).not.toHaveBeenCalled()
  })

  it('switches airy row surfaces to white when the site setting is enabled', async () => {
    const wrapper = mountTable()

    await wrapper.setProps({ whiteSurfaceEnabled: true })

    expect(wrapper.get('.row-style').text()).toContain('"--account-row-bg":"#FFFFFF"')
    expect(wrapper.get('.cell-capacity .capacity-stub').attributes('data-white-surface-enabled')).toBe('true')
    expect(wrapper.get('.status-visual-stub').attributes('data-white-surface-enabled')).toBe('true')
    expect(wrapper.get('.usage-visual-stub').attributes('data-white-surface-enabled')).toBe('true')
  })

  it('falls back to classic visuals without airy row styles', async () => {
    const wrapper = mountTable({
      extra: {
        email_address: 'owner@example.com'
      }
    })

    await wrapper.setProps({ visualStyle: 'classic' })

    expect(wrapper.get('.row-class').text()).toBe('')
    expect(wrapper.get('.row-style').text()).toBe('')
    expect(wrapper.get('.cell-capacity .capacity-stub').attributes('data-visual-variant')).toBe('default')
    expect(wrapper.get('.cell-capacity .capacity-stub').attributes('data-white-surface-enabled')).toBe('false')
    expect(wrapper.get('.cell-capacity .capacity-stub').attributes('data-compact')).toBe('false')
    expect(wrapper.find('.status-classic-stub').exists()).toBe(true)
    expect(wrapper.find('.usage-classic-stub').exists()).toBe(true)
    expect(wrapper.get('.groups-stub').attributes('data-visual-variant')).toBe('default')
    expect(wrapper.find('.row-edit').exists()).toBe(true)
    expect(wrapper.find('.airy-row-actions').exists()).toBe(false)
    expect(wrapper.find('.status-visual-stub').exists()).toBe(false)
    expect(wrapper.find('.usage-visual-stub').exists()).toBe(false)
  })
})
