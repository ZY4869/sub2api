import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
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
})
