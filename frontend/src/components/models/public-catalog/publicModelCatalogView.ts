import type {
  PublicModelCatalogItem,
  PublicModelCatalogPriceEntry,
  PublicModelCatalogStatusItem,
  PublicModelHealthStatus,
} from '@/api/meta'

export type Translate = (key: string, params?: Record<string, unknown>) => string

export interface PublicModelCardView {
  contextLabel: string
  contextSourceLabel: string
  modalityLabel: string
  modalityIcon: 'document' | 'photo' | 'speakerWave' | 'videoCamera' | 'cube'
  modalityClass: string
  statusLabel: string
  statusBadgeClass: string
  statusDotClass: string
  publishedStatusLabel: string
  publishedStatusClass: string
  healthSourceLabel: string
  healthReasonLabel: string
  hasHealthMetrics: boolean
  lifecycleLabel: string
  lifecycleClass: string
  lifecycleSourceLabel: string
  demoLabel: string
}

export function buildPublicModelCardView(
  item: PublicModelCatalogItem,
  health: PublicModelCatalogStatusItem | undefined,
  t: Translate,
): PublicModelCardView {
  return {
    contextLabel: formatContextWindow(item.context_window?.tokens || item.context_window_tokens),
    contextSourceLabel: sourceLabel(t, item.context_window?.source, item.context_window?.verified),
    ...modalityConfig(item.modalities || [], t),
    statusLabel: healthStatusLabel(t, health?.health_status || item.health_status),
    statusBadgeClass: healthBadgeClass(health?.health_status || item.health_status),
    statusDotClass: healthDotClass(health?.health_status || item.health_status),
    publishedStatusLabel: publishedStatusLabel(t, item),
    publishedStatusClass: publishedStatusClass(item),
    healthSourceLabel: healthSourceLabel(t, health?.health_source),
    healthReasonLabel: healthReasonLabel(t, health?.status_reason),
    hasHealthMetrics: hasHealthMetrics(health),
    lifecycleLabel: lifecycleLabel(t, item.lifecycle?.status || item.lifecycle_status),
    lifecycleClass: lifecycleClass(item.lifecycle?.status || item.lifecycle_status),
    lifecycleSourceLabel: lifecycleSourceLabel(t, item.lifecycle),
    demoLabel: item.is_demo ? t('ui.modelCatalog.demo') : '',
  }
}

export function formatContextWindow(tokens?: number): string {
  if (!tokens || !Number.isFinite(tokens) || tokens <= 0) {
    return '-'
  }
  if (tokens >= 1_000_000) {
    return `${formatCompactNumber(tokens / 1_000_000)}M`
  }
  if (tokens >= 1_000) {
    return `${formatCompactNumber(tokens / 1_000)}K`
  }
  return formatCompactNumber(tokens)
}

export function formatRate(value?: number | null): string {
  if (value == null || !Number.isFinite(value)) {
    return '-'
  }
  return `${(value * 100).toFixed(1)}%`
}

export function formatLatency(value?: number | null): string {
  if (value == null || !Number.isFinite(value) || value <= 0) {
    return '-'
  }
  return `${Math.round(value)}ms`
}

export function rateColor(value?: number | null): string {
  if (value == null || !Number.isFinite(value)) {
    return 'text-slate-700 dark:text-slate-200'
  }
  if (value >= 0.99) {
    return 'text-emerald-600 dark:text-emerald-300'
  }
  if (value >= 0.9) {
    return 'text-amber-500 dark:text-amber-300'
  }
  return 'text-rose-600 dark:text-rose-300'
}

export function formatLimit(value?: number | null): string {
  if (value == null || !Number.isFinite(value) || value <= 0) {
    return '-'
  }
  if (value >= 1_000_000) {
    return `${formatCompactNumber(value / 1_000_000)}M`
  }
  if (value >= 1_000) {
    return `${formatCompactNumber(value / 1_000)}K`
  }
  return formatCompactNumber(value)
}

export function priceTheme(entry: PublicModelCatalogPriceEntry): 'blue' | 'emerald' | 'amber' {
  if (entry.id.includes('output')) {
    return 'emerald'
  }
  if (entry.id.includes('cache')) {
    return 'amber'
  }
  return 'blue'
}

