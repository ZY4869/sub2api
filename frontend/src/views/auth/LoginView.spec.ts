import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import LoginView from './LoginView.vue'

const mockGetPublicSettings = vi.fn()
const mockStoreLogin = vi.fn()
const mockRouterPush = vi.fn()
const mockShowSuccess = vi.fn()
const mockShowError = vi.fn()
const mockShowWarning = vi.fn()

vi.mock('vue-router', () => ({
  RouterLink: {
    props: ['to'],
    template: '<a :href="to"><slot /></a>',
  },
  useRouter: () => ({
    push: mockRouterPush,
    currentRoute: {
      value: {
        query: {},
      },
    },
  }),
}))

vi.mock('@/api/auth', () => ({
  getPublicSettings: (...args: any[]) => mockGetPublicSettings(...args),
  isTotp2FARequired: (response: any) => response?.requires_2fa === true,
}))

vi.mock('@/stores', () => ({
  useAuthStore: () => ({
    login: mockStoreLogin,
    login2FA: vi.fn(),
  }),
  useAppStore: () => ({
    showSuccess: mockShowSuccess,
    showError: mockShowError,
    showWarning: mockShowWarning,
  }),
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

const stubs = {
  AuthLayout: {
    template: '<div><slot /><slot name="footer" /></div>',
  },
  AuthMaintenanceNotice: true,
  SocialOAuthSection: true,
  TotpLoginModal: true,
  Icon: true,
  TurnstileWidget: true,
  'router-link': {
    props: ['to'],
    template: '<a :href="to"><slot /></a>',
  },
}

function createAgreementSettings(enabled: boolean) {
  return {
    turnstile_enabled: false,
    turnstile_site_key: '',
    linuxdo_oauth_enabled: false,
    github_oauth_enabled: false,
    google_oauth_enabled: false,
    backend_mode_enabled: false,
    maintenance_mode_enabled: false,
    password_reset_enabled: true,
    login_agreement_enabled: enabled,
    login_agreement_mode: 'checkbox',
    login_agreement_updated_at: '2026-05-08',
    login_agreement_documents: enabled
      ? [
          { id: 'terms', title: 'Terms', page_slug: 'terms' },
          { id: 'privacy', title: 'Privacy', page_slug: 'privacy' },
        ]
      : [],
  }
}

describe('LoginView agreement gating', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    sessionStorage.clear()
  })

  it('blocks submit until agreement is accepted and links point to legal pages', async () => {
    mockGetPublicSettings.mockResolvedValue(createAgreementSettings(true))
    mockStoreLogin.mockResolvedValue({
      access_token: 'token',
      token_type: 'Bearer',
      user: { id: 1, email: 'user@example.com', username: 'user', role: 'user', balance: 0, concurrency: 1, status: 'active', allowed_groups: null, created_at: '', updated_at: '' },
    })

    const wrapper = mount(LoginView, {
      global: {
        stubs,
      },
    })
    await flushPromises()

    const submit = wrapper.get('button[type="submit"]')
    expect(submit.attributes('disabled')).toBeDefined()

    const links = wrapper.findAll('a[href^="/legal/"]')
    expect(links).toHaveLength(2)
    expect(links[0].attributes('href')).toBe('/legal/terms')
    expect(links[1].attributes('href')).toBe('/legal/privacy')

    await wrapper.get('#email').setValue('user@example.com')
    await wrapper.get('#password').setValue('password123')
    await wrapper.get('form').trigger('submit.prevent')
    await flushPromises()

    expect(mockStoreLogin).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('auth.agreementRequired')

    await wrapper.get('input[type="checkbox"]').setValue(true)
    expect(wrapper.get('button[type="submit"]').attributes('disabled')).toBeUndefined()

    await wrapper.get('form').trigger('submit.prevent')
    await flushPromises()

    expect(mockStoreLogin).toHaveBeenCalledWith({
      email: 'user@example.com',
      password: 'password123',
      turnstile_token: undefined,
    })
  })

  it('does not render agreement prompt when public settings disable it', async () => {
    mockGetPublicSettings.mockResolvedValue(createAgreementSettings(false))

    const wrapper = mount(LoginView, {
      global: {
        stubs,
      },
    })
    await flushPromises()

    expect(wrapper.find('input[type="checkbox"]').exists()).toBe(false)
    expect(wrapper.find('a[href^="/legal/"]').exists()).toBe(false)
  })
})
