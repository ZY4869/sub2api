import { mount, flushPromises } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import UserExternalModelCatalogContent from '../UserExternalModelCatalogContent.vue'
import { MODEL_CATALOG_PUBLISHED_EVENT } from '@/utils/modelCatalogPublishedEvent'

const mockState = vi.hoisted(() => ({
  authStore: {
    user: {
      external_model_catalog_view_mode: 'follow_key_binding',
      api_key_model_binding_mode: 'model_required',
    },
  },
  getExternalModelCatalog: vi.fn(),
  closeCatalogEventSubscription: vi.fn(),
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => mockState.authStore,
}))

vi.mock('@/api/groups', () => ({
  default: {
    getExternalModelCatalog: mockState.getExternalModelCatalog,
  },
}))

vi.mock('@/utils/modelCatalogPublishedEvent', async () => {
  const actual = await vi.importActual<typeof import('@/utils/modelCatalogPublishedEvent')>('@/utils/modelCatalogPublishedEvent')
  return {
    ...actual,
    subscribeModelCatalogPublishedEvents: vi.fn(() => ({
      close: mockState.closeCatalogEventSubscription,
    })),
  }
})

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) =>
        params?.name ? `${key}:${params.name}` : params?.count !== undefined ? `${key}:${params.count}` : key,
    }),
  }
})

function catalogView(mode: 'group_first' | 'model_only') {
  return {
    external_model_catalog_view_mode: mode,
    effective_external_model_catalog_view_mode: mode,
    groups: [
      {
        id: 10,
        name: 'OpenAI',
        description: 'OpenAI group',
        platform: 'openai',
        priority: 1,
        model_count: 1,
      },
    ],
    items: [
      {
        model: 'gpt-5.4',
        public_model_id: 'gpt-5.4',
        currency: 'USD',
        price_display: { primary: [] },
        multiplier_summary: { enabled: false, kind: 'disabled' },
      },
    ],
    group_catalogs: {
      '10': [
        {
          model: 'gpt-5.4',
          public_model_id: 'gpt-5.4',
          currency: 'USD',
          price_display: { primary: [] },
          multiplier_summary: { enabled: false, kind: 'disabled' },
        },
      ],
    },
  }
}

function mountComponent() {
  return mount(UserExternalModelCatalogContent, {
    global: {
      stubs: {
        UserModelCatalogGrid: {
          props: ['items'],
          template: '<div data-test="catalog-grid">{{ items.map((item) => item.model).join(",") }}</div>',
        },
      },
    },
  })
}

describe('UserExternalModelCatalogContent', () => {
  const wrappers: ReturnType<typeof mountComponent>[] = []

  beforeEach(() => {
    mockState.getExternalModelCatalog.mockReset()
    mockState.closeCatalogEventSubscription.mockReset()
    mockState.authStore.user = {
      external_model_catalog_view_mode: 'follow_key_binding',
      api_key_model_binding_mode: 'model_required',
    }
  })

  afterEach(() => {
    for (const wrapper of wrappers.splice(0)) {
      wrapper.unmount()
    }
  })

  it('renders model-only users directly as a model list', async () => {
    mockState.getExternalModelCatalog.mockResolvedValue(catalogView('model_only'))

    const wrapper = mountComponent()
    wrappers.push(wrapper)
    await flushPromises()

    expect(wrapper.find('[data-testid="user-model-group-list"]').exists()).toBe(false)
    expect(wrapper.find('[data-test="catalog-grid"]').text()).toContain('gpt-5.4')
  })

  it('renders group-first users as group entry then selected group catalog', async () => {
    mockState.getExternalModelCatalog.mockResolvedValue(catalogView('group_first'))

    const wrapper = mountComponent()
    wrappers.push(wrapper)
    await flushPromises()

    expect(wrapper.find('[data-testid="user-model-group-list"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('ui.modelCatalog.groupFirst.modelCount:1')

    await wrapper.find('[data-testid="user-model-group-10"]').trigger('click')

    expect(wrapper.find('[data-test="catalog-grid"]').text()).toContain('gpt-5.4')
    expect(wrapper.text()).toContain('ui.modelCatalog.groupFirst.selectedGroup:OpenAI')
  })

  it('refreshes catalog after the published event', async () => {
    mockState.getExternalModelCatalog.mockResolvedValue(catalogView('model_only'))

    const wrapper = mountComponent()
    wrappers.push(wrapper)
    await flushPromises()
    window.dispatchEvent(new CustomEvent(MODEL_CATALOG_PUBLISHED_EVENT))
    await flushPromises()

    expect(mockState.getExternalModelCatalog).toHaveBeenCalledTimes(2)
  })

  it('closes the published event subscription when unmounted', async () => {
    mockState.getExternalModelCatalog.mockResolvedValue(catalogView('model_only'))

    const wrapper = mountComponent()
    await flushPromises()
    wrapper.unmount()

    expect(mockState.closeCatalogEventSubscription).toHaveBeenCalledTimes(1)
  })
})
