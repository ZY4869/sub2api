import { defineComponent, ref } from 'vue'
import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AccountsViewDialogsHost from '../AccountsViewDialogsHost.vue'
import AccountsViewToolbar from '../AccountsViewToolbar.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

const Harness = defineComponent({
  components: {
    AccountsViewToolbar,
    AccountsViewDialogsHost
  },
  setup() {
    const showBatchCreate = ref(false)
    const includeProxyOnExport = ref(true)
    return {
      includeProxyOnExport,
      showBatchCreate
    }
  },
  template: `
    <div>
      <AccountsViewToolbar
        :loading="false"
        :usage-refreshing="false"
        search-query=""
        :filters="{ platform: '', type: '', status: '', group: '', search: '' }"
        :groups="[]"
        :has-pending-list-sync="false"
        :archived-count="0"
        :selected-count="0"
        :auto-refresh-enabled="false"
        :auto-refresh-countdown="0"
        :auto-refresh-intervals="[5, 10]"
        :auto-refresh-interval-seconds="5"
        :toggleable-columns="[]"
        @batch-create="showBatchCreate = true"
      />
      <AccountsViewDialogsHost
        v-model:include-proxy-on-export="includeProxyOnExport"
        :show-create="false"
        :show-batch-create="showBatchCreate"
        :show-archive-selected="false"
        :show-archive-group="false"
        :show-edit="false"
        :show-sync="false"
        :show-import-data="false"
        :show-export-data-dialog="false"
        :show-bulk-edit="false"
        :show-temp-unsched="false"
        :show-delete-dialog="false"
        :show-re-auth="false"
        :show-test="false"
        :show-stats="false"
        :show-error-passthrough="false"
        :show-schedule-panel="false"
        :proxies="[]"
        :groups="[]"
        :selected-ids="[]"
        :selected-platforms="[]"
        :selected-types="[]"
        :archive-source-group="null"
        :editing-account="null"
        :temp-unsched-account="null"
        :deleting-account="null"
        :re-auth-account="null"
        :testing-account="null"
        :stats-account="null"
        :schedule-account="null"
        :schedule-model-options="[]"
        :sync-dialog-open="false"
        :sync-dialog-models="[]"
        :sync-dialog-submitting="false"
        :menu-show="false"
        :menu-account="null"
        :menu-position="null"
        @close-batch-create="showBatchCreate = false"
      />
    </div>
  `
})

describe('batch create toolbar flow', () => {
  it('opens batch create modal from the toolbar button', async () => {
    const wrapper = mount(Harness, {
      global: {
        stubs: {
          Icon: true,
          AccountTableFilters: true,
          AccountTableActions: {
            emits: ['refresh', 'sync', 'create'],
            template: `
              <div>
                <slot name="after" />
                <slot name="beforeCreate" />
              </div>
            `
          },
          CreateAccountModal: true,
          ArchiveAccountsModal: true,
          BatchCreateAccountsModal: {
            props: ['show'],
            template: '<div v-if="show" data-test="batch-create-modal">batch modal</div>'
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

    const batchCreateButton = wrapper.findAll('button').find((button) =>
      button.text().includes('admin.accounts.batchCreate')
    )

    expect(batchCreateButton).toBeTruthy()
    await batchCreateButton?.trigger('click')

    expect(wrapper.find('[data-test="batch-create-modal"]').exists()).toBe(true)
  })
})
