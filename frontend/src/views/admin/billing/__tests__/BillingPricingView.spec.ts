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
  getBillingPricingAudit: vi.fn(),
  getBillingPricingDetails: vi.fn(),
  getBillingPricingDetailsWithPreview: vi.fn(),
  refreshBillingPricingCatalog: vi.fn(),
  updateBillingPricingLayer: vi.fn(),
  copyBillingPricingOfficialToSale: vi.fn(),
  applyBillingPricingDiscount: vi.fn(),
  getAllGroups: vi.fn(),
}))

const storeMocks = vi.hoisted(() => ({
  showError: vi.fn(),
  showSuccess: vi.fn(),
}))

vi.mock('@/api/admin/billing', () => ({
  listBillingPricingProviders: apiMocks.listBillingPricingProviders,
  listBillingPricingModels: apiMocks.listBillingPricingModels,
  getBillingPricingAudit: apiMocks.getBillingPricingAudit,
  getBillingPricingDetails: apiMocks.getBillingPricingDetails,
  getBillingPricingDetailsWithPreview: apiMocks.getBillingPricingDetailsWithPreview,
  refreshBillingPricingCatalog: apiMocks.refreshBillingPricingCatalog,
  updateBillingPricingLayer: apiMocks.updateBillingPricingLayer,
  copyBillingPricingOfficialToSale: apiMocks.copyBillingPricingOfficialToSale,
  applyBillingPricingDiscount: apiMocks.applyBillingPricingDiscount,
}))

