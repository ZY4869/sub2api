import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AccountsLayoutView from '../AccountsLayoutView.vue'

const mockState = vi.hoisted(() => ({
  routePath: '/admin/accounts/blacklist',
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({
    path: mockState.routePath,
  }),
  RouterView: {
    template: '<div data-test="router-view" />',
  },
  RouterLink: {
    props: ['to'],
    template: '<a :href="to" v-bind="$attrs"><slot /></a>',
  },
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

describe('AccountsLayoutView', () => {
  it('renders a single parent layout and keeps the blacklist tab active for deep links', () => {
    const wrapper = mount(AccountsLayoutView, {
      global: {
        stubs: {
          AppLayout: { template: '<div data-test="app-layout"><slot /></div>' },
        },
      },
    })

    expect(wrapper.findAll('[data-test="app-layout"]')).toHaveLength(1)
    expect(wrapper.find('[data-test="router-view"]').exists()).toBe(true)

    const links = wrapper.findAll('a')
    expect(links).toHaveLength(3)
    expect(links[0].text()).toContain('admin.accounts.subnav.all')
    expect(links[1].text()).toContain('admin.accounts.subnav.limited')
    expect(links[2].text()).toContain('admin.accounts.subnav.blacklist')
    expect(links[2].classes()).toContain('bg-primary-600')
    expect(links[2].classes()).toContain('text-white')
  })
})
