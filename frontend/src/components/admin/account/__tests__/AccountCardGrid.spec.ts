import { defineComponent } from 'vue'
import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import AccountCardGrid from '../AccountCardGrid.vue'

const virtualState = vi.hoisted(() => ({
  items: [] as Array<{ key?: string | number; index: number; start: number; end: number }>,
  totalSize: 0,
}))

vi.mock('@tanstack/vue-virtual', async () => {
  const vue = await vi.importActual<typeof import('vue')>('vue')
  return {
    useWindowVirtualizer: () =>
      vue.computed(() => ({
        getVirtualItems: () => virtualState.items,
        getTotalSize: () => virtualState.totalSize,
        measureElement: () => {},
      })),
  }
})

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

const cardStub = defineComponent({
  props: ['account', 'visualStyle'],
  template: '<div class="account-card-stub" :data-id="account.id" :data-style="visualStyle">{{ account.name }}</div>',
})

const createAccounts = (count: number) =>
  Array.from({ length: count }, (_, index) => ({
    id: index + 1,
    name: `Account ${index + 1}`,
  }))

function mountGrid() {
  return mount(AccountCardGrid, {
    props: {
      accounts: createAccounts(12),
      loading: false,
      selectedIds: [],
      togglingSchedulable: null,
      todayStatsByAccountId: {},
      todayStatsLoading: false,
      usageManualRefreshToken: 0,
      visualStyle: 'airy',
    } as any,
    global: {
      stubs: {
        AccountCard: cardStub,
      },
    },
  })
}

describe('AccountCardGrid', () => {
  beforeEach(() => {
    virtualState.items = []
    virtualState.totalSize = 0
    Object.defineProperty(window, 'innerWidth', {
      configurable: true,
      value: 1440,
    })
  })

  it('falls back to direct rows before virtual rows are available', async () => {
    const wrapper = mountGrid()

    await wrapper.vm.$nextTick()

    const cards = wrapper.findAll('.account-card-stub')
    expect(cards).toHaveLength(12)
    expect(cards[0].attributes('data-id')).toBe('1')
    expect(cards[11].attributes('data-id')).toBe('12')
  })

  it('renders only virtualized rows when virtual items are available', async () => {
    virtualState.items = [
      { key: 'row-0', index: 0, start: 0, end: 420 },
      { key: 'row-1', index: 1, start: 420, end: 840 },
    ]
    virtualState.totalSize = 1680

    const wrapper = mountGrid()

    await wrapper.vm.$nextTick()

    const cards = wrapper.findAll('.account-card-stub')
    expect(cards).toHaveLength(8)
    expect(cards[0].attributes('data-id')).toBe('1')
    expect(cards[7].attributes('data-id')).toBe('8')
    expect(wrapper.html()).toContain('translateY(420px)')
  })
})
