import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import BlacklistedAccountsView from '../BlacklistedAccountsView.vue'

const {
  listAccounts,
  retestBlacklistedAccounts,
  deleteAccount,
  getAllGroups,
  showSuccess,
  showWarning,
  showError
} = vi.hoisted(() => ({
  listAccounts: vi.fn(),
  retestBlacklistedAccounts: vi.fn(),
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

describe('BlacklistedAccountsView', () => {
  beforeEach(() => {
    listAccounts.mockReset()
    retestBlacklistedAccounts.mockReset()
    deleteAccount.mockReset()
    getAllGroups.mockReset()
    showSuccess.mockReset()
    showWarning.mockReset()
    showError.mockReset()

    listAccounts.mockResolvedValue({
      items: [
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
      ],
      total: 1,
      pages: 1
    })
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
  })

  it('supports batch retest restore from blacklist', async () => {
    const wrapper = mount(BlacklistedAccountsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /></div>' },
          SearchInput: true,
          Select: true,
          DataTable: DataTableStub,
          Pagination: true,
          PlatformTypeBadge: true,
          AccountGroupsCell: true
        }
      }
    })

    await flushPromises()

    const checkboxes = wrapper.findAll('input[type="checkbox"]')
    expect(checkboxes).toHaveLength(2)

    await checkboxes[1].setValue(true)
    await wrapper.get('button.btn-primary').trigger('click')
    await flushPromises()

    expect(retestBlacklistedAccounts).toHaveBeenCalledWith([1])
    expect(showSuccess).toHaveBeenCalledWith('admin.accounts.blacklist.retestSuccess')
  })
})
