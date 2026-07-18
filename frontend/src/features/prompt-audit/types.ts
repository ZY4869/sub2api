import type {
  PromptAuditConfig,
  PromptAuditEndpoint,
  PromptAuditEventFilter,
  PromptAuditUpdateConfig,
  PromptAuditUpdateEndpoint
} from '@/api/admin/promptAudit'

export interface PromptAuditEndpointDraft extends PromptAuditUpdateEndpoint {
  token_mode: 'keep' | 'replace' | 'clear'
}

export interface PromptAuditConfigDraft {
  expected_config_version: number
  enabled: boolean
  blocking_enabled: boolean
  store_pass_events: boolean
  strategy: string
  worker_count: number
  queue_capacity: number
  scanners: string[]
  all_groups: boolean
  group_ids_text: string
  endpoints: PromptAuditEndpointDraft[]
}

export interface PromptAuditState {
  filters: Required<Pick<PromptAuditEventFilter, 'page' | 'page_size'>> &
    Omit<PromptAuditEventFilter, 'page' | 'page_size'>
}

export function draftFromConfig(config: PromptAuditConfig): PromptAuditConfigDraft {
  return {
    expected_config_version: config.config_version,
    enabled: config.enabled,
    blocking_enabled: config.blocking_enabled,
    store_pass_events: config.store_pass_events,
    strategy: config.strategy || 'priority',
    worker_count: config.worker_count || 4,
    queue_capacity: config.queue_capacity || 32768,
    scanners: [...(config.scanners || [])],
    all_groups: config.all_groups,
    group_ids_text: (config.group_ids || []).join(', '),
    endpoints: (config.endpoints || []).map(endpointDraftFromConfig)
  }
}

export function endpointDraftFromConfig(endpoint: PromptAuditEndpoint): PromptAuditEndpointDraft {
  return {
    id: endpoint.id,
    name: endpoint.name,
    protocol: endpoint.protocol || 'openai_compatible',
    base_url: endpoint.base_url,
    model: endpoint.model,
    timeout_ms: endpoint.timeout_ms || 3000,
    input_limit: endpoint.input_limit || 12000,
    enabled: endpoint.enabled,
    has_token: endpoint.has_token,
    token_status: endpoint.token_status,
    token: '',
    clear_token: false,
    token_mode: 'keep'
  }
}

export function payloadFromDraft(draft: PromptAuditConfigDraft): PromptAuditUpdateConfig {
  return {
    expected_config_version: draft.expected_config_version,
    enabled: draft.enabled,
    blocking_enabled: draft.enabled && draft.blocking_enabled,
    store_pass_events: draft.store_pass_events,
    strategy: draft.strategy || 'priority',
    worker_count: Number(draft.worker_count) || 4,
    queue_capacity: Number(draft.queue_capacity) || 32768,
    scanners: [...draft.scanners],
    all_groups: draft.all_groups,
    group_ids: parseIDList(draft.group_ids_text),
    endpoints: draft.endpoints.map(endpointPayloadFromDraft)
  }
}

export function endpointPayloadFromDraft(endpoint: PromptAuditEndpointDraft): PromptAuditUpdateEndpoint {
  return {
    id: endpoint.id.trim(),
    name: endpoint.name.trim(),
    protocol: endpoint.protocol || 'openai_compatible',
    base_url: endpoint.base_url.trim(),
    model: endpoint.model.trim(),
    timeout_ms: Number(endpoint.timeout_ms) || 3000,
    input_limit: Number(endpoint.input_limit) || 12000,
    enabled: endpoint.enabled,
    token: endpoint.token_mode === 'replace' ? endpoint.token?.trim() : undefined,
    clear_token: endpoint.token_mode === 'clear'
  }
}

export function parseIDList(value: string): number[] {
  return value
    .split(/[,\s]+/)
    .map((item) => Number.parseInt(item.trim(), 10))
    .filter((item) => Number.isFinite(item) && item > 0)
}

export function cleanEventFilter(filter: PromptAuditEventFilter): PromptAuditEventFilter {
  const out: PromptAuditEventFilter = { page: filter.page, page_size: filter.page_size }
  for (const key of ['decision', 'risk_level', 'endpoint', 'protocol', 'prompt_hash', 'request_id', 'keyword'] as const) {
    const value = filter[key]?.trim()
    if (value) out[key] = value
  }
  for (const key of ['user_id', 'api_key_id', 'group_id'] as const) {
    const value = filter[key]
    if (typeof value === 'number' && value > 0) out[key] = value
  }
  return out
}
