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

const groups = [
  { id: 1, name: 'VeryLongProductionGroupName' },
  { id: 2, name: '审核组' },
  { id: 3, name: 'Edge' },
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

    expect(wrapper.text()).toContain('VeryLongProductionGroupName')
    expect(wrapper.get('.group-badge-stub').attributes('data-wrap')).toBe('true')
  })

  it('renders compact accessible icons in icon mode', () => {
    const wrapper = mount(AccountGroupsCell, {
      props: {
        groups,
        maxDisplay: 3,
        displayMode: 'icon',
      },
    })

    const buttons = wrapper.findAll('button[aria-label]')
    expect(buttons[0].attributes('aria-label')).toBe('VeryLongProductionGroupName')
    expect(buttons[0].attributes('title')).toBe('VeryLongProductionGroupName')
    expect(buttons[0].text()).toBe('V')
    expect(wrapper.find('[role="tooltip"]').text()).toBe('VeryLongProductionGroupName')
    expect(buttons[1].attributes('aria-label')).toBe('审核组')
    expect(buttons[1].text()).toBe('审')
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
