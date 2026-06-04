import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, h, nextTick, ref } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import ReAuthAccountModal from '../ReAuthAccountModal.vue'
import type { Account } from '@/types'

const {
  updateMock,
  clearErrorMock,
  refreshOpenAITokenMock,
  refreshAntigravityTokenMock,
  showErrorMock,
  showSuccessMock,
  flowState
} = vi.hoisted(() => ({
  updateMock: vi.fn(),
  clearErrorMock: vi.fn(),
  refreshOpenAITokenMock: vi.fn(),
  refreshAntigravityTokenMock: vi.fn(),
  showErrorMock: vi.fn(),
  showSuccessMock: vi.fn(),
  flowState: {
    inputMethod: 'refresh_token',
    refreshToken: 'rt-value',
    authCode: '',
    oauthState: '',
    projectId: '',
    sessionKey: ''
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: showErrorMock,
    showSuccess: showSuccessMock
  })
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      update: updateMock,
      clearError: clearErrorMock,
      refreshOpenAIToken: refreshOpenAITokenMock
    },
    antigravity: {
      generateAuthUrl: vi.fn(),
      exchangeCode: vi.fn(),
      refreshAntigravityToken: refreshAntigravityTokenMock
    }
  }
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

const BaseDialogStub = defineComponent({
  name: 'BaseDialog',
  props: {
    show: {
      type: Boolean,
      default: false
    }
  },
  template: '<div v-if="show"><slot /><slot name="footer" /></div>'
})

const OAuthAuthorizationFlowStub = defineComponent({
  name: 'OAuthAuthorizationFlow',
  props: [
    'showRefreshTokenOption',
    'showRefreshTokenSubmitButton',
    'platform'
  ],
  setup(props, { expose }) {
    const inputMethod = ref(flowState.inputMethod)
    const refreshToken = ref(flowState.refreshToken)
    const authCode = ref(flowState.authCode)
    const oauthState = ref(flowState.oauthState)
    const projectId = ref(flowState.projectId)
    const sessionKey = ref(flowState.sessionKey)

    expose({
      inputMethod,
      refreshToken,
      authCode,
      oauthState,
      projectId,
      sessionKey,
      reset: vi.fn()
    })

    return () =>
      h('div', { 'data-testid': 'oauth-flow' }, [
        h('span', { 'data-testid': 'rt-option' }, String(props.showRefreshTokenOption)),
        h('span', { 'data-testid': 'rt-inline-submit' }, String(props.showRefreshTokenSubmitButton)),
        h('span', { 'data-testid': 'platform' }, String(props.platform))
      ])
  }
})

function createAccount(platform: Account['platform'], overrides: Partial<Account> = {}): Account {
  return {
    id: platform === 'antigravity' ? 22 : 11,
    name: `${platform} account`,
    platform,
    type: 'oauth',
    credentials: {},
    extra: {},
    proxy_id: 9,
    concurrency: 1,
    priority: 1,
    status: 'error',
    error_message: 'expired',
    last_used_at: null,
    expires_at: null,
    auto_pause_on_expired: false,
    auto_renew_enabled: false,
    auto_renew_period: 'month',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    schedulable: false,
    rate_limited_at: null,
    rate_limit_reset_at: null,
    overload_until: null,
    temp_unschedulable_until: null,
    temp_unschedulable_reason: null,
    session_window_start: null,
    session_window_end: null,
    session_window_status: null,
    ...overrides
  } as Account
}

function mountModal(account: Account) {
  return mount(ReAuthAccountModal, {
    props: {
      show: true,
      account
    },
    global: {
      stubs: {
        BaseDialog: BaseDialogStub,
        OAuthAuthorizationFlow: OAuthAuthorizationFlowStub,
        AccountKiroAuthPanel: true,
        Icon: true
      }
    }
  })
}

beforeEach(() => {
  vi.clearAllMocks()
  flowState.inputMethod = 'refresh_token'
  flowState.refreshToken = 'rt-value'
  flowState.authCode = ''
  flowState.oauthState = ''
  flowState.projectId = ''
  flowState.sessionKey = ''
})

