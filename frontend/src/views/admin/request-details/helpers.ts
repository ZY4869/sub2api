import type { LocationQuery } from 'vue-router'
import type { OpsRequestTraceFilter } from '@/api/admin/ops'

const TIME_RANGES = new Set(['5m', '30m', '1h', '6h', '24h', '7d', '30d'])
const SORTS = new Set(['created_at_desc', 'duration_desc'])

function firstQueryValue(value: LocationQuery[string]): string {
  if (Array.isArray(value)) {
    return String(value[0] || '').trim()
  }
  return String(value || '').trim()
}

function parseOptionalNumber(value: string): number | null | undefined {
  if (!value) return undefined
  const parsed = Number.parseInt(value, 10)
  if (!Number.isFinite(parsed) || parsed <= 0) return null
  return parsed
}

function parseOptionalBool(value: string): boolean | undefined {
  if (!value) return undefined
  switch (value.toLowerCase()) {
    case '1':
    case 'true':
    case 'yes':
      return true
    case '0':
    case 'false':
    case 'no':
      return false
    default:
      return undefined
  }
}

export function createDefaultRequestTraceFilter(): OpsRequestTraceFilter {
  return {
    time_range: '1h',
    sort: 'created_at_desc',
    page: 1,
    page_size: 20,
    q: '',
    status: '',
    platform: '',
    protocol_in: '',
    protocol_out: '',
    channel: '',
    route_path: '',
    request_type: '',
    finish_reason: '',
    capture_reason: '',
    requested_model: '',
    upstream_model: '',
    request_id: '',
    client_request_id: '',
    upstream_request_id: ''
  }
}

export function parseRequestTraceFilterFromQuery(query: LocationQuery): OpsRequestTraceFilter {
  const defaults = createDefaultRequestTraceFilter()
  const timeRange = firstQueryValue(query.time_range)
  const sort = firstQueryValue(query.sort)

  const result: OpsRequestTraceFilter = {
    ...defaults,
    time_range: TIME_RANGES.has(timeRange) ? (timeRange as OpsRequestTraceFilter['time_range']) : defaults.time_range,
    sort: SORTS.has(sort) ? sort as OpsRequestTraceFilter['sort'] : defaults.sort,
    start_time: firstQueryValue(query.start_time) || undefined,
    end_time: firstQueryValue(query.end_time) || undefined,
    status: firstQueryValue(query.status),
    platform: firstQueryValue(query.platform),
    protocol_in: firstQueryValue(query.protocol_in),
    protocol_out: firstQueryValue(query.protocol_out),
    channel: firstQueryValue(query.channel),
    route_path: firstQueryValue(query.route_path),
    request_type: firstQueryValue(query.request_type),
    finish_reason: firstQueryValue(query.finish_reason),
    capture_reason: firstQueryValue(query.capture_reason),
    requested_model: firstQueryValue(query.requested_model),
    upstream_model: firstQueryValue(query.upstream_model),
    request_id: firstQueryValue(query.request_id),
    client_request_id: firstQueryValue(query.client_request_id),
    upstream_request_id: firstQueryValue(query.upstream_request_id),
    q: firstQueryValue(query.q),
    page: parseOptionalNumber(firstQueryValue(query.page)) || defaults.page,
    page_size: parseOptionalNumber(firstQueryValue(query.page_size)) || defaults.page_size,
    user_id: parseOptionalNumber(firstQueryValue(query.user_id)),
    api_key_id: parseOptionalNumber(firstQueryValue(query.api_key_id)),
    account_id: parseOptionalNumber(firstQueryValue(query.account_id)),
    group_id: parseOptionalNumber(firstQueryValue(query.group_id)),
    status_code: parseOptionalNumber(firstQueryValue(query.status_code)),
    stream: parseOptionalBool(firstQueryValue(query.stream)),
    has_tools: parseOptionalBool(firstQueryValue(query.has_tools)),
    has_thinking: parseOptionalBool(firstQueryValue(query.has_thinking)),
    raw_available: parseOptionalBool(firstQueryValue(query.raw_available)),
    sampled: parseOptionalBool(firstQueryValue(query.sampled))
  }

  return result
}

export function buildRequestTraceQuery(filter: OpsRequestTraceFilter): Record<string, string> {
  const query: Record<string, string> = {}
  const assignText = (key: keyof OpsRequestTraceFilter) => {
    const value = filter[key]
    if (typeof value === 'string' && value.trim()) {
      query[String(key)] = value.trim()
    }
  }
  const assignNumber = (key: keyof OpsRequestTraceFilter) => {
    const value = filter[key]
    if (typeof value === 'number' && Number.isFinite(value) && value > 0) {
      query[String(key)] = String(value)
    }
  }
  const assignBool = (key: keyof OpsRequestTraceFilter) => {
    const value = filter[key]
    if (typeof value === 'boolean') {
      query[String(key)] = value ? '1' : '0'
    }
  }

  assignText('time_range')
  assignText('start_time')
  assignText('end_time')
  assignText('status')
  assignText('platform')
  assignText('protocol_in')
  assignText('protocol_out')
  assignText('channel')
  assignText('route_path')
  assignText('request_type')
  assignText('finish_reason')
  assignText('capture_reason')
  assignText('requested_model')
  assignText('upstream_model')
  assignText('request_id')
  assignText('client_request_id')
  assignText('upstream_request_id')
  assignText('q')
  assignText('sort')
  assignNumber('page')
  assignNumber('page_size')
  assignNumber('user_id')
  assignNumber('api_key_id')
  assignNumber('account_id')
  assignNumber('group_id')
  assignNumber('status_code')
  assignBool('stream')
  assignBool('has_tools')
  assignBool('has_thinking')
  assignBool('raw_available')
  assignBool('sampled')

  return query
}

export function formatDurationMs(value?: number | null): string {
  if (typeof value !== 'number' || !Number.isFinite(value)) return '-'
  if (value < 1000) return `${Math.round(value)} ms`
  return `${(value / 1000).toFixed(value >= 10000 ? 0 : 2)} s`
}

export function formatPercent(value: number, total: number): string {
  if (!total) return '0%'
  return `${((value / total) * 100).toFixed(1)}%`
}

export function formatPrettyJSON(raw?: string): string {
  if (!raw) return ''
  try {
    return JSON.stringify(JSON.parse(raw), null, 2)
  } catch {
    return raw
  }
}

export function getProtocolPairLabel(protocolIn?: string, protocolOut?: string): string {
  const inbound = (protocolIn || 'unknown').trim() || 'unknown'
  const outbound = (protocolOut || 'unknown').trim() || 'unknown'
  return `${inbound} -> ${outbound}`
}

export function getStatusBadgeClass(status?: string): string {
  switch ((status || '').toLowerCase()) {
    case 'success':
      return 'badge-success'
    case 'error':
      return 'badge-danger'
    case 'blocked':
      return 'badge-warning'
    default:
      return 'badge-gray'
  }
}
