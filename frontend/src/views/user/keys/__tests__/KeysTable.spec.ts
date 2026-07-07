import { mount } from '@vue/test-utils'
import { defineComponent, h } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import KeysTable from '../KeysTable.vue'
import type { Column } from '@/components/common/types'
import type { ApiKey } from '@/types'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

const DataTableStub = defineComponent({
  name: 'DataTable',
  props: {
    columns: { type: Array, required: true },
    data: { type: Array, required: true },
  },
  setup(props, { slots }) {
    return () =>
      h('div', { 'data-testid': 'data-table' }, [
        h(
          'div',
          { 'data-testid': 'columns' },
          (props.columns as Column[]).map((column) => column.key).join(','),
        ),
        ...(props.data as ApiKey[]).flatMap((row) =>
          (props.columns as Column[]).map((column) =>
            h(
              'div',
              { 'data-testid': `cell-${column.key}-${row.id}` },
              slots[`cell-${column.key}`]?.({ row, value: row[column.key as keyof ApiKey] }) ??
                String(row[column.key as keyof ApiKey] ?? ''),
            ),
          ),
        ),
      ])
  },
})

const baseApiKey = (overrides: Partial<ApiKey> = {}): ApiKey => ({
  id: 1,
  user_id: 1,
  key: 'sk-test',
  name: 'test key',
  group_id: null,
  status: 'active',
  ip_whitelist: [],
  ip_blacklist: [],
  last_used_at: null,
  current_concurrency: 4,
  quota: 0,
  quota_used: 0,
  image_only_enabled: false,
  image_count_billing_enabled: false,
  image_max_count: 0,
  image_count_used: 0,
  image_count_weights: { '1K': 1, '2K': 1, '4K': 1 },
  expires_at: null,
  created_at: '2026-07-07T00:00:00Z',
  updated_at: '2026-07-07T00:00:00Z',
  rate_limit_5h: 0,
  rate_limit_1d: 0,
  rate_limit_7d: 0,
  usage_5h: 0,
  usage_1d: 0,
  usage_7d: 0,
  window_5h_start: null,
  window_1d_start: null,
  window_7d_start: null,
  reset_5h_at: null,
  reset_1d_at: null,
  reset_7d_at: null,
  ...overrides,
})

const mountTable = (apiKeys: ApiKey[]) =>
  mount(KeysTable, {
    props: {
      columns: [
        { key: 'name', label: 'Name' },
        { key: 'current_concurrency', label: 'Current concurrency' },
      ],
      apiKeys,
      loading: false,
      copiedKeyId: null,
      usageStats: {},
      userGroupRates: {},
      isAdminMode: false,
      hideCcsImportButton: false,
      resolveGroup: () => undefined,
      getDisplayBindings: () => [],
      maskKey: (key: string) => key,
      formatResetTime: () => '',
    },
    global: {
      stubs: {
        DataTable: DataTableStub,
        Icon: { template: '<span />' },
      },
    },
  })

describe('KeysTable current concurrency', () => {
  it('renders realtime current concurrency and defaults missing values to zero', () => {
    const wrapper = mountTable([
      baseApiKey({ id: 1, current_concurrency: 4 }),
      baseApiKey({ id: 2, current_concurrency: undefined }),
    ])

    expect(wrapper.get('[data-testid="columns"]').text()).toContain('current_concurrency')
    expect(wrapper.get('[data-testid="cell-current_concurrency-1"]').text()).toBe('4')
    expect(wrapper.get('[data-testid="cell-current_concurrency-2"]').text()).toBe('0')
  })
})
