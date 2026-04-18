import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import PublicModelCatalogContent from '../PublicModelCatalogContent.vue'

const apiMocks = vi.hoisted(() => ({
  getModelCatalog: vi.fn(),
  getUSDCNYExchangeRate: vi.fn(),
}))

vi.mock('@/api/meta', () => ({
  getModelCatalog: apiMocks.getModelCatalog,
  getUSDCNYExchangeRate: apiMocks.getUSDCNYExchangeRate,
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

describe('PublicModelCatalogContent', () => {
  beforeEach(() => {
    apiMocks.getModelCatalog.mockReset()
    apiMocks.getUSDCNYExchangeRate.mockReset()
    apiMocks.getModelCatalog.mockResolvedValue({
      notModified: false,
      etag: 'W/"catalog"',
      data: {
        etag: 'W/"catalog"',
        updated_at: '2026-04-18T00:00:00Z',
        items: [
          {
            model: 'gpt-5.4',
            display_name: 'GPT-5.4',
            provider: 'openai',
            provider_icon_key: 'openai',
            request_protocols: ['openai'],
            mode: 'chat',
            currency: 'USD',
            price_display: {
              primary: [{ id: 'input_price', unit: 'input_token', value: 0.000001 }],
            },
            multiplier_summary: {
              enabled: true,
              kind: 'uniform',
              mode: 'shared',
              value: 0.12,
            },
          },
          {
            model: 'claude-sonnet-4.5',
            display_name: 'Claude Sonnet 4.5',
            provider: 'anthropic',
            provider_icon_key: 'anthropic',
            request_protocols: ['anthropic'],
            mode: 'chat',
            currency: 'USD',
            price_display: {
              primary: [{ id: 'input_price', unit: 'input_token', value: 0.000002 }],
            },
            multiplier_summary: {
              enabled: false,
              kind: 'disabled',
            },
          },
        ],
      },
    })
    apiMocks.getUSDCNYExchangeRate.mockResolvedValue({
      base: 'USD',
      quote: 'CNY',
      rate: 7.2,
      date: '2026-04-18',
      updated_at: '2026-04-18T00:00:00Z',
      cached: true,
    })
  })

  it('loads the catalog and filters by provider, protocol, and multiplier', async () => {
    const wrapper = mount(PublicModelCatalogContent, {
      global: {
        stubs: {
          ModelIcon: { template: '<span data-test="model-icon" />' },
          ModelPlatformIcon: { template: '<span data-test="provider-icon" />' },
        },
      },
    })

    await flushPromises()

    expect(wrapper.text()).toContain('GPT-5.4')
    expect(wrapper.text()).toContain('Claude Sonnet 4.5')

    await wrapper.get('[data-testid="models-filter-provider-openai"]').trigger('click')
    expect(wrapper.text()).toContain('GPT-5.4')
    expect(wrapper.text()).not.toContain('Claude Sonnet 4.5')

    await wrapper.get('[data-testid="models-filter-provider-all"]').trigger('click')
    await wrapper.get('[data-testid="models-filter-protocol-anthropic"]').trigger('click')
    expect(wrapper.text()).toContain('Claude Sonnet 4.5')
    expect(wrapper.text()).not.toContain('GPT-5.4')

    await wrapper.get('[data-testid="models-filter-protocol-all"]').trigger('click')
    await wrapper.get('[data-testid="models-filter-multiplier-uniform:0.12"]').trigger('click')
    expect(wrapper.text()).toContain('GPT-5.4')
    expect(wrapper.text()).not.toContain('Claude Sonnet 4.5')
  })
})
