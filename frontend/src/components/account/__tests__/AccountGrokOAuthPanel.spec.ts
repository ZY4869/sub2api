import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import AccountGrokOAuthPanel from '../AccountGrokOAuthPanel.vue'

const {
  generateGrokAuthUrl,
  exchangeGrokAuthCode,
  startGrokDeviceFlow,
  pollGrokDeviceToken,
  copyToClipboard
} = vi.hoisted(() => ({
  generateGrokAuthUrl: vi.fn(),
  exchangeGrokAuthCode: vi.fn(),
  startGrokDeviceFlow: vi.fn(),
  pollGrokDeviceToken: vi.fn(),
  copyToClipboard: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      generateGrokAuthUrl,
      exchangeGrokAuthCode,
      startGrokDeviceFlow,
      pollGrokDeviceToken
    }
  }
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => {
        if (key === 'admin.accounts.grokOauth.devicePollingHint') {
          return `poll every ${params?.seconds}s`
        }
        return key
      }
    })
  }
})

function mountPanel() {
  return mount(AccountGrokOAuthPanel, {
    props: {
      submitLabel: '创建',
      proxyId: 7
    }
  })
}

async function generateAuth(wrapper: ReturnType<typeof mountPanel>) {
  const generateButton = wrapper
    .findAll('button')
    .find((button) => button.text() === 'admin.accounts.grokOauth.generate')
  await generateButton!.trigger('click')
  await flushPromises()
}

