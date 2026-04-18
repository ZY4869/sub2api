import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import ApiDocsView from '../ApiDocsView.vue'

const routeState = vi.hoisted(() => ({
  pageId: 'document-ai',
}))

const { getAPIDocs, copyToClipboard } = vi.hoisted(() => ({
  getAPIDocs: vi.fn(),
  copyToClipboard: vi.fn(),
}))

vi.mock('@/api/docs', () => ({
  default: {
    getAPIDocs,
  },
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard,
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

const markdown = [
  '# API 文档中心',
  '## common',
  '### 概览',
  '通用说明。',
  '## document-ai',
  '### 异步任务',
  '这里是百度智能文档页面。',
  '#### Python',
  '```python',
  'print("document-ai python")',
  '```',
  '#### JavaScript',
  '```javascript',
  'console.log("document-ai js")',
  '```',
  '#### REST',
  '```bash',
  'echo document-ai-rest',
  '```',
].join('\n')

describe('ApiDocsView', () => {
  beforeEach(() => {
    routeState.pageId = 'document-ai'
    getAPIDocs.mockReset()
    copyToClipboard.mockReset()

    getAPIDocs.mockResolvedValue({ content: markdown })
    copyToClipboard.mockResolvedValue(true)

    vi.stubGlobal(
      'IntersectionObserver',
      class {
        observe = vi.fn()
        disconnect = vi.fn()
        unobserve = vi.fn()
      },
    )
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('loads the effective markdown and renders the selected protocol page with code tabs', async () => {
    const wrapper = mount(ApiDocsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div data-test="app-layout"><slot /></div>' },
          RouterLink: RouterLinkStub,
          'router-link': RouterLinkStub,
        },
      },
    })

    await flushPromises()

    expect(getAPIDocs).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('百度智能文档')
    expect(wrapper.text()).toContain('异步任务')
    expect(wrapper.find('a[href="/api-docs/document-ai"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('ui.apiDocs.summary.languages.label')
    expect(wrapper.text()).not.toContain('ui.apiDocs.summary.sync.label')

    const jsTab = wrapper.findAll('button').find((button) => button.text() === 'JavaScript')
    expect(jsTab).toBeTruthy()
    await jsTab!.trigger('click')
    expect(wrapper.findAll('.docs-code-line').length).toBeGreaterThan(0)
  })

  it('copies the effective markdown source', async () => {
    const wrapper = mount(ApiDocsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          RouterLink: RouterLinkStub,
          'router-link': RouterLinkStub,
        },
      },
    })

    await flushPromises()
    await wrapper.get('button.btn-primary').trigger('click')

    expect(copyToClipboard).toHaveBeenCalledWith(markdown, 'ui.apiDocs.copySuccess')
  })
})
