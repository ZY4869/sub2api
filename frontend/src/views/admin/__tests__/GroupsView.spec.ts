import { flushPromises, mount } from '@vue/test-utils'
import { computed, defineComponent, h, nextTick } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import GroupsView from '../GroupsView.vue'

const mockState = vi.hoisted(() => ({
  listGroups: vi.fn(),
  getUsageSummary: vi.fn(),
  getCapacitySummary: vi.fn(),
  getAllGroups: vi.fn(),
  createGroup: vi.fn(),
  updateGroup: vi.fn(),
  deleteGroup: vi.fn(),
  updateSortOrder: vi.fn(),
  listAccounts: vi.fn(),
  getAccountById: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
  isCurrentStep: vi.fn(() => false),
  nextStep: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    groups: {
      list: mockState.listGroups,
      getUsageSummary: mockState.getUsageSummary,
      getCapacitySummary: mockState.getCapacitySummary,
      getAll: mockState.getAllGroups,
      create: mockState.createGroup,
      update: mockState.updateGroup,
      delete: mockState.deleteGroup,
      updateSortOrder: mockState.updateSortOrder
    },
    accounts: {
      list: mockState.listAccounts,
      getById: mockState.getAccountById
    }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: mockState.showError,
    showSuccess: mockState.showSuccess
  })
}))

vi.mock('@/stores/onboarding', () => ({
  useOnboardingStore: () => ({
    isCurrentStep: mockState.isCurrentStep,
    nextStep: mockState.nextStep
  })
}))

vi.mock('@/composables/usePersistedPageSize', () => ({
  getPersistedPageSize: () => 20
}))

