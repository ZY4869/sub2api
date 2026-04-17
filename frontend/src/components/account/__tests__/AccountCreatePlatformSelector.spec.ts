import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AccountCreatePlatformSelector from '../AccountCreatePlatformSelector.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  const messages: Record<string, string> = {
    'admin.accounts.platforms.anthropic': 'Anthropic',
    'admin.accounts.platforms.kiro': 'Kiro',
    'admin.accounts.platforms.openai': 'OpenAI',
    'admin.accounts.platforms.copilot': 'GitHub Copilot',
    'admin.accounts.platforms.grok': 'Grok',
    'admin.accounts.platforms.protocol_gateway': 'Protocol Gateway',
    'admin.accounts.platforms.gemini': 'Google',
    'admin.accounts.platforms.antigravity': 'Antigravity',
    'admin.accounts.platforms.baidu_document_ai': 'Baidu Document AI'
  }
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => messages[key] ?? key
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

    const selector = wrapper.get('[data-tour="account-form-platform"]')
    expect(selector.classes()).toContain('grid')
    expect(selector.classes()).toContain('grid-cols-2')
    expect(selector.classes()).toContain('md:grid-cols-3')
    expect(selector.classes()).toContain('xl:grid-cols-4')

    const buttonTexts = wrapper.findAll('button').map((button) => button.text())
    expect(buttonTexts).toEqual([
      'Anthropic',
      'Kiro',
      'OpenAI',
      'GitHub Copilot',
      'Grok',
      'Protocol Gateway',
      'Google',
      'Antigravity',
      'Baidu Document AI'
    ])
    expect(wrapper.findAll('button')[0].classes()).toContain('min-w-0')
    expect(wrapper.findAll('button')[0].classes()).toContain('whitespace-normal')

    await wrapper.findAll('button')[4].trigger('click')
    expect(wrapper.emitted('update:platform')).toEqual([['grok']])
  })
})
