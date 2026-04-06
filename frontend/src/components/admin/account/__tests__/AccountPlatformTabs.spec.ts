import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AccountPlatformTabs from '../AccountPlatformTabs.vue'
import type { AccountPlatform } from '@/types'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  const messages: Record<string, string> = {
    'admin.accounts.platformTabs.all': 'All',
    'admin.accounts.platforms.gemini': 'Google'
  }
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => messages[key] ?? key
    })
  }
})

const mountTabs = (
  overrides: Partial<{
    modelValue: string
    platformCounts: Partial<Record<AccountPlatform, number>>
  }> = {}
) => mount(AccountPlatformTabs, {
  props: {
    modelValue: '',
    platformCounts: {
      anthropic: 1,
      openai: 4,
      grok: 2
    },
    ...overrides
  },
  global: {
    stubs: {
      PlatformIcon: {
        template: '<span data-testid="platform-icon" />'
      }
    }
  }
})

const resolveOrder = (wrapper: ReturnType<typeof mountTabs>) => (
  wrapper
    .findAll('button[data-tab-value]')
    .map((button) => button.attributes('data-tab-value'))
)

describe('AccountPlatformTabs', () => {
  it('keeps the fixed platform order regardless of platform counts', () => {
    const wrapper = mountTabs()

    expect(resolveOrder(wrapper)).toEqual([
      'all',
      'anthropic',
      'kiro',
      'openai',
      'copilot',
      'grok',
      'protocol_gateway',
      'gemini',
      'antigravity'
    ])
  })

  it('renders the gemini tab with the Google label', () => {
    const wrapper = mountTabs()

    expect(wrapper.get('[data-tab-value="gemini"]').text()).toContain('Google')
  })

  it('does not reorder tabs when counts change', () => {
    const wrapper = mountTabs({
      platformCounts: {
        openai: 99,
        protocol_gateway: 1,
        gemini: 50
      }
    })

    expect(resolveOrder(wrapper)).toEqual([
      'all',
      'anthropic',
      'kiro',
      'openai',
      'copilot',
      'grok',
      'protocol_gateway',
      'gemini',
      'antigravity'
    ])
  })

  it('treats missing counts as zero and still renders the platform badge count', () => {
    const wrapper = mountTabs({
      platformCounts: {
        openai: 3
      }
    })

    expect(wrapper.get('[data-tab-value="anthropic"]').text()).toContain('0')
    expect(wrapper.get('[data-tab-value="openai"]').text()).toContain('3')
  })
})
