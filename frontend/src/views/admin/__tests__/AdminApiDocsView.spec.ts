import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import AdminApiDocsView from '../AdminApiDocsView.vue'

const routeState = vi.hoisted(() => ({
  pageId: 'gemini',
  query: {} as Record<string, unknown>,
}))

const {
  getAPIDocs,
  updateAPIDocs,
  clearAPIDocsOverride,
  copyToClipboard,
  showSuccess,
  showError,
  resolve,
  replace,
} = vi.hoisted(() => ({
  getAPIDocs: vi.fn(),
  updateAPIDocs: vi.fn(),
  clearAPIDocsOverride: vi.fn(),
  copyToClipboard: vi.fn(),
  showSuccess: vi.fn(),
  showError: vi.fn(),
  resolve: vi.fn(() => ({ href: '/api-docs/gemini' })),
  replace: vi.fn((payload: { query?: Record<string, unknown> }) => {
    routeState.query = payload.query || {}
    return Promise.resolve()
  }),
}))

vi.mock('@/api/admin/docs', () => ({
  default: {
    getAPIDocs,
    updateAPIDocs,
    clearAPIDocsOverride,
  },
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard,
  }),
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showSuccess,
    showError,
  }),
}))

vi.mock('vue-router', async () => {
  const actual = await vi.importActual<typeof import('vue-router')>('vue-router')
  return {
    ...actual,
    useRoute: () => ({
      params: {
        pageId: routeState.pageId,
      },
      query: routeState.query,
    }),
    useRouter: () => ({
      resolve,
      replace,
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

const initialDocument = {
  effective_content: '# API Docs\n\n## gemini\n### Model Generation\nGemini page content\n',
  default_content: '# API Docs\n\n## gemini\n### Model Generation\nDefault gemini content\n',
  has_override: true,
}

describe('AdminApiDocsView', () => {
  beforeEach(() => {
    routeState.pageId = 'gemini'
    routeState.query = {}
    getAPIDocs.mockReset()
    updateAPIDocs.mockReset()
    clearAPIDocsOverride.mockReset()
    copyToClipboard.mockReset()
    showSuccess.mockReset()
    showError.mockReset()
    resolve.mockReset()
    replace.mockClear()

    getAPIDocs.mockResolvedValue(initialDocument)
    updateAPIDocs.mockResolvedValue({
      ...initialDocument,
      effective_content: '# API Docs\n\n## gemini\n### Model Generation\nSaved content',
    })
    clearAPIDocsOverride.mockResolvedValue({
      effective_content: initialDocument.default_content,
      default_content: initialDocument.default_content,
      has_override: false,
    })
    copyToClipboard.mockResolvedValue(true)
    resolve.mockReturnValue({ href: '/api-docs/gemini' })

    vi.stubGlobal(
      'IntersectionObserver',
      class {
        observe = vi.fn()
        disconnect = vi.fn()
        unobserve = vi.fn()
      }
    )
    vi.stubGlobal('confirm', vi.fn(() => true))
    vi.stubGlobal('open', vi.fn())
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  function mountView() {
    return mount(AdminApiDocsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          DocsMarkdownContent: { template: '<div data-test="docs-markdown-stub" />' },
        },
      },
    })
  }

  it('defaults to preview tab and syncs tab query to the url', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(getAPIDocs).toHaveBeenCalledWith('gemini')
    expect(replace).toHaveBeenCalledWith(expect.objectContaining({
      query: expect.objectContaining({
        tab: 'preview',
      }),
    }))
    expect(wrapper.find('[data-test="api-docs-preview"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="api-docs-editor"]').exists()).toBe(false)
  })

  it('opens edit tab from query and lets admins save the markdown override', async () => {
    routeState.query = { tab: 'edit' }
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('[data-test="api-docs-editor"]').exists()).toBe(true)

    const textarea = wrapper.get('textarea')
    await textarea.setValue('new markdown')
    await flushPromises()

    await wrapper.findAll('button').find((node) => node.text().includes('admin.apiDocs.save'))!.trigger('click')
    await flushPromises()

    expect(updateAPIDocs).toHaveBeenCalledWith('new markdown', 'gemini')
    expect(showSuccess).toHaveBeenCalledWith('admin.apiDocs.saveSuccess')
  })

  it('updates query when switching tabs and preserves copy/open actions', async () => {
    routeState.query = { tab: 'preview' }
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="api-docs-tab-edit"]').trigger('click')
    expect(replace).toHaveBeenCalledWith(expect.objectContaining({
      query: expect.objectContaining({
        tab: 'edit',
      }),
    }))

    routeState.query = { tab: 'edit' }
    const editWrapper = mountView()
    await flushPromises()

    await editWrapper.findAll('button').find((node) => node.text().includes('admin.apiDocs.copy'))!.trigger('click')
    expect(copyToClipboard).toHaveBeenCalledWith(initialDocument.effective_content, 'admin.apiDocs.copySuccess')

    await editWrapper.findAll('button').find((node) => node.text().includes('admin.apiDocs.openUserPage'))!.trigger('click')
    expect(resolve).toHaveBeenCalledWith('/api-docs/gemini')
    expect(open).toHaveBeenCalledWith('/api-docs/gemini', '_blank', 'noopener')
  })

  it('normalizes invalid tab query values back to preview', async () => {
    routeState.query = { tab: 'invalid' }
    const wrapper = mountView()
    await flushPromises()

    expect(replace).toHaveBeenCalledWith(expect.objectContaining({
      query: expect.objectContaining({
        tab: 'preview',
      }),
    }))
    expect(wrapper.find('[data-test="api-docs-preview"]').exists()).toBe(true)
  })

  it('keeps a valid tab when reopening another docs page', async () => {
    routeState.pageId = 'openai'
    routeState.query = { tab: 'edit' }
    resolve.mockReturnValue({ href: '/api-docs/openai' })
    const wrapper = mountView()
    await flushPromises()

    expect(getAPIDocs).toHaveBeenCalledWith('openai')
    expect(replace).not.toHaveBeenCalled()
    expect(wrapper.find('[data-test="api-docs-editor"]').exists()).toBe(true)

    await wrapper.findAll('button').find((node) => node.text().includes('admin.apiDocs.openUserPage'))!.trigger('click')
    expect(resolve).toHaveBeenCalledWith('/api-docs/openai')
    expect(open).toHaveBeenCalledWith('/api-docs/openai', '_blank', 'noopener')
  })

  it('restores only the current page override', async () => {
    routeState.query = { tab: 'edit' }
    const wrapper = mountView()
    await flushPromises()

    await wrapper.findAll('button').find((node) => node.text().includes('admin.apiDocs.restoreDefault'))!.trigger('click')
    await flushPromises()

    expect(clearAPIDocsOverride).toHaveBeenCalledWith('gemini')
    expect(showSuccess).toHaveBeenCalledWith('admin.apiDocs.restoreSuccess')
  })
})
