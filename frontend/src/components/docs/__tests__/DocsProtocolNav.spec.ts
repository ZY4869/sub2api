import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import DocsProtocolNav from '../DocsProtocolNav.vue'

const RouterLinkStub = {
  props: ['to'],
  template: '<a :href="to"><slot /></a>',
}

const PlatformIconStub = {
  props: ['platform'],
  template: '<span class="platform-icon" :data-platform="platform"></span>',
}

const IconStub = {
  props: ['name'],
  template: '<span class="generic-icon" :data-icon="name"></span>',
}

describe('DocsProtocolNav', () => {
  it('renders document-ai last and keeps openai compat on a generic icon', () => {
    const wrapper = mount(DocsProtocolNav, {
      props: {
        pages: [
          { id: 'common', title: '通用接入', shortTitle: '通用', description: 'desc', rawMarkdown: '', introBlocks: [], sections: [], isMissing: false },
          { id: 'openai-native', title: 'OpenAI 原生', shortTitle: 'OpenAI 原生', description: 'desc', rawMarkdown: '', introBlocks: [], sections: [], isMissing: false },
          { id: 'openai', title: 'OpenAI 兼容', shortTitle: 'OpenAI 兼容', description: 'desc', rawMarkdown: '', introBlocks: [], sections: [], isMissing: false },
          { id: 'document-ai', title: '百度智能文档', shortTitle: '百度文档', description: 'desc', rawMarkdown: '', introBlocks: [], sections: [], isMissing: false },
        ],
        currentPageId: 'common',
        basePath: '/api-docs',
        theme: {
          badgeClass: '',
          glowClass: '',
          navActiveClass: 'active',
          tabActiveClass: '',
          tocActiveClass: '',
        },
      },
      global: {
        stubs: {
          RouterLink: RouterLinkStub,
          'router-link': RouterLinkStub,
          PlatformIcon: PlatformIconStub,
          Icon: IconStub,
        },
      },
    })

    const links = wrapper.findAll('a')
    expect(links.at(-1)?.attributes('href')).toBe('/api-docs/document-ai')
    expect(wrapper.find('[href="/api-docs/openai-native"] .platform-icon').attributes('data-platform')).toBe('openai')
    expect(wrapper.find('[href="/api-docs/openai"] .generic-icon').attributes('data-icon')).toBe('swap')
    expect(wrapper.find('[href="/api-docs/document-ai"]').text()).toContain('百度智能文档')
  })
})
