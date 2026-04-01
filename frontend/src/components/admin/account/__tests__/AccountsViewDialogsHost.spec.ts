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
    showArchiveSelected: false,
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
    showModelDiagnostics: false,
    showErrorPassthrough: false,
    showTlsFingerprintProfiles: false,
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
    diagnosticsAccount: null,
    diagnosticsResult: null,
    diagnosticsLoading: false,
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

function createStubs(overrides: Record<string, unknown> = {}) {
  return {
    CreateAccountModal: true,
    ArchiveAccountsModal: true,
    ModelImportExposureSyncDialog: true,
    EditAccountModal: true,
    ReAuthAccountModal: true,
    AccountTestModal: true,
    AccountStatsModal: true,
    AccountModelDiagnosticsModal: true,
    ScheduledTestsPanel: true,
    AccountActionMenu: true,
    SyncFromCrsModal: true,
    ImportDataModal: true,
    BulkEditAccountModal: true,
    TempUnschedStatusModal: true,
    ConfirmDialog: true,
    ErrorPassthroughRulesModal: true,
    TLSFingerprintProfilesModal: true,
    ...overrides
  }
}

describe('AccountsViewDialogsHost', () => {
  it('forwards create modal events', async () => {
    const wrapper = mount(AccountsViewDialogsHost, {
      props: createProps(),
      global: {
        stubs: createStubs({
          CreateAccountModal: {
            emits: ['close', 'created', 'models-imported'],
            template: `
              <div>
                <button class="create-close" @click="$emit('close')" />
                <button class="create-created" @click="$emit('created')" />
                <button class="create-models" @click="$emit('models-imported', { ok: true })" />
              </div>
            `
          }
        })
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
        stubs: createStubs({
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
        })
      }
    })

    await wrapper.get('input[type="checkbox"]').setValue(false)
    await wrapper.get('.confirm-action').trigger('click')
    await wrapper.get('.cancel-action').trigger('click')

    expect(wrapper.emitted('update:includeProxyOnExport')).toEqual([[false]])
    expect(wrapper.emitted('confirm-export')).toEqual([[]])
    expect(wrapper.emitted('close-export')).toEqual([[]])
  })

  it('forwards archive modal events', async () => {
    const wrapper = mount(AccountsViewDialogsHost, {
      props: {
        ...createProps(),
        showCreate: false,
        showArchiveSelected: true
      },
      global: {
        stubs: createStubs({
          ArchiveAccountsModal: {
            emits: ['close', 'archived'],
            template: `
              <div>
                <button class="archive-close" @click="$emit('close')" />
                <button class="archive-done" @click="$emit('archived', { archived_count: 2, failed_count: 0, archive_group_id: 5, archive_group_name: 'Archive' })" />
              </div>
            `
          }
        })
      }
    })

    await wrapper.get('.archive-close').trigger('click')
    await wrapper.get('.archive-done').trigger('click')

    expect(wrapper.emitted('close-archive-selected')).toEqual([[]])
    expect(wrapper.emitted('archived')).toEqual([
      [{ archived_count: 2, failed_count: 0, archive_group_id: 5, archive_group_name: 'Archive' }]
    ])
  })

  it('does not render an archive current group modal', async () => {
    const wrapper = mount(AccountsViewDialogsHost, {
      props: {
        ...createProps(),
        showCreate: false
      },
      global: {
        stubs: createStubs({
          ArchiveGroupAccountsModal: {
            template: `
              <div class="archive-current-group-modal">
                should not render
              </div>
            `
          }
        })
      }
    })

    expect(wrapper.find('.archive-current-group-modal').exists()).toBe(false)
  })

  it('forwards blacklist events from the action menu', async () => {
    const account = {
      id: 7,
      name: 'openai-7',
      platform: 'openai',
      type: 'apikey',
      status: 'active',
      schedulable: true
    }

    const wrapper = mount(AccountsViewDialogsHost, {
      props: {
        ...createProps(),
        showCreate: false,
        menuShow: true,
        menuAccount: account,
        menuPosition: { top: 10, left: 20 }
      },
      global: {
        stubs: createStubs({
          AccountActionMenu: {
            emits: ['blacklist', 'close'],
            template: `
              <div>
                <button class="menu-blacklist" @click="$emit('blacklist', { id: 7, name: 'openai-7' })" />
                <button class="menu-close" @click="$emit('close')" />
              </div>
            `
          }
        })
      }
    })

    await wrapper.get('.menu-blacklist').trigger('click')
    await wrapper.get('.menu-close').trigger('click')

    expect(wrapper.emitted('blacklist')).toEqual([[{ id: 7, name: 'openai-7' }]])
    expect(wrapper.emitted('close-menu')).toEqual([[]])
  })

  it('forwards downstream diagnostics events from the action menu and diagnostics modal', async () => {
    const account = {
      id: 11,
      name: 'grok-11',
      platform: 'grok',
      type: 'apikey',
      status: 'active',
      schedulable: true
    }

    const wrapper = mount(AccountsViewDialogsHost, {
      props: {
        ...createProps(),
        showCreate: false,
        menuShow: true,
        menuAccount: account,
        menuPosition: { top: 10, left: 20 },
        showModelDiagnostics: true,
        diagnosticsAccount: account
      },
      global: {
        stubs: createStubs({
          AccountActionMenu: {
            emits: ['diagnose-models', 'close'],
            template: `
              <div>
                <button class="menu-diagnose" @click="$emit('diagnose-models', { id: 11, name: 'grok-11' })" />
              </div>
            `
          },
          AccountModelDiagnosticsModal: {
            emits: ['close', 'refresh'],
            template: `
              <div>
                <button class="diag-refresh" @click="$emit('refresh')" />
                <button class="diag-close" @click="$emit('close')" />
              </div>
            `
          }
        })
      }
    })

    await wrapper.get('.menu-diagnose').trigger('click')
    await wrapper.get('.diag-refresh').trigger('click')
    await wrapper.get('.diag-close').trigger('click')

    expect(wrapper.emitted('diagnose-models')).toEqual([[{ id: 11, name: 'grok-11' }]])
    expect(wrapper.emitted('refresh-model-diagnostics')).toEqual([[]])
    expect(wrapper.emitted('close-model-diagnostics')).toEqual([[]])
  })

  it('forwards test modal blacklist events', async () => {
    const wrapper = mount(AccountsViewDialogsHost, {
      props: {
        ...createProps(),
        showCreate: false,
        showTest: true,
        testingAccount: {
          id: 9,
          name: 'openai-9',
          platform: 'openai',
          type: 'apikey',
          status: 'active'
        }
      },
      global: {
        stubs: createStubs({
          AccountTestModal: {
            emits: ['close', 'blacklist'],
            template: `
              <div>
                <button
                  class="test-blacklist"
                  @click="$emit('blacklist', {
                    account: { id: 9, name: 'openai-9' },
                    source: 'test_modal',
                    feedback: { fingerprint: 'fp-9', action: 'blacklist' }
                  })"
                />
              </div>
            `
          }
        })
      }
    })

    await wrapper.get('.test-blacklist').trigger('click')

    expect(wrapper.emitted('test-blacklist')).toEqual([[
      {
        account: { id: 9, name: 'openai-9' },
        source: 'test_modal',
        feedback: { fingerprint: 'fp-9', action: 'blacklist' }
      }
    ]])
  })
})
