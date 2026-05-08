import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import RegisterView from './RegisterView.vue'

const mockGetPublicSettings = vi.fn()
const mockValidatePromoCode = vi.fn()
const mockValidateInvitationCode = vi.fn()
const mockStoreRegister = vi.fn()
const mockRouterPush = vi.fn()
const mockShowSuccess = vi.fn()
const mockShowError = vi.fn()

vi.mock('vue-router', () => ({
  RouterLink: {
    props: ['to'],
    template: '<a :href="to"><slot /></a>',
  },
  useRouter: () => ({
    push: mockRouterPush,
  }),
  useRoute: () => ({
    query: {},
  }),
}))

vi.mock('@/api/auth', () => ({
  getPublicSettings: (...args: any[]) => mockGetPublicSettings(...args),
  validatePromoCode: (...args: any[]) => mockValidatePromoCode(...args),
  validateInvitationCode: (...args: any[]) => mockValidateInvitationCode(...args),
}))

vi.mock('@/stores', () => ({
  useAuthStore: () => ({
    register: mockStoreRegister,
  }),
  useAppStore: () => ({
    showSuccess: mockShowSuccess,
    showError: mockShowError,
  }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, string>) =>
        params?.siteName ? `${key}:${params.siteName}` : key,
      locale: { value: 'zh-CN' },
    }),
  }
})

const stubs = {
  AuthLayout: {
    template: '<div><slot /><slot name="footer" /></div>',
  },
  AuthMaintenanceNotice: true,
  SocialOAuthSection: true,
  Icon: true,
  TurnstileWidget: true,
  'router-link': {
    props: ['to'],
    template: '<a :href="to"><slot /></a>',
  },
}

function createAgreementSettings(enabled: boolean) {
  return {
    registration_enabled: true,
    email_verify_enabled: false,
    promo_code_enabled: false,
    invitation_code_enabled: false,
    affiliate_enabled: false,
    turnstile_enabled: false,
    turnstile_site_key: '',
    site_name: 'Sub2API',
    linuxdo_oauth_enabled: false,
    github_oauth_enabled: false,
    google_oauth_enabled: false,
    maintenance_mode_enabled: false,
    registration_email_suffix_whitelist: [],
    login_agreement_enabled: enabled,
    login_agreement_mode: 'checkbox',
    login_agreement_updated_at: '2026-05-08',
    login_agreement_documents: enabled
      ? [{ id: 'terms', title: 'Terms', page_slug: 'terms' }]
      : [],
  }
}

describe('RegisterView agreement gating', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    mockValidatePromoCode.mockResolvedValue({ valid: true })
    mockValidateInvitationCode.mockResolvedValue({ valid: true })
  })

  it('blocks register until agreement is accepted', async () => {
    mockGetPublicSettings.mockResolvedValue(createAgreementSettings(true))
    mockStoreRegister.mockResolvedValue({
      id: 1,
      email: 'user@example.com',
      username: 'user',
      role: 'user',
      balance: 0,
      concurrency: 1,
      status: 'active',
      allowed_groups: null,
      created_at: '',
      updated_at: '',
    })

    const wrapper = mount(RegisterView, {
      global: {
        stubs,
      },
    })
    await flushPromises()

    expect(wrapper.get('button[type="submit"]').attributes('disabled')).toBeDefined()
    expect(wrapper.find('a[href="/legal/terms"]').exists()).toBe(true)

    await wrapper.get('#email').setValue('user@example.com')
    await wrapper.get('#password').setValue('password123')
    await wrapper.get('form').trigger('submit.prevent')
    await flushPromises()

    expect(mockStoreRegister).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('auth.agreementRequired')

    await wrapper.get('input[type="checkbox"]').setValue(true)
    expect(wrapper.get('button[type="submit"]').attributes('disabled')).toBeUndefined()

    await wrapper.get('form').trigger('submit.prevent')
    await flushPromises()

    expect(mockStoreRegister).toHaveBeenCalledWith({
      email: 'user@example.com',
      password: 'password123',
      turnstile_token: undefined,
      promo_code: undefined,
      invitation_code: undefined,
      aff_code: undefined,
    })
  })

  it('does not render agreement prompt when not enabled', async () => {
    mockGetPublicSettings.mockResolvedValue(createAgreementSettings(false))

    const wrapper = mount(RegisterView, {
      global: {
        stubs,
      },
    })
    await flushPromises()

    expect(wrapper.find('input[type="checkbox"]').exists()).toBe(false)
    expect(wrapper.find('a[href^="/legal/"]').exists()).toBe(false)
  })
})
