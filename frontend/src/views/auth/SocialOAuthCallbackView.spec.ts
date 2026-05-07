import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import SocialOAuthCallbackView from './SocialOAuthCallbackView.vue'

const mocks = vi.hoisted(() => ({
  completeSocialOAuthRegistration: vi.fn(),
  authStore: {
    setToken: vi.fn(),
  },
  appStore: {
    showSuccess: vi.fn(),
    showError: vi.fn(),
  },
  router: {
    replace: vi.fn(),
  },
}))

vi.mock('vue-router', () => ({
  useRouter: () => mocks.router,
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
  completeSocialOAuthRegistration: mocks.completeSocialOAuthRegistration,
}))

vi.mock('@/stores', () => ({
  useAuthStore: () => mocks.authStore,
  useAppStore: () => mocks.appStore,
}))

describe('SocialOAuthCallbackView', () => {
  beforeEach(() => {
    mocks.completeSocialOAuthRegistration.mockReset()
    mocks.authStore.setToken.mockReset()
    mocks.appStore.showSuccess.mockReset()
    mocks.appStore.showError.mockReset()
    mocks.router.replace.mockReset()
    localStorage.clear()
  })

  afterEach(() => {
    window.location.hash = ''
  })

  it('completes invitation flow and redirects to requested path', async () => {
    window.location.hash =
      '#error=invitation_required&pending_oauth_token=pending-1&provider=github&mode=login&redirect=%2Fworkspace'
    mocks.completeSocialOAuthRegistration.mockResolvedValue({
      access_token: 'access-1',
      refresh_token: 'refresh-1',
      expires_in: 3600,
      token_type: 'Bearer',
    })

    const wrapper = mount(SocialOAuthCallbackView, {
      global: {
        stubs: {
          AuthLayout: { template: '<div><slot /></div>' },
        },
      },
    })

    await flushPromises()
    await wrapper.get('input').setValue('INVITE-CODE')
    await wrapper.get('button').trigger('click')
    await flushPromises()

    expect(mocks.completeSocialOAuthRegistration).toHaveBeenCalledWith('github', 'pending-1', 'INVITE-CODE')
    expect(mocks.authStore.setToken).toHaveBeenCalledWith('access-1')
    expect(localStorage.getItem('refresh_token')).toBe('refresh-1')
    expect(localStorage.getItem('token_expires_at')).not.toBeNull()
    expect(mocks.appStore.showSuccess).toHaveBeenCalledWith('auth.loginSuccess')
    expect(mocks.router.replace).toHaveBeenCalledWith('/workspace')
  })

  it('shows callback error from fragment without attempting login', async () => {
    window.location.hash = '#error=invalid_state&error_description=bad%20state'

    mount(SocialOAuthCallbackView, {
      global: {
        stubs: {
          AuthLayout: { template: '<div><slot /></div>' },
        },
      },
    })

    await flushPromises()

    expect(mocks.completeSocialOAuthRegistration).not.toHaveBeenCalled()
    expect(mocks.appStore.showError).toHaveBeenCalledWith('bad state')
    expect(mocks.router.replace).not.toHaveBeenCalled()
  })
})
