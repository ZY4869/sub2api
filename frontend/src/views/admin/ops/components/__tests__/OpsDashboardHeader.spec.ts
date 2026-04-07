import { describe, expect, it, beforeEach, vi } from 'vitest'
import { defineComponent } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'

import OpsDashboardHeader from '../OpsDashboardHeader.vue'

const mockGetRealtimeTrafficSummary = vi.fn()
const mockGetAllGroups = vi.fn()
const mockLoadAllAdminChannelOptions = vi.fn()

vi.mock('@/api', () => ({
  adminAPI: {
    groups: {
      getAll: (...args: any[]) => mockGetAllGroups(...args),
    },
  },
}))

vi.mock('@/api/admin/ops', () => ({
  opsAPI: {
    getRealtimeTrafficSummary: (...args: any[]) => mockGetRealtimeTrafficSummary(...args),
  },
}))

vi.mock('@/stores', () => ({
  useAdminSettingsStore: () => ({
    opsRealtimeMonitoringEnabled: true,
    setOpsRealtimeMonitoringEnabledLocal: vi.fn(),
  }),
}))

vi.mock('@/utils/adminChannelOptions', () => ({
  loadAllAdminChannelOptions: (...args: any[]) => mockLoadAllAdminChannelOptions(...args),
}))

vi.mock('@/utils/format', () => ({
  formatNumber: (value: number) => String(value),
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

const SelectStub = defineComponent({
  name: 'Select',
  props: {
    modelValue: {
      type: [String, Number, Boolean],
      default: null,
    },
  },
  emits: ['update:modelValue'],
  template: '<div class="select-stub" />',
})

describe('OpsDashboardHeader', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockGetAllGroups.mockResolvedValue([])
    mockLoadAllAdminChannelOptions.mockResolvedValue([])
    mockGetRealtimeTrafficSummary.mockResolvedValue({
      enabled: true,
      summary: {
        window: '1min',
        start_time: '2026-04-07T00:00:00Z',
        end_time: '2026-04-07T00:01:00Z',
        platform: 'openai',
        group_id: 7,
        channel_id: 9,
        qps: { current: 0, peak: 0, avg: 0 },
        tps: { current: 0, peak: 0, avg: 0 },
      },
    })
  })

  it('includes channelId when loading realtime traffic summary', async () => {
    mount(OpsDashboardHeader, {
      props: {
        overview: null,
        platform: 'openai',
        groupId: 7,
        channelId: 9,
        timeRange: '1h',
        queryMode: 'auto',
        loading: false,
        lastUpdated: new Date('2026-04-07T00:01:00Z'),
      },
      global: {
        stubs: {
          Select: SelectStub,
          HelpTooltip: true,
          BaseDialog: true,
          Icon: true,
        },
      },
    })

    await flushPromises()

    expect(mockGetRealtimeTrafficSummary).toHaveBeenCalledWith('1min', 'openai', 7, 9)
  })
})
