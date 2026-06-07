import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import UsageFilters from '../UsageFilters.vue'

const adminMocks = vi.hoisted(() => ({
  searchUsers: vi.fn(),
  searchApiKeys: vi.fn(),
  accountsList: vi.fn(),
  groupsList: vi.fn(),
  getModelStats: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    usage: {
      searchUsers: adminMocks.searchUsers,
      searchApiKeys: adminMocks.searchApiKeys
    },
    accounts: {
      list: adminMocks.accountsList
    },
    groups: {
      list: adminMocks.groupsList
    },
    dashboard: {
      getModelStats: adminMocks.getModelStats
    }
  }
}))

vi.mock('@/utils/adminChannelOptions', () => ({
  loadAllAdminChannelOptions: vi.fn(async () => [])
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) =>
      ({
        'admin.usage.deletedUserSuffix': 'Deleted',
        'usage.deletedApiKeySuffix': 'Deleted',
        'admin.usage.userFilter': 'User',
        'usage.apiKeyFilter': 'API Key',
        'admin.usage.searchUserPlaceholder': 'Search user',
        'admin.usage.searchApiKeyPlaceholder': 'Search API key',
        'usage.model': 'Model',
        'usage.platform': 'Platform',
        'admin.usage.account': 'Account',
        'admin.usage.searchAccountPlaceholder': 'Search account',
        'usage.type': 'Type',
        'admin.usage.billingType': 'Billing type',
        'admin.usage.group': 'Group',
        'admin.usage.channel': 'Channel',
        'common.refresh': 'Refresh',
        'common.reset': 'Reset',
        'admin.usage.cleanup.button': 'Cleanup',
        'usage.exportExcel': 'Export'
      })[key] || key
  })
}))

function mountFilters(modelValue: Record<string, unknown> = {}) {
  return mount(UsageFilters, {
    props: {
      modelValue,
      exporting: false,
      startDate: '2026-06-01',
      endDate: '2026-06-07'
    },
    global: {
      stubs: {
        Select: true,
        PlatformIcon: true
      }
    }
  })
}

describe('UsageFilters deleted historical subjects', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    adminMocks.searchUsers.mockReset()
    adminMocks.searchApiKeys.mockReset()
    adminMocks.accountsList.mockReset()
    adminMocks.groupsList.mockReset()
    adminMocks.getModelStats.mockReset()
    adminMocks.groupsList.mockResolvedValue({ items: [] })
    adminMocks.getModelStats.mockResolvedValue({ models: [] })
    adminMocks.accountsList.mockResolvedValue({ items: [] })
  })

  it('shows deleted users and API keys in admin historical usage filters', async () => {
    const filters: Record<string, unknown> = {}
    adminMocks.searchUsers.mockResolvedValue([
      { id: 7, email: 'deleted@example.test', deleted: true }
    ])
    adminMocks.searchApiKeys.mockResolvedValue([
      { id: 19, name: 'old-key', user_id: 7, deleted: true }
    ])

    const wrapper = mountFilters(filters)
    await flushPromises()

    const userInput = wrapper.findAll('input[type="text"]')[0]
    await userInput.trigger('focus')
    await userInput.setValue('deleted')
    await vi.runOnlyPendingTimersAsync()
    await flushPromises()

    expect(wrapper.text()).toContain('deleted@example.test (Deleted)')
    const userOption = wrapper
      .findAll('button[type="button"]')
      .find((button) => button.text().includes('deleted@example.test'))
    expect(userOption).toBeTruthy()
    await userOption!.trigger('click')
    await flushPromises()

    expect(filters.user_id).toBe(7)
    expect((userInput.element as HTMLInputElement).value).toContain('(Deleted)')
    expect(adminMocks.searchApiKeys).toHaveBeenLastCalledWith(7, '')

    const apiKeyInput = wrapper.findAll('input[type="text"]')[1]
    await apiKeyInput.trigger('focus')
    await vi.runOnlyPendingTimersAsync()
    await flushPromises()

    expect(wrapper.text()).toContain('old-key (Deleted)')
    const keyOption = wrapper
      .findAll('button[type="button"]')
      .find((button) => button.text().includes('old-key'))
    expect(keyOption).toBeTruthy()
    await keyOption!.trigger('click')
    await flushPromises()

    expect(filters.api_key_id).toBe(19)
    expect((apiKeyInput.element as HTMLInputElement).value).toContain('(Deleted)')
  })
})
