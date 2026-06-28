import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AccountGrokOAuthPanel from '../AccountGrokOAuthPanel.vue'

const { generateGrokAuthUrl, exchangeGrokAuthCode, copyToClipboard } = vi.hoisted(() => ({
  generateGrokAuthUrl: vi.fn(),
  exchangeGrokAuthCode: vi.fn(),
  copyToClipboard: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      generateGrokAuthUrl,
      exchangeGrokAuthCode
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
      t: (key: string) => key
    })
  }
})

describe('AccountGrokOAuthPanel', () => {
  it('generates auth URL and emits exchanged Grok OAuth credentials', async () => {
    generateGrokAuthUrl.mockResolvedValue({
      auth_url: 'https://auth.x.ai/oauth2/authorize?state=state-1',
      session_id: 'session-1',
      redirect_uri: 'http://127.0.0.1:56121/callback',
      state: 'state-1'
    })
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

    const wrapper = mount(AccountGrokOAuthPanel, {
      props: {
        submitLabel: '创建',
        proxyId: 7
      }
    })

    const generateButton = wrapper
      .findAll('button')
      .find((button) => button.text() === 'admin.accounts.grokOauth.generate')
    expect(generateButton).toBeTruthy()
    await generateButton!.trigger('click')
    await flushPromises()

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
})
