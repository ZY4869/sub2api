import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import OpsSystemLogTable from '../OpsSystemLogTable.vue'

const mockListSystemLogs = vi.fn()
const mockGetSystemLogSinkHealth = vi.fn()
const mockGetRuntimeLogConfig = vi.fn()

vi.mock('@/api/admin/ops', () => ({
  opsAPI: {
    listSystemLogs: (...args: any[]) => mockListSystemLogs(...args),
    getSystemLogSinkHealth: (...args: any[]) => mockGetSystemLogSinkHealth(...args),
    getRuntimeLogConfig: (...args: any[]) => mockGetRuntimeLogConfig(...args)
  }
}))

const mockShowError = vi.fn()
const mockShowSuccess = vi.fn()

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    showError: mockShowError,
    showSuccess: mockShowSuccess
  })
}))

const PaginationStub = defineComponent({
  name: 'PaginationStub',
  template: '<div class="pagination-stub" />'
})

const SelectStub = defineComponent({
  name: 'SelectStub',
  props: ['modelValue', 'options'],
  template: '<div class="select-stub" />'
})

describe('OpsSystemLogTable', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockListSystemLogs.mockResolvedValue({
      items: [],
      total: 0
    })
    mockGetSystemLogSinkHealth.mockResolvedValue({
      queue_depth: 0,
      queue_capacity: 0,
      dropped_count: 0,
      write_failed_count: 0,
      written_count: 0,
      avg_write_delay_ms: 0
    })
    mockGetRuntimeLogConfig.mockResolvedValue({
      level: 'info',
      enable_sampling: false,
      sampling_initial: 100,
      sampling_thereafter: 100,
      caller: true,
      stacktrace_level: 'error',
      retention_days: 30
    })
  })

  it('does not fetch on mount before the first parent refresh token arrives', async () => {
    const wrapper = mount(OpsSystemLogTable, {
      props: {
        refreshToken: 0,
        platformFilter: ''
      },
      global: {
        stubs: {
          Pagination: PaginationStub,
          Select: SelectStub
        }
      }
    })

    await flushPromises()
    expect(mockListSystemLogs).not.toHaveBeenCalled()
    expect(mockGetSystemLogSinkHealth).not.toHaveBeenCalled()
    expect(mockGetRuntimeLogConfig).not.toHaveBeenCalled()

    await wrapper.setProps({ refreshToken: 1 })
    await flushPromises()

    expect(mockListSystemLogs).toHaveBeenCalledTimes(1)
    expect(mockGetSystemLogSinkHealth).toHaveBeenCalledTimes(1)
    expect(mockGetRuntimeLogConfig).toHaveBeenCalledTimes(1)

    await wrapper.setProps({ refreshToken: 2 })
    await flushPromises()

    expect(mockListSystemLogs).toHaveBeenCalledTimes(2)
    expect(mockGetSystemLogSinkHealth).toHaveBeenCalledTimes(2)
    expect(mockGetRuntimeLogConfig).toHaveBeenCalledTimes(1)
  })

  it('loads exactly once when mounted after the parent already emitted the first refresh token', async () => {
    mount(OpsSystemLogTable, {
      props: {
        refreshToken: 1,
        platformFilter: 'openai'
      },
      global: {
        stubs: {
          Pagination: PaginationStub,
          Select: SelectStub
        }
      }
    })

    await flushPromises()
    expect(mockListSystemLogs).toHaveBeenCalledTimes(1)
    expect(mockGetSystemLogSinkHealth).toHaveBeenCalledTimes(1)
    expect(mockGetRuntimeLogConfig).toHaveBeenCalledTimes(1)
    expect(mockListSystemLogs).toHaveBeenCalledWith(
      expect.objectContaining({
        page: 1,
        page_size: 20,
        time_range: '1h',
        platform: 'openai'
      })
    )
  })

  it('includes api_key_id in log filters', async () => {
    const wrapper = mount(OpsSystemLogTable, {
      props: {
        refreshToken: 1,
        platformFilter: ''
      },
      global: {
        stubs: {
          Pagination: PaginationStub,
          Select: SelectStub
        }
      }
    })

    await flushPromises()

    const input = wrapper.findAll('input').find((item) => {
      return item.attributes('type') === 'text' && item.element.previousSibling?.textContent?.includes('api_key_id')
    })
    expect(input).toBeTruthy()
    await input!.setValue('42')
    const searchButton = wrapper
      .findAll('button')
      .find((button) => button.text() === '查询')
    expect(searchButton).toBeTruthy()
    await searchButton!.trigger('click')
    await flushPromises()

    expect(mockListSystemLogs).toHaveBeenLastCalledWith(
      expect.objectContaining({
        api_key_id: 42
      })
    )
  })
})
