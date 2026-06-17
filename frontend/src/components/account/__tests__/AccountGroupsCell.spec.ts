import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AccountGroupsCell from '../AccountGroupsCell.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, number>) =>
        key === 'admin.accounts.groupCountTotal'
          ? `groups-${params?.count ?? 0}`
          : key,
    }),
  }
})

vi.mock('@/components/common/GroupBadge.vue', () => ({
  default: {
    props: {
      name: String,
      wrap: Boolean,
    },
    template: '<span class="group-badge-stub" :data-wrap="String(wrap)" :title="name">{{ name }}</span>',
  },
}))

vi.mock('@/components/common/PlatformIcon.vue', () => ({
  default: {
    props: {
      platform: String,
      size: String,
    },
    template: '<span class="platform-icon-stub" :data-platform="platform" :data-size="size" />',
  },
}))

const groups = [
  { id: 1, name: 'GPT-免费', platform: 'openai' },
  { id: 2, name: '审核组', platform: 'gemini' },
  { id: 3, name: 'Edge', platform: 'anthropic' },
] as any

describe('AccountGroupsCell', () => {
  it('renders full group names with wrapping enabled', () => {
    const wrapper = mount(AccountGroupsCell, {
      props: {
        groups,
        maxDisplay: 3,
        displayMode: 'full',
      },
    })

    expect(wrapper.text()).toContain('GPT-免费')
    expect(wrapper.get('.group-badge-stub').attributes('data-wrap')).toBe('true')
  })

  it('renders platform-only icons with top tooltips in icon mode', () => {
    const wrapper = mount(AccountGroupsCell, {
      props: {
        groups,
        maxDisplay: 3,
        displayMode: 'icon',
      },
    })

    const buttons = wrapper.findAll('button[aria-label]')
    const iconGrid = wrapper.get('.grid')
    expect(iconGrid.classes()).toEqual(expect.arrayContaining(['grid-cols-2', 'gap-1']))
    expect(buttons[0].attributes('aria-label')).toBe('GPT-免费')
    expect(buttons[0].attributes('title')).toBe('GPT-免费')
    expect(buttons[0].text()).toBe('')
    expect(buttons[0].classes()).toEqual(expect.arrayContaining(['h-6', 'w-6', 'p-0']))
    expect(buttons[0].find('.platform-icon-stub').attributes('data-platform')).toBe('openai')
    const tooltip = wrapper.find('[role="tooltip"]')
    expect(tooltip.text()).toBe('GPT-免费')
    expect(tooltip.classes()).toContain('bottom-full')
    expect(tooltip.classes()).not.toContain('top-full')
    expect(tooltip.classes()).toContain('whitespace-nowrap')
    expect(tooltip.classes()).toContain('truncate')
    expect(tooltip.classes()).not.toContain('break-words')
    expect(buttons[1].attributes('aria-label')).toBe('审核组')
    expect(buttons[1].text()).toBe('')
    expect(buttons[1].find('.platform-icon-stub').attributes('data-platform')).toBe('gemini')
  })

  it('assigns different palettes to duplicated abbreviations', () => {
    const wrapper = mount(AccountGroupsCell, {
      props: {
        groups: [
          { id: 10, name: 'GPT-免费', platform: 'openai' },
          { id: 11, name: 'GPT-生图', platform: 'openai' },
        ] as any,
        maxDisplay: 2,
        displayMode: 'icon',
      },
    })

    const buttons = wrapper.findAll('button[aria-label]')
    expect(buttons.map((button) => button.text())).toEqual(['', ''])
    const bgClass = (classes: string[]) => classes.find((className) => className.startsWith('bg-'))
    expect(bgClass(buttons[0].classes())).not.toBe(bgClass(buttons[1].classes()))
  })

  it('keeps the +N popover affordance for hidden groups', async () => {
    const wrapper = mount(AccountGroupsCell, {
      props: {
        groups,
        maxDisplay: 2,
        displayMode: 'icon',
      },
      global: {
        stubs: {
          Teleport: true,
          Transition: false,
        },
      },
    })

    expect(wrapper.text()).toContain('+2')

    await wrapper.get('button:not([aria-label])').trigger('click')

    expect(wrapper.text()).toContain('groups-3')
  })
})
