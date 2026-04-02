import { flushPromises, mount } from '@vue/test-utils'
import { ref } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import OAuthCallbackView from './OAuthCallbackView.vue'

const copyToClipboard = vi.fn()

vi.mock('vue-router', () => ({
  useRoute: () => ({
    query: {
      code: 'sample-code',
      state: 'sample-state'
    }
  })
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
      locale: ref('zh'),
      t: (key: string) => ({
        'ui.oauthCallback.title': '\u6388\u6743\u56de\u8c03',
        'ui.oauthCallback.description': '\u8bf7\u5c06 code \u548c state \u590d\u5236\u56de\u7ba1\u7406\u7aef\u6388\u6743\u6d41\u7a0b\u3002',
        'ui.oauthCallback.copy': '\u590d\u5236',
        'auth.oauth.code': '\u6388\u6743\u7801',
        'auth.oauth.state': '\u72b6\u6001\u53c2\u6570',
        'auth.oauth.fullUrl': '\u5b8c\u6574\u5730\u5740',
        'common.copied': '\u5df2\u590d\u5236'
      }[key] || key)
    })
  }
})

describe('OAuthCallbackView', () => {
  it('renders localized copy guidance and uses localized toast text', async () => {
    window.history.pushState({}, '', '/auth/callback?code=sample-code&state=sample-state')

    const wrapper = mount(OAuthCallbackView)

    expect(wrapper.text()).toContain('\u6388\u6743\u56de\u8c03')
    expect(wrapper.text()).toContain('\u8bf7\u5c06 code \u548c state \u590d\u5236\u56de\u7ba1\u7406\u7aef\u6388\u6743\u6d41\u7a0b\u3002')

    await wrapper.find('button').trigger('click')
    await flushPromises()

    expect(copyToClipboard).toHaveBeenCalledWith('sample-code', '\u5df2\u590d\u5236')
  })
})
