import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import OAuthAuthorizationFlow from '../OAuthAuthorizationFlow.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) =>
        key === 'admin.accounts.oauth.batchCreateAccounts'
          ? `batch:${String(params?.count ?? '')}`
          : key,
    }),
  }
})

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copied: false,
    copyToClipboard: vi.fn(),
  }),
}))

describe('OAuthAuthorizationFlow', () => {
  it('counts unique refresh tokens for batch import hints', async () => {
    const wrapper = mount(OAuthAuthorizationFlow, {
      props: {
        addMethod: 'oauth',
        showRefreshTokenOption: true,
        showCookieOption: false,
        platform: 'openai',
      },
      global: {
        stubs: {
          Icon: true,
        },
      },
    })

    const refreshRadio = wrapper.find('input[type="radio"][value="refresh_token"]')
    await refreshRadio.setValue(true)

    const refreshTextarea = wrapper.find('textarea')
    await refreshTextarea.setValue('rt_dup\nrt_dup\r\n  rt_other  \nrt_dup')

    expect(wrapper.text()).toContain('batch:2')
  })

  it('counts unique session keys for cookie auth batch hints', async () => {
    const wrapper = mount(OAuthAuthorizationFlow, {
      props: {
        addMethod: 'oauth',
        showRefreshTokenOption: false,
        showCookieOption: true,
        allowMultiple: true,
        platform: 'anthropic',
      },
      global: {
        stubs: {
          Icon: true,
        },
      },
    })

    const cookieRadio = wrapper.find('input[type="radio"][value="cookie"]')
    await cookieRadio.setValue(true)

    const sessionTextarea = wrapper.find('textarea')
    await sessionTextarea.setValue('sk-1\nsk-1\r\n  sk-2  ')

    expect(wrapper.text()).toContain('batch:2')
  })
})
