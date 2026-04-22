import { mount } from '@vue/test-utils'
import { defineComponent } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import AccountsViewTable from '../AccountsViewTable.vue'

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
  props: ['data'],
  template: `
    <div>
      <div class="header-select"><slot name="header-select" /></div>
      <div class="cell-name"><slot name="cell-name" :row="data[0]" :value="data[0].name" /></div>
      <div class="cell-select"><slot name="cell-select" :row="data[0]" /></div>
      <div class="cell-status"><slot name="cell-status" :row="data[0]" /></div>
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

function mountTable(accountOverrides: Record<string, unknown> = {}) {
  return mount(AccountsViewTable, {
    props: {
      columns: [
        { key: 'select', label: '' },
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
          lifecycle_state: 'normal',
          extra: {},
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
          template: '<button class="show-temp-unsched" @click="$emit(\'show-temp-unsched\')" />'
        },
        AccountsViewRowActions: RowActionsStub,
        PlatformTypeBadge: true,
        AccountCapacityCell: true,
        AccountTodayStatsCell: true,
        AccountGroupsCell: true,
        AccountUsageCell: true,
        AccountUsageResetCell: true
      }
    }
  })
}

describe('AccountsViewTable', () => {
  it('emits selection, row action and pagination events', async () => {
    const wrapper = mountTable()

    await wrapper.get('.header-select input').setValue(false)
    await wrapper.get('.cell-select input').setValue(false)
    await wrapper.get('.show-temp-unsched').trigger('click')
    await wrapper.get('.row-edit').trigger('click')
    await wrapper.get('.row-delete').trigger('click')
    await wrapper.get('.row-more').trigger('click')
    await wrapper.get('.page-change').trigger('click')
    await wrapper.get('.page-size-change').trigger('click')

    expect(wrapper.emitted('toggle-select-all-visible')).toEqual([[false]])
    expect(wrapper.emitted('toggle-selected')).toEqual([[1]])
    expect(wrapper.emitted('show-temp-unsched')).toEqual([[expect.objectContaining({ id: 1 })]])
    expect(wrapper.emitted('edit')).toEqual([[expect.objectContaining({ id: 1 })]])
    expect(wrapper.emitted('delete')).toEqual([[expect.objectContaining({ id: 1 })]])
    expect(wrapper.emitted('open-menu')?.[0]?.[0].account).toEqual(expect.objectContaining({ id: 1 }))
    expect(wrapper.emitted('page-change')).toEqual([[2]])
    expect(wrapper.emitted('page-size-change')).toEqual([[50]])
  })

  it('renders auto recovery probe summary in the name cell', () => {
    const wrapper = mountTable()

    expect(wrapper.text()).toContain('Temporary gateway error')
    expect(wrapper.text()).toContain('admin.accounts.autoRecoveryProbe.headline')
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
})
