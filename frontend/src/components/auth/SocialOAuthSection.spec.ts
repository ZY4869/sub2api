import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import SocialOAuthSection from './SocialOAuthSection.vue'

const mockState = vi.hoisted(() => ({
  startSocialOAuth: vi.fn(),
  route: {
    query: {},
  },
}))

vi.mock('vue-router', () => ({
  useRoute: () => mockState.route,
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
  startSocialOAuth: mockState.startSocialOAuth,
}))

describe('SocialOAuthSection', () => {
  const originalLocation = window.location

  beforeEach(() => {
    mockState.startSocialOAuth.mockReset()
    mockState.startSocialOAuth.mockResolvedValue({ authorize_url: 'https://oauth.example/start' })
    mockState.route.query = {}
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

  it('starts GitHub social login with redirect and mode', async () => {
    mockState.route.query = { aff_code: 'AFF-123' }
    mockState.startSocialOAuth.mockResolvedValue({ authorize_url: 'https://oauth.example/github' })
    const wrapper = mount(SocialOAuthSection, {
      props: {
        showGitHub: true,
        mode: 'bind',
        redirect: '/profile',
        captchaProvider: 'turnstile',
        turnstileSiteKey: 'site-key',
      },
      global: {
        stubs: {
          CaptchaChallenge: {
            template: '<div data-test="captcha" @click="$emit(\'verify\', { turnstile_token: \'proof-token\' })" />',
          },
          LobeStaticIcon: { template: '<span data-test="icon" />' },
        },
      },
    })

    const oauthButton = wrapper.findAll('button')[0]
    expect((oauthButton.element as HTMLButtonElement).disabled).toBe(true)

    await wrapper.get('[data-test="captcha"]').trigger('click')
    await oauthButton.trigger('click')
    await flushPromises()

    expect(mockState.startSocialOAuth).toHaveBeenCalledWith('github', {
      turnstile_token: 'proof-token',
      mode: 'bind',
      redirect: '/profile',
      aff_code: 'AFF-123',
    })
    expect(window.location.href).toBe('https://oauth.example/github')
  })

  it('uses route redirect fallback for Google login', async () => {
    mockState.route.query = { redirect: '/workspace' }
    mockState.startSocialOAuth.mockResolvedValue({ authorize_url: 'https://oauth.example/google' })

    const wrapper = mount(SocialOAuthSection, {
      props: {
        showGoogle: true,
      },
      global: {
        stubs: {
          LobeStaticIcon: { template: '<span data-test="icon" />' },
        },
      },
    })

    await wrapper.get('button').trigger('click')
    await flushPromises()

    expect(mockState.startSocialOAuth).toHaveBeenCalledWith('google', {
      mode: 'login',
      redirect: '/workspace',
      aff_code: undefined,
    })
    expect(window.location.href).toBe('https://oauth.example/google')
  })
})
