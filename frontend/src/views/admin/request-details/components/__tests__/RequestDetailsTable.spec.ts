import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import RequestDetailsTable from '../RequestDetailsTable.vue'

const copyToClipboard = vi.fn()

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard
  })
}))

vi.mock('@/stores/modelRegistry', () => ({
  getModelRegistrySnapshot: () => ({
    etag: 'test',
    updated_at: '2026-04-04T00:00:00Z',
    models: [
      {
        id: 'claude-opus-4.1',
        display_name: 'Claude Opus 4.1',
        provider: 'anthropic',
        platforms: ['anthropic'],
        protocol_ids: ['claude-opus-4-1-20250805'],
        aliases: [],
        pricing_lookup_ids: [],
        modalities: ['text'],
        capabilities: ['reasoning'],
        ui_priority: 0,
        exposed_in: ['whitelist']
      },
      {
        id: 'gpt-4o-mini',
        display_name: 'GPT-4o mini',
        provider: 'openai',
        platforms: ['openai'],
        protocol_ids: ['gpt-4o-mini-2025-04-01'],
        aliases: [],
        pricing_lookup_ids: [],
        modalities: ['text'],
        capabilities: ['chat'],
        ui_priority: 0,
        exposed_in: ['whitelist']
      }
    ],
    presets: []
  })
}))

const translations: Record<string, string> = {
  'common.total': 'Total',
  'common.refresh': 'Refresh',
  'admin.requestDetails.table.title': 'Trace Table',
  'admin.requestDetails.table.description': 'Trace table description',
  'admin.requestDetails.table.columns.time': 'Time',
  'admin.requestDetails.table.columns.requestId': 'Request ID',
  'admin.requestDetails.table.columns.account': 'Request Account',
  'admin.requestDetails.table.columns.group': 'Request Group',
  'admin.requestDetails.table.columns.protocolPair': 'Protocol Pair',
  'admin.requestDetails.table.columns.route': 'Route',
  'admin.requestDetails.table.columns.models': 'Models',
  'admin.requestDetails.table.columns.status': 'Status / Reason',
  'admin.requestDetails.table.columns.flags': 'Flags',
  'admin.requestDetails.table.columns.performance': 'Performance',
  'admin.requestDetails.table.columns.actions': 'Actions',
  'admin.requestDetails.table.view': 'View',
  'admin.requestDetails.table.summary.user': 'User {id}',
  'admin.requestDetails.table.summary.apiKey': 'API Key {id}',
  'admin.requestDetails.table.summary.account': 'Account {id}',
  'admin.requestDetails.table.summary.group': 'Group {id}',
  'admin.requestDetails.table.summary.ttft': 'TTFT {value}',
  'admin.requestDetails.table.summary.tokens': '{value} Tokens',
  'admin.requestDetails.presentation.labels.requestId': 'Request ID',
  'admin.requestDetails.presentation.labels.clientRequestId': 'Client Request ID',
  'admin.requestDetails.presentation.labels.upstreamRequestId': 'Upstream Request ID',
  'admin.requestDetails.presentation.labels.billingRuleId': 'Billing Rule ID',
  'admin.requestDetails.presentation.labels.geminiSurface': 'Gemini Surface',
  'admin.requestDetails.presentation.labels.probeAction': 'Probe Action',
  'admin.requestDetails.presentation.status.success': 'Success',
  'admin.requestDetails.presentation.finishReasons.stop': 'Completed Normally',
  'admin.requestDetails.presentation.captureReasons.sampled': 'Sampled',
  'admin.requestDetails.presentation.flags.streamEnabled': 'Stream',
  'admin.requestDetails.presentation.flags.streamDisabled': 'Sync',
  'admin.requestDetails.presentation.flags.toolsEnabled': 'Tools',
  'admin.requestDetails.presentation.flags.toolsDisabled': 'No Tools',
  'admin.requestDetails.presentation.flags.thinkingEnabled': 'Thinking',
  'admin.requestDetails.presentation.flags.thinkingDisabled': 'No Thinking',
  'admin.requestDetails.presentation.flags.rawAvailable': 'Raw Saved',
  'admin.requestDetails.presentation.flags.rawUnavailable': 'No Raw',
  'admin.requestDetails.presentation.flags.sampled': 'Sampled',
  'admin.requestDetails.presentation.flags.notSampled': 'Not Sampled',
  'admin.requestDetails.presentation.protocols.openai': 'OpenAI',
  'admin.requestDetails.presentation.thinkingLevels.xhigh': 'Extra High',
  'admin.requestDetails.presentation.thinkingLevels.max': 'Max'
}

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, string | number>) => {
        const template = translations[key] ?? key
        return Object.entries(params || {}).reduce(
          (result, [paramKey, value]) => result.replace(`{${paramKey}}`, String(value)),
          template
        )
      }
    })
  }
})

