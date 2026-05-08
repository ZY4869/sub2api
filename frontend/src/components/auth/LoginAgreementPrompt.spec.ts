import { RouterLinkStub, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import LoginAgreementPrompt from './LoginAgreementPrompt.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

describe('LoginAgreementPrompt', () => {
  it('renders published agreement links and emits acceptance changes', async () => {
    const wrapper = mount(LoginAgreementPrompt, {
      props: {
        enabled: true,
        accepted: false,
        error: 'auth.agreementRequired',
        documents: [
          { id: 'terms', title: 'Terms', page_slug: 'terms' },
          { id: 'privacy', title: 'Privacy', page_slug: 'privacy' },
        ],
      },
      global: {
        stubs: {
          RouterLink: RouterLinkStub,
        },
      },
    })

    expect(wrapper.text()).toContain('auth.agreementPrefix')
    expect(wrapper.text()).toContain('Terms')
    expect(wrapper.text()).toContain('Privacy')
    expect(wrapper.text()).toContain('auth.agreementRequired')

    const links = wrapper.findAllComponents(RouterLinkStub)
    expect(links).toHaveLength(2)
    expect(links[0].props('to')).toBe('/legal/terms')
    expect(links[1].props('to')).toBe('/legal/privacy')

    await wrapper.get('input[type="checkbox"]').setValue(true)
    expect(wrapper.emitted('update:accepted')).toEqual([[true]])
  })

  it('does not render when disabled', () => {
    const wrapper = mount(LoginAgreementPrompt, {
      props: {
        enabled: false,
        accepted: false,
        documents: [],
      },
    })

    expect(wrapper.find('input[type="checkbox"]').exists()).toBe(false)
    expect(wrapper.findComponent(RouterLinkStub).exists()).toBe(false)
  })
})
