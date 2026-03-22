import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key
  })
}))

import ModelProviderModelsPanel from '../ModelProviderModelsPanel.vue'

const SearchInputStub = {
  name: 'SearchInputStub',
  props: ['modelValue'],
  emits: ['update:modelValue', 'search'],
  template: '<input data-test="search-input" :value="modelValue" />'
}

const EmptyStateStub = { template: '<div data-test="empty-state" />' }
const LoadingSpinnerStub = { template: '<div data-test="loading-spinner" />' }
const ModelIconStub = { template: '<span data-test="model-icon" />' }
const ModelPlatformsInlineStub = { template: '<span data-test="platforms-inline" />' }

function createModel(id: string, available: boolean, extra?: Partial<Record<string, unknown>>) {
  return {
    id,
    display_name: id.toUpperCase(),
    provider: 'openai',
    platforms: ['openai'],
    protocol_ids: [id],
    aliases: [],
    pricing_lookup_ids: [id],
    preferred_protocol_ids: {},
    modalities: ['text'],
    capabilities: ['text'],
    ui_priority: 1,
    exposed_in: ['runtime'],
    source: 'runtime',
    hidden: false,
    tombstoned: false,
    available,
    ...extra
  }
}

function mountPanel(props?: Record<string, unknown>) {
  return mount(ModelProviderModelsPanel, {
    props: {
      provider: 'openai',
      models: [],
      selectedIds: [],
      isActivating: () => false,
      isDeactivating: () => false,
      isDeleting: () => false,
      ...props
    },
    global: {
      stubs: {
        SearchInput: SearchInputStub,
        EmptyState: EmptyStateStub,
        LoadingSpinner: LoadingSpinnerStub,
        ModelIcon: ModelIconStub,
        ModelPlatformsInline: ModelPlatformsInlineStub
      }
    }
  })
}

describe('ModelProviderModelsPanel', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('renders grouped categories and row actions based on availability', () => {
    const wrapper = mountPanel({
      models: [
        createModel('gpt-5.4', true),
        createModel('gpt-image', false, {
          modalities: ['text', 'image'],
          capabilities: ['image_generation']
        }),
        createModel('gpt-audio', false, {
          modalities: ['audio'],
          capabilities: ['audio_understanding']
        })
      ]
    })

    expect(wrapper.text()).toContain('admin.models.pages.all.categories.text')
    expect(wrapper.text()).toContain('admin.models.pages.all.categories.image')
    expect(wrapper.text()).toContain('admin.models.pages.all.categories.audio')
    expect(wrapper.text()).toContain('admin.models.registry.actions.deactivate')
    expect(wrapper.text()).toContain('admin.models.registry.actions.activate')
    expect(wrapper.text()).toContain('admin.models.registry.actions.hardDelete')
  })

  it('emits bulk deactivate and hard delete actions after confirmation', async () => {
    vi.spyOn(window, 'confirm').mockReturnValue(true)

    const wrapper = mountPanel({
      models: [
        createModel('gpt-5.4', true),
        createModel('gpt-image', false, {
          modalities: ['text', 'image'],
          capabilities: ['image_generation']
        })
      ],
      selectedIds: ['gpt-5.4', 'gpt-image']
    })

    const buttons = wrapper.findAll('button')
    const bulkDeactivate = buttons.find((button) => button.text() === 'admin.models.pages.all.bulk.deactivate')
    const bulkHardDelete = buttons.find((button) => button.text() === 'admin.models.pages.all.bulk.hardDelete')
    const rowHardDelete = buttons.find((button) => button.text() === 'admin.models.registry.actions.hardDelete')

    expect(bulkDeactivate).toBeDefined()
    expect(bulkHardDelete).toBeDefined()
    expect(rowHardDelete).toBeDefined()

    await bulkDeactivate!.trigger('click')
    await bulkHardDelete!.trigger('click')
    await rowHardDelete!.trigger('click')

    expect(wrapper.emitted('deactivate')).toEqual([[['gpt-5.4']]])
    expect(wrapper.emitted('hard-delete')).toEqual([
      [['gpt-5.4', 'gpt-image']],
      [['gpt-5.4']]
    ])
  })
})
