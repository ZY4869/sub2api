import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AccountsViewDialogsHost from '../AccountsViewDialogsHost.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

function createProps() {
  return {
    showCreate: true,
    showBatchCreate: false,
    showArchiveSelected: false,
    showArchiveGroup: false,
    showEdit: false,
    showSync: false,
    showImportData: false,
    showExportDataDialog: false,
    showBulkEdit: false,
    showTempUnsched: false,
    showDeleteDialog: false,
    showReAuth: false,
    showTest: false,
    showStats: false,
    showErrorPassthrough: false,
    showSchedulePanel: false,
    includeProxyOnExport: true,
    proxies: [],
    groups: [],
    selectedIds: [1, 2],
    selectedPlatforms: ['openai'],
    selectedTypes: ['apikey'],
    archiveSourceGroup: null,
    editingAccount: null,
    tempUnschedAccount: null,
    deletingAccount: null,
    reAuthAccount: null,
    testingAccount: null,
    statsAccount: null,
    scheduleAccount: null,
    scheduleModelOptions: [],
    syncDialogOpen: false,
    syncDialogModels: [],
    syncDialogSubmitting: false,
    menuShow: false,
    menuAccount: null,
    menuPosition: null
  }
}

describe('AccountsViewDialogsHost', () => {
  it('forwards create modal events', async () => {
    const wrapper = mount(AccountsViewDialogsHost, {
      props: createProps(),
      global: {
        stubs: {
          CreateAccountModal: {
            emits: ['close', 'created', 'models-imported'],
            template: `
              <div>
                <button class="create-close" @click="$emit('close')" />
                <button class="create-created" @click="$emit('created')" />
                <button class="create-models" @click="$emit('models-imported', { ok: true })" />
              </div>
            `
          },
          BatchCreateAccountsModal: true,
          ArchiveAccountsModal: true,
          ArchiveGroupAccountsModal: true,
          ModelImportExposureSyncDialog: true,
          EditAccountModal: true,
          ReAuthAccountModal: true,
          AccountTestModal: true,
          AccountStatsModal: true,
          ScheduledTestsPanel: true,
          AccountActionMenu: true,
          SyncFromCrsModal: true,
          ImportDataModal: true,
          BulkEditAccountModal: true,
          TempUnschedStatusModal: true,
          ConfirmDialog: true,
          ErrorPassthroughRulesModal: true
        }
      }
    })

    await wrapper.get('.create-close').trigger('click')
    await wrapper.get('.create-created').trigger('click')
    await wrapper.get('.create-models').trigger('click')

    expect(wrapper.emitted('close-create')).toEqual([[]])
    expect(wrapper.emitted('created')).toEqual([[]])
    expect(wrapper.emitted('models-imported')).toEqual([[{ ok: true }]])
  })

  it('syncs export checkbox state and forwards confirm dialog actions', async () => {
    const wrapper = mount(AccountsViewDialogsHost, {
      props: {
        ...createProps(),
        showCreate: false,
        showExportDataDialog: true
      },
      global: {
        stubs: {
          CreateAccountModal: true,
          BatchCreateAccountsModal: true,
          ArchiveAccountsModal: true,
          ArchiveGroupAccountsModal: true,
          ModelImportExposureSyncDialog: true,
          EditAccountModal: true,
          ReAuthAccountModal: true,
          AccountTestModal: true,
          AccountStatsModal: true,
          ScheduledTestsPanel: true,
          AccountActionMenu: true,
          SyncFromCrsModal: true,
          ImportDataModal: true,
          BulkEditAccountModal: true,
          TempUnschedStatusModal: true,
          ErrorPassthroughRulesModal: true,
          ConfirmDialog: {
            props: ['show', 'title'],
            emits: ['confirm', 'cancel'],
            template: `
              <div v-if="show" class="confirm-dialog">
                <div class="confirm-title">{{ title }}</div>
                <slot />
                <button class="confirm-action" @click="$emit('confirm')" />
                <button class="cancel-action" @click="$emit('cancel')" />
              </div>
            `
          }
        }
      }
    })

    await wrapper.get('input[type="checkbox"]').setValue(false)
    await wrapper.get('.confirm-action').trigger('click')
    await wrapper.get('.cancel-action').trigger('click')

    expect(wrapper.emitted('update:includeProxyOnExport')).toEqual([[false]])
    expect(wrapper.emitted('confirm-export')).toEqual([[]])
    expect(wrapper.emitted('close-export')).toEqual([[]])
  })

  it('forwards batch create modal events', async () => {
    const wrapper = mount(AccountsViewDialogsHost, {
      props: {
        ...createProps(),
        showCreate: false,
        showBatchCreate: true
      },
      global: {
        stubs: {
          CreateAccountModal: true,
          BatchCreateAccountsModal: {
            emits: ['close', 'created'],
            template: `
              <div>
                <button class="batch-close" @click="$emit('close')" />
                <button class="batch-created" @click="$emit('created', { created_count: 2, failed_count: 1, results: [] })" />
              </div>
            `
          },
          ModelImportExposureSyncDialog: true,
          ArchiveAccountsModal: true,
          ArchiveGroupAccountsModal: true,
          EditAccountModal: true,
          ReAuthAccountModal: true,
          AccountTestModal: true,
          AccountStatsModal: true,
          ScheduledTestsPanel: true,
          AccountActionMenu: true,
          SyncFromCrsModal: true,
          ImportDataModal: true,
          BulkEditAccountModal: true,
          TempUnschedStatusModal: true,
          ConfirmDialog: true,
          ErrorPassthroughRulesModal: true
        }
      }
    })

    await wrapper.get('.batch-close').trigger('click')
    await wrapper.get('.batch-created').trigger('click')

    expect(wrapper.emitted('close-batch-create')).toEqual([[]])
    expect(wrapper.emitted('batch-created')).toEqual([
      [{ created_count: 2, failed_count: 1, results: [] }]
    ])
  })

  it('forwards archive modal events', async () => {
    const wrapper = mount(AccountsViewDialogsHost, {
      props: {
        ...createProps(),
        showCreate: false,
        showArchiveSelected: true
      },
      global: {
        stubs: {
          CreateAccountModal: true,
          BatchCreateAccountsModal: true,
          ArchiveAccountsModal: {
            emits: ['close', 'archived'],
            template: `
              <div>
                <button class="archive-close" @click="$emit('close')" />
                <button class="archive-done" @click="$emit('archived', { archived_count: 2, failed_count: 0, archive_group_id: 5, archive_group_name: 'Archive' })" />
              </div>
            `
          },
          ArchiveGroupAccountsModal: true,
          ModelImportExposureSyncDialog: true,
          EditAccountModal: true,
          ReAuthAccountModal: true,
          AccountTestModal: true,
          AccountStatsModal: true,
          ScheduledTestsPanel: true,
          AccountActionMenu: true,
          SyncFromCrsModal: true,
          ImportDataModal: true,
          BulkEditAccountModal: true,
          TempUnschedStatusModal: true,
          ConfirmDialog: true,
          ErrorPassthroughRulesModal: true
        }
      }
    })

    await wrapper.get('.archive-close').trigger('click')
    await wrapper.get('.archive-done').trigger('click')

    expect(wrapper.emitted('close-archive-selected')).toEqual([[]])
    expect(wrapper.emitted('archived')).toEqual([
      [{ archived_count: 2, failed_count: 0, archive_group_id: 5, archive_group_name: 'Archive' }]
    ])
  })

  it('forwards archive current group modal events', async () => {
    const wrapper = mount(AccountsViewDialogsHost, {
      props: {
        ...createProps(),
        showCreate: false,
        showArchiveGroup: true,
        archiveSourceGroup: { id: 9, name: 'Prod Group', platform: 'openai' }
      },
      global: {
        stubs: {
          CreateAccountModal: true,
          BatchCreateAccountsModal: true,
          ArchiveAccountsModal: true,
          ArchiveGroupAccountsModal: {
            emits: ['close', 'archived'],
            template: `
              <div>
                <button class="group-archive-close" @click="$emit('close')" />
                <button class="group-archive-done" @click="$emit('archived', { source_group_id: 9, source_group_name: 'Prod Group', archived_count: 3, failed_count: 0, archive_group_id: 5, archive_group_name: 'Archive' })" />
              </div>
            `
          },
          ModelImportExposureSyncDialog: true,
          EditAccountModal: true,
          ReAuthAccountModal: true,
          AccountTestModal: true,
          AccountStatsModal: true,
          ScheduledTestsPanel: true,
          AccountActionMenu: true,
          SyncFromCrsModal: true,
          ImportDataModal: true,
          BulkEditAccountModal: true,
          TempUnschedStatusModal: true,
          ConfirmDialog: true,
          ErrorPassthroughRulesModal: true
        }
      }
    })

    await wrapper.get('.group-archive-close').trigger('click')
    await wrapper.get('.group-archive-done').trigger('click')

    expect(wrapper.emitted('close-archive-group')).toEqual([[]])
    expect(wrapper.emitted('group-archived')).toEqual([
      [{ source_group_id: 9, source_group_name: 'Prod Group', archived_count: 3, failed_count: 0, archive_group_id: 5, archive_group_name: 'Archive' }]
    ])
  })
})