vi.mock('@/api/admin/groups', () => ({
  getAll: apiMocks.getAllGroups,
  default: {
    getAll: apiMocks.getAllGroups,
  },
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
    pricing_status: 'ok',
    pricing_warnings: [],
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
    multiplier_enabled: false,
    item_multipliers: {},
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
    pricing_status: 'ok',
    pricing_warnings: [],
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
  store.pricingStatusFilter = ''
  store.groupPreviewId = null
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
                    tiered_enabled: false,
                    multiplier_enabled: false,
                    item_multipliers: {}
                  }
                })"
              >
                save
              </button>
            </div>
          `,
        },
        ModelIcon: true,
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
    apiMocks.getBillingPricingAudit.mockResolvedValue({
      total_models: 12,
      pricing_status_counts: {
        ok: 8,
        fallback: 1,
        conflict: 1,
        missing: 2,
      },
      duplicate_model_ids: [],
      aux_identifier_collisions: [{ source: 'aliases', identifier: 'gpt-5', models: ['gpt-5.4', 'gpt-5.4-mini'], count: 2 }],
      collision_counts_by_source: {
        aliases: 1,
        protocol_ids: 0,
        pricing_lookup_ids: 2,
      },
      provider_issue_counts: [
        { provider: 'openai', total: 2, fallback: 1, conflict: 1, missing: 0 },
        { provider: 'gemini', total: 1, fallback: 0, conflict: 0, missing: 1 },
      ],
      pricing_issue_examples: [
        {
          model: 'gpt-5.4-mini',
          display_name: 'GPT-5.4 Mini',
          provider: 'openai',
          pricing_status: 'conflict',
          first_warning: 'aliases identifier "gpt-5" collides with 2 models',
        },
        {
          model: 'gemini-3.1-flash-image',
          display_name: 'Gemini 3.1 Flash Image',
          provider: 'gemini',
          pricing_status: 'missing',
          first_warning: 'No stable upstream pricing source found.',
        },
      ],
      missing_in_snapshot_count: 1,
      missing_in_snapshot_models: ['gpt-5.4'],
      snapshot_only_count: 0,
      snapshot_only_models: [],
      refresh_required: true,
      snapshot_updated_at: '2026-04-16T00:00:00Z',
    })
    apiMocks.getBillingPricingDetails.mockResolvedValue([createDetail()])
    apiMocks.getBillingPricingDetailsWithPreview.mockResolvedValue([
      {
        ...createDetail(),
        preview_group_id: 9,
        preview_rate_multiplier: 1.3,
      },
    ])
    apiMocks.refreshBillingPricingCatalog.mockResolvedValue({
      updated_at: '2026-04-16T00:00:00Z',
      total_models: 12,
      provider_count: 2,
    })
    apiMocks.updateBillingPricingLayer.mockResolvedValue(createDetail())
    apiMocks.copyBillingPricingOfficialToSale.mockResolvedValue([createDetail()])
    apiMocks.applyBillingPricingDiscount.mockResolvedValue([createDetail()])
    apiMocks.getAllGroups.mockResolvedValue([
      { id: 9, name: '图像增强组' },
      { id: 11, name: '标准组' },
    ])
  })

  it('loads providers and paginated list mode data on mount', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(apiMocks.listBillingPricingProviders).toHaveBeenCalledTimes(1)
    expect(apiMocks.getAllGroups).toHaveBeenCalledTimes(1)
    expect(apiMocks.listBillingPricingModels).toHaveBeenCalledWith(expect.objectContaining({
      page: 1,
      page_size: 20,
      sort_by: 'display_name',
      sort_order: 'asc',
    }))
    expect(apiMocks.getBillingPricingAudit).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('计费审计')
    expect(wrapper.text()).toContain('状态分布')
    expect(wrapper.text()).toContain('供应商问题榜')
    expect(wrapper.text()).toContain('重点问题模型')
    expect(wrapper.text()).toContain('GPT-5.4 Mini')
    expect(wrapper.text()).toContain('1')
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

  it('filters list mode by pricing status and passes pricing_status to the list api', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-testid="billing-pricing-status"]').setValue('missing')
    await flushPromises()

    expect(apiMocks.listBillingPricingModels).toHaveBeenLastCalledWith(expect.objectContaining({
      pricing_status: 'missing',
      page: 1,
      page_size: 20,
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

  it('persists page size changes and opens provider worksets from grid cards', async () => {
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
    await wrapper.get('[data-testid="provider-grid-openai"]').trigger('click')
    await flushPromises()

    expect(apiMocks.listBillingPricingModels).toHaveBeenLastCalledWith(expect.objectContaining({
      provider: 'openai',
      page: 1,
      page_size: 100,
    }))
    expect(apiMocks.getBillingPricingDetails).toHaveBeenCalledWith(['openai-model'])
  })

  it('loads pricing details when opening the editor from list mode', async () => {
    const wrapper = mountView()
    await flushPromises()

    wrapper.getComponent(BillingPricingModelList).vm.$emit('open', 'gpt-5.4')
    await flushPromises()

    expect(apiMocks.getBillingPricingDetails).toHaveBeenCalledWith(['gpt-5.4'])
  })

  it('loads preview prices and preview details after selecting a group', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-testid="billing-pricing-group-preview"]').setValue('9')
    await flushPromises()

    expect(apiMocks.listBillingPricingModels).toHaveBeenLastCalledWith(expect.objectContaining({
      group_id: 9,
      page: 1,
      page_size: 20,
    }))

    wrapper.getComponent(BillingPricingModelList).vm.$emit('open', 'gpt-5.4')
    await flushPromises()

    expect(apiMocks.getBillingPricingDetailsWithPreview).toHaveBeenCalledWith({
      models: ['gpt-5.4'],
      group_id: 9,
    })
  })

  it('renders missing pricing badges and warnings in list mode', async () => {
    apiMocks.listBillingPricingModels.mockResolvedValueOnce({
      items: [
        createListItem({
          pricing_status: 'missing',
          pricing_warnings: ['No stable upstream pricing source found.'],
        }),
      ],
      total: 1,
      page: 1,
      page_size: 20,
    })

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('缺价')
    expect(wrapper.text()).toContain('No stable upstream pricing source found.')
  })

  it('imports a pricing patch file and updates only the patched fields', async () => {
    const wrapper = mountView()
    await flushPromises()

    const patch = {
      version: 1,
      kind: 'billing_pricing_patch',
      generated_at: '2026-04-26T00:00:00Z',
      models: [
        {
          model: 'gpt-5.4',
          current: {
            official: createForm(),
            sale: createForm(),
          },
          patch: {
            official: {
              input_price: 9,
            },
          },
          notes: '',
        },
      ],
    }

    const input = wrapper.get('input[type="file"]')
    const file = new File([JSON.stringify(patch)], 'patch.json', { type: 'application/json' })
    Object.defineProperty(input.element, 'files', { value: [file] })
    await input.trigger('change')
    for (let i = 0; i < 5 && apiMocks.updateBillingPricingLayer.mock.calls.length === 0; i += 1) {
      await flushPromises()
    }

    expect(apiMocks.updateBillingPricingLayer).toHaveBeenCalledTimes(1)
    expect(apiMocks.updateBillingPricingLayer).toHaveBeenCalledWith('gpt-5.4', 'official', expect.objectContaining({
      form: expect.objectContaining({
        input_price: 9,
        output_price: 2,
      }),
      currency: 'USD',
    }))
  })

  it('imports source currency from the pricing patch file', async () => {
    const wrapper = mountView()
    await flushPromises()

    const patch = {
      version: 1,
      kind: 'billing_pricing_patch',
      generated_at: '2026-04-27T00:00:00Z',
      models: [
        {
          model: 'gpt-5.4',
          currency: 'CNY',
          current: {
            official: createForm(),
            sale: createForm(),
          },
          patch: {
            official: {
              input_price: 0.3,
            },
          },
          notes: '',
        },
      ],
    }

    const input = wrapper.get('input[type="file"]')
    const file = new File([JSON.stringify(patch)], 'patch-cny.json', { type: 'application/json' })
    Object.defineProperty(input.element, 'files', { value: [file] })
    await input.trigger('change')
    for (let i = 0; i < 5 && apiMocks.updateBillingPricingLayer.mock.calls.length === 0; i += 1) {
      await flushPromises()
    }

    expect(apiMocks.updateBillingPricingLayer).toHaveBeenCalledWith('gpt-5.4', 'official', expect.objectContaining({
      currency: 'CNY',
      form: expect.objectContaining({
        input_price: 0.3,
      }),
    }))
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
    expect(apiMocks.getBillingPricingAudit).toHaveBeenCalledTimes(2)
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

    grid.vm.$emit('open-provider', 'anthropic')
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
        multiplier_enabled: false,
        item_multipliers: {},
      },
      currency: 'CNY',
      group_id: null,
    })
    expect(apiMocks.listBillingPricingProviders).toHaveBeenCalledTimes(2)
    expect(apiMocks.listBillingPricingModels).toHaveBeenCalledTimes(2)
  })
})
