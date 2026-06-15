import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const virtualState = vi.hoisted(() => ({
  items: [] as Array<{ index: number; start: number; end: number }>,
  totalSize: 0
}))

vi.mock('@tanstack/vue-virtual', async () => {
  const vue = await vi.importActual<typeof import('vue')>('vue')
  return {
    useWindowVirtualizer: () =>
      vue.computed(() => ({
        getVirtualItems: () => virtualState.items,
        getTotalSize: () => virtualState.totalSize,
        measureElement: () => {}
      })),
    useVirtualizer: () =>
      vue.computed(() => ({
        getVirtualItems: () => virtualState.items,
        getTotalSize: () => virtualState.totalSize,
        measureElement: () => {}
      }))
  }
})

import DataTable from '../DataTable.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

const columns = [
  { key: 'name', label: 'Name', sortable: true }
]

const rows = [
  { id: 1, name: 'Beta' },
  { id: 2, name: 'Alpha' }
]

describe('DataTable', () => {
  beforeEach(() => {
    localStorage.clear()
    virtualState.items = []
    virtualState.totalSize = 0
  })

  it('preserves input order on initial render even when a persisted sort exists', async () => {
    localStorage.setItem('account-table-sort', JSON.stringify({ key: 'name', order: 'desc' }))

    const wrapper = mount(DataTable, {
      props: {
        columns,
        data: rows,
        rowKey: 'id',
        sortStorageKey: 'account-table-sort',
        defaultSortKey: 'name',
        defaultSortOrder: 'asc',
        preserveInputOrder: true
      },
      global: {
        stubs: {
          Icon: true
        }
      }
    })

    await wrapper.vm.$nextTick()

    expect((wrapper.vm as any).sortedData.map((row: { name: string }) => row.name)).toEqual(['Beta', 'Alpha'])
    expect(localStorage.getItem('account-table-sort')).toBeNull()
  })

  it('still allows manual sorting after preserving the input order', async () => {
    const wrapper = mount(DataTable, {
      props: {
        columns,
        data: rows,
        rowKey: 'id',
        sortStorageKey: 'account-table-sort',
        defaultSortKey: 'name',
        defaultSortOrder: 'asc',
        preserveInputOrder: true
      },
      global: {
        stubs: {
          Icon: true
        }
      }
    })

    await wrapper.vm.$nextTick()
    await wrapper.get('th').trigger('click')
    await wrapper.vm.$nextTick()

    expect((wrapper.vm as any).sortedData.map((row: { name: string }) => row.name)).toEqual(['Alpha', 'Beta'])
    expect(localStorage.getItem('account-table-sort')).toBeNull()
  })

  it('falls back to direct row rendering when the virtualizer has no visible items yet', async () => {
    const wrapper = mount(DataTable, {
      props: {
        columns,
        data: rows,
        rowKey: 'id'
      },
      global: {
        stubs: {
          Icon: true
        }
      }
    })

    await wrapper.vm.$nextTick()

    expect(wrapper.findAll('tbody tr[data-row-id]')).toHaveLength(2)
    expect(wrapper.text()).toContain('Beta')
    expect(wrapper.text()).toContain('Alpha')
  })

  it('renders desktop rows directly when virtual scrolling is disabled', async () => {
    virtualState.items = [{ index: 0, start: 0, end: 56 }]
    virtualState.totalSize = 56

    const wrapper = mount(DataTable, {
      props: {
        columns,
        data: rows,
        rowKey: 'id',
        virtualScroll: false
      },
      global: {
        stubs: {
          Icon: true
        }
      }
    })

    await wrapper.vm.$nextTick()

    expect(wrapper.findAll('tbody tr[data-row-id]')).toHaveLength(2)
    expect(wrapper.text()).toContain('Beta')
    expect(wrapper.text()).toContain('Alpha')
  })

  it('keeps existing behavior when row visuals are not provided and applies visuals when configured', async () => {
    const wrapper = mount(DataTable, {
      props: {
        columns,
        data: rows,
        rowKey: 'id',
        virtualScroll: false,
        rowClass: (row: { id: number }) => (row.id === 1 ? 'visual-row' : ''),
        rowStyle: (row: { id: number }) =>
          row.id === 1 ? ({ '--account-row-sticky-bg': '#fff7ed' } as any) : undefined
      },
      global: {
        stubs: {
          Icon: true
        }
      }
    })

    await wrapper.vm.$nextTick()

    const row = wrapper.get('tbody tr[data-row-id="1"]')
    expect(row.classes()).toContain('visual-row')
    expect(row.attributes('style')).toContain('--account-row-sticky-bg')
  })

  it('supports window virtual scroll mode without breaking direct fallback rendering', async () => {
    const wrapper = mount(DataTable, {
      props: {
        columns,
        data: rows,
        rowKey: 'id',
        virtualScrollTarget: 'window'
      },
      global: {
        stubs: {
          Icon: true
        }
      }
    })

    await wrapper.vm.$nextTick()

    expect(wrapper.findAll('tbody tr[data-row-id]')).toHaveLength(2)
    expect(wrapper.text()).toContain('Beta')
    expect(wrapper.text()).toContain('Alpha')
  })

  it('uses the default horizontal scrollbar treatment unless subtle mode is requested', async () => {
    const wrapper = mount(DataTable, {
      props: {
        columns,
        data: rows,
        rowKey: 'id'
      },
      global: {
        stubs: {
          Icon: true
        }
      }
    })

    await wrapper.vm.$nextTick()

    expect(wrapper.get('.table-wrapper').classes()).not.toContain(
      'table-wrapper-scrollbar-subtle'
    )
  })

  it('adds the subtle horizontal scrollbar class when requested', async () => {
    const wrapper = mount(DataTable, {
      props: {
        columns,
        data: rows,
        rowKey: 'id',
        horizontalScrollbar: 'subtle'
      },
      global: {
        stubs: {
          Icon: true
        }
      }
    })

    await wrapper.vm.$nextTick()

    expect(wrapper.get('.table-wrapper').classes()).toContain(
      'table-wrapper-scrollbar-subtle'
    )
  })

  it('supports desktop cell colspan and skipped cells', async () => {
    const wrapper = mount(DataTable, {
      props: {
        columns: [
          { key: 'name', label: 'Name' },
          { key: 'today_stats', label: 'Today' },
          { key: 'usage', label: 'Usage' },
          { key: 'usage_reset_dates', label: 'Reset' },
        ],
        data: [{ id: 1, name: 'Key Row', today_stats: 'summary' }],
        rowKey: 'id',
        virtualScroll: false,
        cellSpan: (_row, column) => {
          if (column.key === 'today_stats') return { colspan: 3 }
          if (column.key === 'usage' || column.key === 'usage_reset_dates') return { skip: true }
          return undefined
        },
      },
      global: {
        stubs: {
          Icon: true
        }
      }
    })

    await wrapper.vm.$nextTick()

    const cells = wrapper.findAll('tbody tr[data-row-id="1"] td')
    expect(cells).toHaveLength(2)
    expect(cells[1].attributes('colspan')).toBe('3')
    expect(wrapper.text()).toContain('summary')
    expect(wrapper.text()).not.toContain('undefined')
  })

  it('ignores stale virtual rows that point outside the current filtered data', async () => {
    virtualState.items = [{ index: 12, start: 672, end: 728 }]
    virtualState.totalSize = 728

    const wrapper = mount(DataTable, {
      props: {
        columns,
        data: rows,
        rowKey: 'id',
        virtualScrollTarget: 'window'
      },
      global: {
        stubs: {
          Icon: true
        }
      }
    })

    await wrapper.vm.$nextTick()

    expect(wrapper.findAll('tbody tr[data-row-id]')).toHaveLength(2)
    expect(wrapper.text()).toContain('Beta')
    expect(wrapper.text()).toContain('Alpha')
  })
})
