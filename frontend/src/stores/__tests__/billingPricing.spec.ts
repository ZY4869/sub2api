import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useBillingPricingStore } from '../billingPricing'

const apiMocks = vi.hoisted(() => ({
  listBillingPricingProviders: vi.fn(),
  listBillingPricingModels: vi.fn(),
  refreshBillingPricingCatalog: vi.fn(),
}))

vi.mock('@/api/admin/billing', () => ({
  listBillingPricingProviders: apiMocks.listBillingPricingProviders,
  listBillingPricingModels: apiMocks.listBillingPricingModels,
  refreshBillingPricingCatalog: apiMocks.refreshBillingPricingCatalog,
}))

function resetStore() {
  const store = useBillingPricingStore()
  store.invalidate()
  store.viewMode = 'list'
  store.search = ''
  store.providerFilter = ''
  store.modeFilter = ''
  store.pricingStatusFilter = ''
  store.sortBy = 'display_name'
  store.sortOrder = 'asc'
  store.page = 1
  store.pageSize = 20
  store.expandedProvider = ''
  return store
}

describe('billingPricing store', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
    setActivePinia(createPinia())
    resetStore()

    apiMocks.listBillingPricingProviders.mockResolvedValue([
      { provider: 'openai', label: 'OpenAI', total_count: 1, official_count: 2, sale_count: 1 },
    ])
    apiMocks.listBillingPricingModels.mockResolvedValue({
      items: [
        {
          model: 'gpt-5.4',
          display_name: 'GPT-5.4',
          provider: 'openai',
          mode: 'chat',
          price_item_count: 3,
          official_count: 2,
          sale_count: 1,
          capabilities: {
            supports_tiered_pricing: true,
            supports_batch_pricing: true,
            supports_service_tier: false,
            supports_prompt_caching: true,
            supports_provider_special: true,
          },
        },
      ],
      total: 1,
      page: 1,
      page_size: 20,
    })
    apiMocks.refreshBillingPricingCatalog.mockResolvedValue({
      updated_at: '2026-04-16T00:00:00Z',
      total_models: 1,
      provider_count: 1,
    })
  })

  it('reuses cached providers and list responses until invalidated', async () => {
    const store = resetStore()

    await store.loadProviders()
    await store.loadProviders()
    await store.loadModels()
    await store.loadModels()

    expect(apiMocks.listBillingPricingProviders).toHaveBeenCalledTimes(1)
    expect(apiMocks.listBillingPricingModels).toHaveBeenCalledTimes(1)

    store.invalidate()

    await store.loadProviders()
    await store.loadModels()

    expect(apiMocks.listBillingPricingProviders).toHaveBeenCalledTimes(2)
    expect(apiMocks.listBillingPricingModels).toHaveBeenCalledTimes(2)
  })

  it('separates provider model cache by current sort scope', async () => {
    const store = resetStore()

    await store.loadProviderModels('openai')
    await store.loadProviderModels('openai')

    expect(apiMocks.listBillingPricingModels).toHaveBeenCalledTimes(1)
    expect(apiMocks.listBillingPricingModels).toHaveBeenLastCalledWith(expect.objectContaining({
      provider: 'openai',
      sort_by: 'display_name',
      sort_order: 'asc',
      page: 1,
      page_size: 100,
    }))

    store.sortBy = 'provider'
    store.sortOrder = 'desc'

    await store.loadProviderModels('openai')

    expect(apiMocks.listBillingPricingModels).toHaveBeenCalledTimes(2)
    expect(apiMocks.listBillingPricingModels).toHaveBeenLastCalledWith(expect.objectContaining({
      provider: 'openai',
      sort_by: 'provider',
      sort_order: 'desc',
      page: 1,
      page_size: 100,
    }))
  })

  it('invalidates cached providers and models after manual refresh', async () => {
    const store = resetStore()

    await store.loadProviders()
    await store.loadModels()
    await store.refreshCatalog()
    await store.loadProviders()
    await store.loadModels()

    expect(apiMocks.refreshBillingPricingCatalog).toHaveBeenCalledTimes(1)
    expect(apiMocks.listBillingPricingProviders).toHaveBeenCalledTimes(2)
    expect(apiMocks.listBillingPricingModels).toHaveBeenCalledTimes(2)
  })
})
