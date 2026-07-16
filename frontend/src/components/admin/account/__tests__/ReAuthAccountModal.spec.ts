import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, h, nextTick, ref } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import ReAuthAccountModal from '../ReAuthAccountModal.vue'
import type { Account } from '@/types'

const {
  updateMock,
  clearErrorMock,
  reauthorizeGrokAccountFromOAuthMock,
  refreshOpenAITokenMock,
  refreshAntigravityTokenMock,
  showErrorMock,
  showSuccessMock,
  flowState
} = vi.hoisted(() => ({
  updateMock: vi.fn(),
  clearErrorMock: vi.fn(),
  reauthorizeGrokAccountFromOAuthMock: vi.fn(),
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
      reauthorizeGrokAccountFromOAuth: reauthorizeGrokAccountFromOAuthMock,
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

const AccountGrokOAuthPanelStub = defineComponent({
  name: 'AccountGrokOAuthPanel',
  props: ['submitMode', 'allowDevice'],
  emits: ['submit'],
  setup(props, { emit, expose }) {
    expose({ reset: vi.fn() })
    return () =>
      h('div', [
        h('span', { 'data-testid': 'grok-submit-mode' }, String(props.submitMode)),
        h('span', { 'data-testid': 'grok-allow-device' }, String(props.allowDevice)),
        h('button', {
          'data-testid': 'grok-reauth-submit',
          onClick: () => emit('submit', {
            sessionId: 'session-1',
            code: 'auth-code',
            state: 'state-1'
          })
        }, 'submit grok')
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

function mountModal(account: Account | null, show = true) {
  return mount(ReAuthAccountModal, {
    props: {
      show,
      account
    },
    global: {
      stubs: {
        BaseDialog: BaseDialogStub,
        OAuthAuthorizationFlow: OAuthAuthorizationFlowStub,
        AccountKiroAuthPanel: true,
        AccountGrokOAuthPanel: AccountGrokOAuthPanelStub,
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
  it('mounts hidden and resets initial state without throwing', () => {
    expect(() => {
      const wrapper = mountModal(null, false)
      wrapper.unmount()
    }).not.toThrow()

    expect(updateMock).not.toHaveBeenCalled()
    expect(clearErrorMock).not.toHaveBeenCalled()
  })

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

  it('reauthorizes Grok accounts through the Grok OAuth panel payload', async () => {
    const updatedAccount = createAccount('grok', {
      type: 'oauth',
      status: 'active',
      error_message: null,
      credentials: {
        access_token: 'grok-access',
        refresh_token: 'grok-refresh',
        base_url: 'https://cli-chat-proxy.grok.com/v1'
      },
      extra: {
        provider: 'xai',
        source: 'grok_browser_oauth',
        model_scope_v2: {
          policy_mode: 'whitelist',
          entries: []
        }
      }
    })
    reauthorizeGrokAccountFromOAuthMock.mockResolvedValue(updatedAccount)

    const wrapper = mountModal(createAccount('grok', {
      type: 'sso',
      credentials: {
        sso_token: 'legacy-token'
      },
      extra: {
        grok_tier: 'super'
      }
    }))
    await nextTick()

    expect(wrapper.get('[data-testid="grok-submit-mode"]').text()).toBe('authorization')
    expect(wrapper.get('[data-testid="grok-allow-device"]').text()).toBe('false')

    await wrapper.get('[data-testid="grok-reauth-submit"]').trigger('click')
    await flushPromises()

    expect(reauthorizeGrokAccountFromOAuthMock).toHaveBeenCalledWith(11, {
      session_id: 'session-1',
      code: 'auth-code',
      state: 'state-1',
      proxy_id: 9
    })
    expect(updateMock).not.toHaveBeenCalled()
    expect(clearErrorMock).not.toHaveBeenCalled()
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
