import type { LocationQuery } from 'vue-router'
import type { OpsRequestTraceDetail, OpsRequestTraceFilter, OpsRequestTraceListItem } from '@/api/admin/ops'
import { getModelRegistrySnapshot } from '@/stores/modelRegistry'
import { parseRequestPreviewContent } from '@/utils/requestPreview'

const TIME_RANGES = new Set(['5m', '30m', '1h', '6h', '24h', '7d', '30d'])
const SORTS = new Set(['created_at_desc', 'duration_desc'])
const EMPTY_VALUE = '-'

type TraceRow = OpsRequestTraceListItem | OpsRequestTraceDetail
type TranslateFn = (key: string, params?: Record<string, unknown>) => string

export interface RequestTraceField {
  label: string
  value: string
  mono?: boolean
}

export interface RequestTraceBadge {
  key: string
  label: string
  className: string
}

export interface RequestTraceModelPresentation {
  modelId: string
  displayName: string
  provider: string
}

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

function toDisplayValue(value?: string | number | boolean | null): string {
  if (value === null || typeof value === 'undefined') return EMPTY_VALUE
  if (typeof value === 'string') {
    const trimmed = value.trim()
    return trimmed || EMPTY_VALUE
  }
  return String(value)
}

function normalizeTranslationKey(value?: string | null): string {
  return String(value || '')
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '_')
    .replace(/^_+|_+$/g, '')
}

function translateMappedValue(t: TranslateFn, baseKey: string, value?: string | null): string {
  const raw = toDisplayValue(value)
  if (raw === EMPTY_VALUE) return raw
  const normalizedKey = normalizeTranslationKey(raw)
  if (!normalizedKey) return raw
  const translationKey = `${baseKey}.${normalizedKey}`
  const translated = t(translationKey)
  return translated === translationKey ? raw : translated
}

function createField(label: string, value?: string | number | boolean | null, mono = false): RequestTraceField {
  return {
    label,
    value: toDisplayValue(value),
    mono
  }
}

function findModelEntry(modelId: string) {
  const normalizedId = modelId.trim().toLowerCase()
  if (!normalizedId) return null
  const snapshot = getModelRegistrySnapshot()
  return snapshot.models.find((entry) => {
    const candidates = [entry.id, ...entry.protocol_ids, ...entry.aliases]
    return candidates.some((candidate) => candidate.trim().toLowerCase() === normalizedId)
  }) || null
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
    upstream_request_id: '',
    gemini_surface: '',
    billing_rule_id: '',
    probe_action: ''
  }
}

