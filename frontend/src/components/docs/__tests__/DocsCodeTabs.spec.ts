import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import DocsCodeTabs from '../DocsCodeTabs.vue'

const { copyToClipboard } = vi.hoisted(() => ({
  copyToClipboard: vi.fn(),
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard,
  }),
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

describe('DocsCodeTabs', () => {
  it('copies the currently active tab and preserves focus-line rendering', async () => {
    copyToClipboard.mockReset()
    copyToClipboard.mockResolvedValue(true)

    const wrapper = mount(DocsCodeTabs, {
      props: {
        group: {
          id: 'group-1',
          tabs: [
            {
              id: 'tab-python',
              label: 'Python',
              language: 'python',
              focusLines: [2],
              code: 'print("one")\nprint("two")',
            },
            {
              id: 'tab-rest',
              label: 'REST',
              language: 'bash',
              focusLines: [1],
              code: 'curl https://api.zyxai.de',
            },
          ],
        },
        theme: {
          badgeClass: '',
          glowClass: '',
          navActiveClass: '',
          tabActiveClass: 'bg-sky-600 text-white',
          tocActiveClass: '',
        },
      },
    })

    expect(wrapper.findAll('.docs-code-line-focus')).toHaveLength(1)

    await wrapper.get('[data-test="docs-code-copy"]').trigger('click')
    expect(copyToClipboard).toHaveBeenCalledWith('print("one")\nprint("two")')

    await wrapper.findAll('button').find((button) => button.text() === 'REST')!.trigger('click')
    expect(wrapper.find('.docs-code-line-focus .docs-code-line-content').text()).toContain('curl')

    await wrapper.get('[data-test="docs-code-copy"]').trigger('click')
    expect(copyToClipboard).toHaveBeenLastCalledWith('curl https://api.zyxai.de')
  })
})