vi.mock('@/composables/useKeyedDebouncedSearch', () => ({
  useKeyedDebouncedSearch: () => ({
    trigger: vi.fn(),
    clearKey: vi.fn(),
    clearAll: vi.fn()
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

const BaseSelectStub = defineComponent({
  name: 'BaseSelectStub',
  props: {
    modelValue: {
      type: [String, Number, Boolean, Object, null],
      default: null
    },
    options: {
      type: Array,
      default: () => []
    },
    valueKey: {
      type: String,
      default: 'value'
    }
  },
  emits: ['update:modelValue', 'change'],
  setup(props, { slots }) {
    const selectedOption = computed(() =>
      (props.options as Array<Record<string, unknown>>).find(
        (option) => option?.[props.valueKey] === props.modelValue
      ) ?? null
    )

    return () =>
      h('div', { class: 'select-stub' }, [
        h('div', { class: 'selected-slot' }, slots.selected?.({ option: selectedOption.value })),
        ...(props.options as Array<Record<string, unknown>>).map((option) =>
          h(
            'div',
            {
              class: 'option-slot',
              'data-option-value': String(option?.[props.valueKey] ?? '')
            },
            slots.option?.({
              option,
              selected: option?.[props.valueKey] === props.modelValue
            })
          )
        )
      ])
  }
})

const BaseDialogStub = {
  props: ['show', 'title', 'width'],
  template: `
    <section v-if="show" data-testid="base-dialog">
      <slot />
      <slot name="footer" />
    </section>
  `
}

const PlatformLabelStub = {
  name: 'PlatformLabel',
  props: ['platform', 'label', 'description'],
  template: `
    <div
      data-testid="platform-label"
      :data-platform="platform"
      :data-label="label"
    >
      <span>{{ label }}</span>
      <span v-if="description">{{ description }}</span>
    </div>
  `
}

const GroupOptionItemStub = {
  name: 'GroupOptionItem',
  props: ['name', 'platform', 'description', 'selected'],
  template: `
    <div
      data-testid="group-option-item"
      :data-name="name"
      :data-platform="platform"
      :data-selected="selected"
    >
      <span>{{ name }}</span>
      <span v-if="description">{{ description }}</span>
    </div>
  `
}

const GroupBadgeStub = {
  name: 'GroupBadge',
  props: ['name', 'platform'],
  template: `
    <div
      data-testid="group-badge"
      :data-name="name"
      :data-platform="platform"
    >
      {{ name }}
    </div>
  `
}

function createGroup(id: number, name: string, accountCount: number) {
  return {
    id,
    name,
    description: null,
    platform: 'anthropic',
    priority: id,
    rate_multiplier: 1,
    is_exclusive: false,
    status: 'active',
    subscription_type: 'standard',
    daily_limit_usd: null,
    weekly_limit_usd: null,
    monthly_limit_usd: null,
    image_price_1k: null,
    image_price_2k: null,
    image_price_4k: null,
    web_search_price_per_call: null,
    claude_code_only: false,
    fallback_group_id: null,
    fallback_group_id_on_invalid_request: null,
    created_at: '2026-04-18T00:00:00Z',
    updated_at: '2026-04-18T00:00:00Z',
    model_routing: null,
    model_routing_enabled: false,
    mcp_xml_inject: true,
    supported_model_scopes: ['claude'],
    account_count: accountCount,
    active_account_count: accountCount,
    rate_limited_account_count: 0,
    default_mapped_model: '',
    sort_order: id * 10
  }
}

const groupsFixture = [
  createGroup(1, 'Anthropic Main', 6),
  createGroup(2, 'Anthropic Fallback', 4),
  createGroup(3, 'Anthropic Invalid Fallback', 3),
  createGroup(4, 'Anthropic Copy Source', 5)
]

function mountView() {
  return mount(GroupsView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        TablePageLayout: {
          template:
            '<div><slot name="actions" /><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>'
        },
        DataTable: { template: '<div><slot /><slot name="empty" /></div>' },
        Pagination: true,
        EmptyState: true,
        BaseDialog: BaseDialogStub,
        ConfirmDialog: true,
        Select: BaseSelectStub,
        PlatformIcon: true,
        PlatformLabel: PlatformLabelStub,
        GroupBadge: GroupBadgeStub,
        GroupOptionItem: GroupOptionItemStub,
        Icon: true,
        GroupRateMultipliersModal: true,
        GroupCapacityBadge: true,
        VueDraggable: { template: '<div><slot /></div>' },
        Teleport: true
      }
    }
  })
}

function setupState(wrapper: any) {
  return (wrapper.vm as any).$?.setupState
}

describe('GroupsView iconized selections', () => {
  beforeEach(() => {
    mockState.listGroups.mockReset()
    mockState.getUsageSummary.mockReset()
    mockState.getCapacitySummary.mockReset()
    mockState.getAllGroups.mockReset()
    mockState.createGroup.mockReset()
    mockState.updateGroup.mockReset()
    mockState.deleteGroup.mockReset()
    mockState.updateSortOrder.mockReset()
    mockState.listAccounts.mockReset()
    mockState.getAccountById.mockReset()
    mockState.showError.mockReset()
    mockState.showSuccess.mockReset()
    mockState.isCurrentStep.mockReset()
    mockState.nextStep.mockReset()
    window.localStorage.removeItem('sub2api.admin.groups.hiddenColumns')

    mockState.isCurrentStep.mockReturnValue(false)
    mockState.listGroups.mockResolvedValue({
      items: groupsFixture,
      total: groupsFixture.length,
      pages: 1
    })
    mockState.getUsageSummary.mockResolvedValue([])
    mockState.getCapacitySummary.mockResolvedValue([])
    mockState.getAllGroups.mockResolvedValue(groupsFixture)
    mockState.listAccounts.mockResolvedValue({ items: [] })
    mockState.getAccountById.mockResolvedValue({ id: 11, name: 'unused' })
  })

  it('renders platform, fallback, invalid fallback, and copy-account selections through icon-aware helpers', async () => {
    const wrapper = mountView()

    await flushPromises()

    const state = setupState(wrapper)
    state.showCreateModal = true
    state.showEditModal = true
    state.editingGroup = groupsFixture[0]

    state.createForm.platform = 'anthropic'
    state.createForm.claude_code_only = true
    state.createForm.subscription_type = 'standard'
    state.createForm.fallback_group_id = 2
    state.createForm.fallback_group_id_on_invalid_request = 3
    state.createForm.copy_accounts_from_group_ids.splice(0)
    state.createForm.copy_accounts_from_group_ids.push(4)

    state.editForm.platform = 'anthropic'
    state.editForm.claude_code_only = true
    state.editForm.subscription_type = 'standard'
    state.editForm.fallback_group_id = 2
    state.editForm.fallback_group_id_on_invalid_request = 3
    state.editForm.copy_accounts_from_group_ids.splice(0)
    state.editForm.copy_accounts_from_group_ids.push(4)

    await nextTick()

    const platformLabels = wrapper.findAll('[data-testid="platform-label"][data-platform="anthropic"]')
    expect(platformLabels.length).toBeGreaterThan(0)

    expect(
      wrapper.findAll('[data-testid="group-badge"][data-name="Anthropic Fallback"][data-platform="anthropic"]').length
    ).toBeGreaterThan(0)
    expect(
      wrapper.findAll('[data-testid="group-badge"][data-name="Anthropic Invalid Fallback"][data-platform="anthropic"]').length
    ).toBeGreaterThan(0)
    expect(
      wrapper.findAll('[data-testid="group-badge"][data-name="Anthropic Copy Source"][data-platform="anthropic"]').length
    ).toBeGreaterThan(0)

    expect(
      wrapper.findAll('[data-testid="group-option-item"][data-name="Anthropic Fallback"][data-platform="anthropic"]').length
    ).toBeGreaterThan(0)
    expect(
      wrapper.findAll('[data-testid="group-option-item"][data-name="Anthropic Invalid Fallback"][data-platform="anthropic"]').length
    ).toBeGreaterThan(0)
    expect(
      wrapper.findAll('[data-testid="group-option-item"][data-name="Anthropic Copy Source"][data-platform="anthropic"]').length
    ).toBeGreaterThan(0)
  })

  it('uses the shared english-name platform order in both create and filter selects', async () => {
    const wrapper = mountView()
    await flushPromises()

    const platformLabels = wrapper
      .findAll('[data-testid="platform-label"]')
      .map((node) => node.attributes('data-label'))
      .filter(Boolean)

    expect(platformLabels).toEqual(
      expect.arrayContaining([
        'admin.groups.platforms.anthropic',
        'admin.groups.platforms.antigravity',
        'admin.groups.platforms.baidu_document_ai',
        'admin.groups.platforms.deepseek',
        'admin.groups.platforms.gemini',
        'admin.groups.platforms.grok',
        'admin.groups.platforms.kiro',
        'admin.groups.platforms.openai',
        'admin.groups.platforms.protocol_gateway'
      ])
    )
  })

  it('keeps name/actions columns fixed and allows peak-rate column persistence toggles', async () => {
    const wrapper = mountView()
    await flushPromises()

    const state = setupState(wrapper)
    const allColumns = state.columns.value ?? state.columns
    expect(allColumns.map((column: { key: string }) => column.key)).toContain('id')
    expect(allColumns.map((column: { key: string }) => column.key)).toContain('peak_rate')

    state.toggleGroupColumn('id')
    await nextTick()
    let visibleColumns = state.visibleColumns.value ?? state.visibleColumns
    expect(visibleColumns.map((column: { key: string }) => column.key)).not.toContain('id')
    expect(window.localStorage.getItem('sub2api.admin.groups.hiddenColumns')).toContain('id')

    state.toggleGroupColumn('name')
    await nextTick()
    visibleColumns = state.visibleColumns.value ?? state.visibleColumns
    expect(visibleColumns.map((column: { key: string }) => column.key)).toContain('name')

    state.toggleGroupColumn('actions')
    await nextTick()
    visibleColumns = state.visibleColumns.value ?? state.visibleColumns
    expect(visibleColumns.map((column: { key: string }) => column.key)).toContain('actions')

    state.toggleGroupColumn('peak_rate')
    await nextTick()
    visibleColumns = state.visibleColumns.value ?? state.visibleColumns
    expect(visibleColumns.map((column: { key: string }) => column.key)).not.toContain('peak_rate')
    expect(window.localStorage.getItem('sub2api.admin.groups.hiddenColumns')).toContain('peak_rate')
  })

  it('clears peak-rate payload for standard groups and validates subscription peak windows', async () => {
    const wrapper = mountView()
    await flushPromises()

    const state = setupState(wrapper)
    state.createForm.name = 'Standard Group'
    state.createForm.subscription_type = 'standard'
    state.createForm.peak_rate_enabled = true
    state.createForm.peak_start = '09:00'
    state.createForm.peak_end = '18:00'
    state.createForm.peak_rate_multiplier = 2

    await state.handleCreateGroup()
    await flushPromises()

    expect(mockState.createGroup).toHaveBeenCalledWith(
      expect.objectContaining({
        peak_rate_enabled: false,
        peak_start: '',
        peak_end: '',
        peak_rate_multiplier: 1
      })
    )

    mockState.createGroup.mockClear()
    mockState.showError.mockClear()
    state.createForm.name = 'Bad Peak Group'
    state.createForm.subscription_type = 'subscription'
    state.createForm.peak_rate_enabled = true
    state.createForm.peak_start = '9:00'
    state.createForm.peak_end = '18:00'

    await state.handleCreateGroup()
    await flushPromises()

    expect(mockState.createGroup).not.toHaveBeenCalled()
    expect(mockState.showError).toHaveBeenCalledWith('admin.groups.peakRate.invalidTime')
  })

  it('shows Grok image and web search pricing without supported scopes', async () => {
    const wrapper = mountView()
    await flushPromises()

    const state = setupState(wrapper)
    state.showCreateModal = true
    state.createForm.platform = 'grok'
    await nextTick()

    expect(wrapper.text()).toContain('admin.groups.imagePricing.title')
    expect(wrapper.text()).toContain('admin.groups.webSearchPricing.label')
    expect(wrapper.text()).not.toContain('admin.groups.supportedScopes.title')

    state.showCreateModal = false
    state.showEditModal = true
    state.editingGroup = { ...groupsFixture[0], platform: 'grok' }
    state.editForm.platform = 'grok'
    await nextTick()

    expect(wrapper.text()).toContain('admin.groups.imagePricing.title')
    expect(wrapper.text()).toContain('admin.groups.webSearchPricing.label')
    expect(wrapper.text()).not.toContain('admin.groups.supportedScopes.title')
  })

  it('does not submit Antigravity supported scopes for default Anthropic creates', async () => {
    const wrapper = mountView()
    await flushPromises()

    const state = setupState(wrapper)
    state.createForm.name = 'Anthropic Group'
    state.createForm.platform = 'anthropic'
    state.createForm.supported_model_scopes = ['claude', 'gemini_text']
    state.createForm.mcp_xml_inject = false
    state.createForm.allow_messages_dispatch = true
    state.createForm.default_mapped_model = 'gpt-5.4'
    state.createForm.image_protocol_mode = 'native'
    state.createForm.gemini_mixed_protocol_enabled = true
    state.createForm.image_batch_enabled = true

    await state.handleCreateGroup()
    await flushPromises()

    const payload = mockState.createGroup.mock.calls.at(-1)?.[0]
    expect(payload).toEqual(
      expect.objectContaining({
        platform: 'anthropic',
        supported_model_scopes: []
      })
    )
    expect(payload).not.toHaveProperty('mcp_xml_inject')
    expect(payload).not.toHaveProperty('allow_messages_dispatch')
    expect(payload).not.toHaveProperty('default_mapped_model')
    expect(payload).not.toHaveProperty('image_protocol_mode')
    expect(payload).not.toHaveProperty('gemini_mixed_protocol_enabled')
    expect(payload).not.toHaveProperty('image_batch_enabled')
  })

  it('cleans stale platform-only fields when creating and updating Grok groups', async () => {
    const wrapper = mountView()
    await flushPromises()

    const state = setupState(wrapper)
    state.createForm.name = 'Grok Group'
    state.createForm.platform = 'grok'
    state.createForm.image_price_1k = 0.1
    state.createForm.web_search_price_per_call = 0
    state.createForm.claude_code_only = true
    state.createForm.fallback_group_id = 2
    state.createForm.fallback_group_id_on_invalid_request = 3
    state.createForm.model_routing_enabled = true
    state.createForm.allow_messages_dispatch = true
    state.createForm.default_mapped_model = 'gpt-5.4'
    state.createForm.image_protocol_mode = 'native'
    state.createForm.mcp_xml_inject = false
    state.createForm.supported_model_scopes = ['claude']
    state.createForm.gemini_mixed_protocol_enabled = true
    state.createForm.image_batch_enabled = true
    state.createForm.image_batch_allowed_providers = ['gemini']

    await state.handleCreateGroup()
    await flushPromises()

    const createPayload = mockState.createGroup.mock.calls.at(-1)?.[0]
    expect(createPayload).toEqual(
      expect.objectContaining({
        platform: 'grok',
        image_price_1k: 0.1,
        web_search_price_per_call: 0,
        supported_model_scopes: []
      })
    )
    expect(createPayload).not.toHaveProperty('claude_code_only')
    expect(createPayload).not.toHaveProperty('fallback_group_id')
    expect(createPayload).not.toHaveProperty('fallback_group_id_on_invalid_request')
    expect(createPayload).not.toHaveProperty('model_routing_enabled')
    expect(createPayload).not.toHaveProperty('model_routing')
    expect(createPayload).not.toHaveProperty('allow_messages_dispatch')
    expect(createPayload).not.toHaveProperty('default_mapped_model')
    expect(createPayload).not.toHaveProperty('image_protocol_mode')
    expect(createPayload).not.toHaveProperty('mcp_xml_inject')
    expect(createPayload).not.toHaveProperty('gemini_mixed_protocol_enabled')
    expect(createPayload).not.toHaveProperty('image_batch_enabled')
    expect(createPayload).not.toHaveProperty('image_batch_allowed_providers')

    mockState.updateGroup.mockResolvedValue({})
    state.editingGroup = { ...groupsFixture[0], id: 20, platform: 'grok' }
    state.editForm.name = 'Grok Group'
    state.editForm.platform = 'grok'
    state.editForm.status = 'active'
    state.editForm.image_price_2k = 0.2
    state.editForm.web_search_price_per_call = ''
    state.editForm.claude_code_only = true
    state.editForm.fallback_group_id = 2
    state.editForm.fallback_group_id_on_invalid_request = 3
    state.editForm.allow_messages_dispatch = true
    state.editForm.supported_model_scopes = ['claude']
    state.editForm.image_batch_enabled = true

    await state.handleUpdateGroup()
    await flushPromises()

    const updatePayload = mockState.updateGroup.mock.calls.at(-1)?.[1]
    expect(mockState.updateGroup).toHaveBeenLastCalledWith(
      20,
      expect.objectContaining({
        platform: 'grok',
        image_price_2k: 0.2,
        web_search_price_per_call: null,
        supported_model_scopes: []
      })
    )
    expect(updatePayload).not.toHaveProperty('claude_code_only')
    expect(updatePayload).not.toHaveProperty('fallback_group_id')
    expect(updatePayload).not.toHaveProperty('fallback_group_id_on_invalid_request')
    expect(updatePayload).not.toHaveProperty('allow_messages_dispatch')
    expect(updatePayload).not.toHaveProperty('image_batch_enabled')
  })

  it('preserves OpenAI and Grok web search price values including free and cleared states', async () => {
    const wrapper = mountView()
    await flushPromises()

    const state = setupState(wrapper)
    state.createForm.name = 'OpenAI Search'
    state.createForm.platform = 'openai'
    state.createForm.web_search_price_per_call = 0

    await state.handleCreateGroup()
    await flushPromises()

    expect(mockState.createGroup).toHaveBeenCalledWith(
      expect.objectContaining({
        web_search_price_per_call: 0
      })
    )

    mockState.createGroup.mockClear()
    state.createForm.name = 'Grok Search'
    state.createForm.platform = 'grok'
    state.createForm.web_search_price_per_call = 0

    await state.handleCreateGroup()
    await flushPromises()

    expect(mockState.createGroup).toHaveBeenCalledWith(
      expect.objectContaining({
        platform: 'grok',
        web_search_price_per_call: 0
      })
    )

    mockState.updateGroup.mockResolvedValue({})
    state.editingGroup = {
      ...groupsFixture[0],
      id: 10,
      platform: 'openai',
      web_search_price_per_call: 0.025
    }
    state.editForm.name = 'OpenAI Search'
    state.editForm.platform = 'openai'
    state.editForm.web_search_price_per_call = ''

    await state.handleUpdateGroup()
    await flushPromises()

    expect(mockState.updateGroup).toHaveBeenCalledWith(
      10,
      expect.objectContaining({
        web_search_price_per_call: null
      })
    )
  })
})
