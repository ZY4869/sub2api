import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AccountCreateOAuthStep from '../AccountCreateOAuthStep.vue'

const resetMock = vi.fn()

const oauthFlowStub = defineComponent({
  name: 'OAuthAuthorizationFlow',
  props: [
    'addMethod',
    'authUrl',
    'sessionId',
    'loading',
    'error',
    'showHelp',
    'showProxyWarning',
    'allowMultiple',
    'showCookieOption',
    'showRefreshTokenOption',
    'platform',
    'showProjectId'
  ],
  emits: [
    'generate-url',
    'cookie-auth',
    'validate-refresh-token'
  ],
  setup(props, { emit, expose }) {
    expose({
      authCode: 'auth-code',
      oauthState: 'oauth-state',
      projectId: 'project-id',
      sessionKey: 'session-key',
      refreshToken: 'refresh-token',
      inputMethod: 'manual',
      reset: resetMock
    })

    return () =>
      h('div', { 'data-testid': 'oauth-step-stub' }, [
        h('span', { 'data-testid': 'auth-url' }, String(props.authUrl)),
        h('span', { 'data-testid': 'platform' }, String(props.platform)),
        h('button', { 'data-testid': 'emit-generate', onClick: () => emit('generate-url') }),
        h('button', { 'data-testid': 'emit-cookie', onClick: () => emit('cookie-auth', 'cookie-value') }),
        h('button', { 'data-testid': 'emit-refresh', onClick: () => emit('validate-refresh-token', 'rt-value') })
      ])
  }
})

const createWrapper = () =>
  mount(AccountCreateOAuthStep, {
    props: {
      addMethod: 'oauth',
      authUrl: 'https://example.com/auth',
      sessionId: 'session-id',
      loading: false,
      error: '',
      showHelp: true,
      showProxyWarning: false,
      allowMultiple: true,
      showCookieOption: true,
      showRefreshTokenOption: true,
      platform: 'anthropic',
      showProjectId: false
    },
    global: {
      stubs: {
        OAuthAuthorizationFlow: oauthFlowStub
      }
    }
  })

describe('AccountCreateOAuthStep', () => {
  it('forwards props, events and exposed flow state', async () => {
    resetMock.mockClear()
    const wrapper = createWrapper()

    expect(wrapper.get('[data-testid="auth-url"]').text()).toBe('https://example.com/auth')
    expect(wrapper.get('[data-testid="platform"]').text()).toBe('anthropic')

    await wrapper.get('[data-testid="emit-generate"]').trigger('click')
    await wrapper.get('[data-testid="emit-cookie"]').trigger('click')
    await wrapper.get('[data-testid="emit-refresh"]').trigger('click')

    expect(wrapper.emitted('generateUrl')).toEqual([[]])
    expect(wrapper.emitted('cookieAuth')).toEqual([['cookie-value']])
    expect(wrapper.emitted('validateRefreshToken')).toEqual([['rt-value']])

    const vm = wrapper.vm as unknown as {
      authCode: string
      oauthState: string
      projectId: string
      sessionKey: string
      refreshToken: string
      inputMethod: string
      reset: () => void
    }

    expect(vm.authCode).toBe('auth-code')
    expect(vm.oauthState).toBe('oauth-state')
    expect(vm.projectId).toBe('project-id')
    expect(vm.sessionKey).toBe('session-key')
    expect(vm.refreshToken).toBe('refresh-token')
    expect(vm.inputMethod).toBe('manual')

    vm.reset()
    expect(resetMock).toHaveBeenCalledTimes(1)
  })
})