export function normalizeHealthStatus(status?: string): PublicModelHealthStatus {
  if (status === 'healthy' || status === 'warning' || status === 'error') {
    return status
  }
  return 'pending'
}

export function hasHealthMetrics(health?: PublicModelCatalogStatusItem): boolean {
  if (!health) return false
  if (
    health.success_rate_today != null ||
    health.success_rate_7d != null ||
    health.latency_ms != null ||
    (health.trend || []).length > 0
  ) {
    return true
  }
  return (health.daily || []).some((day) => day.success_rate != null || day.latency_ms != null || day.status !== 'pending')
}

export function healthStatusLabel(t: Translate, status?: string): string {
  switch (normalizeHealthStatus(status)) {
    case 'healthy':
      return t('ui.modelCatalog.health.healthy')
    case 'warning':
      return t('ui.modelCatalog.health.warning')
    case 'error':
      return t('ui.modelCatalog.health.error')
    default:
      return t('ui.modelCatalog.health.pending')
  }
}

export function healthSourceLabel(t: Translate, source?: string): string {
  switch (source) {
    case 'traffic':
      return t('ui.modelCatalog.healthSource.traffic')
    case 'probe':
      return t('ui.modelCatalog.healthSource.probe')
    default:
      return t('ui.modelCatalog.healthSource.none')
  }
}

export function healthReasonLabel(t: Translate, reason?: string): string {
  switch (reason) {
    case 'traffic_recent':
      return t('ui.modelCatalog.healthReason.trafficRecent')
    case 'probe_recent':
      return t('ui.modelCatalog.healthReason.probeRecent')
    case 'monitor_disabled':
      return t('ui.modelCatalog.healthReason.monitorDisabled')
    case 'no_history':
      return t('ui.modelCatalog.healthReason.noHistory')
    case 'stale_history':
      return t('ui.modelCatalog.healthReason.staleHistory')
    case 'checking':
      return t('ui.modelCatalog.healthReason.checking')
    default:
      return t('ui.modelCatalog.healthReason.noHistory')
  }
}

export function publishedStatusLabel(t: Translate, item: PublicModelCatalogItem): string {
  if (item.publication_status === 'published') {
    return t('ui.modelCatalog.publishStatus.published')
  }
  if (item.verification_source === 'live_fallback') {
    return t('ui.modelCatalog.publishStatus.liveFallback')
  }
  return t('ui.modelCatalog.publishStatus.unknown')
}

export function publishedStatusClass(item: PublicModelCatalogItem): string {
  if (item.publication_status === 'published') {
    return 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-500/30 dark:bg-emerald-500/10 dark:text-emerald-200'
  }
  if (item.verification_source === 'live_fallback') {
    return 'border-sky-200 bg-sky-50 text-sky-700 dark:border-sky-500/30 dark:bg-sky-500/10 dark:text-sky-200'
  }
  return 'border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-200'
}

export function healthLabels(t: Translate): Record<PublicModelHealthStatus, string> {
  return {
    healthy: healthStatusLabel(t, 'healthy'),
    warning: healthStatusLabel(t, 'warning'),
    error: healthStatusLabel(t, 'error'),
    pending: healthStatusLabel(t, 'pending'),
  }
}

export function healthBadgeClass(status?: string): string {
  switch (normalizeHealthStatus(status)) {
    case 'healthy':
      return 'border-teal-200 bg-teal-50/80 text-teal-600 dark:border-teal-500/30 dark:bg-teal-500/10 dark:text-teal-200'
    case 'warning':
      return 'border-amber-200 bg-amber-50/80 text-amber-600 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-200'
    case 'error':
      return 'border-rose-200 bg-rose-50/80 text-rose-600 dark:border-rose-500/30 dark:bg-rose-500/10 dark:text-rose-200'
    default:
      return 'border-cyan-200 bg-cyan-50/50 text-cyan-600 dark:border-cyan-500/30 dark:bg-cyan-500/10 dark:text-cyan-200'
  }
}

export function healthDotClass(status?: string): string {
  switch (normalizeHealthStatus(status)) {
    case 'healthy':
      return 'bg-teal-500'
    case 'warning':
      return 'bg-amber-500'
    case 'error':
      return 'bg-rose-500 animate-pulse'
    default:
      return 'bg-cyan-400'
  }
}

