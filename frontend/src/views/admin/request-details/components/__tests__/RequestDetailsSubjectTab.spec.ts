import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import RequestDetailsSubjectTab from '../RequestDetailsSubjectTab.vue'

const routeQuery = vi.hoisted(() => ({ value: {} as Record<string, string> }))

const adminMocks = vi.hoisted(() => ({
  accountsList: vi.fn(),
  accountsGetById: vi.fn(),
  groupsList: vi.fn(),
  groupsGetById: vi.fn(),
  searchApiKeys: vi.fn(),
  usageList: vi.fn(),
  getSubjectInsights: vi.fn()
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({
    query: routeQuery.value
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

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      list: adminMocks.accountsList,
      getById: adminMocks.accountsGetById
    },
    groups: {
      list: adminMocks.groupsList,
      getById: adminMocks.groupsGetById
    },
    usage: {
      searchApiKeys: adminMocks.searchApiKeys
    }
  }
}))

vi.mock('@/api/admin/usage', () => ({
  adminUsageAPI: {
    list: adminMocks.usageList
  }
}))

vi.mock('@/api/admin/ops', () => ({
  opsAPI: {
    getSubjectInsights: adminMocks.getSubjectInsights
  }
}))

vi.mock('@/composables/useTokenDisplayMode', () => ({
  useTokenDisplayMode: () => ({
    formatTokenDisplay: (value: number) => String(value)
  })
}))

function mountTab() {
  return mount(RequestDetailsSubjectTab, {
    global: {
      stubs: {
        ModelDistributionChart: true,
        EndpointDistributionChart: true,
        RequestDetailsSubjectTrendChart: true,
        RequestDetailsSubjectUsageTable: true
      }
    }
  })
}

describe('RequestDetailsSubjectTab', () => {
  beforeEach(() => {
    routeQuery.value = {}
    adminMocks.accountsList.mockReset()
    adminMocks.accountsGetById.mockReset()
    adminMocks.groupsList.mockReset()
    adminMocks.groupsGetById.mockReset()
    adminMocks.searchApiKeys.mockReset()
    adminMocks.usageList.mockReset()
    adminMocks.getSubjectInsights.mockReset()

    adminMocks.accountsList.mockResolvedValue({
      items: [{ id: 11, name: 'Account Alpha' }]
    })
    adminMocks.accountsGetById.mockResolvedValue({ id: 11, name: 'Account Alpha' })
    adminMocks.groupsList.mockResolvedValue({
      items: [{ id: 22, name: 'Group Beta' }]
    })
    adminMocks.groupsGetById.mockResolvedValue({ id: 22, name: 'Group Beta' })
    adminMocks.searchApiKeys.mockResolvedValue([{ id: 33, name: 'Key Gamma', deleted: false }])
    adminMocks.getSubjectInsights.mockResolvedValue({
      subject: { id: 1, type: 'account', name: 'Account Alpha' },
      summary: {
        total_account_cost: 0,
        total_user_cost: 0,
        total_standard_cost: 0,
        total_requests: 0,
        total_tokens: 0,
        avg_duration_ms: 0,
        active_days: 0,
        window_days: 30
      },
      history: [],
      models: [],
      endpoints: [],
      upstream_endpoints: [],
      request_preview_coverage: {
        preview_available_rate: 0,
        preview_available_count: 0,
        normalized_count: 0,
        upstream_request_count: 0,
        upstream_response_count: 0,
        gateway_response_count: 0
      }
    })
    adminMocks.usageList.mockResolvedValue({
      items: [],
      total: 0
    })
  })

  it('lists existing candidates for account, group, and API key subjects and refreshes data after selection', async () => {
    const wrapper = mountTab()
    await flushPromises()

    const subjectInput = wrapper.get('input[type="text"]')
    await subjectInput.trigger('focus')
    await flushPromises()
    expect(wrapper.text()).toContain('Account Alpha')

    await wrapper.get('button[type="button"]').trigger('click')
    await flushPromises()

    expect(adminMocks.getSubjectInsights).toHaveBeenLastCalledWith(
      expect.objectContaining({ subject_type: 'account', subject_id: 11 })
    )
    expect(adminMocks.usageList).toHaveBeenLastCalledWith(
      expect.objectContaining({ account_id: 11 })
    )

    await wrapper.findAll('select')[0].setValue('group')
    await subjectInput.setValue('Group')
    await flushPromises()
    expect(adminMocks.groupsList).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('Group Beta')

    const groupOption = wrapper
      .findAll('button[type="button"]')
      .find((button) => button.text().includes('Group Beta'))
    expect(groupOption).toBeTruthy()
    await groupOption!.trigger('click')
    await flushPromises()

    expect(adminMocks.getSubjectInsights).toHaveBeenLastCalledWith(
      expect.objectContaining({ subject_type: 'group', subject_id: 22 })
    )
    expect(adminMocks.usageList).toHaveBeenLastCalledWith(
      expect.objectContaining({ group_id: 22 })
    )

    await wrapper.findAll('select')[0].setValue('api_key')
    await subjectInput.setValue('Key')
    await flushPromises()
    expect(adminMocks.searchApiKeys).toHaveBeenLastCalledWith(undefined, 'Key')

    const apiKeyOption = wrapper
      .findAll('button[type="button"]')
      .find((button) => button.text().includes('Key Gamma'))
    expect(apiKeyOption).toBeTruthy()
    await apiKeyOption!.trigger('click')
    await flushPromises()

    expect(adminMocks.getSubjectInsights).toHaveBeenLastCalledWith(
      expect.objectContaining({ subject_type: 'api_key', subject_id: 33 })
    )
    expect(adminMocks.usageList).toHaveBeenLastCalledWith(
      expect.objectContaining({ api_key_id: 33 })
    )
  })

  it('hydrates route-prefilled account labels through getById', async () => {
    routeQuery.value = { account_id: '77' }
    adminMocks.accountsGetById.mockResolvedValueOnce({ id: 77, name: 'Prefilled Account' })

    const wrapper = mountTab()
    await flushPromises()

    const subjectInput = wrapper.get('input[type="text"]')
    expect((subjectInput.element as HTMLInputElement).value).toContain('Prefilled Account')
    expect(adminMocks.getSubjectInsights).toHaveBeenCalledWith(
      expect.objectContaining({ subject_type: 'account', subject_id: 77 })
    )
  })

  it('falls back to raw IDs for route-prefilled API keys without a detail lookup API', async () => {
    routeQuery.value = { api_key_id: '88' }

    const wrapper = mountTab()
    await flushPromises()

    const subjectInput = wrapper.get('input[type="text"]')
    expect((subjectInput.element as HTMLInputElement).value).toBe('#88')
    expect(adminMocks.getSubjectInsights).toHaveBeenCalledWith(
      expect.objectContaining({ subject_type: 'api_key', subject_id: 88 })
    )
  })
})
