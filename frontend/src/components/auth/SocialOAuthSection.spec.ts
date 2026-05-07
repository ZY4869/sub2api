import { mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import SocialOAuthSection from './SocialOAuthSection.vue'

const mockState = vi.hoisted(() => ({
  buildSocialOAuthStartURL: vi.fn((provider: string, options?: Record<string, string>) => {
    const params = new URLSearchParams(options as Record<string, string>)
    return `/api/v1/auth/oauth/${provider}/start?${params.toString()}`
  }),
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
  buildSocialOAuthStartURL: mockState.buildSocialOAuthStartURL,
}))

describe('SocialOAuthSection', () => {
  const originalLocation = window.location

  beforeEach(() => {
    mockState.buildSocialOAuthStartURL.mockClear()
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
    const wrapper = mount(SocialOAuthSection, {
      props: {
        showGitHub: true,
        mode: 'bind',
        redirect: '/profile',
      },
      global: {
        stubs: {
          LobeStaticIcon: { template: '<span data-test="icon" />' },
        },
      },
    })

    ;(wrapper.get('button').element as HTMLButtonElement).click()

    expect(mockState.buildSocialOAuthStartURL).toHaveBeenCalledWith('github', {
      mode: 'bind',
      redirect: '/profile',
    })
    expect(window.location.href).toContain('/api/v1/auth/oauth/github/start')
  })

  it('uses route redirect fallback for Google login', async () => {
    mockState.route.query = { redirect: '/workspace' }

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

    ;(wrapper.get('button').element as HTMLButtonElement).click()

    expect(mockState.buildSocialOAuthStartURL).toHaveBeenCalledWith('google', {
      mode: 'login',
      redirect: '/workspace',
    })
    expect(window.location.href).toContain('/api/v1/auth/oauth/google/start')
  })
})
