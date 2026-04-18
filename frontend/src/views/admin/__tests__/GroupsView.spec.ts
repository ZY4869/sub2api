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

const SelectStub = defineComponent({
  name: 'Select',
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
        Select: SelectStub,
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

    const state = (wrapper.vm as any).$?.setupState
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
})
