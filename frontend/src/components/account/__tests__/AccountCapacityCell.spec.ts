import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { ref } from 'vue'
import AccountCapacityCell from '../AccountCapacityCell.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      locale: ref('zh'),
      t: (key: string) => key,
    }),
  }
})

const baseAccount = {
  id: 1,
  name: 'Primary',
  platform: 'openai',
  type: 'apikey',
  concurrency: 3,
  current_concurrency: 1,
  extra: {},
}

const anthroAccount = {
  ...baseAccount,
  platform: 'anthropic',
  type: 'oauth',
  window_cost_limit: 15,
  current_window_cost: 9.8,
  window_cost_sticky_reserve: 10,
  max_sessions: 12,
  active_sessions: 7,
  session_idle_timeout_minutes: 5,
  base_rpm: 100,
  current_rpm: 82,
  rpm_strategy: 'tiered',
  rpm_sticky_buffer: 20,
  quota_daily_limit: 20,
  quota_daily_used: 5,
  quota_weekly_limit: 80,
  quota_weekly_used: 31,
  quota_limit: 200,
  quota_used: 96,
}

describe('AccountCapacityCell', () => {
  it('keeps the default variant compatible with the legacy animated slot badge', () => {
    const wrapper = mount(AccountCapacityCell, {
      props: {
        account: baseAccount,
      } as any,
      global: {
        stubs: {
          QuotaBadge: true,
        },
      },
    })

    expect(wrapper.html()).toContain('backdrop-blur-sm')
    expect(wrapper.html()).toContain('animate-ping')
  })

  it('renders the glass variant with the migrated airy signal-bar capacity primary block', () => {
    const wrapper = mount(AccountCapacityCell, {
      props: {
        account: baseAccount,
        visualVariant: 'glass',
      } as any,
      global: {
        stubs: {
          QuotaBadge: true,
        },
      },
    })

    expect(wrapper.get('[data-testid="airy-capacity-cell"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="airy-capacity-primary"]').attributes('title')).toBe('当前占用并发/最大上限: 1/3')
    expect(wrapper.findAll('[data-testid="airy-capacity-bar"]')).toHaveLength(3)
    expect(wrapper.html()).toContain('account-capacity-breathe')
    expect(wrapper.html()).not.toContain('animate-ping')
    expect(wrapper.text()).toContain('01')
    expect(wrapper.text()).toContain('03')
  })

  it('pads concurrency values to at least two digits for display only', () => {
    const wrapper = mount(AccountCapacityCell, {
      props: {
        account: baseAccount,
      } as any,
      global: {
        stubs: {
          QuotaBadge: true,
        },
      },
    })

    expect(wrapper.text()).toContain('01')
    expect(wrapper.text()).toContain('/')
    expect(wrapper.text()).toContain('03')
  })

  it('matches the total concurrency width when the capacity exceeds two digits', () => {
    const wrapper = mount(AccountCapacityCell, {
      props: {
        account: {
          ...baseAccount,
          concurrency: 12,
          current_concurrency: 2,
        },
      } as any,
      global: {
        stubs: {
          QuotaBadge: true,
        },
      },
    })

    expect(wrapper.text()).toContain('02')
    expect(wrapper.text()).toContain('/')
    expect(wrapper.text()).toContain('12')
  })

  it('compresses airy capacity bars to a maximum of ten cells for large totals', () => {
    const wrapper = mount(AccountCapacityCell, {
      props: {
        account: {
          ...baseAccount,
          concurrency: 100,
          current_concurrency: 12,
        },
        visualVariant: 'glass',
      } as any,
      global: {
        stubs: {
          QuotaBadge: true,
        },
      },
    })

    expect(wrapper.findAll('[data-testid="airy-capacity-bar"]')).toHaveLength(10)
    expect(wrapper.text()).toContain('12')
    expect(wrapper.text()).toContain('100')
  })

  it.each([
    { used: 0, total: 2, filled: 0, cells: 2, current: '00', max: '02' },
    { used: 1, total: 5, filled: 1, cells: 5, current: '01', max: '05' },
    { used: 5, total: 5, filled: 5, cells: 5, current: '05', max: '05' },
    { used: 12, total: 100, filled: 1, cells: 10, current: '012', max: '100' },
  ])('renders compact airy capacity bars for $used/$total', ({ used, total, filled, cells, current, max }) => {
    const wrapper = mount(AccountCapacityCell, {
      props: {
        account: {
          ...baseAccount,
          concurrency: total,
          current_concurrency: used,
        },
        visualVariant: 'glass',
        compact: true,
      } as any,
      global: {
        stubs: {
          QuotaBadge: true,
        },
      },
    })

    const bars = wrapper.findAll('[data-testid="airy-capacity-bar"]')
    expect(bars).toHaveLength(cells)
    expect(bars.filter((bar) => !bar.classes().includes('bg-slate-200/80'))).toHaveLength(filled)
    expect(wrapper.text()).toContain(current)
    expect(wrapper.text()).toContain(max)
    expect(wrapper.find('[data-testid="airy-capacity-metrics"]').exists()).toBe(false)
  })

  it('shows the urgent airy state when concurrency is fully occupied', () => {
    const wrapper = mount(AccountCapacityCell, {
      props: {
        account: {
          ...baseAccount,
          concurrency: 5,
          current_concurrency: 5,
        },
        visualVariant: 'glass',
      } as any,
      global: {
        stubs: {
          QuotaBadge: true,
        },
      },
    })

    expect(wrapper.html()).toContain('account-capacity-urgent')
    expect(wrapper.text()).toContain('05')
  })

  it('reflows airy capacity sub-metrics into unified cards and passes white surface styling through', () => {
    const wrapper = mount(AccountCapacityCell, {
      props: {
        account: anthroAccount,
        visualVariant: 'glass',
        whiteSurfaceEnabled: true,
      } as any,
    })

    expect(wrapper.findAll('[data-testid="airy-capacity-metric-card"]').length).toBeGreaterThanOrEqual(6)
    expect(wrapper.get('[data-testid="airy-capacity-metrics"]').text()).toContain('admin.accounts.capacity.cards.windowCost')
    expect(wrapper.get('[data-testid="airy-capacity-metrics"]').text()).toContain('admin.accounts.capacity.cards.sessions')
    expect(wrapper.get('[data-testid="airy-capacity-metrics"]').text()).toContain('admin.accounts.capacity.cards.rpm')
    expect(wrapper.get('[data-testid="airy-capacity-metrics"]').text()).toContain('admin.accounts.capacity.cards.quota')
    expect(wrapper.html()).toContain('bg-white')
  })

  it('keeps airy table capacity compact while leaving room for the main meter', () => {
    const wrapper = mount(AccountCapacityCell, {
      props: {
        account: anthroAccount,
        visualVariant: 'glass',
        compact: true,
      } as any,
      global: {
        stubs: {
          QuotaBadge: true,
        },
      },
    })

    expect(wrapper.get('[data-testid="airy-capacity-cell"]').classes()).toContain('max-w-[148px]')
    expect(wrapper.find('[data-testid="airy-capacity-metrics"]').exists()).toBe(false)
    expect(wrapper.findAll('[data-testid="airy-capacity-bar"]')).toHaveLength(3)
    expect(wrapper.text()).toContain('1')
    expect(wrapper.text()).toContain('3')
    expect(wrapper.text()).not.toContain('admin.accounts.capacity.cards.windowCost')
    expect(wrapper.text()).not.toContain('admin.accounts.capacity.cards.sessions')
    expect(wrapper.text()).not.toContain('admin.accounts.capacity.cards.rpm')
  })
})
