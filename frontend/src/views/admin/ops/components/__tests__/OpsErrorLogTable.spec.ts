import { describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import { mount } from '@vue/test-utils'
import type { OpsErrorLog } from '@/api/admin/ops'
import OpsErrorLogTable from '../OpsErrorLogTable.vue'

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => {
        if (key === 'admin.ops.errorLog.billingRule') return 'Billing Rule'
        return key
      }
    })
  }
})

const PaginationStub = defineComponent({
  name: 'PaginationStub',
  template: '<div class="pagination-stub" />'
})

const ElTooltipStub = defineComponent({
  name: 'ElTooltipStub',
  template: '<div><slot /></div>'
})

function createErrorLog(overrides: Partial<OpsErrorLog> = {}): OpsErrorLog {
  return {
    id: 1,
    created_at: '2026-04-12T10:00:00Z',
    phase: 'upstream',
    type: 'upstream_error',
    error_owner: 'provider',
    error_source: 'upstream_http',
    severity: 'error',
    status_code: 503,
    platform: 'gemini',
    model: 'gemini-2.5-flash',
    is_retryable: true,
    retry_count: 0,
    resolved: false,
    client_request_id: 'client-1',
    request_id: 'req-1',
    message: 'upstream unavailable',
    user_email: 'user@example.com',
    account_name: 'Gemini API Key',
    group_name: 'Gemini Group',
    request_path: '/v1beta/models/gemini-2.5-flash:generateContent',
    inbound_endpoint: '/v1beta/models',
    upstream_endpoint: '/v1beta/models',
    requested_model: 'gemini-2.5-flash',
    upstream_model: 'gemini-2.5-flash',
    request_type: 1,
    gemini_surface: 'live',
    billing_rule_id: 'rule-42',
    probe_action: 'recovery_probe',
    ...overrides
  }
}

describe('OpsErrorLogTable', () => {
  it('shows gemini surface, probe action, and billing rule id', () => {
    const wrapper = mount(OpsErrorLogTable, {
      props: {
        rows: [createErrorLog()],
        total: 1,
        loading: false,
        page: 1,
        pageSize: 10
      },
      global: {
        stubs: {
          Pagination: PaginationStub,
          'el-tooltip': ElTooltipStub
        }
      }
    })

    expect(wrapper.text()).toContain('live / recovery_probe')
    expect(wrapper.text()).toContain('Billing Rule: rule-42')
  })

  it('falls back to upstream model when requested model is empty', () => {
    const wrapper = mount(OpsErrorLogTable, {
      props: {
        rows: [
          createErrorLog({
            requested_model: '',
            upstream_model: 'gpt-5.1-codex'
          })
        ],
        total: 1,
        loading: false,
        page: 1,
        pageSize: 10
      },
      global: {
        stubs: {
          Pagination: PaginationStub,
          'el-tooltip': ElTooltipStub
        }
      }
    })

    expect(wrapper.text()).toContain('gpt-5.1-codex')
  })
})
