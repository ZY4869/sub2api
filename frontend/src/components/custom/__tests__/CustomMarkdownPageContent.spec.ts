import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import CustomMarkdownPageContent from '../CustomMarkdownPageContent.vue'

describe('CustomMarkdownPageContent', () => {
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

  it('hardens links and images after markdown rendering', async () => {
    const markdown = [
      '# Safety',
      '',
      '[External](https://example.com/docs)',
      '[Relative](/docs/guide)',
      '[Unsafe](javascript:alert(1))',
      '',
      '![Inline](data:image/png;base64,abc)',
      '![Unsafe](javascript:alert(1))',
    ].join('\n')

    const wrapper = mount(CustomMarkdownPageContent, {
      props: {
        markdown,
        title: 'Safety',
        tocTitle: 'On This Page',
      },
      global: {
        stubs: {
          DocsToc: { template: '<nav />' },
        },
      },
    })

    await flushPromises()

    const anchors = wrapper.findAll('a')
    expect(anchors).toHaveLength(3)
    expect(anchors[0].attributes('href')).toBe('https://example.com/docs')
    expect(anchors[0].attributes('target')).toBe('_blank')
    expect(anchors[0].attributes('rel')).toContain('noopener')
    expect(anchors[1].attributes('href')).toBe('/docs/guide')
    expect(anchors[2].attributes('href')).toBeUndefined()

    const images = wrapper.findAll('img')
    const safeImages = images.filter((image) => image.attributes('src'))
    expect(safeImages).toHaveLength(1)
    expect(safeImages[0].attributes('src')).toBe('data:image/png;base64,abc')
    expect(safeImages[0].attributes('loading')).toBe('lazy')
    expect(images.some((image) => String(image.attributes('src') || '').startsWith('javascript:'))).toBe(false)
  })
})
