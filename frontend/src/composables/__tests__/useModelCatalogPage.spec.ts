import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises } from '@vue/test-utils'

const {
  listModels,
  getModelDetail,
  updateOfficialPricingOverride,
  deleteOfficialPricingOverride,
  updatePricingOverride,
  deletePricingOverride,
  copyOfficialPricingToSale,
  showError,
  showSuccess
} = vi.hoisted(() => ({
  listModels: vi.fn(),
  getModelDetail: vi.fn(),
  updateOfficialPricingOverride: vi.fn(),
  deleteOfficialPricingOverride: vi.fn(),
  updatePricingOverride: vi.fn(),
  deletePricingOverride: vi.fn(),
  copyOfficialPricingToSale: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn()
}))

const modelInventoryStore = { revision: 0 }

vi.stubGlobal('localStorage', {
  getItem: vi.fn(() => '500'),
  setItem: vi.fn()
})

vi.mock('@/api/admin/models', () => ({
  modelsAPI: {
    listModels,
    getModelDetail,
    updateOfficialPricingOverride,
    deleteOfficialPricingOverride,
    updatePricingOverride,
    deletePricingOverride,
    copyOfficialPricingToSale
  }
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    showError,
    showSuccess
  }),
  useModelInventoryStore: () => modelInventoryStore
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key
  })
}))

import { useModelCatalogPage } from '../useModelCatalogPage'

describe('useModelCatalogPage', () => {
  beforeEach(() => {
    listModels.mockReset()
    getModelDetail.mockReset()
    updateOfficialPricingOverride.mockReset()
    deleteOfficialPricingOverride.mockReset()
    updatePricingOverride.mockReset()
    deletePricingOverride.mockReset()
    copyOfficialPricingToSale.mockReset()
    showError.mockReset()
    showSuccess.mockReset()
    modelInventoryStore.revision = 0
  })

  it('uses backend pagination and clamps the persisted page size to 100', async () => {
    listModels.mockResolvedValue({
      items: [],
      total: 240,
      page: 1,
      page_size: 100,
      pages: 3
    })

    const subject = useModelCatalogPage('official')

    expect(subject.pagination.page_size).toBe(100)

    await subject.loadModels()

    expect(listModels).toHaveBeenCalledWith({
      search: undefined,
      provider: undefined,
      mode: undefined,
      availability: 'available',
      pricing_source: undefined,
      page: 1,
      page_size: 100
    })

    listModels.mockResolvedValue({
      items: [],
      total: 240,
      page: 3,
      page_size: 100,
      pages: 3
    })
    subject.handlePageChange(3)
    await flushPromises()

    expect(listModels).toHaveBeenLastCalledWith({
      search: undefined,
      provider: undefined,
      mode: undefined,
      availability: 'available',
      pricing_source: undefined,
      page: 3,
      page_size: 100
    })

    listModels.mockResolvedValue({
      items: [],
      total: 240,
      page: 1,
      page_size: 50,
      pages: 5
    })
    subject.handlePageSizeChange(50)
    await flushPromises()

    expect(listModels).toHaveBeenLastCalledWith({
      search: undefined,
      provider: undefined,
      mode: undefined,
      availability: 'available',
      pricing_source: undefined,
      page: 1,
      page_size: 50
    })
    expect(subject.pagination.page_size).toBe(50)
  })
})
