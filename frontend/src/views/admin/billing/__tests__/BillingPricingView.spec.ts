import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { BillingPricingLayerForm } from '@/api/admin/billing'
import BillingPricingModeToggle from '@/components/admin/billing/BillingPricingModeToggle.vue'
import BillingPricingModelList from '@/components/admin/billing/BillingPricingModelList.vue'
import BillingPricingProviderGrid from '@/components/admin/billing/BillingPricingProviderGrid.vue'
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

function mountView() {
  return mount(BillingPricingView, {
    global: {
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

    wrapper.getComponent(BillingPricingModeToggle).vm.$emit('update:modelValue', 'grid')
    await flushPromises()
    wrapper.getComponent(BillingPricingProviderGrid).vm.$emit('toggle-provider', 'openai')
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

    wrapper.getComponent(BillingPricingModelList).vm.$emit('open', 'gpt-5.4')
    await flushPromises()

    expect(apiMocks.getBillingPricingDetails).toHaveBeenCalledWith(['gpt-5.4'])
  })

  it('sends simplified form payloads to the save-layer api', async () => {
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
    })
  })
})
