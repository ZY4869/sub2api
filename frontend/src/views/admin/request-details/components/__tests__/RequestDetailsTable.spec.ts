import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import RequestDetailsTable from '../RequestDetailsTable.vue'

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
  'common.total': '总计',
  'admin.requestDetails.table.title': '请求表格',
  'admin.requestDetails.table.description': '请求表格描述',
  'admin.requestDetails.table.columns.requestId': '请求标识',
  'admin.requestDetails.table.columns.subject': '主体对象',
  'admin.requestDetails.table.columns.route': '路由来源',
  'admin.requestDetails.table.columns.models': '模型',
  'admin.requestDetails.table.columns.status': '状态 / 原因',
  'admin.requestDetails.table.columns.flags': '能力标记',
  'admin.requestDetails.table.columns.actions': '操作',
  'admin.requestDetails.table.view': '查看',
  'admin.requestDetails.presentation.labels.requestId': '请求 ID',
  'admin.requestDetails.presentation.labels.clientRequestId': '客户端请求 ID',
  'admin.requestDetails.presentation.labels.upstreamRequestId': '上游请求 ID',
  'admin.requestDetails.presentation.labels.requestType': '请求类型',
  'admin.requestDetails.presentation.labels.userId': '用户 ID',
  'admin.requestDetails.presentation.labels.apiKeyId': 'API Key ID',
  'admin.requestDetails.presentation.labels.accountId': '账号 ID',
  'admin.requestDetails.presentation.labels.groupId': '分组 ID',
  'admin.requestDetails.presentation.labels.routePath': '路由',
  'admin.requestDetails.presentation.labels.channel': '通道',
  'admin.requestDetails.presentation.labels.platform': '平台',
  'admin.requestDetails.presentation.labels.protocolPair': '协议对',
  'admin.requestDetails.presentation.labels.requestedModel': '请求模型',
  'admin.requestDetails.presentation.labels.upstreamModel': '上游模型',
  'admin.requestDetails.presentation.labels.finishReason': '完成原因',
  'admin.requestDetails.presentation.labels.captureReason': '采集原因',
  'admin.requestDetails.presentation.labels.duration': '总耗时',
  'admin.requestDetails.presentation.labels.ttft': '首字耗时',
  'admin.requestDetails.presentation.labels.totalTokens': '总 Tokens',
  'admin.requestDetails.presentation.labels.thinkingLevel': 'Thinking 强度',
  'admin.requestDetails.presentation.labels.toolKinds': '工具类型',
  'admin.requestDetails.presentation.flags.streamEnabled': '流式',
  'admin.requestDetails.presentation.flags.toolsEnabled': '工具调用',
  'admin.requestDetails.presentation.flags.thinkingEnabled': 'Thinking',
  'admin.requestDetails.presentation.flags.rawAvailable': '有原文',
  'admin.requestDetails.presentation.flags.sampled': '已采样',
  'admin.requestDetails.presentation.status.success': '成功',
  'admin.requestDetails.presentation.requestTypes.chat_completions': '聊天补全',
  'admin.requestDetails.presentation.finishReasons.stop': '正常结束',
  'admin.requestDetails.presentation.captureReasons.sampled': '采样命中',
  'admin.requestDetails.presentation.protocols.openai': 'OpenAI'
}

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => translations[key] ?? key
    })
  }
})

const item = {
  id: 1,
  created_at: '2026-04-04T00:00:00Z',
  request_id: 'req-1',
  client_request_id: 'client-1',
  upstream_request_id: 'upstream-1',
  platform: 'openai',
  protocol_in: 'openai',
  protocol_out: 'openai',
  channel: 'main',
  route_path: '/v1/chat/completions',
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
  raw_access_allowed: true
}

describe('RequestDetailsTable', () => {
  it('renders translated stacked labels and model icons', () => {
    const wrapper = mount(RequestDetailsTable, {
      props: {
        items: [item],
        total: 1,
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

    expect(wrapper.text()).toContain('用户 ID')
    expect(wrapper.text()).toContain('路由')
    expect(wrapper.text()).toContain('请求模型')
    expect(wrapper.text()).toContain('上游模型')
    expect(wrapper.text()).toContain('Claude Opus 4.1')
    expect(wrapper.text()).toContain('成功')
    expect(wrapper.text()).toContain('流式')
    expect(wrapper.text()).toContain('有原文')
    expect(wrapper.findAll('[data-test="model-icon"]')).toHaveLength(2)
  })
})
