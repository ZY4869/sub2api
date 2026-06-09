import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import ProxiesTable from '../ProxiesTable.vue'
import type { Proxy } from '@/types'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => {
        if (key === 'admin.proxies.expired') return 'Expired'
        if (key === 'admin.proxies.expiringSoon') return 'Expiring soon'
        if (key === 'admin.proxies.expiryActive') return 'Active expiry'
        if (key === 'admin.proxies.fallbackProxyShort') return `Fallback #${params?.id}`
        if (key === 'admin.groups.accountsCount') return `${params?.count ?? 0} accounts`
        return key
      },
      locale: 'en'
    })
  }
})

const baseProxy = (overrides: Partial<Proxy>): Proxy => ({
  id: 1,
  name: 'proxy',
  protocol: 'http',
  host: '127.0.0.1',
  port: 8080,
  username: null,
  password: null,
  status: 'active',
  created_at: '2026-06-01T00:00:00Z',
  updated_at: '2026-06-01T00:00:00Z',
  ...overrides
})

describe('ProxiesTable', () => {
  it('shows expired and expiring lifecycle badges with fallback identity', () => {
    const wrapper = mount(ProxiesTable, {
      props: {
        columns: [{ key: 'expiry', label: 'Expiry' }],
        proxies: [
          baseProxy({
            id: 1,
            expires_at: '2020-01-01T00:00:00Z',
            expiry_remind_days: 3,
            fallback_proxy_id: 12
          }),
          baseProxy({
            id: 2,
            expires_at: '2999-01-02T00:00:00Z',
            expiry_remind_days: 365000
          })
        ],
        loading: false,
        allVisibleSelected: false,
        selectedProxyIds: new Set<number>(),
        visiblePasswordIds: new Set<number>(),
        copyMenuProxyId: null,
        testingProxyIds: new Set<number>(),
        qualityCheckingProxyIds: new Set<number>(),
        locale: 'en-US',
        qualityOverallClass: () => 'badge-success',
        qualityOverallLabel: () => 'healthy'
      },
      global: {
        stubs: {
          Icon: true,
          EmptyState: true
        }
      }
    })

    expect(wrapper.text()).toContain('Expired')
    expect(wrapper.text()).toContain('Expiring soon')
    expect(wrapper.text()).toContain('Fallback #12')
    expect(wrapper.find('.badge-danger').exists()).toBe(true)
    expect(wrapper.find('.badge-warning').exists()).toBe(true)
  })
})
