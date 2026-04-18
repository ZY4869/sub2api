import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import PublicModelsView from '../PublicModelsView.vue'

const mockState = vi.hoisted(() => ({
  router: {
    replace: vi.fn(),
  },
  route: {
    path: '/models',
    fullPath: '/models',
  },
  authStore: {
    isAuthenticated: false,
  },
  appStore: {
    publicSettingsLoaded: true,
    publicModelCatalogEnabled: true,
    fetchPublicSettings: vi.fn(),
    siteName: 'Sub2API',
    siteLogo: '/logo.png',
  },
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => mockState.authStore,
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => mockState.appStore,
}))

vi.mock('vue-router', () => ({
  useRouter: () => mockState.router,
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

function mountView() {
  return mount(PublicModelsView, {
    global: {
      stubs: {
        AppLayout: { template: '<div data-test="app-layout"><slot /></div>' },
        PublicModelCatalogContent: { template: '<div data-test="catalog-content" />' },
        'router-link': { template: '<a data-test="login-link"><slot /></a>' },
      },
    },
  })
}

describe('PublicModelsView', () => {
  beforeEach(() => {
    mockState.router.replace.mockReset()
    mockState.route.path = '/models'
    mockState.route.fullPath = '/models'
    mockState.authStore.isAuthenticated = false
    mockState.appStore.publicSettingsLoaded = true
    mockState.appStore.publicModelCatalogEnabled = true
    mockState.appStore.fetchPublicSettings.mockReset()
    mockState.appStore.siteName = 'Sub2API'
    mockState.appStore.siteLogo = '/logo.png'
  })

  it('renders the public shell and sign-in entry for guests', () => {
    const wrapper = mountView()

    expect(wrapper.find('[data-test="app-layout"]').exists()).toBe(false)
    expect(wrapper.find('[data-test="catalog-content"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="login-link"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('Sub2API')
    expect(wrapper.text()).toContain('auth.signIn')
    expect(mockState.router.replace).not.toHaveBeenCalled()
  })

  it('renders the shared catalog content inside AppLayout for authenticated users', () => {
    mockState.authStore.isAuthenticated = true

    const wrapper = mountView()

    expect(wrapper.find('[data-test="app-layout"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="catalog-content"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="login-link"]').exists()).toBe(false)
    expect(mockState.router.replace).not.toHaveBeenCalled()
  })

  it('redirects guests to login when public model catalog is disabled', () => {
    mockState.appStore.publicModelCatalogEnabled = false

    mountView()

    expect(mockState.router.replace).toHaveBeenCalledWith({
      path: '/login',
      query: {
        redirect: '/models',
      },
    })
  })
})
