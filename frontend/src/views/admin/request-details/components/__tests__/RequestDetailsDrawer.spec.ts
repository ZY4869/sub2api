import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import RequestDetailsDrawer from '../RequestDetailsDrawer.vue'

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard: vi.fn()
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
      }
    ],
    presets: []
  })
}))

const translations: Record<string, string> = {
  'common.loading': 'Loading',
  'common.close': 'Close',
  'common.refresh': 'Refresh',
  'common.noData': 'No Data',
  'admin.requestDetails.drawer.title': 'Trace Detail',
  'admin.requestDetails.drawer.noSelection': 'No selection',
  'admin.requestDetails.drawer.copyContent': 'Copy Content',
  'admin.requestDetails.drawer.viewFull': 'View Full Details',
  'admin.requestDetails.drawer.previewOnlyNotice': 'Only the captured preview is available here. Raw full text is not accessible for this tab.',
  'admin.requestDetails.drawer.fullDialogTitle': '{tab} · {id}',
  'admin.requestDetails.drawer.loadRaw': 'Load Raw Payload',
  'admin.requestDetails.drawer.rawNotAllowed': 'Raw payload access is restricted to configured audit users.',
  'admin.requestDetails.drawer.auditOperator': 'Operator #{id}',
  'admin.requestDetails.drawer.payload.previewReady': 'Preview is ready',
  'admin.requestDetails.drawer.payload.empty': 'No content available',
  'admin.requestDetails.drawer.emptyStates.inbound': 'No inbound request preview was captured for this trace.',
  'admin.requestDetails.drawer.emptyStates.normalized': 'No normalized request content is available for this trace.',
  'admin.requestDetails.drawer.emptyStates.upstreamRequest': 'No upstream request content is available for this trace.',
  'admin.requestDetails.drawer.emptyStates.upstreamResponse': 'No upstream response content is available for this trace.',
  'admin.requestDetails.drawer.emptyStates.gatewayResponse': 'No gateway response content is available for this trace.',
  'admin.requestDetails.drawer.emptyStates.tools': 'No tool or thinking trace was captured for this trace.',
  'admin.requestDetails.drawer.emptyStates.rawRequest': 'No raw request content is available.',
  'admin.requestDetails.drawer.emptyStates.rawResponse': 'No raw response content is available.',
  'admin.requestDetails.drawer.tabs.overview': 'Overview',
  'admin.requestDetails.drawer.tabs.inbound': 'Inbound Request',
  'admin.requestDetails.drawer.tabs.normalized': 'Normalized Request',
  'admin.requestDetails.drawer.tabs.upstreamRequest': 'Upstream Request',
  'admin.requestDetails.drawer.tabs.upstreamResponse': 'Upstream Response',
  'admin.requestDetails.drawer.tabs.gatewayResponse': 'Gateway Response',
  'admin.requestDetails.drawer.tabs.tools': 'Tools / Thinking',
  'admin.requestDetails.drawer.tabs.audits': 'Audit Log',
  'admin.requestDetails.drawer.tabs.raw': 'Raw Payload',
  'admin.requestDetails.presentation.labels.rawRequest': 'Raw Request',
  'admin.requestDetails.presentation.labels.rawResponse': 'Raw Response',
  'admin.requestDetails.presentation.labels.createdAt': 'Created At',
  'admin.requestDetails.presentation.labels.requestType': 'Request Type',
  'admin.requestDetails.presentation.labels.routePath': 'Route',
  'admin.requestDetails.presentation.labels.channel': 'Channel',
  'admin.requestDetails.presentation.labels.platform': 'Platform',
  'admin.requestDetails.presentation.labels.protocolPair': 'Protocol Pair',
  'admin.requestDetails.presentation.labels.requestId': 'Request ID',
  'admin.requestDetails.presentation.labels.clientRequestId': 'Client Request ID',
  'admin.requestDetails.presentation.labels.upstreamRequestId': 'Upstream Request ID',
  'admin.requestDetails.presentation.labels.userId': 'User ID',
  'admin.requestDetails.presentation.labels.apiKeyId': 'API Key ID',
  'admin.requestDetails.presentation.labels.accountId': 'Account ID',
  'admin.requestDetails.presentation.labels.groupId': 'Group ID',
  'admin.requestDetails.presentation.labels.status': 'Status',
  'admin.requestDetails.presentation.labels.finishReason': 'Finish Reason',
  'admin.requestDetails.presentation.labels.captureReason': 'Capture Reason',
  'admin.requestDetails.presentation.labels.statusCode': 'Status Code',
  'admin.requestDetails.presentation.labels.upstreamStatusCode': 'Upstream Status Code',
  'admin.requestDetails.presentation.labels.duration': 'Duration',
  'admin.requestDetails.presentation.labels.ttft': 'TTFT',
  'admin.requestDetails.presentation.labels.totalTokens': 'Total Tokens',
  'admin.requestDetails.presentation.labels.tokenBreakdown': 'Input / Output Tokens',
  'admin.requestDetails.presentation.labels.thinkingSource': 'Thinking Source',
  'admin.requestDetails.presentation.labels.thinkingLevel': 'Thinking Level',
  'admin.requestDetails.presentation.labels.tokenSource': 'Token Counting Source',
  'admin.requestDetails.presentation.labels.mediaResolution': 'Media Resolution',
  'admin.requestDetails.presentation.labels.requestHeaders': 'Request Headers',
  'admin.requestDetails.presentation.labels.responseHeaders': 'Response Headers',
  'admin.requestDetails.presentation.cards.status': 'Status Summary',
  'admin.requestDetails.presentation.cards.requestedModel': 'Requested Model',
  'admin.requestDetails.presentation.cards.upstreamModel': 'Upstream Model',
  'admin.requestDetails.presentation.cards.performance': 'Performance',
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
  'admin.requestDetails.presentation.status.success': 'Success',
  'admin.requestDetails.presentation.protocols.openai': 'OpenAI',
  'admin.requestDetails.presentation.requestTypes.chat_completions': 'Chat Completions',
  'admin.requestDetails.presentation.finishReasons.stop': 'Completed Normally',
  'admin.requestDetails.presentation.captureReasons.sampled': 'Sampled',
  'admin.requestDetails.presentation.thinkingSources.request': 'Request Parameter',
  'admin.requestDetails.presentation.thinkingLevels.high': 'High',
  'admin.requestDetails.drawer.sections.basicInfo': 'Basic Info',
  'admin.requestDetails.drawer.sections.identifiers': 'Identifiers',
  'admin.requestDetails.drawer.sections.execution': 'Execution',
  'admin.requestDetails.drawer.sections.flags': 'Flags',
  'admin.requestDetails.drawer.sections.headers': 'Headers'
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

const detail = {
  id: 1,
  created_at: '2026-04-04T00:00:00Z',
  request_id: 'req-1',
  client_request_id: 'client-1',
  upstream_request_id: 'upstream-1',
  platform: 'openai',
  protocol_in: 'openai',
  protocol_out: 'openai',
  channel: 'main',
  route_path: '/responses',
  request_type: 'chat_completions',
  user_id: 10,
  api_key_id: 20,
  account_id: 30,
  group_id: 40,
  requested_model: 'claude-opus-4-1-20250805',
  upstream_model: 'claude-opus-4-1-20250805',
  actual_upstream_model: 'claude-opus-4-1-20250805',
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
  raw_access_allowed: true,
  inbound_request_json: '{"preview":true}',
  normalized_request_json: '{"normalized":true}',
  upstream_request_json: '',
  upstream_response_json: '{"upstream":true}',
  gateway_response_json: '{"gateway":true}',
  tool_trace_json: '{"tool":"web_search"}',
  request_headers_json: '{"x-request-id":"req-1"}',
  response_headers_json: '{"content-type":"application/json"}',
  audits: []
}

function createWrapper(overrides?: Record<string, unknown>) {
  return mount(RequestDetailsDrawer, {
    props: {
      open: true,
      detail,
      rawDetail: null,
      loading: false,
      rawLoading: false,
      ...overrides
    },
    global: {
      stubs: {
        ModelIcon: true,
        BaseDialog: {
          props: ['show', 'title'],
          template: '<div v-if="show" data-test="base-dialog"><h3>{{ title }}</h3><slot /></div>'
        }
      }
    }
  })
}

describe('RequestDetailsDrawer', () => {
  it('renders payload empty state instead of a blank pre block', async () => {
    const wrapper = createWrapper()

    await wrapper.findAll('button').find((button) => button.text() === 'Upstream Request')?.trigger('click')

    expect(wrapper.find('[data-test="upstreamRequest-panel"] [data-test="payload-empty"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('No upstream request content is available for this trace.')
    expect(wrapper.find('[data-test="upstreamRequest-panel"] pre').exists()).toBe(false)
  })

  it('loads raw request content before opening the inbound full dialog', async () => {
    const wrapper = createWrapper({
      detail: {
        ...detail,
        inbound_request_json: '',
        raw_access_allowed: true,
        raw_available: true
      }
    })

    await wrapper.findAll('button').find((button) => button.text() === 'Inbound Request')?.trigger('click')
    const inboundPanelButtons = wrapper.find('[data-test="inbound-panel"]').findAll('button')

    await inboundPanelButtons[1].trigger('click')

    expect(wrapper.emitted('loadRaw')).toHaveLength(1)

    await wrapper.setProps({
      rawDetail: {
        id: 1,
        request_id: 'req-1',
        raw_request: '{"raw":true}',
        raw_response: '{"ok":true}'
      }
    })

    expect(wrapper.find('[data-test="request-details-full-dialog"]').text()).toContain('"raw": true')
  })

  it('falls back to preview content when raw access is not allowed', async () => {
    const wrapper = createWrapper({
      detail: {
        ...detail,
        raw_access_allowed: false,
        raw_available: true,
        inbound_request_json: '{"preview":"only"}'
      }
    })

    await wrapper.findAll('button').find((button) => button.text() === 'Inbound Request')?.trigger('click')
    const inboundPanelButtons = wrapper.find('[data-test="inbound-panel"]').findAll('button')

    await inboundPanelButtons[1].trigger('click')

    expect(wrapper.emitted('loadRaw')).toBeUndefined()
    expect(wrapper.text()).toContain('Only the captured preview is available here. Raw full text is not accessible for this tab.')
    expect(wrapper.find('[data-test="request-details-full-dialog"]').text()).toContain('"preview": "only"')
  })
})