describe('AccountGrokOAuthPanel', () => {
  beforeEach(() => {
    vi.useRealTimers()
    vi.clearAllMocks()
    generateGrokAuthUrl.mockResolvedValue({
      auth_url: 'https://auth.x.ai/oauth2/authorize?state=state-1',
      session_id: 'session-1',
      redirect_uri: 'http://127.0.0.1:56121/callback',
      state: 'state-1'
    })
    startGrokDeviceFlow.mockResolvedValue({
      session_id: 'device-session',
      user_code: 'ABCD-EFGH',
      verification_uri: 'https://auth.x.ai/activate',
      verification_uri_complete: 'https://auth.x.ai/activate?user_code=ABCD-EFGH',
      interval: 2,
      expires_at: Math.floor(Date.now() / 1000) + 600
    })
  })

  it('resets stale callback state after Grok OAuth exchange fails', async () => {
    exchangeGrokAuthCode.mockRejectedValue(new Error('authorization code expired or already used'))

    const wrapper = mountPanel()
    await generateAuth(wrapper)

    await wrapper.find('textarea').setValue('http://127.0.0.1:56121/callback?code=auth-code&state=state-1')
    await wrapper.get('[data-testid="grok-oauth-submit"]').trigger('click')
    await flushPromises()

    expect(wrapper.find('textarea').element.value).toBe('')
    expect(wrapper.text()).toContain('authorization code expired or already used')
    expect(wrapper.text()).not.toContain('auth-code')
    expect(wrapper.get('[data-testid="grok-oauth-submit"]').attributes('disabled')).toBeDefined()
  })

  it('generates auth URL and emits exchanged Grok OAuth credentials', async () => {
    exchangeGrokAuthCode.mockResolvedValue({
      access_token: 'at',
      refresh_token: 'rt',
      token_type: 'Bearer',
      expires_at: 1798761600,
      base_url: 'https://api.x.ai/v1',
      email: 'grok@example.com',
      subject: 'user-1',
      name: 'Grok User'
    })

    const wrapper = mountPanel()
    await generateAuth(wrapper)

    expect(generateGrokAuthUrl).toHaveBeenCalledWith({ proxy_id: 7 })

    await wrapper.find('textarea').setValue('http://127.0.0.1:56121/callback?code=auth-code&state=state-1')
    await wrapper.get('[data-testid="grok-oauth-submit"]').trigger('click')
    await flushPromises()

    expect(exchangeGrokAuthCode).toHaveBeenCalledWith({
      session_id: 'session-1',
      code: 'auth-code',
      state: 'state-1',
      proxy_id: 7
    })
    expect(wrapper.emitted('submit')?.[0]?.[0]).toEqual({
      credentials: {
        access_token: 'at',
        refresh_token: 'rt',
        token_type: 'Bearer',
        expires_at: 1798761600,
        base_url: 'https://api.x.ai/v1',
        email: 'grok@example.com',
        subject: 'user-1',
        name: 'Grok User'
      },
      extra: {
        provider: 'xai',
        source: 'grok_browser_oauth',
        email: 'grok@example.com',
        subject: 'user-1',
        display_name: 'Grok User'
      },
      suggestedName: 'grok@example.com'
    })
  })

  it('recognizes external device short codes and does not exchange them as auth codes', async () => {
    const wrapper = mountPanel()
    await generateAuth(wrapper)

    await wrapper.find('textarea').setValue('ABCD-EFGH')
    await flushPromises()

    expect(wrapper.text()).toContain('admin.accounts.grokOauth.externalDeviceCodeHint')
    expect(exchangeGrokAuthCode).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('admin.accounts.grokOauth.deviceMode')
  })

  it('starts device flow and emits credentials after pending and slow_down polling', async () => {
    vi.useFakeTimers()
    pollGrokDeviceToken
      .mockResolvedValueOnce({
        status: 'pending',
        interval: 2,
        expires_at: Math.floor(Date.now() / 1000) + 600,
        error_message: 'pending'
      })
      .mockResolvedValueOnce({
        status: 'slow_down',
        interval: 7,
        expires_at: Math.floor(Date.now() / 1000) + 600,
        error_message: 'slow down'
      })
      .mockResolvedValueOnce({
        status: 'authorized',
        interval: 7,
        expires_at: Math.floor(Date.now() / 1000) + 600,
        token_info: {
          access_token: 'device-at',
          refresh_token: 'device-rt',
          base_url: 'https://api.x.ai/v1',
          email: 'device@example.com'
        }
      })

    const wrapper = mountPanel()
    await wrapper.findAll('button').find((button) => button.text() === 'admin.accounts.grokOauth.deviceMode')!.trigger('click')
    await wrapper.get('[data-testid="grok-device-start"]').trigger('click')
    await flushPromises()

    expect(startGrokDeviceFlow).toHaveBeenCalledWith({ proxy_id: 7 })
    expect(wrapper.get('[data-testid="grok-device-user-code"]').text()).toBe('ABCD-EFGH')
    expect(wrapper.get('[data-testid="grok-device-countdown"]').text()).toMatch(/\d{2}:\d{2}/)

    await wrapper.get('[data-testid="grok-device-poll"]').trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('pending')

    await wrapper.get('[data-testid="grok-device-poll"]').trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('slow down')
    expect(wrapper.text()).toContain('poll every 7s')

    await wrapper.get('[data-testid="grok-device-poll"]').trigger('click')
    await flushPromises()

    expect(wrapper.emitted('submit')?.[0]?.[0]).toMatchObject({
      credentials: {
        access_token: 'device-at',
        refresh_token: 'device-rt',
        base_url: 'https://api.x.ai/v1',
        email: 'device@example.com'
      },
      suggestedName: 'device@example.com'
    })
    vi.useRealTimers()
  })

  it('stops device polling on expired status', async () => {
    pollGrokDeviceToken.mockResolvedValue({
      status: 'expired',
      interval: 2,
      expires_at: Math.floor(Date.now() / 1000) + 600,
      error_message: 'expired in test'
    })

    const wrapper = mountPanel()
    await wrapper.findAll('button').find((button) => button.text() === 'admin.accounts.grokOauth.deviceMode')!.trigger('click')
    await wrapper.get('[data-testid="grok-device-start"]').trigger('click')
    await flushPromises()

    await wrapper.get('[data-testid="grok-device-poll"]').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('expired in test')
    expect(wrapper.get('[data-testid="grok-device-poll"]').attributes('disabled')).toBeDefined()
    expect(wrapper.emitted('submit')).toBeUndefined()
  })
})
