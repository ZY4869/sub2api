import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import AdminModulesView from '../AdminModulesView.vue'

const mockState = vi.hoisted(() => ({
  authStore: {
    isSimpleMode: false
  }
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => mockState.authStore
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

const RouterLinkStub = {
  props: ['to'],
  template: '<a :href="to"><slot /></a>'
}

function mountView() {
  return mount(AdminModulesView, {
    global: {
      stubs: {
        AppLayout: { template: '<main><slot /></main>' },
        RouterLink: RouterLinkStub
      }
    }
  })
}

describe('AdminModulesView', () => {
  beforeEach(() => {
    mockState.authStore.isSimpleMode = false
  })

  it('renders the three expanded module sections and their entry cards', () => {
    const wrapper = mountView()

    expect(wrapper.findAll('[data-testid="module-section"]')).toHaveLength(3)
    expect(wrapper.find('[data-testid="module-card-promo-codes"]').attributes('href')).toBe('/admin/promo-codes')
    expect(wrapper.find('[data-testid="module-card-redeem-codes"]').attributes('href')).toBe('/admin/redeem')
    expect(wrapper.find('[data-testid="module-card-proxies"]').attributes('href')).toBe('/admin/proxies')
    expect(wrapper.find('[data-testid="module-card-settings-email"]').attributes('href')).toBe('/admin/settings?tab=email')
  })

  it('collapses and expands a section from the header button', async () => {
    const wrapper = mountView()
    const toggle = wrapper.find('[data-testid="module-section-toggle-growth"]')

    expect(toggle.attributes('aria-expanded')).toBe('true')
    await toggle.trigger('click')
    await nextTick()
    expect(toggle.attributes('aria-expanded')).toBe('false')
    await toggle.trigger('click')
    await nextTick()
    expect(toggle.attributes('aria-expanded')).toBe('true')
  })

  it('filters simple-mode-only hidden cards while keeping allowed module cards', () => {
    mockState.authStore.isSimpleMode = true
    const wrapper = mountView()

    expect(wrapper.find('[data-testid="module-card-promo-codes"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="module-card-redeem-codes"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="module-card-invitation"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="module-card-proxies"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="module-card-registration"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="module-card-settings-general"]').exists()).toBe(true)
  })
})
