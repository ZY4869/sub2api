import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { BillingPricingLayerForm } from '@/api/admin/billing'
import BillingPricingModeToggle from '@/components/admin/billing/BillingPricingModeToggle.vue'
import BillingPricingModelList from '@/components/admin/billing/BillingPricingModelList.vue'
import BillingPricingProviderGrid from '@/components/admin/billing/BillingPricingProviderGrid.vue'
import { useBillingPricingStore } from '@/stores'
import BillingPricingView from '../BillingPricingView.vue'

const apiMocks = vi.hoisted(() => ({
  listBillingPricingProviders: vi.fn(),
  listBillingPricingModels: vi.fn(),
  getBillingPricingDetails: vi.fn(),
  refreshBillingPricingCatalog: vi.fn(),
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
  refreshBillingPricingCatalog: apiMocks.refreshBillingPricingCatalog,
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
      supports_service_tier: false,
      supports_prompt_caching: true,
      supports_provider_special: true,
    },
    ...overrides,
  }
}

function createForm(overrides: Partial<BillingPricingLayerForm> = {}): BillingPricingLayerForm {
  return {
    input_price: 1,
    output_price: 2,
    cache_price: 0.1,
    special_enabled: false,
    special: {
      ...(overrides.special || {}),
    },
    tiered_enabled: false,
    ...overrides,
  }
}

function createDetail() {
  return {
    model: 'gpt-5.4',
    display_name: 'GPT-5.4',
    provider: 'openai',
    mode: 'chat',
    currency: 'USD',
    input_supported: true,
    output_charge_slot: 'text_output',
    supports_prompt_caching: true,
    supports_service_tier: false,
    long_context_input_token_threshold: 200000,
    long_context_input_cost_multiplier: 2,
    long_context_output_cost_multiplier: 2,
    capabilities: {
      supports_tiered_pricing: true,
      supports_batch_pricing: true,
      supports_service_tier: false,
      supports_prompt_caching: true,
      supports_provider_special: true,
    },
    official_form: createForm(),
    sale_form: createForm({
      input_price: 1.5,
      output_price: 2.5,
      cache_price: 0.2,
    }),
  }
}

function resetBillingPricingStore() {
  const store = useBillingPricingStore()
  store.invalidate()
  store.viewMode = 'list'
  store.search = ''
  store.providerFilter = ''
  store.modeFilter = ''
  store.sortBy = 'display_name'
  store.sortOrder = 'asc'
  store.page = 1
  store.pageSize = 20
  store.expandedProvider = ''
}

