import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import TokenUsageTrend from '../TokenUsageTrend.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

vi.mock('@/composables/useTokenDisplayMode', () => ({
  useTokenDisplayMode: () => ({
    formatTokenDisplay: (value: number) => String(value),
  }),
}))

vi.mock('vue-chartjs', () => ({
  Line: {
    props: ['data'],
    template: '<div class="chart-data">{{ JSON.stringify(data) }}</div>',
  },
}))

describe('TokenUsageTrend', () => {
  it('adds cache hit rate dataset and tolerates missing cache fields', () => {
    const wrapper = mount(TokenUsageTrend, {
      props: {
        trendData: [
          {
            date: '2026-03-30',
            requests: 1,
            input_tokens: 100,
            output_tokens: 20,
            cache_creation_tokens: 20,
            cache_read_tokens: 80,
            total_tokens: 200,
            cost: 1,
            actual_cost: 1,
          },
          {
            date: '2026-03-31',
            requests: 1,
            input_tokens: 50,
            output_tokens: 10,
            total_tokens: 60,
            cost: 0.5,
            actual_cost: 0.5,
          } as any,
        ],
      },
      global: {
        stubs: {
          LoadingSpinner: true,
        },
      },
    })

    const chartData = JSON.parse(wrapper.find('.chart-data').text())
    const cacheHitRate = chartData.datasets.find((dataset: any) => dataset.label === 'Cache Hit Rate')

    expect(cacheHitRate).toBeTruthy()
    expect(cacheHitRate.data).toEqual([80, 0])
    expect(cacheHitRate.yAxisID).toBe('yPercent')

    const options = (wrapper.vm as any).$?.setupState.lineOptions
    expect(options.scales.yPercent.max).toBe(100)
    expect(
      options.plugins.tooltip.callbacks.label({
        dataset: { label: 'Cache Hit Rate', yAxisID: 'yPercent' },
        raw: 33.333,
      }),
    ).toBe('Cache Hit Rate: 33.3%')
  })
})
