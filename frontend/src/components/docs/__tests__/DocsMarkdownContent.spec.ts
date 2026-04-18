import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import DocsMarkdownContent from '../DocsMarkdownContent.vue'

const RouterLinkStub = {
  props: ['to'],
  template: '<a :href="to"><slot /></a>',
}

const markdown = [
  '# API 文档中心',
  '## document-ai',
  '### 异步任务',
  '说明一。',
  '### 直连解析',
  '说明二。',
].join('\n')

describe('DocsMarkdownContent', () => {
  beforeEach(() => {
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

  it('keeps the page toc visible in preview mode', async () => {
    const wrapper = mount(DocsMarkdownContent, {
      props: {
        markdown,
        pageId: 'document-ai',
        basePath: '/admin/api-docs',
        previewMode: true,
        navTitle: '支持协议',
        tocTitle: '本页内容',
      },
      global: {
        stubs: {
          RouterLink: RouterLinkStub,
          'router-link': RouterLinkStub,
        },
      },
    })

    await flushPromises()

    expect(wrapper.text()).toContain('支持协议')
    expect(wrapper.text()).toContain('本页内容')
    expect(wrapper.text()).toContain('百度智能文档')
    expect(wrapper.find('a[href="/admin/api-docs/document-ai"]').exists()).toBe(true)
    expect(wrapper.find('a[href="#异步任务"]').exists()).toBe(true)
    expect(wrapper.find('a[href="#直连解析"]').exists()).toBe(true)
  })
})
