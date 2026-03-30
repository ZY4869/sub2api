import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import BlacklistedAccountsView from '../BlacklistedAccountsView.vue'

const {
  listAccounts,
  retestBlacklistedAccounts,
  batchDeleteBlacklistedAccounts,
  deleteAccount,
  getAllGroups,
  showSuccess,
  showWarning,
  showError
} = vi.hoisted(() => ({
  listAccounts: vi.fn(),
  retestBlacklistedAccounts: vi.fn(),
  batchDeleteBlacklistedAccounts: vi.fn(),
  deleteAccount: vi.fn(),
  getAllGroups: vi.fn(),
  showSuccess: vi.fn(),
  showWarning: vi.fn(),
  showError: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      list: listAccounts,
      retestBlacklistedAccounts,
      batchDeleteBlacklistedAccounts,
      delete: deleteAccount
    },
    groups: {
      getAll: getAllGroups
    }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showSuccess,
    showWarning,
    showError
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

const DataTableStub = {
  props: ['data'],
  template: `
    <div>
      <slot name="header-select" />
      <div v-for="row in data" :key="row.id" class="table-row">
        <slot name="cell-select" :row="row" />
        <slot name="cell-name" :row="row" />
        <slot name="cell-platform" :row="row" />
        <slot name="cell-groups" :row="row" />
        <slot name="cell-blacklisted_at" :value="row.blacklisted_at" />
        <slot name="cell-blacklist_purge_at" :value="row.blacklist_purge_at" />
        <slot name="cell-actions" :row="row" />
      </div>
    </div>
  `
}

const BlacklistRetestModalStub = {
  props: ['show', 'accounts'],
  emits: ['close', 'confirm'],
  methods: {
    emitConfirm() {
      this.$emit('confirm', {
        account_ids: this.accounts.map((account: { id: number }) => account.id),
        model_input_mode: 'catalog',
        model_id: 'gpt-5.4'
      })
    }
  },
  template: `
    <div v-if="show" data-test="blacklist-retest-modal">
      <button type="button" data-test="confirm-blacklist-retest" @click="emitConfirm">confirm</button>
      <button type="button" data-test="close-blacklist-retest" @click="$emit('close')">close</button>
    </div>
  `
}

const globalStubs = {
  AppLayout: { template: '<div><slot /></div>' },
  TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /></div>' },
  SearchInput: true,
  Select: true,
  DataTable: DataTableStub,
  Pagination: true,
  PlatformTypeBadge: true,
  AccountGroupsCell: true,
  BlacklistRetestModal: BlacklistRetestModalStub
}

describe('BlacklistedAccountsView', () => {
  beforeEach(() => {
    listAccounts.mockReset()
    retestBlacklistedAccounts.mockReset()
    batchDeleteBlacklistedAccounts.mockReset()
    deleteAccount.mockReset()
    getAllGroups.mockReset()
    showSuccess.mockReset()
    showWarning.mockReset()
    showError.mockReset()

    listAccounts.mockImplementation(async (_page: number, pageSize: number, _filters: unknown, options?: { signal?: AbortSignal }) => ({
      items: options?.signal
        ? [
            {
              id: 1,
              name: 'blacklisted-1',
              platform: 'openai',
              type: 'apikey',
              groups: [],
              lifecycle_reason_message: 'account deactivated',
              blacklisted_at: '2026-03-23T12:00:00Z',
              blacklist_purge_at: '2026-03-26T12:00:00Z'
            }
          ]
        : [],
      total: 1,
      page: 1,
      page_size: pageSize,
      pages: 1
    }))
    getAllGroups.mockResolvedValue([{ id: 1, name: 'OpenAI Group' }])
    retestBlacklistedAccounts.mockResolvedValue({
      results: [
        {
          account_id: 1,
          success: true,
          restored: true,
          error_message: '',
          response_text: 'ok',
          latency_ms: 120
        }
      ]
    })
    batchDeleteBlacklistedAccounts.mockResolvedValue({
      deleted_ids: [1],
      failed: [],
      deleted_count: 1,
      failed_count: 0
    })
    vi.stubGlobal('confirm', vi.fn(() => true))
    deleteAccount.mockResolvedValue({ message: 'ok' })
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('supports batch retest restore from blacklist', async () => {
    const wrapper = mount(BlacklistedAccountsView, {
      global: {
        stubs: globalStubs
      }
    })

    await flushPromises()

    expect(wrapper.text()).toContain('admin.accounts.blacklist.totalCountLabel')
    expect(wrapper.text()).toContain('admin.accounts.blacklist.currentResultLabel')

    const checkboxes = wrapper.findAll('input[type="checkbox"]')
    expect(checkboxes).toHaveLength(2)

    await checkboxes[1].setValue(true)
    await wrapper.get('button.btn-primary').trigger('click')
    await flushPromises()

    expect(wrapper.find('[data-test="blacklist-retest-modal"]').exists()).toBe(true)
    expect(retestBlacklistedAccounts).not.toHaveBeenCalled()

    await wrapper.get('[data-test="confirm-blacklist-retest"]').trigger('click')
    await flushPromises()

    expect(retestBlacklistedAccounts).toHaveBeenCalledWith({
      account_ids: [1],
      model_input_mode: 'catalog',
      model_id: 'gpt-5.4'
    })
    expect(showSuccess).toHaveBeenCalledWith('admin.accounts.blacklist.retestSuccess')
    expect(listAccounts.mock.calls.length).toBeGreaterThan(2)
  })

  it('supports batch delete for selected blacklisted accounts', async () => {
    const wrapper = mount(BlacklistedAccountsView, {
      global: {
        stubs: globalStubs
      }
    })

    await flushPromises()

    const checkboxes = wrapper.findAll('input[type="checkbox"]')
    await checkboxes[1].setValue(true)

    const buttons = wrapper.findAll('button')
    const batchDeleteButton = buttons.find((button) => button.text().includes('admin.accounts.blacklist.batchDelete'))
    expect(batchDeleteButton).toBeTruthy()

    await batchDeleteButton!.trigger('click')
    await flushPromises()

    expect(confirm).toHaveBeenCalledWith('admin.accounts.blacklist.batchDeleteConfirm')
    expect(batchDeleteBlacklistedAccounts).toHaveBeenCalledWith({ ids: [1] })
    expect(showSuccess).toHaveBeenCalledWith('admin.accounts.blacklist.batchDeleteSuccess')
    expect(listAccounts.mock.calls.length).toBeGreaterThan(2)
  })

  it('supports deleting the entire blacklist without selection', async () => {
    listAccounts.mockImplementation(async (_page: number, pageSize: number, _filters: unknown, options?: { signal?: AbortSignal }) => ({
      items: options?.signal
        ? [
            {
              id: 1,
              name: 'blacklisted-1',
              platform: 'openai',
              type: 'apikey',
              groups: [],
              lifecycle_reason_message: 'account deactivated',
              blacklisted_at: '2026-03-23T12:00:00Z',
              blacklist_purge_at: '2026-03-26T12:00:00Z'
            },
            {
              id: 2,
              name: 'blacklisted-2',
              platform: 'openai',
              type: 'apikey',
              groups: [],
              lifecycle_reason_message: 'workspace deactivated',
              blacklisted_at: '2026-03-23T13:00:00Z',
              blacklist_purge_at: '2026-03-26T13:00:00Z'
            }
          ]
        : [],
      total: 2,
      page: 1,
      page_size: pageSize,
      pages: 1
    }))
    batchDeleteBlacklistedAccounts.mockResolvedValue({
      deleted_ids: [1, 2],
      failed: [],
      deleted_count: 2,
      failed_count: 0
    })

    const wrapper = mount(BlacklistedAccountsView, {
      global: {
        stubs: globalStubs
      }
    })

    await flushPromises()

    const deleteAllButton = wrapper.findAll('button').find((button) =>
      button.text().includes('admin.accounts.blacklist.deleteAll')
    )
    expect(deleteAllButton).toBeTruthy()

    await deleteAllButton!.trigger('click')
    await flushPromises()

    expect(confirm).toHaveBeenCalledWith('admin.accounts.blacklist.deleteAllConfirm')
    expect(batchDeleteBlacklistedAccounts).toHaveBeenCalledWith({ delete_all: true })
    expect(showSuccess).toHaveBeenCalledWith('admin.accounts.blacklist.deleteAllSuccess')
  })

  it('shows partial warning when batch delete only succeeds for some accounts', async () => {
    listAccounts.mockImplementation(async (_page: number, pageSize: number, _filters: unknown, options?: { signal?: AbortSignal }) => ({
      items: options?.signal
        ? [
            {
              id: 1,
              name: 'blacklisted-1',
              platform: 'openai',
              type: 'apikey',
              groups: [],
              lifecycle_reason_message: 'account deactivated',
              blacklisted_at: '2026-03-23T12:00:00Z',
              blacklist_purge_at: '2026-03-26T12:00:00Z'
            },
            {
              id: 2,
              name: 'blacklisted-2',
              platform: 'openai',
              type: 'apikey',
              groups: [],
              lifecycle_reason_message: 'workspace deactivated',
              blacklisted_at: '2026-03-23T13:00:00Z',
              blacklist_purge_at: '2026-03-26T13:00:00Z'
            }
          ]
        : [],
      total: 2,
      page: 1,
      page_size: pageSize,
      pages: 1
    }))
    batchDeleteBlacklistedAccounts.mockResolvedValue({
      deleted_ids: [1],
      failed: [{ id: 2, reason: 'delete failed' }],
      deleted_count: 1,
      failed_count: 1
    })

    const wrapper = mount(BlacklistedAccountsView, {
      global: {
        stubs: globalStubs
      }
    })

    await flushPromises()

    const checkboxes = wrapper.findAll('input[type="checkbox"]')
    await checkboxes[1].setValue(true)
    await checkboxes[2].setValue(true)

    const batchDeleteButton = wrapper.findAll('button').find((button) =>
      button.text().includes('admin.accounts.blacklist.batchDelete')
    )
    expect(batchDeleteButton).toBeTruthy()

    await batchDeleteButton!.trigger('click')
    await flushPromises()

    expect(batchDeleteBlacklistedAccounts).toHaveBeenCalledWith({ ids: [1, 2] })
    expect(showWarning).toHaveBeenCalledWith('admin.accounts.blacklist.batchDeletePartial')
  })
})
