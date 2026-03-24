import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AccountGroupedView from '../AccountGroupedView.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, string | number>) =>
        key === 'admin.accounts.groupView.stats' && params?.count
          ? `${params.count} accounts`
          : key
    })
  }
})

describe('AccountGroupedView', () => {
  it('duplicates multi-group accounts across sections and keeps ungrouped accounts separate', () => {
    const wrapper = mount(AccountGroupedView, {
      props: {
        accounts: [
          {
            id: 1,
            name: 'Shared',
            platform: 'openai',
            type: 'apikey',
            status: 'active',
            schedulable: true,
            groups: [{ id: 1, name: 'A' }, { id: 2, name: 'B' }],
            group_ids: [1, 2]
          },
          {
            id: 2,
            name: 'Solo',
            platform: 'openai',
            type: 'apikey',
            status: 'active',
            schedulable: true,
            groups: [],
            group_ids: []
          }
        ],
        groups: [
          { id: 1, name: 'A' },
          { id: 2, name: 'B' }
        ],
        groupFilter: '',
        viewMode: 'table',
        columns: [],
        selectedIds: [],
        loading: false,
        togglingSchedulable: null,
        todayStatsByAccountId: {},
        todayStatsLoading: false,
        todayStatsError: null,
        usageManualRefreshToken: 0,
        sortStorageKey: 'account-table-sort'
      } as any,
      global: {
        stubs: {
          AccountGroupSection: {
            props: ['title', 'accounts'],
            template: '<div class="group-section">{{ title }}:{{ accounts[0]?.id }}</div>'
          }
        }
      }
    })

    const text = wrapper.text()
    expect(text).toContain('A:1')
    expect(text).toContain('B:1')
    expect(text).toContain('admin.accounts.groupView.ungrouped:2')
  })
})
