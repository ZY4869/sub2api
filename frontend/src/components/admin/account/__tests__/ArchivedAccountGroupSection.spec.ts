import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi, beforeEach } from 'vitest'
import ArchivedAccountGroupSection from '../ArchivedAccountGroupSection.vue'

const {
  listAccounts,
  unarchiveAccounts,
  showSuccess,
  showWarning,
  showError
} = vi.hoisted(() => ({
  listAccounts: vi.fn(),
  unarchiveAccounts: vi.fn(),
  showSuccess: vi.fn(),
  showWarning: vi.fn(),
  showError: vi.fn()
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

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      list: listAccounts,
      unarchiveAccounts
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

function mountSection() {
  return mount(ArchivedAccountGroupSection, {
    props: {
      summary: {
        group_id: 9,
        group_name: 'Archive 2026',
        total_count: 123,
        available_count: 7,
        invalid_count: 23,
        latest_updated_at: '2026-03-23T00:00:00Z'
      },
      filters: {
        platform: '',
        type: '',
        status: '',
        group: '',
        search: ''
      },
      columns: [
        { key: 'name', label: 'Name' },
        { key: 'actions', label: 'Actions' }
      ],
      togglingSchedulable: null,
      todayStatsByAccountId: {},
      todayStatsLoading: false,
      todayStatsError: null,
      usageManualRefreshToken: 0,
      sortStorageKey: 'account-table-sort-archived',
      refreshToken: 0
    },
    global: {
      stubs: {
        AccountsViewTable: {
          props: ['accounts'],
          template: `
            <div data-test="archived-table">
              <div v-for="account in accounts" :key="account.id" class="account-row">{{ account.name }}</div>
            </div>
          `
        },
        AccountsViewRowActions: true
      }
    }
  })
}

describe('ArchivedAccountGroupSection', () => {
  beforeEach(() => {
    listAccounts.mockReset()
    unarchiveAccounts.mockReset()
    showSuccess.mockReset()
    showWarning.mockReset()
    showError.mockReset()
  })

  it('defaults to collapsed and pads counts from total digits', () => {
    const wrapper = mountSection()

    expect(wrapper.find('[data-test="archived-table"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('007')
    expect(wrapper.text()).toContain('023')
    expect(listAccounts).not.toHaveBeenCalled()
  })

  it('loads accounts on expand and unarchives the whole group', async () => {
    listAccounts.mockResolvedValue({
      items: [
        { id: 1, name: 'account-one' },
        { id: 2, name: 'account-two' }
      ],
      total: 2,
      page: 1,
      page_size: 10,
      pages: 1
    })
    unarchiveAccounts.mockResolvedValue({
      restored_count: 2,
      failed_count: 0,
      restored_to_original_group_count: 2,
      restored_in_place_count: 0,
      results: []
    })

    const wrapper = mountSection()
    await wrapper.get('button[type="button"]').trigger('click')
    await flushPromises()

    expect(listAccounts).toHaveBeenCalledWith(1, 10, {
      platform: '',
      type: '',
      status: '',
      search: '',
      group: '9',
      lifecycle: 'archived'
    })
    expect(wrapper.text()).toContain('account-one')
    expect(wrapper.text()).toContain('account-two')

    const buttons = wrapper.findAll('button')
    const unarchiveGroupButton = buttons.find((button) =>
      button.text().includes('admin.accounts.unarchiveGroup')
    )

    expect(unarchiveGroupButton).toBeTruthy()
    await unarchiveGroupButton?.trigger('click')
    await flushPromises()

    expect(unarchiveAccounts).toHaveBeenCalledWith([1, 2])
    expect(showSuccess).toHaveBeenCalledWith('admin.accounts.unarchiveSuccess')
    expect(wrapper.emitted('changed')).toEqual([[]])
  })
})
