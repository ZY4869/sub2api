import { describe, expect, it, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'

import UsageTable from '../UsageTable.vue'

const messages: Record<string, string> = {
  'usage.costDetails': 'Cost Breakdown',
  'usage.contextBadgeRequested1M': 'Requested 1M',
  'usage.statusFailed': 'Failed',
  'usage.statusSucceeded': 'Succeeded',
  'usage.httpStatus': 'HTTP Status',
  'usage.errorCode': 'Error Code',
  'usage.errorMessage': 'Error Message',
  'usage.simulatedClientCodex': 'Codex',
  'usage.simulatedClientGeminiCli': 'Gemini CLI',
  'admin.usage.inputCost': 'Input Cost',
  'admin.usage.outputCost': 'Output Cost',
  'admin.usage.cacheCreationCost': 'Cache Creation Cost',
  'admin.usage.cacheReadCost': 'Cache Read Cost',
  'usage.inputTokenPrice': 'Input price',
  'usage.outputTokenPrice': 'Output price',
  'usage.perMillionTokens': '/ 1M tokens',
  'usage.serviceTier': 'Service tier',
  'usage.serviceTierPriority': 'Fast',
  'usage.serviceTierFlex': 'Flex',
  'usage.serviceTierStandard': 'Standard',
  'usage.rate': 'Rate',
  'usage.accountMultiplier': 'Account rate',
  'usage.original': 'Original',
  'usage.userBilled': 'User billed',
  'usage.accountBilled': 'Account billed',
  'admin.usage.cacheCreationTokens': 'Cache Creation Tokens',
  'admin.usage.cacheReadTokens': 'Cache Read Tokens',
  'usage.millionContextRequested': '1M Requested',
  'usage.millionContextEffective': '1M Effective',
  'usage.millionContextSource': '1M Source',
  'usage.millionContextBetaToken': '1M Beta Token',
  'usage.nativeContext': 'Native Context',
  'usage.stream': 'Stream',
  'usage.operationTypeAccountTest': 'Account Test',
  'usage.operationTypeBatchTest': 'Batch Test',
  'usage.operationTypeScheduledTest': 'Scheduled Test',
  'usage.operationTypeAutoRecoveryTest': 'Auto Recovery Probe',
  'common.copy': 'Copy',
}

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => messages[key] ?? key,
    }),
  }
})

vi.mock('@/stores/modelRegistry', () => ({
  getModelRegistrySnapshot: () => ({
    etag: 'usage-table-test-etag',
    updated_at: '2026-05-09T00:00:00Z',
    provider_labels: {},
    presets: [],
    models: [
      {
        id: 'deepseek-v4-pro',
        display_name: 'DeepSeek V4 Pro',
        provider: 'deepseek',
        platforms: ['deepseek'],
        protocol_ids: ['deepseek-v4-pro'],
        aliases: [],
        pricing_lookup_ids: ['deepseek-v4-pro'],
        context_window_tokens: 1_048_576,
        modalities: ['text'],
        capabilities: ['reasoning'],
        ui_priority: 1,
        exposed_in: ['runtime'],
      },
      {
        id: 'doubao-pro-256k',
        display_name: 'Doubao Pro 256K',
        provider: 'doubao',
        platforms: ['doubao'],
        protocol_ids: ['doubao-pro-256k'],
        aliases: [],
        pricing_lookup_ids: ['doubao-pro-256k'],
        context_window_tokens: 262_144,
        modalities: ['text'],
        capabilities: [],
        ui_priority: 1,
        exposed_in: ['runtime'],
      },
    ],
  }),
}))

vi.mock('@/composables/useUsageModelDisplayModePreference', () => ({
  useUsageModelDisplayModePreference: () => ({
    usageModelDisplayMode: 'model_only',
    updatingUsageModelDisplayMode: false,
    setUsageModelDisplayMode: vi.fn(),
  }),
}))

vi.mock('@/composables/useUsageContextBadgeDisplayModePreference', () => ({
  useUsageContextBadgeDisplayModePreference: () => ({
    usageContextBadgeDisplayMode: 'request_only',
    updatingUsageContextBadgeDisplayMode: false,
    setUsageContextBadgeDisplayMode: vi.fn(),
  }),
}))

