import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, h, ref } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import KeysView from '../KeysView.vue'

const mocks = vi.hoisted(() => ({
  listKeys: vi.fn(),
  createKey: vi.fn(),
  updateKey: vi.fn(),
  getAvailableGroups: vi.fn(),
  getModelOptions: vi.fn(),
  getModelCatalog: vi.fn(),
  getUserGroupRates: vi.fn(),
  getPublicSettings: vi.fn(),
  getUsage: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
  authState: {
    isAdmin: false,
    user: {
      id: 1,
      role: 'user',
      api_key_model_binding_mode: 'model_required',
    },
  },
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

vi.mock('@/api', () => ({
  keysAPI: {
    list: mocks.listKeys,
    getModelCatalog: vi.fn(),
    createWithPayload: mocks.createKey,
    update: mocks.updateKey,
    toggleStatus: vi.fn(),
    delete: vi.fn(),
  },
  authAPI: {
    getPublicSettings: mocks.getPublicSettings,
  },
  usageAPI: {
    getDashboardApiKeysUsage: mocks.getUsage,
  },
  userGroupsAPI: {
    getAvailable: mocks.getAvailableGroups,
    getModelOptions: mocks.getModelOptions,
    getModelCatalog: mocks.getModelCatalog,
    getUserGroupRates: mocks.getUserGroupRates,
  },
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    groups: {
      getAll: vi.fn(),
    },
  },
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => mocks.authState,
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: mocks.showError,
    showSuccess: mocks.showSuccess,
  }),
}))

vi.mock('@/stores/onboarding', () => ({
  useOnboardingStore: () => ({
    isCurrentStep: vi.fn(() => false),
    nextStep: vi.fn(),
  }),
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard: vi.fn(),
  }),
}))

vi.mock('@/composables/usePersistedPageSize', () => ({
  getPersistedPageSize: () => 20,
}))

const bindingsRef = ref([
  {
    group_id: 10,
    quota: 0,
    model_patterns_text: '',
    selected_models: [],
    model_selection_dirty: true,
  },
])

const APIKeyGroupBindingsEditorStub = defineComponent({
  name: 'APIKeyGroupBindingsEditor',
  props: ['modelValue'],
  emits: ['update:modelValue'],
  setup(props, { emit }) {
    emit('update:modelValue', bindingsRef.value)
    return () => h('div', { 'data-testid': 'bindings-editor' }, JSON.stringify(props.modelValue))
  },
})

const BaseDialogStub = {
  props: ['show'],
  template: `
    <section v-if="show">
      <slot />
      <slot name="footer" />
    </section>
  `,
}

const EmptyStateStub = {
  emits: ['action'],
  template: '<button data-testid="empty-create" @click="$emit(\'action\')">create</button>',
}

const SelectStub = {
  props: ['modelValue'],
  emits: ['update:modelValue'],
  template: '<div />',
}

const DataTableStub = {
  props: ['data'],
  template: '<div><slot name="empty" /></div>',
}

function resetMocks(mode: 'model_required' | 'group_allowed') {
  mocks.authState.isAdmin = false
  mocks.authState.user = {
    id: 1,
    role: 'user',
    api_key_model_binding_mode: mode,
  }
  mocks.listKeys.mockResolvedValue({ items: [], total: 0, pages: 0 })
  mocks.getAvailableGroups.mockResolvedValue([
    { id: 10, name: 'OpenAI', platform: 'openai', priority: 1, status: 'active' },
  ])
  mocks.getModelOptions.mockResolvedValue([
    {
      group_id: 10,
      name: 'OpenAI',
      platform: 'openai',
      priority: 1,
      model_count: 101,
      models: [
        { public_id: 'gpt-5.4', display_name: 'GPT 5.4', request_protocols: ['chat'] },
        ...Array.from({ length: 100 }, (_, index) => ({
          public_id: `gpt-image-${index}`,
          display_name: `GPT Image ${index}`,
          request_protocols: ['images'],
        })),
      ],
    },
  ])
  mocks.getModelCatalog.mockResolvedValue({ items: [] })
  mocks.getUserGroupRates.mockResolvedValue({})
  mocks.getPublicSettings.mockResolvedValue({})
  mocks.getUsage.mockResolvedValue({ stats: {} })
  mocks.createKey.mockResolvedValue({})
  mocks.updateKey.mockResolvedValue({})
  mocks.showError.mockReset()
  mocks.showSuccess.mockReset()
  mocks.createKey.mockClear()
}