function mountView() {
  const pinia = createPinia()
  setActivePinia(pinia)
  resetBillingPricingStore()

  return mount(BillingPricingView, {
    global: {
      plugins: [pinia],
      stubs: {
        BillingPricingEditorDialog: {
          props: ['show', 'activeModel'],
          emits: ['save-layer', 'copy-official', 'apply-discount', 'close', 'update:activeModel'],
          template: `
            <div v-if="show" data-testid="editor-dialog-stub">
              <button
                data-testid="emit-save-layer"
                @click="$emit('save-layer', {
                  model: activeModel || 'gpt-5.4',
                  layer: 'official',
                  currency: 'CNY',
                  form: {
                    input_price: 1.25,
                    output_price: 2.5,
                    cache_price: 0.3,
                    special_enabled: false,
                    special: {},
                    tiered_enabled: false
                  }
                })"
              >
                save
              </button>
            </div>
          `,
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
    setActivePinia(createPinia())
    resetBillingPricingStore()

    apiMocks.listBillingPricingProviders.mockResolvedValue([
      { provider: 'openai', label: 'OpenAI', total_count: 1, official_count: 2, sale_count: 1 },
      { provider: 'anthropic', label: 'Anthropic', total_count: 2, official_count: 3, sale_count: 2 },
    ])
    apiMocks.listBillingPricingModels.mockImplementation(async (params: Record<string, unknown> = {}) => ({
      items: [createListItem(params.provider ? { model: `${params.provider}-model`, display_name: `${params.provider} model`, provider: params.provider } : {})],
      total: 1,
      page: Number(params.page || 1),
      page_size: Number(params.page_size || 20),
    }))
    apiMocks.getBillingPricingDetails.mockResolvedValue([createDetail()])
    apiMocks.refreshBillingPricingCatalog.mockResolvedValue({
      updated_at: '2026-04-16T00:00:00Z',
      total_models: 12,
      provider_count: 2,
    })
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
      sort_by: 'display_name',
      sort_order: 'asc',
    }))
  })

  it('changes sort mode and reloads the model list with provider sorting', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-testid="billing-pricing-sort"]').setValue('provider:desc')
    await flushPromises()

    expect(apiMocks.listBillingPricingModels).toHaveBeenLastCalledWith(expect.objectContaining({
      page: 1,
      page_size: 20,
      sort_by: 'provider',
      sort_order: 'desc',
    }))
  })

  it('filters list mode with provider quick cards and resets to page 1', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-testid="provider-quick-filter-openai"]').trigger('click')
    await flushPromises()

    expect(apiMocks.listBillingPricingModels).toHaveBeenLastCalledWith(expect.objectContaining({
      provider: 'openai',
      page: 1,
      page_size: 20,
    }))
  })

  it('persists page size changes and expands the selected provider in grid mode quick filters', async () => {
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

    wrapper.getComponent(BillingPricingModeToggle).vm.$emit('update:modelValue', 'grid')
    await flushPromises()
    await wrapper.get('[data-testid="provider-quick-filter-openai"]').trigger('click')
    await flushPromises()

    const grid = wrapper.getComponent(BillingPricingProviderGrid)
    expect(grid.props('expandedProvider')).toBe('openai')
    expect(apiMocks.listBillingPricingModels).toHaveBeenLastCalledWith(expect.objectContaining({
      provider: 'openai',
      page: 1,
      page_size: 100,
    }))
  })

  it('loads pricing details when opening the editor from list mode', async () => {
    const wrapper = mountView()
    await flushPromises()

    wrapper.getComponent(BillingPricingModelList).vm.$emit('open', 'gpt-5.4')
    await flushPromises()

    expect(apiMocks.getBillingPricingDetails).toHaveBeenCalledWith(['gpt-5.4'])
  })

  it('refreshes the persisted catalog and reloads the current filters', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-testid="provider-quick-filter-openai"]').trigger('click')
    await flushPromises()
    await wrapper.get('[data-testid="billing-pricing-sort"]').setValue('provider:desc')
    await flushPromises()
    await wrapper.get('[data-testid="billing-pricing-refresh"]').trigger('click')
    await flushPromises()

    expect(apiMocks.refreshBillingPricingCatalog).toHaveBeenCalledTimes(1)
    expect(apiMocks.listBillingPricingProviders).toHaveBeenCalledTimes(2)
    expect(apiMocks.listBillingPricingModels).toHaveBeenLastCalledWith(expect.objectContaining({
      provider: 'openai',
      sort_by: 'provider',
      sort_order: 'desc',
      page: 1,
      page_size: 20,
    }))
    expect(storeMocks.showSuccess).toHaveBeenCalledWith('模型列表已刷新，共 12 个模型')
  })

  it('keeps grid mode sorted by provider labels and loads provider models with the active sort', async () => {
    const wrapper = mountView()
    await flushPromises()

    wrapper.getComponent(BillingPricingModeToggle).vm.$emit('update:modelValue', 'grid')
    await flushPromises()
    await wrapper.get('[data-testid="billing-pricing-sort"]').setValue('provider:asc')
    await flushPromises()

    const grid = wrapper.getComponent(BillingPricingProviderGrid)
    expect((grid.props('providers') as Array<{ provider: string }>).map((item) => item.provider)).toEqual([
      'anthropic',
      'openai',
    ])

    grid.vm.$emit('toggle-provider', 'anthropic')
    await flushPromises()

    expect(apiMocks.listBillingPricingModels).toHaveBeenLastCalledWith(expect.objectContaining({
      provider: 'anthropic',
      sort_by: 'provider',
      sort_order: 'asc',
      page: 1,
      page_size: 100,
    }))
  })

  it('sends canonical form payloads and currency to the save-layer api', async () => {
    const wrapper = mountView()
    await flushPromises()

    wrapper.getComponent(BillingPricingModelList).vm.$emit('open', 'gpt-5.4')
    await flushPromises()
    await wrapper.get('[data-testid="emit-save-layer"]').trigger('click')
    await flushPromises()

    expect(apiMocks.updateBillingPricingLayer).toHaveBeenCalledWith('gpt-5.4', 'official', {
      form: {
        input_price: 1.25,
        output_price: 2.5,
        cache_price: 0.3,
        special_enabled: false,
        special: {},
        tiered_enabled: false,
      },
      currency: 'CNY',
    })
    expect(apiMocks.listBillingPricingProviders).toHaveBeenCalledTimes(2)
    expect(apiMocks.listBillingPricingModels).toHaveBeenCalledTimes(2)
  })
})