const DataTableStub = {
  props: ['data', 'rowKey'],
  template: `
    <div>
      <span data-testid="row-key-prop">{{ rowKey }}</span>
      <div v-for="(row, index) in data" :key="(row && row[rowKey]) || row.request_id || index">
        <slot name="cell-model" :row="row" />
        <slot name="cell-native_context" :row="row" />
        <slot name="cell-status" :row="row" />
        <slot name="cell-reasoning_effort" :row="row" />
        <slot name="cell-request_protocol" :row="row" />
        <slot name="cell-stream" :row="row" />
        <slot name="cell-tokens" :row="row" />
        <slot name="cell-cost" :row="row" />
      </div>
    </div>
  `,
}

function mountUsageTable(
  data: Record<string, unknown>[],
  options: {
    props?: Record<string, unknown>
    stubs?: Record<string, unknown>
  } = {},
) {
  return mount(UsageTable, {
    props: {
      data,
      loading: false,
      columns: [],
      usageModelDisplayMode: 'model_only',
      usageContextBadgeDisplayMode: 'request_only',
      ...options.props,
    },
    global: {
      stubs: {
        DataTable: DataTableStub,
        EmptyState: true,
        Icon: true,
        AccountErrorTooltipButton: false,
        ModelIcon: true,
        Teleport: true,
        UsageContextBadge: {
          props: ['badge'],
          template:
            '<span v-if="badge" data-test="usage-context-badge">{{ badge.labelKey || badge.label }}</span>',
        },
        ...options.stubs,
      },
    },
  })
}

