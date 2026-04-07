import { describe, expect, it, beforeEach, afterEach, vi } from 'vitest'
import { defineComponent, reactive } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'

import OpsDashboard from '../OpsDashboard.vue'

const route = reactive<{ query: Record<string, any> }>({
  query: {},
})

const routerReplace = vi.fn(async (target: any) => {
  if (target?.query) {
    route.query = { ...target.query }
  }
})

const mockGetDashboardSnapshotV2 = vi.fn()
const mockGetThroughputTrend = vi.fn()
const mockGetLatencyHistogram = vi.fn()
const mockGetErrorDistribution = vi.fn()
const mockGetMetricThresholds = vi.fn()
const mockGetAdvancedSettings = vi.fn()
const mockFetchAdminSettings = vi.fn()

vi.mock('vue-router', () => ({
  useRoute: () => route,
  useRouter: () => ({
    replace: routerReplace,
  }),
}))

vi.mock('@/api/admin/ops', () => ({
  opsAPI: {
    getDashboardSnapshotV2: (...args: any[]) => mockGetDashboardSnapshotV2(...args),
    getThroughputTrend: (...args: any[]) => mockGetThroughputTrend(...args),
    getLatencyHistogram: (...args: any[]) => mockGetLatencyHistogram(...args),
    getErrorDistribution: (...args: any[]) => mockGetErrorDistribution(...args),
    getMetricThresholds: (...args: any[]) => mockGetMetricThresholds(...args),
    getAdvancedSettings: (...args: any[]) => mockGetAdvancedSettings(...args),
  },
  default: {
    getDashboardSnapshotV2: (...args: any[]) => mockGetDashboardSnapshotV2(...args),
    getThroughputTrend: (...args: any[]) => mockGetThroughputTrend(...args),
    getLatencyHistogram: (...args: any[]) => mockGetLatencyHistogram(...args),
    getErrorDistribution: (...args: any[]) => mockGetErrorDistribution(...args),
    getMetricThresholds: (...args: any[]) => mockGetMetricThresholds(...args),
    getAdvancedSettings: (...args: any[]) => mockGetAdvancedSettings(...args),
  },
}))

vi.mock('@/stores', () => ({
  useAdminSettingsStore: () => ({
    opsMonitoringEnabled: true,
    opsQueryModeDefault: 'auto',
    fetch: mockFetchAdminSettings,
  }),
  useAppStore: () => ({
    showError: vi.fn(),
    showWarning: vi.fn(),
    showSuccess: vi.fn(),
    showInfo: vi.fn(),
  }),
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

const OpsDashboardHeaderStub = defineComponent({
  name: 'OpsDashboardHeader',
  emits: ['update:channel'],
  template: '<button data-test="change-channel" @click="$emit(\'update:channel\', 11)">change channel</button>',
})

const AppLayoutStub = defineComponent({
  name: 'AppLayout',
  template: '<div><slot /></div>',
})

describe('OpsDashboard', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    route.query = { channel_id: '9' }
    routerReplace.mockClear()
    mockFetchAdminSettings.mockResolvedValue(undefined)
    mockGetMetricThresholds.mockResolvedValue(null)
    mockGetAdvancedSettings.mockResolvedValue({
      display_alert_events: true,
      display_openai_token_stats: true,
      auto_refresh_enabled: false,
      auto_refresh_interval_seconds: 30,
    })
    mockGetDashboardSnapshotV2.mockResolvedValue({
      overview: {
        start_time: '2026-04-07T00:00:00Z',
        end_time: '2026-04-07T01:00:00Z',
        platform: '',
        channel_id: 9,
        success_count: 0,
        error_count_total: 0,
        business_limited_count: 0,
        error_count_sla: 0,
        request_count_total: 0,
        request_count_sla: 0,
        token_consumed: 0,
        sla: 1,
        error_rate: 0,
        upstream_error_rate: 0,
        upstream_error_count_excl_429_529: 0,
        upstream_429_count: 0,
        upstream_529_count: 0,
        qps: { current: 0, peak: 0, avg: 0 },
        tps: { current: 0, peak: 0, avg: 0 },
        duration: {},
        ttft: {},
      },
      throughput_trend: {
        bucket: 'minute',
        points: [],
        by_platform: [],
        top_groups: [],
      },
      error_trend: {
        bucket: 'minute',
        points: [],
      },
    })
    mockGetThroughputTrend.mockResolvedValue({
      bucket: 'minute',
      points: [],
      by_platform: [],
      top_groups: [],
    })
    mockGetLatencyHistogram.mockResolvedValue({
      start_time: '2026-04-07T00:00:00Z',
      end_time: '2026-04-07T01:00:00Z',
      platform: '',
      channel_id: 9,
      total_requests: 0,
      buckets: [],
    })
    mockGetErrorDistribution.mockResolvedValue({
      total: 0,
      items: [],
    })
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('reads channel_id from route query and syncs updates back to the URL', async () => {
    const wrapper = mount(OpsDashboard, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
          BaseDialog: true,
          OpsDashboardHeader: OpsDashboardHeaderStub,
          OpsDashboardSkeleton: true,
          OpsConcurrencyCard: true,
          OpsErrorDetailModal: true,
          OpsErrorDistributionChart: true,
          OpsErrorDetailsModal: true,
          OpsErrorTrendChart: true,
          OpsLatencyChart: true,
          OpsThroughputTrendChart: true,
          OpsSwitchRateTrendChart: true,
          OpsAlertEventsCard: true,
          OpsOpenAITokenStatsCard: true,
          OpsSystemLogTable: true,
          OpsRequestDetailsModal: true,
          OpsSettingsDialog: true,
          OpsAlertRulesCard: true,
        },
      },
    })

    await flushPromises()

    expect(mockGetDashboardSnapshotV2).toHaveBeenCalledWith(
      expect.objectContaining({ channel_id: 9 }),
      expect.any(Object)
    )

    await wrapper.get('[data-test="change-channel"]').trigger('click')
    await flushPromises()
    vi.advanceTimersByTime(300)
    await flushPromises()

    expect(mockGetDashboardSnapshotV2).toHaveBeenCalledWith(
      expect.objectContaining({ channel_id: 11 }),
      expect.any(Object)
    )
    expect(routerReplace).toHaveBeenCalledWith(
      expect.objectContaining({
        query: expect.objectContaining({
          channel_id: '11',
        }),
      })
    )
  })
})
