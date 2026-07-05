import type { OpsErrorLog } from '@/api/admin/ops'
import { statusCodeBadgeClass } from '@/utils/errorBadges'

type Translate = (key: string) => string

export function isUpstreamRow(log: OpsErrorLog): boolean {
  const phase = String(log.phase || '').toLowerCase()
  const owner = String(log.error_owner || '').toLowerCase()
  return phase === 'upstream' && owner === 'provider'
}

export function getTypeBadge(log: OpsErrorLog, t: Translate): { label: string; className: string } {
  const phase = String(log.phase || '').toLowerCase()
  const owner = String(log.error_owner || '').toLowerCase()

  if (isUpstreamRow(log)) {
    return { label: t('admin.ops.errorLog.typeUpstream'), className: 'bg-red-50 text-red-700 ring-red-600/20 dark:bg-red-900/30 dark:text-red-400 dark:ring-red-500/30' }
  }
  if (phase === 'request' && owner === 'client') {
    return { label: t('admin.ops.errorLog.typeRequest'), className: 'bg-amber-50 text-amber-700 ring-amber-600/20 dark:bg-amber-900/30 dark:text-amber-400 dark:ring-amber-500/30' }
  }
  if (phase === 'auth' && owner === 'client') {
    return { label: t('admin.ops.errorLog.typeAuth'), className: 'bg-blue-50 text-blue-700 ring-blue-600/20 dark:bg-blue-900/30 dark:text-blue-400 dark:ring-blue-500/30' }
  }
  if (phase === 'routing' && owner === 'platform') {
    return { label: t('admin.ops.errorLog.typeRouting'), className: 'bg-purple-50 text-purple-700 ring-purple-600/20 dark:bg-purple-900/30 dark:text-purple-400 dark:ring-purple-500/30' }
  }
  if (phase === 'internal' && owner === 'platform') {
    return { label: t('admin.ops.errorLog.typeInternal'), className: 'bg-gray-100 text-gray-800 ring-gray-600/20 dark:bg-dark-700 dark:text-gray-200 dark:ring-dark-500/40' }
  }

  const fallback = phase || owner || t('common.unknown')
  return { label: fallback, className: 'bg-gray-50 text-gray-700 ring-gray-600/10 dark:bg-dark-900 dark:text-gray-300 dark:ring-dark-700' }
}

export function getStatusClass(code: number): string {
  return statusCodeBadgeClass(code)
}

export function getInboundEndpoint(log: OpsErrorLog): string {
  return String(log.inbound_endpoint || log.request_path || '').trim()
}

export function shouldShowUpstreamEndpoint(log: OpsErrorLog): boolean {
  const inbound = getInboundEndpoint(log)
  const upstream = String(log.upstream_endpoint || '').trim()
  return upstream !== '' && upstream !== inbound
}

export function getEndpointTooltip(log: OpsErrorLog): string {
  const inbound = getInboundEndpoint(log)
  const upstream = String(log.upstream_endpoint || '').trim()
  return upstream && upstream !== inbound ? `${inbound} -> ${upstream}` : inbound
}

export function getRequestedModel(log: OpsErrorLog): string {
  return String(log.requested_model || log.upstream_model || log.model || '').trim()
}

export function shouldShowModelMapping(log: OpsErrorLog): boolean {
  const requested = getRequestedModel(log)
  const upstream = String(log.upstream_model || '').trim()
  return requested !== '' && upstream !== '' && upstream !== requested
}

export function getModelTooltip(log: OpsErrorLog): string {
  const requested = getRequestedModel(log)
  const upstream = String(log.upstream_model || '').trim()
  return upstream && upstream !== requested ? `${requested} -> ${upstream}` : requested
}

export function formatRequestType(value: number | null | undefined, t: Translate): string {
  switch (value) {
    case 1:
      return t('admin.ops.errorLog.requestTypeSync')
    case 2:
      return t('admin.ops.errorLog.requestTypeStream')
    case 3:
      return t('admin.ops.errorLog.requestTypeWsV2')
    default:
      return t('common.unknown')
  }
}

export function getRequestTypeClass(value: number | null | undefined): string {
  switch (value) {
    case 3:
      return 'bg-sky-50 text-sky-700 dark:bg-sky-900/30 dark:text-sky-300'
    case 2:
      return 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300'
    case 1:
      return 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-200'
    default:
      return 'bg-gray-100 text-gray-500 dark:bg-dark-700 dark:text-gray-400'
  }
}

export function formatSmartMessage(msg: string, t: Translate): string {
  if (!msg) return ''
  if (msg.startsWith('{') || msg.startsWith('[')) {
    try {
      const obj = JSON.parse(msg)
      if (obj?.error?.message) return String(obj.error.message)
      if (obj?.message) return String(obj.message)
      if (obj?.detail) return String(obj.detail)
      if (typeof obj === 'object') return JSON.stringify(obj).substring(0, 150)
    } catch {
      // Keep the original message if the upstream body is not JSON.
    }
  }
  if (msg.includes('context deadline exceeded')) return t('admin.ops.errorLog.commonErrors.contextDeadlineExceeded')
  if (msg.includes('connection refused')) return t('admin.ops.errorLog.commonErrors.connectionRefused')
  if (msg.toLowerCase().includes('rate limit')) return t('admin.ops.errorLog.commonErrors.rateLimit')
  return msg.length > 200 ? msg.substring(0, 200) + '...' : msg
}
