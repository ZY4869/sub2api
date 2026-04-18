import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import AppSidebar from '../AppSidebar.vue'

const mockState = vi.hoisted(() => ({
  routePath: '/dashboard',
  appStore: {
    sidebarCollapsed: false,
    mobileOpen: false,
    siteName: 'Sub2API',
    siteLogo: '',
    siteVersion: '1.0.0',
    publicSettingsLoaded: true,
    backendModeEnabled: false,
    cachedPublicSettings: {
      purchase_subscription_enabled: false,
      custom_menu_items: [],
    },
    toggleSidebar: vi.fn(),
    setMobileOpen: vi.fn(),
  },
  authStore: {
    isAdmin: false,
    canReviewRequestDetails: false,
    isSimpleMode: false,
  },
  adminSettingsStore: {
    opsMonitoringEnabled: false,
    customMenuItems: [],
    fetch: vi.fn(),
  },
  onboardingStore: {
    isCurrentStep: vi.fn(() => false),
    nextStep: vi.fn(),
  },
}))

vi.mock('vue-router', async () => {
  const actual = await vi.importActual<typeof import('vue-router')>('vue-router')
  return {
    ...actual,
    useRoute: () => ({
      path: mockState.routePath,
    }),
  }
})

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

vi.mock('@/stores', () => ({
  useAppStore: () => mockState.appStore,
  useAuthStore: () => mockState.authStore,
  useAdminSettingsStore: () => mockState.adminSettingsStore,
  useOnboardingStore: () => mockState.onboardingStore,
}))

vi.mock('@/utils/sanitize', () => ({
  sanitizeSvg: (value: string) => value,
}))

const RouterLinkStub = {
  props: ['to'],
  template: '<a :href="to" v-bind="$attrs"><slot /></a>',
}

describe('AppSidebar', () => {
  beforeEach(() => {
    mockState.routePath = '/dashboard'
    mockState.authStore.isAdmin = false
    mockState.authStore.canReviewRequestDetails = false
    mockState.authStore.isSimpleMode = false
    mockState.appStore.backendModeEnabled = false
    mockState.adminSettingsStore.fetch.mockReset()
    mockState.onboardingStore.isCurrentStep.mockReset()
    mockState.onboardingStore.isCurrentStep.mockReturnValue(false)
    mockState.onboardingStore.nextStep.mockReset()

    vi.stubGlobal('matchMedia', vi.fn(() => ({
      matches: false,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      addListener: vi.fn(),
      removeListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })))
  })

  it('shows the API docs entry for regular users', () => {
    mockState.routePath = '/api-docs/common'

    const wrapper = mount(AppSidebar, {
      global: {
        stubs: {
          'router-link': RouterLinkStub,
          VersionBadge: { template: '<span data-test="version-badge" />' },
        },
      },
    })

    expect(wrapper.text()).toContain('nav.apiDocs')
    expect(wrapper.find('a[href="/api-docs"]').classes()).toContain('sidebar-link-active')
  })

  it('shows the models catalog entry for regular users', () => {
    mockState.routePath = '/models'

    const wrapper = mount(AppSidebar, {
      global: {
        stubs: {
          'router-link': RouterLinkStub,
          VersionBadge: { template: '<span data-test="version-badge" />' },
        },
      },
    })

    expect(wrapper.text()).toContain('nav.modelsCatalog')
    expect(wrapper.find('a[href="/models"]').classes()).toContain('sidebar-link-active')
  })

  it('shows consolidated admin navigation and keeps the nested accounts items out of the top level', () => {
    mockState.authStore.isAdmin = true
    mockState.routePath = '/admin/api-docs/gemini'

    const wrapper = mount(AppSidebar, {
      global: {
        stubs: {
          'router-link': RouterLinkStub,
          VersionBadge: { template: '<span data-test="version-badge" />' },
        },
      },
    })

    expect(wrapper.text()).toContain('nav.accounts')
    expect(wrapper.text()).toContain('nav.apiDocs')
    expect(wrapper.text()).not.toContain('nav.limitedAccounts')
    expect(wrapper.text()).not.toContain('nav.blacklist')
    expect(wrapper.find('a[href="/admin/api-docs"]').classes()).toContain('sidebar-link-active')
    expect(mockState.adminSettingsStore.fetch).toHaveBeenCalled()
  })
})
