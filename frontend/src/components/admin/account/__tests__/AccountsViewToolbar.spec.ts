import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AccountsViewToolbar from '../AccountsViewToolbar.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, string | number>) =>
        key === 'admin.accounts.autoRefreshCountdown' && params?.seconds
          ? `countdown-${params.seconds}`
          : key
    })
  }
})

function mountToolbar() {
  return mount(AccountsViewToolbar, {
    props: {
      loading: false,
      usageRefreshing: false,
      searchQuery: '',
      filters: { platform: '', type: '', status: '', group: '', search: '' },
      groups: [{ id: 1, name: 'Default' }],
      hasPendingListSync: true,
      selectedCount: 2,
      autoRefreshEnabled: true,
      autoRefreshCountdown: 15,
      autoRefreshIntervals: [5, 10, 15, 30],
      autoRefreshIntervalSeconds: 10,
      toggleableColumns: [
        { key: 'proxy', label: 'Proxy', visible: true },
        { key: 'notes', label: 'Notes', visible: false }
      ]
    },
    global: {
      stubs: {
        Icon: true,
        AccountTableFilters: {
          emits: ['update:filters', 'update:searchQuery', 'change'],
          template: `
            <div>
              <button class="filters-update" @click="$emit('update:filters', { platform: 'openai' })" />
              <button class="filters-search" @click="$emit('update:searchQuery', 'claude')" />
              <button class="filters-change" @click="$emit('change')" />
            </div>
          `
        },
        AccountTableActions: {
          props: ['loading'],
          emits: ['refresh', 'sync', 'create'],
          template: `
            <div>
              <button class="refresh" @click="$emit('refresh')" />
              <slot name="after" />
              <slot name="beforeCreate" />
              <button class="sync" @click="$emit('sync')" />
              <button class="create" @click="$emit('create')" />
            </div>
          `
        }
      }
    }
  })
}

describe('AccountsViewToolbar', () => {
  it('forwards filter, search and toolbar actions', async () => {
    const wrapper = mountToolbar()

    await wrapper.get('.filters-update').trigger('click')
    await wrapper.get('.filters-search').trigger('click')
    await wrapper.get('.filters-change').trigger('click')
    await wrapper.get('.refresh').trigger('click')
    await wrapper.findAll('button').find((button) =>
      button.text().includes('admin.accounts.batchCreate')
    )?.trigger('click')
    await wrapper.findAll('button').find((button) =>
      button.text().includes('admin.accounts.refreshActualUsage')
    )?.trigger('click')
    await wrapper.get('.sync').trigger('click')
    await wrapper.get('.create').trigger('click')

    expect(wrapper.emitted('update:filters')).toEqual([[{ platform: 'openai' }]])
    expect(wrapper.emitted('update:searchQuery')).toEqual([['claude']])
    expect(wrapper.emitted('change')).toEqual([[]])
    expect(wrapper.emitted('refresh')).toEqual([[]])
    expect(wrapper.emitted('batch-create')).toEqual([[]])
    expect(wrapper.emitted('refresh-usage')).toEqual([[]])
    expect(wrapper.emitted('sync')).toEqual([[]])
    expect(wrapper.emitted('create')).toEqual([[]])
  })

  it('emits dropdown and pending sync actions', async () => {
    const wrapper = mountToolbar()

    await wrapper.get('button[title="admin.accounts.autoRefresh"]').trigger('click')
    await wrapper.findAll('button').find((button) =>
      button.text().includes('admin.accounts.enableAutoRefresh')
    )?.trigger('click')
    await wrapper.findAll('button').find((button) =>
      button.text().includes('admin.accounts.refreshInterval5s')
    )?.trigger('click')

    await wrapper.get('button[title="admin.users.columnSettings"]').trigger('click')
    await wrapper.findAll('button').find((button) => button.text().includes('Proxy'))?.trigger('click')
    await wrapper.findAll('button').find((button) =>
      button.text().includes('admin.accounts.dataImport')
    )?.trigger('click')
    await wrapper.findAll('button').find((button) =>
      button.text().includes('admin.accounts.dataExportSelected')
    )?.trigger('click')
    await wrapper.findAll('button').find((button) =>
      button.text().includes('admin.errorPassthrough.title')
    )?.trigger('click')
    await wrapper.findAll('button').find((button) =>
      button.text().includes('admin.accounts.listPendingSyncAction')
    )?.trigger('click')

    expect(wrapper.emitted('set-auto-refresh-enabled')).toEqual([[false]])
    expect(wrapper.emitted('set-auto-refresh-interval')).toEqual([[5]])
    expect(wrapper.emitted('toggle-column')).toEqual([['proxy']])
    expect(wrapper.emitted('import-data')).toEqual([[]])
    expect(wrapper.emitted('export-data')).toEqual([[]])
    expect(wrapper.emitted('show-error-passthrough')).toEqual([[]])
    expect(wrapper.emitted('sync-pending-list')).toEqual([[]])
  })
})
