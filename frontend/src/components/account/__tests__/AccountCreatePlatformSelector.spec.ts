import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AccountCreatePlatformSelector from '../AccountCreatePlatformSelector.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

describe('AccountCreatePlatformSelector', () => {
  it('renders all platform buttons and emits updates', async () => {
    const wrapper = mount(AccountCreatePlatformSelector, {
      props: {
        platform: 'anthropic'
      },
      global: {
        stubs: {
          PlatformIcon: {
            template: '<span data-testid="platform-icon" />'
          }
        }
      }
    })

    const buttonTexts = wrapper.findAll('button').map((button) => button.text())
    expect(buttonTexts).toEqual(['Anthropic', 'OpenAI', 'Sora', 'Gemini', 'Antigravity'])

    await wrapper.findAll('button')[3].trigger('click')
    expect(wrapper.emitted('update:platform')).toEqual([['gemini']])
  })
})
