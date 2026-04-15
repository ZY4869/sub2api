import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import BillingPricingView from '../BillingPricingView.vue'

const apiMocks = vi.hoisted(() => ({
  listBillingPricingProviders: vi.fn(),
  listBillingPricingModels: vi.fn(),
  getBillingPricingDetails: vi.fn(),
  updateBillingPricingLayer: vi.fn(),
  copyBillingPricingOfficialToSale: vi.fn(),
  applyBillingPricingDiscount: vi.fn(),
}))

const storeMocks = vi.hoisted(() => ({
  showError: vi.fn(),
  showSuccess: vi.fn(),
}))

vi.mock('@/api/admin/billing', () => ({
  listBillingPricingProviders: apiMocks.listBillingPricingProviders,
  listBillingPricingModels: apiMocks.listBillingPricingModels,
  getBillingPricingDetails: apiMocks.getBillingPricingDetails,
  updateBillingPricingLayer: apiMocks.updateBillingPricingLayer,
  copyBillingPricingOfficialToSale: apiMocks.copyBillingPricingOfficialToSale,
  applyBillingPricingDiscount: apiMocks.applyBillingPricingDiscount,
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: storeMocks.showError,
    showSuccess: storeMocks.showSuccess,
  }),
}))

function createListItem(overrides: Record<string, unknown> = {}) {
  return {
    model: 'gpt-5.4',
    display_name: 'GPT-5.4',
    provider: 'openai',
    mode: 'chat',
    price_item_count: 4,
    official_count: 2,
    sale_count: 1,
    capabilities: {
      supports_tiered_pricing: true,
      supports_batch_pricing: true,
      supports_service_tier: true,
      supports_prompt_caching: false,
      supports_provider_special: true,
    },
    ...overrides,
  }
}

function createDetail() {
  return {
    model: 'gpt-5.4',
    display_name: 'GPT-5.4',
    provider: 'openai',
    mode: 'chat',
    supports_prompt_caching: false,
    supports_service_tier: true,
    long_context_input_token_threshold: 200000,
    long_context_input_cost_multiplier: 2,
    long_context_output_cost_multiplier: 2,
    capabilities: {
      supports_tiered_pricing: true,
      supports_batch_pricing: true,
      supports_service_tier: true,
      supports_prompt_caching: false,
      supports_provider_special: true,
    },
    official_items: [],
    sale_items: [],
  }
}

function mountView() {
  return mount(BillingPricingView, {
    global: {
      stubs: {
        BillingPricingEditorDialog: {
          template: '<div data-testid="editor-dialog-stub" />',
        },
        ModelPlatformIcon: true,
      },
    },
  })
}

describe('BillingPricingView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()

    apiMocks.listBillingPricingProviders.mockResolvedValue([
      { provider: 'openai', label: 'OpenAI', total_count: 1, official_count: 2, sale_count: 1 },
    ])
    apiMocks.listBillingPricingModels.mockImplementation(async (params: Record<string, unknown> = {}) => ({
      items: [createListItem(params.provider ? { model: 'gpt-5.4-mini', display_name: 'GPT-5.4 Mini' } : {})],
      total: 1,
      page: Number(params.page || 1),
      page_size: Number(params.page_size || 20),
    }))
    apiMocks.getBillingPricingDetails.mockResolvedValue([createDetail()])
    apiMocks.updateBillingPricingLayer.mockResolvedValue(createDetail())
    apiMocks.copyBillingPricingOfficialToSale.mockResolvedValue([createDetail()])
    apiMocks.applyBillingPricingDiscount.mockResolvedValue([createDetail()])
  })

  it('loads providers and paginated list mode data on mount', async () => {
    mountView()
    await flushPromises()

    expect(apiMocks.listBillingPricingProviders).toHaveBeenCalledTimes(1)
    expect(apiMocks.listBillingPricingModels).toHaveBeenCalledWith(expect.objectContaining({
      page: 1,
      page_size: 20,
    }))
  })

  it('persists page size changes and loads provider grid data on demand', async () => {
    const wrapper = mountView()
    await flushPromises()

    const pageSizeSelect = wrapper.findAll('select').find((node) => node.find('option[value="100"]').exists())
    expect(pageSizeSelect).toBeTruthy()

    await pageSizeSelect!.setValue('50')
    await flushPromises()

    expect(localStorage.getItem('admin.billing.pricing.page_size')).toBe('50')
    expect(apiMocks.listBillingPricingModels).toHaveBeenLastCalledWith(expect.objectContaining({
      page: 1,
      page_size: 50,
    }))

    const gridButton = wrapper.findAll('button').find((button) => button.text() === '九宫格模式')
    expect(gridButton).toBeTruthy()
    await gridButton!.trigger('click')
    await flushPromises()

    const providerButton = wrapper.findAll('button').find((button) => button.text().includes('OpenAI'))
    expect(providerButton).toBeTruthy()
    await providerButton!.trigger('click')
    await flushPromises()

    expect(apiMocks.listBillingPricingModels).toHaveBeenLastCalledWith(expect.objectContaining({
      provider: 'openai',
      page: 1,
      page_size: 100,
    }))
  })

  it('loads pricing details when opening the editor from list mode', async () => {
    const wrapper = mountView()
    await flushPromises()

    const openButton = wrapper.findAll('button').find((button) => button.text() === '编辑定价')
    expect(openButton).toBeTruthy()
    await openButton!.trigger('click')
    await flushPromises()

    expect(apiMocks.getBillingPricingDetails).toHaveBeenCalledWith(['gpt-5.4'])
  })
})
