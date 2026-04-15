import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const virtualState = vi.hoisted(() => ({
  items: [] as Array<{ index: number; start: number; end: number }>,
  totalSize: 0
}))

vi.mock('@tanstack/vue-virtual', async () => {
  const vue = await vi.importActual<typeof import('vue')>('vue')
  return {
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
})
