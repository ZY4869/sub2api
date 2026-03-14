import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AccountGeminiHelpDialog from '../AccountGeminiHelpDialog.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

const baseDialogStub = {
  name: 'BaseDialog',
  props: ['show', 'title', 'maxWidth'],
  template: `
    <div data-testid="base-dialog">
      <span data-testid="dialog-title">{{ title }}</span>
      <slot />
      <slot name="footer" />
    </div>
  `
}

describe('AccountGeminiHelpDialog', () => {
  it('renders help content and emits close from footer button', async () => {
    const wrapper = mount(AccountGeminiHelpDialog, {
      props: {
        show: true
      },
      global: {
        stubs: {
          BaseDialog: baseDialogStub
        }
      }
    })

    expect(wrapper.get('[data-testid="dialog-title"]').text()).toBe('admin.accounts.gemini.helpDialog.title')
    expect(wrapper.text()).toContain('admin.accounts.gemini.quotaPolicy.title')

    const links = wrapper.findAll('a').map((link) => link.attributes('href'))
    expect(links).toContain('https://aistudio.google.com/app/apikey')
    expect(links).toContain('https://ai.google.dev/pricing')

    await wrapper.get('.btn-primary').trigger('click')
    expect(wrapper.emitted('close')).toEqual([[]])
  })
})
