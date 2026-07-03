import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import SubscriptionsView from '../SubscriptionsView.vue'
import { FILTER_PLATFORM_ORDER } from '@/utils/platformBranding'

const mockState = vi.hoisted(() => ({
  listSubscriptions: vi.fn(),
  getAllGroups: vi.fn(),
  searchUsers: vi.fn(),
  assignSubscription: vi.fn(),
  extendSubscription: vi.fn(),
  revokeSubscription: vi.fn(),
  restoreSubscription: vi.fn(),
  resetQuota: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    subscriptions: {
      list: mockState.listSubscriptions,
      assign: mockState.assignSubscription,
      extend: mockState.extendSubscription,
      revoke: mockState.revokeSubscription,
      restore: mockState.restoreSubscription,
      resetQuota: mockState.resetQuota
    },
    groups: {
      getAll: mockState.getAllGroups
    },
    usage: {
      searchUsers: mockState.searchUsers
    }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: mockState.showError,
    showSuccess: mockState.showSuccess,
    showWarning: vi.fn(),
    showInfo: vi.fn()
  })
}))

vi.mock('@/composables/usePersistedPageSize', () => ({
  getPersistedPageSize: () => 20
}))

vi.mock('@/utils/format', () => ({
  formatDateOnly: (value: string) => value
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) =>
        ({
          'admin.subscriptions.allPlatforms': 'All Platforms',
          'admin.subscriptions.allStatus': 'All Status',
          'admin.subscriptions.allGroups': 'All Groups',
          'admin.users.searchUsers': 'Search users',
          'admin.subscriptions.assignSubscription': 'Assign Subscription',
          'admin.subscriptions.guide.showGuide': 'Guide',
          'admin.users.columnSettings': 'Column Settings',
          'common.refresh': 'Refresh',
          'common.clear': 'Clear',
          'admin.accounts.platforms.anthropic': 'anthropic',
          'admin.accounts.platforms.antigravity': 'antigravity',
          'admin.accounts.platforms.baidu_document_ai': 'baidu_document_ai',
          'admin.accounts.platforms.deepseek': 'deepseek',
          'admin.accounts.platforms.gemini': 'gemini',
          'admin.accounts.platforms.grok': 'grok',
          'admin.accounts.platforms.kiro': 'kiro',
          'admin.accounts.platforms.openai': 'openai',
          'admin.accounts.platforms.protocol_gateway': 'protocol_gateway',
        })[key] ?? key
    })
  }
})

const selectPropsHistory: Array<Record<string, any>> = []

const SelectStub = defineComponent({
  name: 'SelectStub',
  props: {
    modelValue: {
      type: [String, Number, Boolean],
      default: null
    },
    options: {
      type: Array,
      default: () => []
    },
    placeholder: {
      type: String,
      default: ''
    }
  },
  emits: ['update:modelValue', 'change'],
  setup(props) {
    selectPropsHistory.push(props as unknown as Record<string, any>)
    return () => null
  }
})

function mountView() {
  return mount(SubscriptionsView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        TablePageLayout: {
          template:
            '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>'
        },
        DataTable: { template: '<div><slot /><slot name="empty" /></div>' },
        Pagination: true,
        BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' },
        ConfirmDialog: true,
        EmptyState: true,
        Select: SelectStub,
        GroupBadge: true,
        GroupOptionItem: true,
        Icon: true,
        RouterLink: true,
        Teleport: true
      }
    }
  })
}

describe('SubscriptionsView platform filter', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    selectPropsHistory.length = 0
    mockState.listSubscriptions.mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      page_size: 20,
      pages: 0
    })
    mockState.getAllGroups.mockResolvedValue([])
    mockState.searchUsers.mockResolvedValue([])
    mockState.restoreSubscription.mockResolvedValue({})
  })

  it('uses All plus FILTER_PLATFORM_ORDER for platform options and excludes copilot', async () => {
    mountView()
    await flushPromises()

    const platformSelectProps = selectPropsHistory.find(
      (props) => props.placeholder === 'All Platforms'
    )
    expect(platformSelectProps).toBeTruthy()

    const optionValues = (platformSelectProps!.options as Array<{ value: string }>).map(
      (option) => option.value
    )

    expect(optionValues).toEqual(['', ...FILTER_PLATFORM_ORDER])
    expect(optionValues).not.toContain('copilot')
  })

  it('restores a revoked subscription after confirmation and refreshes the list', async () => {
    const wrapper = mountView()
    await flushPromises()

    const state = (wrapper.vm as any).$?.setupState
    state.restoringSubscription = {
      id: 123,
      user: { email: 'revoked@example.com' },
      status: 'revoked'
    }

    await state.confirmRestore()
    await flushPromises()

    expect(mockState.restoreSubscription).toHaveBeenCalledWith(123)
    expect(mockState.showSuccess).toHaveBeenCalledWith('admin.subscriptions.subscriptionRestored')
    expect(state.showRestoreDialog).toBe(false)
    expect(state.restoringSubscription).toBeNull()
    expect(mockState.listSubscriptions).toHaveBeenCalledTimes(2)
  })

  it('keeps restore dialog open and shows a friendly error when restore fails', async () => {
    mockState.restoreSubscription.mockRejectedValueOnce({
      response: { data: { detail: '同用户同分组已有有效订阅' } }
    })
    const wrapper = mountView()
    await flushPromises()

    const state = (wrapper.vm as any).$?.setupState
    state.showRestoreDialog = true
    state.restoringSubscription = {
      id: 124,
      user: { email: 'conflict@example.com' },
      status: 'revoked'
    }

    await state.confirmRestore()
    await flushPromises()

    expect(mockState.restoreSubscription).toHaveBeenCalledWith(124)
    expect(mockState.showError).toHaveBeenCalledWith('同用户同分组已有有效订阅')
    expect(state.showRestoreDialog).toBe(true)
    expect(state.restoringSubscription.id).toBe(124)
  })
})
