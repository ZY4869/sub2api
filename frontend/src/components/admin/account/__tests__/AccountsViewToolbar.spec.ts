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

function mountToolbar(overrides: Record<string, unknown> = {}) {
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
      viewMode: 'table',
      groupViewEnabled: false,
      platformCountSortOrder: 'count_asc',
      toggleableColumns: [
        { key: 'proxy', label: 'Proxy', visible: true },
        { key: 'notes', label: 'Notes', visible: false }
      ],
      ...overrides
    },
    global: {
      stubs: {
        Icon: true,
        AccountViewModeToggle: {
          props: ['modelValue'],
          emits: ['update:modelValue'],
          template: '<button class="view-mode-toggle" @click="$emit(\'update:modelValue\', \'card\')" />'
        },
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

    await wrapper.get('.view-mode-toggle').trigger('click')
    await wrapper.get('.filters-update').trigger('click')
    await wrapper.get('.filters-search').trigger('click')
    await wrapper.get('.filters-change').trigger('click')
    await wrapper.get('.refresh').trigger('click')
    await wrapper.findAll('button').find((button) =>
      button.text().includes('admin.accounts.refreshActualUsage')
    )?.trigger('click')
    await wrapper.get('.sync').trigger('click')
    await wrapper.get('.create').trigger('click')

    expect(wrapper.text()).not.toContain('admin.accounts.viewArchived')
    expect(wrapper.text()).not.toContain('admin.accounts.batchCreate')
    expect(wrapper.emitted('update:view-mode')).toEqual([['card']])
    expect(wrapper.emitted('update:filters')).toEqual([[{ platform: 'openai' }]])
    expect(wrapper.emitted('update:searchQuery')).toEqual([['claude']])
    expect(wrapper.emitted('change')).toEqual([[]])
    expect(wrapper.emitted('refresh')).toEqual([[]])
    expect(wrapper.emitted('refresh-usage')).toEqual([[]])
    expect(wrapper.emitted('sync')).toEqual([[]])
    expect(wrapper.emitted('create')).toEqual([[]])
  })

  it('emits archive-group when a concrete group filter is selected', async () => {
    const wrapper = mountToolbar({
      filters: { platform: '', type: '', status: '', group: '1', search: '' },
      groups: [{ id: 1, name: 'Default', platform: 'openai' }]
    })

    const archiveGroupButton = wrapper.findAll('button').find((button) =>
      button.text().includes('admin.accounts.bulkActions.archiveCurrentGroup')
    )

    expect(archiveGroupButton).toBeTruthy()
    expect(archiveGroupButton?.attributes('disabled')).toBeUndefined()

    await archiveGroupButton?.trigger('click')

    expect(wrapper.emitted('archive-group')).toEqual([[]])
  })

  it('keeps archive current group disabled for the ungrouped filter', () => {
    const wrapper = mountToolbar({
      filters: { platform: '', type: '', status: '', group: 'ungrouped', search: '' }
    })

    const archiveGroupButton = wrapper.findAll('button').find((button) =>
      button.text().includes('admin.accounts.bulkActions.archiveCurrentGroup')
    )

    expect(archiveGroupButton?.attributes('disabled')).toBeDefined()
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
      button.text().includes('admin.tlsFingerprintProfiles.title')
    )?.trigger('click')
    await wrapper.findAll('button').find((button) =>
      button.text().includes('admin.accounts.groupView.enable')
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
    expect(wrapper.emitted('show-tls-fingerprint-profiles')).toEqual([[]])
    expect(wrapper.emitted('toggle-group-view')).toEqual([[]])
    expect(wrapper.emitted('sync-pending-list')).toEqual([[]])
  })

  it('renders limited account controls and forwards their actions', async () => {
    const wrapper = mountToolbar({
      showLimitedControls: true,
      hideLimitedAccounts: true,
      limitedAccountsCount: 7
    })

    expect(wrapper.text()).toContain('admin.accounts.limited.hideToggleOn')
    expect(wrapper.text()).toContain('admin.accounts.limited.entry')

    await wrapper.findAll('button').find((button) =>
      button.text().includes('admin.accounts.limited.hideToggleOn')
    )?.trigger('click')
    await wrapper.findAll('button').find((button) =>
      button.text().includes('admin.accounts.limited.entry')
    )?.trigger('click')

    expect(wrapper.emitted('toggle-hide-limited')).toEqual([[]])
    expect(wrapper.emitted('open-limited-page')).toEqual([[]])
  })

  it('renders the platform count sort toggle and emits the next mode', async () => {
    const wrapper = mountToolbar({
      platformCountSortOrder: 'count_asc'
    })

    const button = wrapper.get('[data-platform-sort-button="true"]')
    expect(button.text()).toContain('admin.accounts.platformSort.countAsc')
    expect(button.attributes('title')).toBe('admin.accounts.platformSort.toggleDesc')

    await button.trigger('click')

    expect(wrapper.emitted('update:platform-count-sort-order')).toEqual([['count_desc']])
  })
})
