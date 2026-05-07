import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import AuthIdentitiesCard from './AuthIdentitiesCard.vue'

const mocks = vi.hoisted(() => ({
  buildSocialOAuthStartURL: vi.fn((provider: string, options?: Record<string, string>) => {
    const params = new URLSearchParams(options as Record<string, string>)
    return `/api/v1/auth/oauth/${provider}/start?${params.toString()}`
  }),
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
  buildSocialOAuthStartURL: mocks.buildSocialOAuthStartURL,
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
    mocks.buildSocialOAuthStartURL.mockReset()
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
      },
      global: {
        stubs: {
          LobeStaticIcon: { template: '<span />' },
        },
      },
    })

    ;(wrapper.get('button').element as HTMLButtonElement).click()

    expect(mocks.buildSocialOAuthStartURL).toHaveBeenCalledWith('github', {
      mode: 'bind',
      redirect: '/profile',
    })
    if (typeof window.location.href === 'string') {
      expect(window.location.href).toContain('/api/v1/auth/oauth/github/start')
    }
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
