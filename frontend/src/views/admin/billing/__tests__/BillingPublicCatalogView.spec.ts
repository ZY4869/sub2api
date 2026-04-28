import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi, beforeEach } from 'vitest'
import BillingPublicCatalogView from '../BillingPublicCatalogView.vue'

const apiMocks = vi.hoisted(() => ({
  getBillingPublicCatalogDraft: vi.fn(),
  saveBillingPublicCatalogDraft: vi.fn(),
  publishBillingPublicCatalog: vi.fn(),
}))

const storeMocks = vi.hoisted(() => ({
  showError: vi.fn(),
  showSuccess: vi.fn(),
}))

vi.mock('@/api/admin/billing', () => ({
  getBillingPublicCatalogDraft: apiMocks.getBillingPublicCatalogDraft,
  saveBillingPublicCatalogDraft: apiMocks.saveBillingPublicCatalogDraft,
  publishBillingPublicCatalog: apiMocks.publishBillingPublicCatalog,
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: storeMocks.showError,
    showSuccess: storeMocks.showSuccess,
  }),
}))

function createCatalogItem(model: string, displayName: string) {
  return {
    model,
    display_name: displayName,
    provider: 'openai',
    provider_icon_key: 'openai',
    request_protocols: ['openai'],
    mode: 'chat',
    currency: 'USD',
    price_display: {
      primary: [
        { id: 'input_price', unit: 'input_token', value: 1e-6 },
        { id: 'output_price', unit: 'output_token', value: 2e-6 },
      ],
    },
    multiplier_summary: {
      enabled: false,
      kind: 'disabled',
    },
  }
}

describe('BillingPublicCatalogView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    apiMocks.getBillingPublicCatalogDraft.mockResolvedValue({
      draft: {
        selected_models: ['gpt-5.4'],
        page_size: 10,
        updated_at: '2026-04-20T10:00:00Z',
      },
      available_items: [
        createCatalogItem('gpt-5.4', 'GPT-5.4'),
        createCatalogItem('gpt-4.1-mini', 'GPT-4.1 Mini'),
      ],
      available_updated_at: '2026-04-20T10:30:00Z',
      available_source: 'persisted_snapshot',
      published: {
        etag: 'W/"published"',
        updated_at: '2026-04-19T09:00:00Z',
        page_size: 10,
        model_count: 1,
      },
    })
    apiMocks.saveBillingPublicCatalogDraft.mockResolvedValue({
      selected_models: ['gpt-5.4'],
      page_size: 10,
      updated_at: '2026-04-20T10:00:00Z',
    })
    apiMocks.publishBillingPublicCatalog.mockResolvedValue({
      etag: 'W/"published-next"',
      updated_at: '2026-04-20T11:00:00Z',
      page_size: 20,
      model_count: 1,
    })
  })

  it('loads without force by default and forces refresh when manually reloading', async () => {
    const wrapper = mount(BillingPublicCatalogView, {
      global: {
        stubs: {
          ModelIcon: true,
        },
      },
    })

    await flushPromises()

    expect(apiMocks.getBillingPublicCatalogDraft).toHaveBeenCalledTimes(1)
    expect(apiMocks.getBillingPublicCatalogDraft).toHaveBeenNthCalledWith(1, { force: false })

    const reloadButton = wrapper.findAll('button').find((node) => node.text().includes('同步当前可用模型'))
    expect(reloadButton).toBeTruthy()
    await reloadButton!.trigger('click')
    await flushPromises()

    expect(apiMocks.getBillingPublicCatalogDraft).toHaveBeenLastCalledWith({ force: true })
  })

  it('publishes the current draft payload so page_size changes are included in the new snapshot', async () => {
    const wrapper = mount(BillingPublicCatalogView, {
      global: {
        stubs: {
          ModelIcon: true,
        },
      },
    })

    await flushPromises()

    await wrapper.get('[data-testid="billing-public-catalog-page-size"]').setValue('20')
    await wrapper.get('[data-testid="billing-public-catalog-publish"]').trigger('click')
    await flushPromises()

    expect(apiMocks.publishBillingPublicCatalog).toHaveBeenCalledWith({
      selected_models: ['gpt-5.4'],
      page_size: 20,
      updated_at: '2026-04-20T10:00:00Z',
    })
    expect(storeMocks.showSuccess).toHaveBeenCalledWith('公开模型库已推送更新')
    expect(wrapper.text()).toContain('每页 20 条')
  })
})
