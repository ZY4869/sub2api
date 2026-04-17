import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import AdminApiDocsView from '../AdminApiDocsView.vue'

const routeState = vi.hoisted(() => ({
  pageId: 'gemini',
}))

const {
  getAPIDocs,
  updateAPIDocs,
  clearAPIDocsOverride,
  copyToClipboard,
  showSuccess,
  showError,
  resolve,
} = vi.hoisted(() => ({
  getAPIDocs: vi.fn(),
  updateAPIDocs: vi.fn(),
  clearAPIDocsOverride: vi.fn(),
  copyToClipboard: vi.fn(),
  showSuccess: vi.fn(),
  showError: vi.fn(),
  resolve: vi.fn(() => ({ href: '/api-docs/gemini' })),
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
    }),
    useRouter: () => ({
      resolve,
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

const RouterLinkStub = {
  props: ['to'],
  template: '<a :href="to"><slot /></a>',
}

const initialDocument = {
  effective_content: [
    '# API 文档中心',
    '## common',
    '### 概览',
    '通用说明。',
    '## gemini',
    '### 模型生成',
    '这里是 Gemini 页面。',
    '#### Python',
    '```python',
    'print("gemini py")',
    '```',
    '#### JavaScript',
    '```javascript',
    'console.log("gemini js")',
    '```',
    '#### REST',
    '```bash',
    'echo gemini-rest',
    '```',
  ].join('\n'),
  default_content: '# API 文档中心\n\n## common\n### 概览\n默认模板\n',
  has_override: true,
}

const findButtonByText = (wrapper: ReturnType<typeof mount>, text: string) => {
  const button = wrapper.findAll('button').find((item) => item.text().includes(text))
  expect(button, `button ${text} should exist`).toBeTruthy()
  return button!
}

describe('AdminApiDocsView', () => {
  beforeEach(() => {
    routeState.pageId = 'gemini'
    getAPIDocs.mockReset()
    updateAPIDocs.mockReset()
    clearAPIDocsOverride.mockReset()
    copyToClipboard.mockReset()
    showSuccess.mockReset()
    showError.mockReset()
    resolve.mockReset()

    getAPIDocs.mockResolvedValue(initialDocument)
    updateAPIDocs.mockResolvedValue({
      ...initialDocument,
      effective_content: initialDocument.effective_content.replace('这里是 Gemini 页面。', '已保存的新 Gemini 页面。'),
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

  it('loads the document, renders the selected preview page, and saves the override', async () => {
    const wrapper = mount(AdminApiDocsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div data-test="app-layout"><slot /></div>' },
          RouterLink: RouterLinkStub,
          'router-link': RouterLinkStub,
        },
      },
    })

    await flushPromises()

    const textarea = wrapper.get('textarea')
    expect((textarea.element as HTMLTextAreaElement).value).toContain('这里是 Gemini 页面。')
    expect(wrapper.text()).toContain('Gemini 原生')
    expect(wrapper.text()).toContain('模型生成')

    await textarea.setValue(initialDocument.effective_content.replace('这里是 Gemini 页面。', '待保存的 Gemini 页面。'))
    await flushPromises()

    await findButtonByText(wrapper, 'admin.apiDocs.save').trigger('click')
    await flushPromises()

    expect(updateAPIDocs).toHaveBeenCalledWith(
      initialDocument.effective_content.replace('这里是 Gemini 页面。', '待保存的 Gemini 页面。')
    )
    expect(showSuccess).toHaveBeenCalledWith('admin.apiDocs.saveSuccess')
  })

  it('restores the default template from the runtime override', async () => {
    const wrapper = mount(AdminApiDocsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          RouterLink: RouterLinkStub,
          'router-link': RouterLinkStub,
        },
      },
    })

    await flushPromises()
    await findButtonByText(wrapper, 'admin.apiDocs.restoreDefault').trigger('click')
    await flushPromises()

    expect(confirm).toHaveBeenCalledWith('admin.apiDocs.restoreConfirm')
    expect(clearAPIDocsOverride).toHaveBeenCalledTimes(1)
    expect(showSuccess).toHaveBeenCalledWith('admin.apiDocs.restoreSuccess')
    expect((wrapper.get('textarea').element as HTMLTextAreaElement).value).toBe(initialDocument.default_content)
  })

  it('copies markdown, opens the matching user page, and supports panel switching', async () => {
    const wrapper = mount(AdminApiDocsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          RouterLink: RouterLinkStub,
          'router-link': RouterLinkStub,
        },
      },
    })

    await flushPromises()

    await findButtonByText(wrapper, 'admin.apiDocs.copy').trigger('click')
    expect(copyToClipboard).toHaveBeenCalledWith(initialDocument.effective_content, 'admin.apiDocs.copySuccess')

    await findButtonByText(wrapper, 'admin.apiDocs.openUserPage').trigger('click')
    expect(resolve).toHaveBeenCalledWith('/api-docs/gemini')
    expect(open).toHaveBeenCalledWith('/api-docs/gemini', '_blank', 'noopener')

    await findButtonByText(wrapper, 'admin.apiDocs.previewTab').trigger('click')
    expect(wrapper.get('[data-test="api-docs-editor"]').classes()).toContain('hidden')
    expect(wrapper.get('[data-test="api-docs-preview"]').classes()).toContain('block')
  })
})
