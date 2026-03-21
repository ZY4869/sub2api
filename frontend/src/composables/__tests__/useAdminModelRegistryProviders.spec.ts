import { beforeEach, describe, expect, it, vi } from 'vitest'

const {
  listModelRegistry,
  listModelRegistryProviders,
  activateModelRegistryEntries,
  showError,
  showSuccess,
  modelInventoryInvalidate,
  ensureModelRegistryFresh,
  invalidateModelRegistry
} = vi.hoisted(() => ({
  listModelRegistry: vi.fn(),
  listModelRegistryProviders: vi.fn(),
  activateModelRegistryEntries: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
  modelInventoryInvalidate: vi.fn(),
  ensureModelRegistryFresh: vi.fn(() => Promise.resolve()),
  invalidateModelRegistry: vi.fn()
}))

vi.mock('@/api/admin/modelRegistry', () => ({
  listModelRegistry,
  listModelRegistryProviders,
  activateModelRegistryEntries
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showSuccess
  })
}))

vi.mock('@/stores', () => ({
  useModelInventoryStore: () => ({
    invalidate: modelInventoryInvalidate
  })
}))

vi.mock('@/stores/modelRegistry', () => ({
  ensureModelRegistryFresh,
  invalidateModelRegistry
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key
  })
}))

import { useAdminModelRegistryProviders } from '../useAdminModelRegistryProviders'

function createRegistryModel(id: string, provider: string, available = false, uiPriority = 0) {
  return {
    id,
    display_name: id.toUpperCase(),
    provider,
    platforms: [provider],
    protocol_ids: [id],
    aliases: [],
    pricing_lookup_ids: [id],
    preferred_protocol_ids: {},
    modalities: ['text'],
    capabilities: [],
    ui_priority: uiPriority,
    exposed_in: ['runtime'],
    source: 'runtime',
    hidden: false,
    tombstoned: false,
    available
  }
}

describe('useAdminModelRegistryProviders', () => {
  beforeEach(() => {
    listModelRegistry.mockReset()
    listModelRegistryProviders.mockReset()
    activateModelRegistryEntries.mockReset()
    showError.mockReset()
    showSuccess.mockReset()
    modelInventoryInvalidate.mockReset()
    ensureModelRegistryFresh.mockClear()
    invalidateModelRegistry.mockReset()
  })

  it('loads only provider summaries on first screen and defers provider models until requested', async () => {
    listModelRegistryProviders.mockResolvedValue({
      items: [
        { provider: 'openai', total_count: 60, available_count: 24 },
        { provider: 'anthropic', total_count: 20, available_count: 8 }
      ],
      total: 2,
      page: 1,
      page_size: 24,
      pages: 1
    })
    listModelRegistry.mockResolvedValue({
      items: [createRegistryModel('gpt-5.4', 'openai', true)],
      total: 1,
      page: 1,
      page_size: 50,
      pages: 1
    })

    const subject = useAdminModelRegistryProviders()

    await subject.loadAll()

    expect(listModelRegistryProviders).toHaveBeenCalledWith({
      page: 1,
      page_size: 24
    })
    expect(listModelRegistry).not.toHaveBeenCalled()
    expect(subject.providerGroups.value.map((group) => group.provider)).toEqual(['openai', 'anthropic'])

    await subject.ensureProviderModels('openai')

    expect(listModelRegistry).toHaveBeenCalledTimes(1)
    expect(listModelRegistry).toHaveBeenCalledWith({
      provider: 'openai',
      availability: 'all',
      include_hidden: false,
      include_tombstoned: false,
      page: 1,
      page_size: 50
    })
    expect(subject.getProviderModels('openai').map((item) => item.id)).toEqual(['gpt-5.4'])

    await subject.ensureProviderModels('openai')

    expect(listModelRegistry).toHaveBeenCalledTimes(1)
  })

  it('continues loading provider summaries and provider models page by page', async () => {
    listModelRegistryProviders
      .mockResolvedValueOnce({
        items: [{ provider: 'openai', total_count: 55, available_count: 20 }],
        total: 2,
        page: 1,
        page_size: 24,
        pages: 2
      })
      .mockResolvedValueOnce({
        items: [{ provider: 'gemini', total_count: 30, available_count: 12 }],
        total: 2,
        page: 2,
        page_size: 24,
        pages: 2
      })

    listModelRegistry
      .mockResolvedValueOnce({
        items: [createRegistryModel('gpt-5.4', 'openai', true, 2)],
        total: 2,
        page: 1,
        page_size: 50,
        pages: 2
      })
      .mockResolvedValueOnce({
        items: [createRegistryModel('gpt-5.4-mini', 'openai', false, 1)],
        total: 2,
        page: 2,
        page_size: 50,
        pages: 2
      })

    const subject = useAdminModelRegistryProviders()

    await subject.loadAll()
    expect(subject.hasMoreProviders.value).toBe(true)

    await subject.loadMoreProviders()

    expect(subject.providerGroups.value.map((group) => group.provider)).toEqual(['openai', 'gemini'])
    expect(listModelRegistryProviders).toHaveBeenCalledTimes(2)

    await subject.ensureProviderModels('openai')
    expect(subject.providerHasMoreModels('openai')).toBe(true)

    await subject.loadMoreProviderModels('openai')

    expect(subject.providerHasMoreModels('openai')).toBe(false)
    expect(subject.getProviderModels('openai').map((item) => item.id)).toEqual([
      'gpt-5.4',
      'gpt-5.4-mini'
    ])
  })
})
