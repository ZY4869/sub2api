import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import AuthIdentitiesCard from './AuthIdentitiesCard.vue'

const mocks = vi.hoisted(() => ({
  startSocialOAuth: vi.fn(),
  deleteAuthIdentity: vi.fn(),
  appStore: {
    showSuccess: vi.fn(),
    showError: vi.fn(),
  },
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

vi.mock('@/api/auth', () => ({
  startSocialOAuth: mocks.startSocialOAuth,
}))

vi.mock('@/api', () => ({
  userAPI: {
    deleteAuthIdentity: mocks.deleteAuthIdentity,
  },
}))

vi.mock('@/stores', () => ({
  useAppStore: () => mocks.appStore,
}))

describe('AuthIdentitiesCard', () => {
  const originalLocation = window.location

  beforeEach(() => {
    mocks.startSocialOAuth.mockReset()
    mocks.startSocialOAuth.mockResolvedValue({ authorize_url: 'https://oauth.example/bind' })
    mocks.deleteAuthIdentity.mockReset()
    mocks.appStore.showSuccess.mockReset()
    mocks.appStore.showError.mockReset()
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: { ...originalLocation, href: '' },
    })
  })

  afterEach(() => {
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: originalLocation,
    })
  })

  it('starts bind flow for GitHub', async () => {
    const wrapper = mount(AuthIdentitiesCard, {
      props: {
        identities: [],
        githubEnabled: true,
        captchaProvider: 'turnstile',
        turnstileSiteKey: 'site-key',
      },
      global: {
        stubs: {
          CaptchaChallenge: {
            template: '<div data-test="captcha" @click="$emit(\'verify\', { turnstile_token: \'profile-proof\' })" />',
          },
          LobeStaticIcon: { template: '<span />' },
        },
      },
    })

    const bindButton = wrapper.findAll('button')[0]
    expect((bindButton.element as HTMLButtonElement).disabled).toBe(true)

    await wrapper.get('[data-test="captcha"]').trigger('click')
    await bindButton.trigger('click')
    await flushPromises()

    expect(mocks.startSocialOAuth).toHaveBeenCalledWith('github', {
      turnstile_token: 'profile-proof',
      mode: 'bind',
      redirect: '/profile',
    })
    expect(window.location.href).toBe('https://oauth.example/bind')
  })

  it('unbinds identity and emits refresh on success', async () => {
    mocks.deleteAuthIdentity.mockResolvedValue({ message: 'ok' })

    const wrapper = mount(AuthIdentitiesCard, {
      props: {
        identities: [
          {
            id: 1,
            provider: 'github',
            provider_user_id: 'gh-1',
            email: 'alice@example.com',
            email_verified: true,
            display_name: 'Alice',
            avatar_url: '',
            created_at: '',
            updated_at: '',
          },
        ],
      },
      global: {
        stubs: {
          LobeStaticIcon: { template: '<span />' },
        },
      },
    })

    await wrapper.get('button.btn-sm').trigger('click')
    await flushPromises()

    expect(mocks.deleteAuthIdentity).toHaveBeenCalledWith('github')
    expect(mocks.appStore.showSuccess).toHaveBeenCalledWith('profile.identities.unbindSuccess')
    expect(wrapper.emitted('refresh')).toHaveLength(1)
  })
})
