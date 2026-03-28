import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AccountPlatformTabs from '../AccountPlatformTabs.vue'
import type { AccountPlatform, AccountPlatformCountSortOrder } from '@/types'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

vi.mock('@/utils/lobeIconResolver', () => ({
  buildLobeIconSources: vi.fn(() => [])
}))

const mountTabs = (
  overrides: Partial<{
    modelValue: string
    platformCounts: Partial<Record<AccountPlatform, number>>
    sortOrder: AccountPlatformCountSortOrder
  }> = {}
) => mount(AccountPlatformTabs, {
  props: {
    modelValue: '',
    platformCounts: {
      anthropic: 1,
      openai: 4,
      grok: 2
    },
    sortOrder: 'count_asc',
    ...overrides
  },
  global: {
    stubs: {
      LobeStaticIcon: true
    }
  }
})

const resolveOrder = (wrapper: ReturnType<typeof mountTabs>) => (
  wrapper
    .findAll('button[data-tab-value]')
    .map((button) => button.attributes('data-tab-value'))
)

describe('AccountPlatformTabs', () => {
  it('keeps the all tab first and sorts non-zero platforms from few to many', () => {
    const wrapper = mountTabs()

    expect(resolveOrder(wrapper)).toEqual([
      'all',
      'anthropic',
      'grok',
      'openai',
      'kiro',
      'copilot',
      'protocol_gateway',
      'gemini',
      'antigravity',
      'sora'
    ])
  })

  it('sorts non-zero platforms from many to few when count_desc is selected', () => {
    const wrapper = mountTabs({
      sortOrder: 'count_desc'
    })

    expect(resolveOrder(wrapper).slice(0, 4)).toEqual([
      'all',
      'openai',
      'grok',
      'anthropic'
    ])
  })

  it('keeps static platform order for equal counts and places zero-count platforms last', () => {
    const wrapper = mountTabs({
      platformCounts: {
        openai: 2,
        grok: 2
      }
    })

    expect(resolveOrder(wrapper)).toEqual([
      'all',
      'openai',
      'grok',
      'anthropic',
      'kiro',
      'copilot',
      'protocol_gateway',
      'gemini',
      'antigravity',
      'sora'
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
