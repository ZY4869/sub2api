import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AccountKiroAuthPanel from '../AccountKiroAuthPanel.vue'

const { generateKiroAuthUrl, exchangeKiroAuthCode, copyToClipboard } = vi.hoisted(() => ({
  generateKiroAuthUrl: vi.fn(),
  exchangeKiroAuthCode: vi.fn(),
  copyToClipboard: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      generateKiroAuthUrl,
      exchangeKiroAuthCode
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

describe('AccountKiroAuthPanel', () => {
  it('generates oauth state and emits exchanged tokens with membership metadata', async () => {
    generateKiroAuthUrl.mockResolvedValue({
      auth_url: 'https://kiro.example.com/auth',
      session_id: 'session-1',
      redirect_uri: 'http://127.0.0.1:19877/oauth/callback',
      state: 'state-1'
    })
    exchangeKiroAuthCode.mockResolvedValue({
      access_token: 'at',
      refresh_token: 'rt',
      auth_method: 'builder_id',
      provider: 'aws',
      email: 'kiro@example.com',
      username: 'kiro-user'
    })

    const wrapper = mount(AccountKiroAuthPanel, {
      props: {
        submitLabel: '授权',
        proxyId: 7,
        initialExtra: {
          kiro_member_level: 'kiro_free',
          kiro_member_credits: 50
        }
      }
    })

    const generateButton = wrapper
      .findAll('button')
      .find((button) => button.text() === 'admin.accounts.kiroAuth.generate')
    expect(generateButton).toBeTruthy()
    await generateButton!.trigger('click')
    await flushPromises()

    expect(generateKiroAuthUrl).toHaveBeenCalledWith({
      proxy_id: 7,
      method: 'builder_id',
      start_url: undefined,
      region: 'us-east-1'
    })

    await wrapper.find('select').setValue('kiro_power')
    await wrapper.find('input[type="number"]').setValue('8888')
    await wrapper.find('textarea').setValue('http://127.0.0.1:19877/oauth/callback?code=auth-code&state=state-1')
    const submitButton = wrapper
      .findAll('button')
      .find((button) => button.text() === '授权')
    expect(submitButton).toBeTruthy()
    await submitButton!.trigger('click')
    await flushPromises()

    expect(exchangeKiroAuthCode).toHaveBeenCalledWith({
      session_id: 'session-1',
      code: 'auth-code',
      state: 'state-1',
      proxy_id: 7
    })
    expect(wrapper.emitted('submit')?.[0]?.[0]).toEqual({
      credentials: {
        access_token: 'at',
        refresh_token: 'rt',
        auth_method: 'builder_id'
      },
      extra: {
        source: 'kiro_browser_oauth',
        provider: 'aws',
        email: 'kiro@example.com',
        username: 'kiro-user',
        kiro_member_level: 'kiro_power',
        kiro_member_credits: 8888
      },
      suggestedName: 'kiro@example.com'
    })
  })
})
