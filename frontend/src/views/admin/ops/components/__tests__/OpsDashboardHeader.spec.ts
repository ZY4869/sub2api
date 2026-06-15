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

const SelectStubComponent = defineComponent({
  name: 'SelectStubComponent',
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
          Select: SelectStubComponent,
          HelpTooltip: true,
          BaseDialog: true,
          Icon: true,
        },
      },
    })

    await flushPromises()

    expect(mockGetRealtimeTrafficSummary).toHaveBeenCalledWith('1min', 'openai', 7, 9)
  })

  it('renders the health and realtime traffic sections without collapsing text', async () => {
    const wrapper = mount(OpsDashboardHeader, {
      props: {
        overview: {
          start_time: '2026-04-07T00:00:00Z',
          end_time: '2026-04-07T01:00:00Z',
          platform: 'openai',
          group_id: 7,
          channel_id: 9,
          health_score: 96,
          success_count: 119,
          error_count_total: 1,
          business_limited_count: 0,
          error_count_sla: 1,
          request_count_total: 120,
          request_count_sla: 120,
          token_consumed: 42000,
          sla: 0.991,
          error_rate: 0.008,
          upstream_error_rate: 0.004,
          upstream_error_count_excl_429_529: 1,
          upstream_429_count: 0,
          upstream_529_count: 0,
          qps: { current: 2.1, peak: 8.5, avg: 3.2 },
          tps: { current: 120.3, peak: 900.4, avg: 300.2 },
          duration: {
            p50_ms: 100,
            p90_ms: 200,
            p95_ms: 250,
            p99_ms: 400,
            avg_ms: 150,
            max_ms: 800,
          },
          ttft: {
            p50_ms: 80,
            p90_ms: 160,
            p95_ms: 180,
            p99_ms: 260,
            avg_ms: 120,
            max_ms: 400,
          },
        },
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
          Select: SelectStubComponent,
          HelpTooltip: true,
          BaseDialog: true,
          Icon: true,
        },
      },
    })

    await flushPromises()

    expect(wrapper.text()).toContain('96')
    expect(wrapper.text()).toContain('admin.ops.health')
    expect(wrapper.text()).toContain('admin.ops.realtime.title')
    expect(wrapper.text()).toContain('QPS')
    expect(wrapper.text()).toContain('admin.ops.tps')
    expect(wrapper.find('.xl\\:grid-cols-\\[180px_minmax\\(0\\,1fr\\)\\]').exists()).toBe(true)
  })
})
