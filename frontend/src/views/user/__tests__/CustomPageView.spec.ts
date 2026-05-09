import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import CustomPageView from '../CustomPageView.vue'

const routeState = vi.hoisted(() => ({
  id: 'page-1',
}))

const testState = vi.hoisted(() => ({
  getCustomPageMock: vi.fn(),
  appStoreState: {
    publicSettingsLoaded: true,
    fetchPublicSettings: vi.fn(),
    cachedPublicSettings: {
      custom_menu_items: [] as Array<Record<string, unknown>>,
    },
  },
  authStoreState: {
    isAdmin: false,
    user: { id: 7 },
  },
  adminSettingsStoreState: {
    loaded: true,
    customMenuItems: [] as Array<Record<string, unknown>>,
    fetch: vi.fn(),
  },
}))

vi.mock('vue-router', async () => {
  const actual = await vi.importActual<typeof import('vue-router')>('vue-router')
  return {
    ...actual,
    useRoute: () => ({
      params: {
        id: routeState.id,
      },
    }),
  }
})

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      locale: { value: 'zh-CN' },
      t: (key: string) => key,
    }),
  }
})

vi.mock('@/api', () => ({
  pagesAPI: {
    getCustomPage: testState.getCustomPageMock,
  },
}))

vi.mock('@/stores', () => ({
  useAppStore: () => testState.appStoreState,
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => testState.authStoreState,
}))

vi.mock('@/stores/adminSettings', () => ({
  useAdminSettingsStore: () => testState.adminSettingsStoreState,
}))

describe('CustomPageView', () => {
  beforeEach(() => {
    routeState.id = 'page-1'
    testState.getCustomPageMock.mockReset()
    testState.appStoreState.fetchPublicSettings.mockReset()
    testState.adminSettingsStoreState.fetch.mockReset()
    testState.appStoreState.cachedPublicSettings.custom_menu_items = []
    testState.adminSettingsStoreState.customMenuItems = []
    testState.authStoreState.isAdmin = false
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('loads published markdown pages through the public page api', async () => {
    testState.appStoreState.cachedPublicSettings.custom_menu_items = [
      {
        id: 'page-1',
        label: 'Guide',
        url: '',
        visibility: 'user',
        sort_order: 0,
        page_mode: 'markdown',
        page_slug: 'guide',
      },
    ]
    testState.getCustomPageMock.mockResolvedValue({
      id: 'page-1',
      slug: 'guide',
      label: 'Guide',
      visibility: 'user',
      page_mode: 'markdown',
      content: '# Hello Guide',
    })

    const wrapper = mount(CustomPageView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          Icon: { template: '<span />' },
          CustomMarkdownPageContent: {
            props: ['markdown'],
            template: '<div data-testid="markdown">{{ markdown }}</div>',
          },
        },
      },
    })

    await flushPromises()

    expect(testState.getCustomPageMock).toHaveBeenCalledWith('guide')
    expect(wrapper.get('[data-testid="markdown"]').text()).toContain('# Hello Guide')
  })

  it('uses admin local markdown content without fetching the public page api', async () => {
    testState.authStoreState.isAdmin = true
    testState.appStoreState.cachedPublicSettings.custom_menu_items = []
    testState.adminSettingsStoreState.customMenuItems = [
      {
        id: 'page-1',
        label: 'Draft Guide',
        url: '',
        visibility: 'admin',
        sort_order: 0,
        page_mode: 'markdown',
        page_slug: 'draft-guide',
        page_content: '# Draft Content',
      },
    ]

    const wrapper = mount(CustomPageView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          Icon: { template: '<span />' },
          CustomMarkdownPageContent: {
            props: ['markdown'],
            template: '<div data-testid="markdown">{{ markdown }}</div>',
          },
        },
      },
    })

    await flushPromises()

    expect(testState.getCustomPageMock).not.toHaveBeenCalled()
    expect(wrapper.get('[data-testid="markdown"]').text()).toContain('# Draft Content')
  })

  it('falls back to iframe mode for legacy custom menu urls', async () => {
    testState.appStoreState.cachedPublicSettings.custom_menu_items = [
      {
        id: 'page-1',
        label: 'Legacy Page',
        url: 'https://example.com/help',
        visibility: 'user',
        sort_order: 0,
        page_mode: 'iframe',
      },
    ]

    const wrapper = mount(CustomPageView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          Icon: { template: '<span />' },
          CustomMarkdownPageContent: true,
        },
      },
    })

    await flushPromises()

    expect(testState.getCustomPageMock).not.toHaveBeenCalled()
    const iframe = wrapper.get('iframe')
    expect(iframe.attributes('src')).toContain('https://example.com/help')
    expect(iframe.attributes('src')).toContain('ui_mode=embedded')
    expect(iframe.attributes('src')).not.toContain('user_id=')
    expect(iframe.attributes('src')).not.toContain('token=')
    expect(iframe.attributes('src')).not.toContain('src_url=')
  })
})
