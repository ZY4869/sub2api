import type {
  PublicModelCatalogItem,
  PublicModelCatalogPriceEntry,
  PublicModelCatalogStatusItem,
  PublicModelHealthStatus,
} from '@/api/meta'

export type Translate = (key: string, params?: Record<string, unknown>) => string

export interface PublicModelCardView {
  contextLabel: string
  modalityLabel: string
  modalityIcon: 'document' | 'photo' | 'speakerWave' | 'videoCamera' | 'cube'
  modalityClass: string
  statusLabel: string
  statusBadgeClass: string
  statusDotClass: string
  lifecycleLabel: string
  lifecycleClass: string
}

export function buildPublicModelCardView(
  item: PublicModelCatalogItem,
  health: PublicModelCatalogStatusItem | undefined,
  t: Translate,
): PublicModelCardView {
  return {
    contextLabel: formatContextWindow(item.context_window_tokens),
    ...modalityConfig(item.modalities || [], t),
    statusLabel: healthStatusLabel(t, health?.status),
    statusBadgeClass: healthBadgeClass(health?.status),
    statusDotClass: healthDotClass(health?.status),
    lifecycleLabel: lifecycleLabel(t, item.lifecycle_status),
    lifecycleClass: lifecycleClass(item.lifecycle_status),
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