const baseItem = {
  id: 1,
  created_at: '2026-04-04T00:00:00Z',
  request_id: 'req-1',
  client_request_id: 'client-1',
  upstream_request_id: 'upstream-1',
  platform: 'protocol_gateway',
  protocol_in: '/v1/responses',
  protocol_out: '/v1/chat/completions',
  channel: 'openai_compat',
  route_path: '/v1/responses',
  request_type: 'chat_completions',
  user_id: 10,
  api_key_id: 20,
  account_id: 30,
  group_id: 40,
  account_name: 'Test Account',
  group_name: 'Test Group',
  requested_model: 'claude-opus-4-1-20250805',
  upstream_model: 'gpt-4o-mini-2025-04-01',
  actual_upstream_model: 'gpt-4o-mini-2025-04-01',
  gemini_surface: 'live',
  billing_rule_id: 'rule-live-1',
  probe_action: 'recovery_probe',
  status: 'success',
  status_code: 200,
  upstream_status_code: 200,
  duration_ms: 1250,
  ttft_ms: 180,
  input_tokens: 100,
  output_tokens: 40,
  total_tokens: 140,
  finish_reason: 'stop',
  prompt_block_reason: '',
  stream: true,
  has_tools: true,
  tool_kinds: ['web_search'],
  has_thinking: true,
  thinking_source: 'request',
  thinking_level: 'high',
  thinking_budget: 2048,
  media_resolution: '',
  count_tokens_source: 'gateway',
  capture_reason: 'sampled',
  sampled: true,
  raw_available: true,
  raw_access_allowed: true
}

describe('RequestDetailsTable', () => {
  it('renders single-line aggregated columns and different upstream models', () => {
    const wrapper = mount(RequestDetailsTable, {
      props: {
        items: [
          baseItem,
          {
            ...baseItem,
            id: 2,
            request_id: 'req-2',
            client_request_id: 'client-2',
            upstream_request_id: 'upstream-2',
            actual_upstream_model: 'claude-opus-4-1-20250805',
            upstream_model: 'claude-opus-4-1-20250805'
          }
        ],
        total: 2,
        page: 1,
        pageSize: 20,
        loading: false,
        selectedId: null
      },
      global: {
        stubs: {
          Pagination: true,
          ModelIcon: { template: '<span data-test="model-icon" />' }
        }
      }
    })

    expect(wrapper.text()).toContain('Test Account')
    expect(wrapper.text()).toContain('Test Group')
    expect(wrapper.text()).toContain('/v1/responses')
    expect(wrapper.text()).toContain('/v1/responses -> /v1/chat/completions')
    expect(wrapper.text()).toContain('openai_compat')
    expect(wrapper.text()).toContain('protocol_gateway')
    expect(wrapper.text()).toContain('live')
    expect(wrapper.text()).toContain('recovery_probe')
    expect(wrapper.text()).toContain('rule-live-1')
    expect(wrapper.text()).toContain('Claude Opus 4.1')
    expect(wrapper.text()).toContain('GPT-4o mini')
    expect(wrapper.text()).toContain('Success')
    expect(wrapper.text()).toContain('TTFT 180 ms')
    expect(wrapper.text()).toContain('140 Tokens')
    expect(wrapper.findAll('[data-test="model-icon"]')).toHaveLength(3)
    expect(wrapper.findAll('tbody tr')).toHaveLength(2)
  })

  it('copies text without opening the row and keeps request id tooltip details', async () => {
    const wrapper = mount(RequestDetailsTable, {
      props: {
        items: [baseItem],
        total: 1,
        page: 1,
        pageSize: 20,
        loading: false,
        selectedId: null
      },
      global: {
        stubs: {
          Pagination: true,
          ModelIcon: true
        }
      }
    })

    const requestIdButton = wrapper.find('tbody tr td:nth-child(2) button')
    expect(requestIdButton.attributes('title')).toContain('Client Request ID: client-1')
    expect(requestIdButton.attributes('title')).toContain('Upstream Request ID: upstream-1')
    expect(requestIdButton.attributes('title')).toContain('Billing Rule ID: rule-live-1')
    expect(requestIdButton.attributes('title')).toContain('Gemini Surface: live')
    expect(requestIdButton.attributes('title')).toContain('Probe Action: recovery_probe')
    expect(requestIdButton.attributes('title')).toContain('User 10')
    expect(requestIdButton.attributes('title')).toContain('API Key 20')
    expect(requestIdButton.attributes('title')).toContain('Account 30')
    expect(requestIdButton.attributes('title')).toContain('Group 40')

    await requestIdButton.trigger('click')

    expect(copyToClipboard).toHaveBeenCalledWith('req-1', undefined)
    expect(wrapper.emitted('select')).toBeUndefined()

    await wrapper.find('tbody tr').trigger('click')

    expect(wrapper.emitted('select')).toHaveLength(1)
  })

  it('emits refresh from the table toolbar', async () => {
    const wrapper = mount(RequestDetailsTable, {
      props: {
        items: [baseItem],
        total: 1,
        page: 1,
        pageSize: 20,
        loading: false,
        refreshing: false,
        selectedId: null
      },
      global: {
        stubs: {
          Pagination: true,
          ModelIcon: true
        }
      }
    })

    const refreshButton = wrapper.findAll('button').find((button) => button.text() === 'Refresh')
    expect(refreshButton).toBeTruthy()

    await refreshButton!.trigger('click')

    expect(wrapper.emitted('refresh')).toHaveLength(1)
    expect(wrapper.emitted('select')).toBeUndefined()
  })
})
