import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AccountStatusSummaryBar from '../AccountStatusSummaryBar.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

const summary = {
  total: 88,
  by_status: {
    active: 67,
    inactive: 10,
    error: 4
  },
  rate_limited: 5,
  temp_unschedulable: 1,
  overloaded: 1,
  paused: 2,
  in_use: 3,
  by_platform: {
    openai: 50
  },
  limited_breakdown: {
    total: 5,
    rate_429: 2,
    usage_5h: 2,
    usage_7d: 1
  }
}

describe('AccountStatusSummaryBar', () => {
  it('renders six horizontal summary cards with icon slots and right-aligned counts', () => {
    const wrapper = mount(AccountStatusSummaryBar, {
      props: {
        summary
      },
      global: {
        stubs: {
          Icon: true
        }
      }
    })

    const cards = wrapper.findAll('button[data-card-key]')
    expect(cards).toHaveLength(6)
    expect(cards[0].attributes('data-card-key')).toBe('total')
    expect(cards[1].attributes('data-card-key')).toBe('active')
    expect(cards[2].attributes('data-card-key')).toBe('in_use')
    expect(cards[0].classes()).toContain('justify-between')
    expect(cards[1].text()).toContain('admin.accounts.summary.active')
    expect(cards[1].text()).toContain('67')
  })

  it('keeps the summary cards clickable for filtering', async () => {
    const wrapper = mount(AccountStatusSummaryBar, {
      props: {
        summary,
        activeStatus: ''
      },
      global: {
        stubs: {
          Icon: true
        }
      }
    })

    await wrapper.get('[data-card-key="rate_limited"]').trigger('click')

    expect(wrapper.emitted('select-status')).toEqual([['rate_limited']])
  })

  it('emits runtime view selection when clicking the in-use card', async () => {
    const wrapper = mount(AccountStatusSummaryBar, {
      props: {
        summary,
        activeRuntimeView: 'all'
      },
      global: {
        stubs: {
          Icon: true
        }
      }
    })

    await wrapper.get('[data-card-key="in_use"]').trigger('click')

    expect(wrapper.emitted('select-runtime-view')).toEqual([['in_use_only']])
  })

  it('only highlights the in-use card when runtime view is active', () => {
    const wrapper = mount(AccountStatusSummaryBar, {
      props: {
        summary,
        activeStatus: '',
        activeRuntimeView: 'in_use_only'
      },
      global: {
        stubs: {
          Icon: true
        }
      }
    })

    expect(wrapper.get('[data-card-key="in_use"]').classes()).toContain('ring-2')
    expect(wrapper.get('[data-card-key="total"]').classes()).not.toContain('ring-2')
  })
})