export function parseRequestTraceFilterFromQuery(query: LocationQuery): OpsRequestTraceFilter {
  const defaults = createDefaultRequestTraceFilter()
  const timeRange = firstQueryValue(query.time_range)
  const sort = firstQueryValue(query.sort)

  return {
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
    gemini_surface: firstQueryValue(query.gemini_surface),
    billing_rule_id: firstQueryValue(query.billing_rule_id),
    probe_action: firstQueryValue(query.probe_action),
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
  assignText('gemini_surface')
  assignText('billing_rule_id')
  assignText('probe_action')
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
  if (typeof value !== 'number' || !Number.isFinite(value)) return EMPTY_VALUE
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

export function getRequestTraceStatusLabel(t: TranslateFn, status?: string): string {
  return translateMappedValue(t, 'admin.requestDetails.presentation.status', status)
}

export function getRequestTraceProtocolLabel(t: TranslateFn, protocol?: string): string {
  return translateMappedValue(t, 'admin.requestDetails.presentation.protocols', protocol)
}

export function getRequestTraceRequestTypeLabel(t: TranslateFn, requestType?: string): string {
  return translateMappedValue(t, 'admin.requestDetails.presentation.requestTypes', requestType)
}

export function getRequestTraceFinishReasonLabel(t: TranslateFn, finishReason?: string): string {
  return translateMappedValue(t, 'admin.requestDetails.presentation.finishReasons', finishReason)
}

export function getRequestTraceCaptureReasonLabel(t: TranslateFn, captureReason?: string): string {
  return translateMappedValue(t, 'admin.requestDetails.presentation.captureReasons', captureReason)
}

export function getRequestTraceThinkingSourceLabel(t: TranslateFn, thinkingSource?: string): string {
  return translateMappedValue(t, 'admin.requestDetails.presentation.thinkingSources', thinkingSource)
}

export function getRequestTraceThinkingLevelLabel(t: TranslateFn, thinkingLevel?: string): string {
  return translateMappedValue(t, 'admin.requestDetails.presentation.thinkingLevels', thinkingLevel)
}

export function resolveRequestTraceModelPresentation(model?: string | null): RequestTraceModelPresentation | null {
  const modelId = String(model || '').trim()
  if (!modelId) return null
  const entry = findModelEntry(modelId)
  if (!entry) {
    return {
      modelId,
      displayName: modelId,
      provider: ''
    }
  }

  return {
    modelId,
    displayName: entry.display_name || entry.id || modelId,
    provider: entry.provider || ''
  }
}

export function getProtocolPairLabel(t: TranslateFn, protocolIn?: string, protocolOut?: string): string {
  const inbound = getRequestTraceProtocolLabel(t, protocolIn || 'unknown')
  const outbound = getRequestTraceProtocolLabel(t, protocolOut || 'unknown')
  return `${inbound} -> ${outbound}`
}

export function getRequestTraceSubjectFields(t: TranslateFn, item: TraceRow): RequestTraceField[] {
  return [
    createField(t('admin.requestDetails.presentation.labels.userId'), item.user_id),
    createField(t('admin.requestDetails.presentation.labels.apiKeyId'), item.api_key_id),
    createField(t('admin.requestDetails.presentation.labels.accountId'), item.account_id),
    createField(t('admin.requestDetails.presentation.labels.groupId'), item.group_id)
  ]
}

export function getRequestTraceRouteFields(t: TranslateFn, item: TraceRow): RequestTraceField[] {
  return [
    createField(t('admin.requestDetails.presentation.labels.routePath'), item.route_path),
    createField(t('admin.requestDetails.presentation.labels.channel'), item.channel),
    createField(t('admin.requestDetails.presentation.labels.platform'), item.platform),
    createField(t('admin.requestDetails.presentation.labels.geminiSurface'), item.gemini_surface),
    createField(t('admin.requestDetails.presentation.labels.probeAction'), item.probe_action),
    createField(
      t('admin.requestDetails.presentation.labels.protocolPair'),
      getProtocolPairLabel(t, item.protocol_in, item.protocol_out)
    )
  ]
}

export function getRequestTraceExecutionFields(t: TranslateFn, item: TraceRow): RequestTraceField[] {
  return [
    createField(t('admin.requestDetails.presentation.labels.status'), getRequestTraceStatusLabel(t, item.status)),
    createField(t('admin.requestDetails.presentation.labels.requestType'), getRequestTraceRequestTypeLabel(t, item.request_type)),
    createField(t('admin.requestDetails.presentation.labels.finishReason'), getRequestTraceFinishReasonLabel(t, item.finish_reason)),
    createField(t('admin.requestDetails.presentation.labels.captureReason'), getRequestTraceCaptureReasonLabel(t, item.capture_reason)),
    createField(t('admin.requestDetails.presentation.labels.statusCode'), item.status_code),
    createField(t('admin.requestDetails.presentation.labels.upstreamStatusCode'), item.upstream_status_code),
    createField(t('admin.requestDetails.presentation.labels.duration'), formatDurationMs(item.duration_ms)),
    createField(t('admin.requestDetails.presentation.labels.ttft'), formatDurationMs(item.ttft_ms)),
    createField(t('admin.requestDetails.presentation.labels.totalTokens'), item.total_tokens),
    createField(
      t('admin.requestDetails.presentation.labels.tokenBreakdown'),
      `${toDisplayValue(item.input_tokens)} / ${toDisplayValue(item.output_tokens)}`
    )
  ]
}

export function getRequestTraceIdentityFields(t: TranslateFn, item: TraceRow): RequestTraceField[] {
  return [
    createField(t('admin.requestDetails.presentation.labels.requestId'), item.request_id, true),
    createField(t('admin.requestDetails.presentation.labels.clientRequestId'), item.client_request_id, true),
    createField(t('admin.requestDetails.presentation.labels.upstreamRequestId'), item.upstream_request_id, true),
    createField(t('admin.requestDetails.presentation.labels.billingRuleId'), item.billing_rule_id, true)
  ]
}

export function getRequestTraceCapabilityFields(t: TranslateFn, item: TraceRow): RequestTraceField[] {
  return [
    createField(t('admin.requestDetails.presentation.labels.thinkingSource'), getRequestTraceThinkingSourceLabel(t, item.thinking_source)),
    createField(t('admin.requestDetails.presentation.labels.thinkingLevel'), getRequestTraceThinkingLevelLabel(t, item.thinking_level)),
    createField(t('admin.requestDetails.presentation.labels.tokenSource'), item.count_tokens_source),
    createField(t('admin.requestDetails.presentation.labels.mediaResolution'), item.media_resolution)
  ]
}

export function getRequestTraceFlagBadges(t: TranslateFn, item: TraceRow): RequestTraceBadge[] {
  return [
    {
      key: 'stream',
      label: item.stream
        ? t('admin.requestDetails.presentation.flags.streamEnabled')
        : t('admin.requestDetails.presentation.flags.streamDisabled'),
      className: item.stream ? 'badge-primary' : 'badge-gray'
    },
    {
      key: 'tools',
      label: item.has_tools
        ? t('admin.requestDetails.presentation.flags.toolsEnabled')
        : t('admin.requestDetails.presentation.flags.toolsDisabled'),
      className: item.has_tools ? 'badge-success' : 'badge-gray'
    },
    {
      key: 'thinking',
      label: item.has_thinking
        ? t('admin.requestDetails.presentation.flags.thinkingEnabled')
        : t('admin.requestDetails.presentation.flags.thinkingDisabled'),
      className: item.has_thinking ? 'badge-warning' : 'badge-gray'
    },
    {
      key: 'raw',
      label: item.raw_available
        ? t('admin.requestDetails.presentation.flags.rawAvailable')
        : t('admin.requestDetails.presentation.flags.rawUnavailable'),
      className: item.raw_available ? 'badge-success' : 'badge-gray'
    },
    {
      key: 'sampled',
      label: item.sampled
        ? t('admin.requestDetails.presentation.flags.sampled')
        : t('admin.requestDetails.presentation.flags.notSampled'),
      className: item.sampled ? 'badge-primary' : 'badge-gray'
    }
  ]
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

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function resolveFirstNonEmptyString(payload: unknown, keys: string[]): string {
  if (!isRecord(payload)) {
    return ''
  }
  for (const key of keys) {
    const value = payload[key]
    if (typeof value === 'string') {
      const trimmed = value.trim()
      if (trimmed) return trimmed
    }
  }
  return ''
}

function extractErrorMessageFromPayload(payload: unknown): string {
  if (typeof payload === 'string') {
    return payload.trim()
  }
  if (!payload) {
    return ''
  }
  const direct = resolveFirstNonEmptyString(payload, [
    'error_message',
    'message',
    'error',
    'detail',
    'msg',
    'reason',
    'description',
  ])
  if (direct) return direct

  if (isRecord(payload)) {
    const nested = payload.error
    if (typeof nested === 'string') {
      return nested.trim()
    }
    const nestedDirect = resolveFirstNonEmptyString(nested, [
      'error_message',
      'message',
      'error',
      'detail',
      'msg',
      'reason',
      'description',
    ])
    if (nestedDirect) return nestedDirect
  }

  return ''
}

function extractErrorCodeFromPayload(payload: unknown): string {
  const direct = resolveFirstNonEmptyString(payload, ['error_code', 'code', 'type', 'reason'])
  if (direct) return direct

  if (isRecord(payload)) {
    const nested = payload.error
    const nestedDirect = resolveFirstNonEmptyString(nested, ['error_code', 'code', 'type', 'reason'])
    if (nestedDirect) return nestedDirect
  }

  return ''
}

function truncateForCopy(text: string, limit = 8000): string {
  const normalized = String(text || '').trim()
  if (!normalized) return ''
  if (normalized.length <= limit) return normalized
  return `${normalized.slice(0, limit)}...`
}

export function extractRequestTraceErrorText(detail?: OpsRequestTraceDetail | null): string {
  if (!detail) return ''
  const candidates = [
    parseRequestPreviewContent(detail.gateway_response_json),
    parseRequestPreviewContent(detail.upstream_response_json),
  ]

  for (const parsed of candidates) {
    if (!parsed.hasContent) continue
    const extracted = extractErrorMessageFromPayload(parsed.payload)
    if (extracted) return truncateForCopy(extracted)
    if (parsed.displayContent) return truncateForCopy(parsed.displayContent)
  }

  return ''
}

export function extractRequestTraceErrorCode(detail?: OpsRequestTraceDetail | null): string {
  if (!detail) return ''
  const candidates = [
    parseRequestPreviewContent(detail.gateway_response_json),
    parseRequestPreviewContent(detail.upstream_response_json),
  ]

  for (const parsed of candidates) {
    if (!parsed.hasContent) continue
    const extracted = extractErrorCodeFromPayload(parsed.payload)
    if (extracted) return extracted
  }

  return ''
}

export function buildCopyableRequestTraceErrorSummary(detail?: OpsRequestTraceDetail | null): string {
  const requestID = String(detail?.request_id || '').trim()
  const statusCode = detail?.status_code == null ? '' : String(detail.status_code)
  const upstreamStatusCode = detail?.upstream_status_code == null ? '' : String(detail.upstream_status_code)
  const httpStatus = upstreamStatusCode ? `${statusCode || '-'} (upstream ${upstreamStatusCode})` : (statusCode || '-')

  const errorCode = extractRequestTraceErrorCode(detail)
  const errorText = extractRequestTraceErrorText(detail)

  return [
    `request_id: ${requestID || '-'}`,
    `http_status: ${httpStatus || '-'}`,
    `error_code: ${errorCode || '-'}`,
    `error_message: ${errorText || '-'}`,
  ].join('\n')
}
