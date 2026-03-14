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
})
