import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { BillingPricingLayerForm } from '@/api/admin/billing'
import BillingPricingModeToggle from '@/components/admin/billing/BillingPricingModeToggle.vue'
import BillingPricingModelList from '@/components/admin/billing/BillingPricingModelList.vue'
import BillingPricingProviderGrid from '@/components/admin/billing/BillingPricingProviderGrid.vue'
import { useBillingPricingStore } from '@/stores'
import BillingPricingView from '../BillingPricingView.vue'
import { materializeBillingPricingPatchFileV1 } from '@/utils/billingPricingPatch'

const apiMocks = vi.hoisted(() => ({
  listBillingPricingProviders: vi.fn(),
  listBillingPricingModels: vi.fn(),
  getBillingPricingAudit: vi.fn(),
  getBillingPricingDetails: vi.fn(),
  refreshBillingPricingCatalog: vi.fn(),
  updateBillingPricingLayer: vi.fn(),
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
  refreshBillingPricingCatalog: apiMocks.refreshBillingPricingCatalog,
  updateBillingPricingLayer: apiMocks.updateBillingPricingLayer,
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
          emits: ['save-layer', 'close', 'update:activeModel'],
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
    apiMocks.refreshBillingPricingCatalog.mockResolvedValue({
      updated_at: '2026-04-16T00:00:00Z',
      total_models: 12,
      provider_count: 2,
    })
    apiMocks.updateBillingPricingLayer.mockResolvedValue(createDetail())
  })

  it('presents pricing as official cost management instead of a sale entry point', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('官方成本/价格源管理')
    expect(wrapper.text()).toContain('对外出售价格请在“对外模型展示”页按公开条目设定')
    expect(wrapper.text()).not.toContain('在同一处查看并编辑真实价格与出售价格')
  })

  it('loads providers and paginated list mode data on mount', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(apiMocks.listBillingPricingProviders).toHaveBeenCalledTimes(1)
    expect(apiMocks.listBillingPricingModels).toHaveBeenCalledWith(expect.objectContaining({
      page: 1,
      page_size: 20,
      sort_by: 'display_name',
      sort_order: 'asc',
    }))
    expect(apiMocks.getBillingPricingAudit).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('计费审计')
    expect(wrapper.text()).toContain('状态分布')
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

  it('does not show sale counts in provider grid cards', async () => {
    const wrapper = mountView()
    await flushPromises()

    wrapper.getComponent(BillingPricingModeToggle).vm.$emit('update:modelValue', 'grid')
    await flushPromises()

    expect(wrapper.text()).toContain('官方 2')
    expect(wrapper.text()).not.toContain('出售 1')
  })

  it('loads pricing details when opening the editor from list mode', async () => {
    const wrapper = mountView()
    await flushPromises()

    wrapper.getComponent(BillingPricingModelList).vm.$emit('open', 'gpt-5.4')
    await flushPromises()

    expect(apiMocks.getBillingPricingDetails).toHaveBeenCalledWith(['gpt-5.4'])
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

  it('materializes empty worklist patches before import and downloads the confirmed json', async () => {
    const createObjectURL = vi.fn(() => 'blob:pricing-confirmed')
    const revokeObjectURL = vi.fn()
    const originalURL = window.URL
    const appendChildSpy = vi.spyOn(document.body, 'appendChild')
    const removeSpy = vi.spyOn(HTMLAnchorElement.prototype, 'remove').mockImplementation(() => {})
    const clickSpy = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})
    // @ts-expect-error test override
    window.URL = {
      ...originalURL,
      createObjectURL,
      revokeObjectURL,
    }

    try {
      const wrapper = mountView()
      await flushPromises()

      const worklist = {
        version: 1,
        kind: 'billing_pricing_patch',
        generated_at: '2026-05-06T12:37:48Z',
        export_mode: 'issue_worklist',
        models: [
          {
            model: 'gpt-5.4',
            currency: 'USD',
            current: {
              official: createForm({
                input_price: 1.5e-6,
                output_price: undefined,
                cache_price: undefined,
              }),
              sale: createForm({
                input_price: undefined,
                output_price: 2.5e-6,
                cache_price: undefined,
              }),
            },
            patch: {},
            notes: '',
          },
          {
            model: 'missing-model',
            currency: 'CNY',
            current: {
              official: createForm({
                input_price: undefined,
                output_price: undefined,
                cache_price: undefined,
              }),
              sale: createForm({
                input_price: undefined,
                output_price: undefined,
                cache_price: undefined,
              }),
            },
            patch: {},
            notes: '',
          },
        ],
      }

      const input = wrapper.get('input[type="file"]')
      const file = new File([JSON.stringify(worklist)], 'issue-worklist.json', { type: 'application/json' })
      Object.defineProperty(input.element, 'files', { value: [file] })
      await input.trigger('change')
      for (let i = 0; i < 5 && apiMocks.updateBillingPricingLayer.mock.calls.length === 0; i += 1) {
        await flushPromises()
      }

      expect(apiMocks.updateBillingPricingLayer).toHaveBeenCalledTimes(1)
      expect(apiMocks.updateBillingPricingLayer).toHaveBeenNthCalledWith(1, 'gpt-5.4', 'official', expect.objectContaining({
        currency: 'USD',
        form: expect.objectContaining({
          input_price: 1.5e-6,
        }),
      }))
      expect(clickSpy).toHaveBeenCalledTimes(1)
      expect(createObjectURL).toHaveBeenCalledTimes(1)
      expect(revokeObjectURL).toHaveBeenCalledTimes(1)
      expect(storeMocks.showSuccess).toHaveBeenCalledWith('批量导入完成', expect.objectContaining({
        title: '导入结果',
        details: expect.arrayContaining([
          '更新层数：1',
          '自动补全：1',
          '跳过模型：1',
          '已忽略出售价格补丁：1，请在“对外模型展示”页设置',
        ]),
      }))
      expect(appendChildSpy).toHaveBeenCalled()
    } finally {
      window.URL = originalURL
      appendChildSpy.mockRestore()
      removeSpy.mockRestore()
      clickSpy.mockRestore()
    }
  })

  it('ignores sale pricing patch fields and keeps sale edits on the public catalog page', async () => {
    apiMocks.getBillingPricingDetails.mockResolvedValueOnce([{
      ...createDetail(),
      sale_form: createForm({
        input_price: 1.5,
        output_price: 2.5,
        special_enabled: true,
        special: {
          grounding_search: 0.01,
        },
        multiplier_enabled: true,
        multiplier_mode: 'item',
        item_multipliers: {
          input_price: 0.9,
          output_price: 0.8,
        },
      }),
    }])

    const createObjectURL = vi.fn(() => 'blob:pricing-confirmed')
    const revokeObjectURL = vi.fn()
    const originalURL = window.URL
    const removeSpy = vi.spyOn(HTMLAnchorElement.prototype, 'remove').mockImplementation(() => {})
    const clickSpy = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})
    // @ts-expect-error test override
    window.URL = {
      ...originalURL,
      createObjectURL,
      revokeObjectURL,
    }

    try {
      const wrapper = mountView()
      await flushPromises()

      const patch = {
        version: 1,
        kind: 'billing_pricing_patch',
        generated_at: '2026-05-05T00:00:00Z',
        models: [
          {
            model: 'gpt-5.4',
            current: {
              official: createForm(),
              sale: createForm(),
            },
            patch: {
              sale: {
                special_enabled: true,
                special: {
                  grounding_search: 0.03,
                },
                multiplier_enabled: true,
                multiplier_mode: 'item',
                item_multipliers: {
                  input_price: null,
                },
              },
            },
            notes: '',
          },
        ],
      }

      const input = wrapper.get('input[type="file"]')
      const file = new File([JSON.stringify(patch)], 'patch-nested.json', { type: 'application/json' })
      Object.defineProperty(input.element, 'files', { value: [file] })
      await input.trigger('change')
      for (let i = 0; i < 5 && storeMocks.showSuccess.mock.calls.length === 0; i += 1) {
        await flushPromises()
      }

      expect(apiMocks.updateBillingPricingLayer).not.toHaveBeenCalled()
      expect(storeMocks.showSuccess).toHaveBeenCalledWith('批量导入完成', expect.objectContaining({
        details: expect.arrayContaining([
          '已忽略出售价格补丁：1，请在“对外模型展示”页设置',
        ]),
      }))
      expect(clickSpy).toHaveBeenCalledTimes(1)
      expect(createObjectURL).toHaveBeenCalledTimes(1)
    } finally {
      window.URL = originalURL
      removeSpy.mockRestore()
      clickSpy.mockRestore()
    }
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

  it('materializes empty issue worklist patches from current prices and skips models without price data', () => {
    const result = materializeBillingPricingPatchFileV1({
      version: 1,
      kind: 'billing_pricing_patch',
      generated_at: '2026-05-06T12:37:48Z',
      export_mode: 'issue_worklist',
      models: [
        {
          model: 'ernie-3.5-8k',
          currency: 'CNY',
          current: {
            official: createForm({
              input_price: 0.0000008,
              output_price: 0.000002,
              cache_price: undefined,
            }),
            sale: createForm({
              input_price: undefined,
              output_price: undefined,
              cache_price: undefined,
            }),
          },
          patch: {},
          notes: '',
        } as any,
        {
          model: 'missing-model',
          currency: 'USD',
          current: {
            official: createForm({
              input_price: undefined,
              output_price: undefined,
              cache_price: undefined,
            }),
            sale: createForm({
              input_price: undefined,
              output_price: undefined,
              cache_price: undefined,
            }),
          },
          patch: {},
          notes: '',
        } as any,
      ],
    })

    expect(result.updated).toBe(1)
    expect(result.skipped).toBe(1)
    expect(result.file.export_mode).toBe('executable_template')
    expect(result.file.models).toHaveLength(1)
    expect(result.file.models[0]?.patch.official).toEqual({
      input_price: 0.0000008,
      output_price: 0.000002,
    })
    expect(result.file.models[0]?.currency).toBe('CNY')
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
    })
    expect(apiMocks.listBillingPricingProviders).toHaveBeenCalledTimes(2)
    expect(apiMocks.listBillingPricingModels).toHaveBeenCalledTimes(2)
  })

  it('maps server metadata field errors back into the editor instead of only showing a generic toast', async () => {
    apiMocks.updateBillingPricingLayer.mockRejectedValueOnce({
      message: 'pricing layer form contains invalid fields',
      reason: 'BILLING_PRICE_INVALID',
      metadata: {
        'field_errors.tier_threshold_tokens': '共享阈值必须是正整数',
        'field_errors.input_price_above_threshold': '至少填写一个阈值后价格',
      },
    })

    const wrapper = mount(BillingPricingView, {
      global: {
        plugins: [createPinia()],
        stubs: {
          BillingPricingEditorDialog: {
            props: ['show', 'activeModel', 'serverErrors'],
            emits: ['save-layer', 'close', 'update:activeModel'],
            template: `
              <div v-if="show" data-testid="editor-dialog-stub">
                <div data-testid="server-error-threshold">{{ serverErrors?.official?.tier_threshold_tokens || '' }}</div>
                <button
                  data-testid="emit-save-layer"
                  @click="$emit('save-layer', {
                    model: activeModel || 'gpt-5.4',
                    layer: 'official',
                    currency: 'USD',
                    form: {
                      input_price: 1.25,
                      output_price: 2.5,
                      special_enabled: false,
                      special: {},
                      tiered_enabled: true,
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
    await flushPromises()

    wrapper.getComponent(BillingPricingModelList).vm.$emit('open', 'gpt-5.4')
    await flushPromises()
    await wrapper.get('[data-testid="emit-save-layer"]').trigger('click')
    await flushPromises()

    expect(wrapper.get('[data-testid="server-error-threshold"]').text()).toContain('共享阈值必须是正整数')
    expect(storeMocks.showError).toHaveBeenCalledWith(
      '官方价格保存失败，请先修正标记字段。',
      expect.objectContaining({
        title: '官方价格校验失败',
        details: expect.arrayContaining(['共享阈值必须是正整数', '至少填写一个阈值后价格']),
      }),
    )
  })

  it('exports executable pricing patch templates', async () => {
    apiMocks.getBillingPricingDetails.mockResolvedValueOnce([
      createDetail(),
      {
        ...createDetail(),
        model: 'gpt-5.4-mini',
        display_name: 'GPT-5.4 Mini',
      },
      {
        ...createDetail(),
        model: 'gemini-3.1-flash-image',
        display_name: 'Gemini 3.1 Flash Image',
        provider: 'gemini',
      },
    ])

    const createObjectURL = vi.fn(() => 'blob:pricing-template')
    const revokeObjectURL = vi.fn()
    const originalURL = window.URL
    const appendChildSpy = vi.spyOn(document.body, 'appendChild')
    const removeSpy = vi.spyOn(HTMLAnchorElement.prototype, 'remove').mockImplementation(() => {})
    const clickSpy = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})
    // @ts-expect-error test override
    window.URL = {
      ...originalURL,
      createObjectURL,
      revokeObjectURL,
    }

    try {
      const wrapper = mountView()
      await flushPromises()

      await wrapper.get('[data-testid="billing-pricing-export-template"]').trigger('click')
      await flushPromises()

      expect(createObjectURL).toHaveBeenCalledTimes(1)
      expect(clickSpy).toHaveBeenCalledTimes(1)
      expect(revokeObjectURL).toHaveBeenCalledTimes(1)
      expect(storeMocks.showSuccess).toHaveBeenCalledWith('已导出 3 个模型的可执行模板')
      expect(appendChildSpy).toHaveBeenCalled()
    } finally {
      window.URL = originalURL
      appendChildSpy.mockRestore()
      removeSpy.mockRestore()
      clickSpy.mockRestore()
    }
  })
})