describe('admin UsageTable tooltip', () => {
  beforeEach(() => {
    vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockReturnValue({
      x: 0,
      y: 0,
      top: 20,
      left: 20,
      right: 120,
      bottom: 40,
      width: 100,
      height: 20,
      toJSON: () => ({}),
    } as DOMRect)
  })

  it('shows service tier and billing breakdown in cost tooltip', async () => {
    const row = {
      id: 1,
      request_id: 'req-admin-1',
      actual_cost: 0.092883,
      total_cost: 0.092883,
      account_rate_multiplier: 1,
      rate_multiplier: 1,
      service_tier: 'priority',
      input_cost: 0.020285,
      output_cost: 0.00303,
      cache_creation_cost: 0,
      cache_read_cost: 0.069568,
      input_tokens: 4057,
      output_tokens: 101,
    }

    const wrapper = mountUsageTable([row])

    await wrapper.findAll('.group.relative')[1]?.trigger('mouseenter')
    await nextTick()

    const text = wrapper.text()
    expect(text).toContain('Service tier')
    expect(text).toContain('Fast')
    expect(text).toContain('Rate')
    expect(text).toContain('1.00x')
    expect(text).toContain('Account rate')
    expect(text).toContain('User billed')
    expect(text).toContain('Account billed')
    expect(text).toContain('$0.092883')
    expect(text).toContain('$5.0000 / 1M tokens')
    expect(text).toContain('$30.0000 / 1M tokens')
    expect(text).toContain('$0.069568')
  })

  it('renders failed status rows with simulated client and tooltip error details', async () => {
    const row = {
      id: 2,
      request_id: 'req-admin-failed',
      status: 'failed',
      simulated_client: 'codex',
      http_status: 429,
      error_code: 'rate_limited',
      error_message: 'Rate limit exceeded for this account',
      actual_cost: 0,
      total_cost: 0,
      account_rate_multiplier: 1,
      rate_multiplier: 1,
      input_cost: 0,
      output_cost: 0,
      cache_creation_cost: 0,
      cache_read_cost: 0,
      input_tokens: 0,
      output_tokens: 0,
    }

    const wrapper = mountUsageTable([row])

    const text = wrapper.text()
    expect(text).toContain('Failed')
    expect(text).toContain('Codex')
    expect(text).toContain('Copy')
    expect(text).not.toContain('HTTP Status')
    expect(text).not.toContain('rate_limited')
    expect(text).not.toContain('Rate limit exceeded for this account')

    await wrapper.get('.error-info-trigger').trigger('mouseenter')
    await nextTick()

    expect(wrapper.text()).toContain('http_status: 429')
    expect(wrapper.text()).toContain('error_code: rate_limited')
    expect(wrapper.text()).toContain('error_message: Rate limit exceeded for this account')
  })

  it('renders the request protocol cell with badge text and normalized path', () => {
    const row = {
      id: 3,
      request_id: 'req-admin-protocol',
      inbound_endpoint: '/v1/chat/completions',
      upstream_endpoint: '/v1/responses',
      actual_cost: 0,
      total_cost: 0,
      account_rate_multiplier: 1,
      rate_multiplier: 1,
      input_cost: 0,
      output_cost: 0,
      cache_creation_cost: 0,
      cache_read_cost: 0,
      input_tokens: 0,
      output_tokens: 0,
    }

    const wrapper = mountUsageTable([row])

    const text = wrapper.text()
    expect(text).toContain('OpenAI')
    expect(text).toContain('/v1/chat/completions')
  })

  it('passes the stable id row key to the shared DataTable', () => {
    const wrapper = mountUsageTable([
      {
        id: 99,
        request_id: '',
        input_tokens: 0,
        output_tokens: 0,
        actual_cost: 0,
        total_cost: 0,
      },
    ])

    expect(wrapper.get('[data-testid="row-key-prop"]').text()).toBe('id')
  })

  it('shows DeepSeek cache labels as Cache Hit and Cache Miss', async () => {
    const row = {
      id: 4,
      request_id: 'req-admin-deepseek-cache',
      upstream_service: 'deepseek',
      input_tokens: 12,
      output_tokens: 34,
      cache_creation_tokens: 56,
      cache_read_tokens: 78,
      cache_creation_1h_tokens: 0,
      cache_ttl_overridden: false,
      actual_cost: 0,
      total_cost: 0,
      account_rate_multiplier: 1,
      rate_multiplier: 1,
      input_cost: 0,
      output_cost: 0,
      cache_creation_cost: 0,
      cache_read_cost: 0,
    }

    const wrapper = mountUsageTable([row])

    await wrapper.findAll('.group.relative')[0]?.trigger('mouseenter')
    await nextTick()

    expect(wrapper.text()).toContain('Cache Hit')
    expect(wrapper.text()).toContain('Cache Miss')
  })

  it('does not render 1M capability lines in the reasoning effort cell anymore', () => {
    const row = {
      id: 5,
      request_id: 'req-admin-1m',
      reasoning_effort_raw: 'max',
      reasoning_effort_effective: 'xhigh',
      reasoning_effort: 'xhigh',
      million_context_requested: true,
      million_context_effective: false,
      million_context_source: 'model_suffix_[1m]',
      million_context_beta_token: 'context-1m-2025-08-07',
      actual_cost: 0,
      total_cost: 0,
      account_rate_multiplier: 1,
      rate_multiplier: 1,
      input_cost: 0,
      output_cost: 0,
      cache_creation_cost: 0,
      cache_read_cost: 0,
      input_tokens: 0,
      output_tokens: 0,
    }

    const wrapper = mountUsageTable([row])

    const text = wrapper.text()
    expect(text).toContain('Max -> Xhigh')
    expect(text).not.toContain('1M Requested')
    expect(text).not.toContain('1M Effective')
    expect(text).not.toContain('1M Source')
  })

  it('renders system operation badge alongside transport label', () => {
    const row = {
      id: 6,
      request_id: 'req-admin-system-op',
      request_type: 'stream',
      stream: true,
      operation_type: 'account_test',
      actual_cost: 0,
      total_cost: 0,
      account_rate_multiplier: 1,
      rate_multiplier: 1,
      input_cost: 0,
      output_cost: 0,
      cache_creation_cost: 0,
      cache_read_cost: 0,
      input_tokens: 0,
      output_tokens: 0,
    }

    const wrapper = mountUsageTable([row])

    const text = wrapper.text()
    expect(text).toContain('Stream')
    expect(text).toContain('Account Test')
  })

  it('renders request, native, and combined context badges with the expected order', () => {
    const row = {
      id: 7,
      request_id: 'req-admin-context-badges',
      model: 'deepseek-v4-pro',
      million_context_requested: true,
      million_context_effective: false,
      actual_cost: 0,
      total_cost: 0,
      account_rate_multiplier: 1,
      rate_multiplier: 1,
      input_cost: 0,
      output_cost: 0,
      cache_creation_cost: 0,
      cache_read_cost: 0,
      input_tokens: 0,
      output_tokens: 0,
    }

    const requestOnlyWrapper = mountUsageTable([row], {
      props: {
        usageContextBadgeDisplayMode: 'request_only',
      },
    })
    expect(
      requestOnlyWrapper
        .findAll('[data-test="usage-context-badge"]')
        .map((badge) => badge.text()),
    ).toEqual(['usage.contextBadgeRequested1M'])

    const nativeOnlyWrapper = mountUsageTable([row], {
      props: {
        usageContextBadgeDisplayMode: 'native_only',
      },
    })
    expect(
      nativeOnlyWrapper
        .findAll('[data-test="usage-context-badge"]')
        .map((badge) => badge.text()),
    ).toEqual(['1M'])

    const bothWrapper = mountUsageTable([row], {
      props: {
        usageContextBadgeDisplayMode: 'both',
      },
    })
    expect(
      bothWrapper
        .findAll('[data-test="usage-context-badge"]')
        .map((badge) => badge.text()),
    ).toEqual(['usage.contextBadgeRequested1M', '1M'])
  })

  it('renders native context through the dedicated context badge cell instead of the model cell', () => {
    const row = {
      id: 10,
      request_id: 'req-admin-native-context-column',
      model: 'deepseek-v4-pro',
      million_context_requested: true,
      million_context_effective: false,
      actual_cost: 0,
      total_cost: 0,
      account_rate_multiplier: 1,
      rate_multiplier: 1,
      input_cost: 0,
      output_cost: 0,
      cache_creation_cost: 0,
      cache_read_cost: 0,
      input_tokens: 0,
      output_tokens: 0,
    }

    const wrapper = mountUsageTable([row], {
      stubs: {
        UsageModelCell: {
          props: ['row', 'mode'],
          template: '<div data-test="usage-model-cell">{{ row.model }}|{{ mode }}</div>',
        },
      },
    })

    expect(wrapper.get('[data-test="usage-model-cell"]').text()).toBe('deepseek-v4-pro|model_only')
    expect(wrapper.get('[data-test="usage-context-badges-cell"]').exists()).toBe(true)
    expect(
      wrapper
        .findAll('[data-test="usage-context-badge"]')
        .map((badge) => badge.text()),
    ).toEqual(['usage.contextBadgeRequested1M'])
  })

  it('keeps showing the existing badge in both mode when only one side is available', () => {
    const nativeOnlyRow = {
      id: 8,
      request_id: 'req-admin-native-badge-only',
      model: 'doubao-pro-256k',
      million_context_requested: false,
      million_context_effective: false,
      actual_cost: 0,
      total_cost: 0,
      account_rate_multiplier: 1,
      rate_multiplier: 1,
      input_cost: 0,
      output_cost: 0,
      cache_creation_cost: 0,
      cache_read_cost: 0,
      input_tokens: 0,
      output_tokens: 0,
    }
    const requestOnlyRow = {
      ...nativeOnlyRow,
      id: 9,
      request_id: 'req-admin-request-badge-only',
      model: 'unknown-model',
      million_context_requested: true,
      million_context_effective: false,
    }

    const nativeOnlyWrapper = mountUsageTable([nativeOnlyRow], {
      props: {
        usageContextBadgeDisplayMode: 'both',
      },
    })
    expect(
      nativeOnlyWrapper
        .findAll('[data-test="usage-context-badge"]')
        .map((badge) => badge.text()),
    ).toEqual(['256K'])

    const requestOnlyWrapper = mountUsageTable([requestOnlyRow], {
      props: {
        usageContextBadgeDisplayMode: 'both',
      },
    })
    expect(
      requestOnlyWrapper
        .findAll('[data-test="usage-context-badge"]')
        .map((badge) => badge.text()),
    ).toEqual(['usage.contextBadgeRequested1M'])
  })
})
