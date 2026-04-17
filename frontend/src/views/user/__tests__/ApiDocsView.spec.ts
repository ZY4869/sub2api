import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import ApiDocsView from '../ApiDocsView.vue'

const routeState = vi.hoisted(() => ({
  pageId: 'openai',
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
  '## openai',
  '### Responses 规则',
  '这里是 OpenAI 页面。',
  '#### Python',
  '```python',
  'print("openai python")',
  '```',
  '#### JavaScript',
  '```javascript',
  'console.log("openai js")',
  '```',
  '#### REST',
  '```bash',
  'echo openai-rest',
  '```',
].join('\n')

describe('ApiDocsView', () => {
  beforeEach(() => {
    routeState.pageId = 'openai'
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
      }
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
    expect(wrapper.text()).toContain('OpenAI 兼容')
    expect(wrapper.text()).toContain('Responses 规则')
    expect(wrapper.find('a[href="/api-docs/openai"]').exists()).toBe(true)

    const jsTab = wrapper.findAll('button').find((button) => button.text() === 'Javascript')
    expect(jsTab).toBeTruthy()
    await jsTab!.trigger('click')
    expect(wrapper.text()).toContain('console.log("openai js")')
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