describe('admin ReAuthAccountModal', () => {
  it('reauthorizes OpenAI accounts from a refresh token', async () => {
    const updatedAccount = createAccount('openai', { status: 'active', error_message: null })
    refreshOpenAITokenMock.mockResolvedValue({
      access_token: 'new-access',
      refresh_token: 'new-refresh',
      token_type: 'Bearer',
      expires_at: 1800000000,
      email: 'user@example.com',
      plan_type: 'plus'
    })
    clearErrorMock.mockResolvedValue(updatedAccount)

    const wrapper = mountModal(createAccount('openai', {
      extra: {
        gateway_test_provider: 'openai',
        gateway_test_model_id: 'gpt-5.4'
      }
    }))
    await nextTick()

    expect(wrapper.get('[data-testid="rt-option"]').text()).toBe('true')
    expect(wrapper.get('[data-testid="rt-inline-submit"]').text()).toBe('false')

    const completeButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('admin.accounts.oauth.completeAuth'))

    expect(completeButton?.attributes('disabled')).toBeUndefined()

    await completeButton?.trigger('click')
    await flushPromises()

    expect(refreshOpenAITokenMock).toHaveBeenCalledWith(
      'rt-value',
      9,
      '/admin/openai/refresh-token'
    )
    expect(updateMock).toHaveBeenCalledWith(11, {
      type: 'oauth',
      credentials: expect.objectContaining({
        access_token: 'new-access',
        refresh_token: 'new-refresh',
        plan_type: 'plus'
      }),
      extra: expect.objectContaining({
        email: 'user@example.com',
        gateway_test_provider: 'openai',
        gateway_test_model_id: 'gpt-5.4'
      })
    })
    expect(clearErrorMock).toHaveBeenCalledWith(11)
    expect(wrapper.emitted('reauthorized')).toEqual([[updatedAccount]])
  })

  it('reauthorizes Antigravity accounts from a refresh token', async () => {
    const updatedAccount = createAccount('antigravity', { status: 'active', error_message: null })
    refreshAntigravityTokenMock.mockResolvedValue({
      access_token: 'ag-access',
      refresh_token: 'ag-refresh',
      token_type: 'Bearer',
      expires_at: 1800000000,
      project_id: 'project-1',
      email: 'ag@example.com'
    })
    clearErrorMock.mockResolvedValue(updatedAccount)

    const wrapper = mountModal(createAccount('antigravity'))
    await nextTick()

    expect(wrapper.get('[data-testid="rt-option"]').text()).toBe('true')
    expect(wrapper.get('[data-testid="rt-inline-submit"]').text()).toBe('false')

    const completeButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('admin.accounts.oauth.completeAuth'))

    expect(completeButton?.attributes('disabled')).toBeUndefined()

    await completeButton?.trigger('click')
    await flushPromises()

    expect(refreshAntigravityTokenMock).toHaveBeenCalledWith('rt-value', 9)
    expect(updateMock).toHaveBeenCalledWith(22, {
      type: 'oauth',
      credentials: expect.objectContaining({
        access_token: 'ag-access',
        refresh_token: 'ag-refresh',
        project_id: 'project-1',
        email: 'ag@example.com'
      })
    })
    expect(clearErrorMock).toHaveBeenCalledWith(22)
    expect(wrapper.emitted('reauthorized')).toEqual([[updatedAccount]])
  })

  it('does not show the generic completion button for Kiro and Gemini Vertex', async () => {
    const kiroWrapper = mountModal(createAccount('kiro'))
    expect(kiroWrapper.text()).not.toContain('admin.accounts.oauth.completeAuth')

    const vertexWrapper = mountModal(createAccount('gemini', {
      credentials: {
        oauth_type: 'vertex_ai'
      }
    }))
    await nextTick()

    expect(vertexWrapper.text()).toContain('admin.accounts.reauthUnavailableForPlatform')
    expect(vertexWrapper.text()).not.toContain('admin.accounts.oauth.completeAuth')
  })
})