async function mountKeysView(mode: 'model_required' | 'group_allowed') {
  resetMocks(mode)
  bindingsRef.value = [
    {
      group_id: 10,
      quota: 0,
      model_patterns_text: '',
      selected_models: [],
      model_selection_dirty: true,
    },
  ]
  const wrapper = mount(KeysView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        TablePageLayout: {
          template: '<div><slot name="filters" /><slot name="actions" /><slot name="table" /><slot name="pagination" /></div>',
        },
        DataTable: DataTableStub,
        Pagination: { template: '<div />' },
        BaseDialog: BaseDialogStub,
        ConfirmDialog: { template: '<div />' },
        EmptyState: EmptyStateStub,
        Select: SelectStub,
        SearchInput: { template: '<input />' },
        Icon: { template: '<span />' },
        UseKeyModal: { template: '<div />' },
        ApiKeyGroupPill: { template: '<span />' },
        APIKeyGroupBindingsEditor: APIKeyGroupBindingsEditorStub,
      },
    },
  })
  await flushPromises()
  await wrapper.get('[data-testid="empty-create"]').trigger('click')
  await flushPromises()
  await wrapper.get('input[required]').setValue('narrow key')
  return wrapper
}

async function enableImageOnlyKey(wrapper: Awaited<ReturnType<typeof mountKeysView>>) {
  const imageOnlyToggle = wrapper
    .findAllComponents({ name: 'ToggleField' })
    .find((component) => component.props('label') === 'keys.imageOnlyKey')
  if (!imageOnlyToggle) {
    throw new Error('image-only toggle was not rendered')
  }
  await imageOnlyToggle.find('button').trigger('click')
  await flushPromises()
}

describe('KeysView api key model binding policy', () => {
  beforeEach(() => {
    vi.useRealTimers()
  })

  it('allows whole-group payloads in model_required mode', async () => {
    const wrapper = await mountKeysView('model_required')

    await wrapper.get('form#key-form').trigger('submit.prevent')

    expect(mocks.showError).not.toHaveBeenCalledWith('keys.modelSelectionRequired')
    expect(mocks.createKey).toHaveBeenCalledWith(
      expect.objectContaining({
        name: 'narrow key',
        groups: [{ group_id: 10 }],
      }),
    )
  })

  it('allows whole-group payloads in group_allowed mode', async () => {
    const wrapper = await mountKeysView('group_allowed')

    await wrapper.get('form#key-form').trigger('submit.prevent')

    expect(mocks.showError).not.toHaveBeenCalledWith('keys.modelSelectionRequired')
    expect(mocks.createKey).toHaveBeenCalledWith(
      expect.objectContaining({
        name: 'narrow key',
        groups: [{ group_id: 10 }],
      }),
    )
  })

  it('keeps image-only create payload as whole-group when no model is selected', async () => {
    const wrapper = await mountKeysView('group_allowed')

    await enableImageOnlyKey(wrapper)
    await wrapper.get('form#key-form').trigger('submit.prevent')

    const payload = mocks.createKey.mock.calls[0]?.[0]
    expect(payload).toEqual(
      expect.objectContaining({
        name: 'narrow key',
        image_only_enabled: true,
        groups: [{ group_id: 10 }],
      }),
    )
    expect(payload.groups[0]).not.toHaveProperty('model_patterns')
  })
})