export function lifecycleLabel(t: Translate, status?: string): string {
  switch (status) {
    case 'beta':
      return t('ui.modelCatalog.lifecycle.beta')
    case 'deprecated':
      return t('ui.modelCatalog.lifecycle.deprecated')
    default:
      return t('ui.modelCatalog.lifecycle.stable')
  }
}

export function lifecycleClass(status?: string): string {
  switch (status) {
    case 'beta':
      return 'border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-200'
    case 'deprecated':
      return 'border-rose-200 bg-rose-50 text-rose-700 dark:border-rose-500/30 dark:bg-rose-500/10 dark:text-rose-200'
    default:
      return 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-500/30 dark:bg-emerald-500/10 dark:text-emerald-200'
  }
}

export function supportLabel(t: Translate, support?: string): string {
  switch (support) {
    case 'supported':
      return t('ui.modelCatalog.support.supported')
    case 'partial':
      return t('ui.modelCatalog.support.partial')
    case 'unsupported':
      return t('ui.modelCatalog.support.unsupported')
    default:
      return t('ui.modelCatalog.support.unknown')
  }
}

export function sourceLabel(t: Translate, source?: string, verified?: boolean): string {
  if (verified) {
    return t('ui.modelCatalog.source.verified')
  }
  switch (source) {
    case 'verified_probe':
    case 'runtime_observed':
    case 'account_probe':
      return t('ui.modelCatalog.source.probe')
    case 'official_registry':
    case 'manual_config':
      return t('ui.modelCatalog.source.declared')
    case 'pricing_catalog':
      return t('ui.modelCatalog.source.pricing')
    case 'published_snapshot':
      return t('ui.modelCatalog.source.snapshot')
    case 'inferred':
      return t('ui.modelCatalog.source.inferred')
    default:
      return t('ui.modelCatalog.source.unknown')
  }
}

export function lifecycleSourceLabel(t: Translate, lifecycle?: PublicModelCatalogItem['lifecycle']): string {
  if (lifecycle?.confidence === 'inferred' || lifecycle?.source === 'inferred') {
    return t('ui.modelCatalog.source.inferred')
  }
  return sourceLabel(t, lifecycle?.source, lifecycle?.confidence === 'verified')
}

function modalityConfig(modalities: string[], t: Translate) {
  const normalized = modalities.map((item) => item.toLowerCase())
  if (normalized.length > 1) {
    return {
      modalityLabel: t('ui.modelCatalog.modalities.multi'),
      modalityIcon: 'cube' as const,
      modalityClass: 'border-fuchsia-200 bg-fuchsia-50/80 text-fuchsia-600 dark:border-fuchsia-500/30 dark:bg-fuchsia-500/10 dark:text-fuchsia-200',
    }
  }
  if (normalized.includes('video')) {
    return {
      modalityLabel: t('ui.modelCatalog.modalities.video'),
      modalityIcon: 'videoCamera' as const,
      modalityClass: 'border-rose-200 bg-rose-50 text-rose-600 dark:border-rose-500/30 dark:bg-rose-500/10 dark:text-rose-200',
    }
  }
  if (normalized.includes('image')) {
    return {
      modalityLabel: t('ui.modelCatalog.modalities.image'),
      modalityIcon: 'photo' as const,
      modalityClass: 'border-violet-200 bg-violet-50 text-violet-600 dark:border-violet-500/30 dark:bg-violet-500/10 dark:text-violet-200',
    }
  }
  if (normalized.includes('audio')) {
    return {
      modalityLabel: t('ui.modelCatalog.modalities.audio'),
      modalityIcon: 'speakerWave' as const,
      modalityClass: 'border-cyan-200 bg-cyan-50 text-cyan-600 dark:border-cyan-500/30 dark:bg-cyan-500/10 dark:text-cyan-200',
    }
  }
  return {
    modalityLabel: t('ui.modelCatalog.modalities.text'),
    modalityIcon: 'document' as const,
    modalityClass: 'border-sky-200 bg-sky-50 text-sky-600 dark:border-sky-500/30 dark:bg-sky-500/10 dark:text-sky-200',
  }
}

function formatCompactNumber(value: number): string {
  return new Intl.NumberFormat(undefined, {
    maximumFractionDigits: value >= 10 ? 0 : 1,
  }).format(value)
}
