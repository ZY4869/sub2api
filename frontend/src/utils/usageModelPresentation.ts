import type { ModelRegistryEntry } from '@/generated/modelRegistry'
import { getModelRegistrySnapshot } from '@/stores/modelRegistry'
import type { UsageContextBadgeDisplayMode, UsageLog, UsageModelDisplayMode } from '@/types'

export interface UsageContextBadgeInfo {
  label: string
  labelKey?: string
  tokens: number
  tier:
    | '4k'
    | '8k'
    | '16k'
    | '32k'
    | '64k'
    | '128k'
    | '200k'
    | '512k'
    | '1m'
    | '2m'
    | '10m'
  muted?: boolean
  title?: string
  titleKey?: string
  titleParams?: Record<string, string | number>
}

export interface UsageModelLinePresentation {
  modelId: string
  displayName: string
  provider: string
  primaryText: string
  secondaryText: string
  requestBadge: UsageContextBadgeInfo | null
  nativeContextBadge: UsageContextBadgeInfo | null
  nativeContextLabel: string
}

export interface UsageModelPresentation {
  requested: UsageModelLinePresentation
  upstream: UsageModelLinePresentation | null
}

const CONTEXT_WINDOW_MARKETING_LABELS: Array<{ min: number; max: number; label: string }> = [
  { min: 3_800, max: 4_200, label: '4K' },
  { min: 7_600, max: 8_400, label: '8K' },
  { min: 15_500, max: 16_500, label: '16K' },
  { min: 31_000, max: 33_000, label: '32K' },
  { min: 62_000, max: 66_000, label: '64K' },
  { min: 120_000, max: 136_000, label: '128K' },
  { min: 190_000, max: 210_000, label: '200K' },
  { min: 240_000, max: 272_000, label: '256K' },
  { min: 500_000, max: 536_000, label: '512K' },
  { min: 980_000, max: 1_080_000, label: '1M' },
  { min: 1_950_000, max: 2_150_000, label: '2M' },
  { min: 9_500_000, max: 10_500_000, label: '10M' },
]

function normalizeModelId(modelId: string | null | undefined): string {
  return String(modelId || '').trim().toLowerCase()
}

export function normalizeUsageModelDisplayMode(mode: string | null | undefined): UsageModelDisplayMode {
  switch (mode) {
    case 'display_only':
      return 'display_only'
    case 'display_and_model':
      return 'display_and_model'
    case 'model_only':
    default:
      return 'model_only'
  }
}

export function findUsageModelEntry(modelId: string | null | undefined): ModelRegistryEntry | null {
  const normalizedId = normalizeModelId(modelId)
  if (!normalizedId) {
    return null
  }
  const snapshot = getModelRegistrySnapshot()
  return snapshot.models.find((entry) => {
    const candidates = [
      entry.id,
      ...entry.protocol_ids,
      ...entry.aliases,
      ...entry.pricing_lookup_ids,
    ]
    return candidates.some((candidate) => normalizeModelId(candidate) === normalizedId)
  }) || null
}

export function formatContextWindowLabel(tokens: number): string {
  const matched = CONTEXT_WINDOW_MARKETING_LABELS.find(
    (candidate) => tokens >= candidate.min && tokens <= candidate.max
  )
  if (matched) {
    return matched.label
  }
  if (tokens >= 10_000_000) return '10M'
  if (tokens >= 1_000_000) {
    const millions = tokens / 1_000_000
    return Number.isInteger(millions) ? `${millions}M` : `${millions.toFixed(1)}M`
  }
  if (tokens >= 1_000) {
    const thousands = tokens / 1_000
    return Number.isInteger(thousands) ? `${thousands}K` : `${Math.round(thousands)}K`
  }
  return String(tokens)
}

export function resolveContextWindowTier(tokens: number): UsageContextBadgeInfo['tier'] {
  if (tokens >= 10_000_000) return '10m'
  if (tokens >= 2_000_000) return '2m'
  if (tokens >= 1_000_000) return '1m'
  if (tokens >= 512_000) return '512k'
  if (tokens >= 200_000) return '200k'
  if (tokens >= 128_000) return '128k'
  if (tokens >= 64_000) return '64k'
  if (tokens >= 32_000) return '32k'
  if (tokens >= 16_000) return '16k'
  if (tokens >= 8_000) return '8k'
  return '4k'
}

function resolveMillionContextBadge(log: Pick<UsageLog, 'million_context_requested' | 'million_context_effective'>): UsageContextBadgeInfo | null {
  if (!log.million_context_requested) {
    return null
  }
  const effective = Boolean(log.million_context_effective)
  return {
    label: effective ? '1M' : 'usage.contextBadgeRequested1M',
    labelKey: effective ? undefined : 'usage.contextBadgeRequested1M',
    tokens: 1_000_000,
    tier: '1m',
    muted: !effective,
    title: effective
      ? 'usage.contextBadgeRequested1MEffective'
      : 'usage.contextBadgeRequested1MPending',
    titleKey: effective
      ? 'usage.contextBadgeRequested1MEffective'
      : 'usage.contextBadgeRequested1MPending'
  }
}

function resolveRegistryContextBadge(modelId: string): UsageContextBadgeInfo | null {
  const entry = findUsageModelEntry(modelId)
  const tokens = Number(entry?.context_window_tokens || 0)
  if (!Number.isFinite(tokens) || tokens <= 0) {
    return null
  }
  return {
    label: formatContextWindowLabel(tokens),
    tokens,
    tier: resolveContextWindowTier(tokens),
    titleKey: 'usage.nativeContextTooltip',
    titleParams: {
      context: formatContextWindowLabel(tokens),
    },
  }
}

function resolvePresentationText(displayName: string, modelId: string, mode: UsageModelDisplayMode) {
  switch (mode) {
    case 'display_only':
      return {
        primaryText: displayName || modelId,
        secondaryText: ''
      }
    case 'display_and_model':
      return {
        primaryText: displayName || modelId,
        secondaryText: displayName && displayName !== modelId ? modelId : ''
      }
    case 'model_only':
    default:
      return {
        primaryText: modelId,
        secondaryText: ''
      }
  }
}

export function buildUsageModelLinePresentation(
  modelId: string | null | undefined,
  mode: UsageModelDisplayMode,
  log?: Pick<UsageLog, 'million_context_requested' | 'million_context_effective'>
): UsageModelLinePresentation {
  const safeModelId = String(modelId || '').trim() || '-'
  const entry = findUsageModelEntry(safeModelId)
  const displayName = String(entry?.display_name || '').trim() || safeModelId
  const provider = String(entry?.provider || '').trim()
  const { primaryText, secondaryText } = resolvePresentationText(displayName, safeModelId, mode)
  const requestBadge = log ? resolveMillionContextBadge(log) : null
  const nativeContextBadge = resolveRegistryContextBadge(safeModelId)
  const nativeContextLabel = nativeContextBadge?.label || '-'

  return {
    modelId: safeModelId,
    displayName,
    provider,
    primaryText,
    secondaryText,
    requestBadge,
    nativeContextBadge,
    nativeContextLabel,
  }
}

export function buildUsageModelPresentation(
  log: Pick<UsageLog, 'model' | 'upstream_model' | 'million_context_requested' | 'million_context_effective'>,
  mode: UsageModelDisplayMode
): UsageModelPresentation {
  const requested = buildUsageModelLinePresentation(log.model, mode, log)
  const upstreamModel = String(log.upstream_model || '').trim()
  const upstream = upstreamModel && upstreamModel !== requested.modelId
    ? buildUsageModelLinePresentation(upstreamModel, mode)
    : null
  return { requested, upstream }
}

export function normalizeUsageContextBadgeDisplayMode(mode: string | null | undefined): UsageContextBadgeDisplayMode {
  switch (mode) {
    case 'native_only':
      return 'native_only'
    case 'both':
      return 'both'
    case 'request_only':
    default:
      return 'request_only'
  }
}

export function resolveUsageContextBadge(
  line: Pick<UsageModelLinePresentation, 'requestBadge' | 'nativeContextBadge'>,
  mode: UsageContextBadgeDisplayMode
): UsageContextBadgeInfo | null {
  const normalized = normalizeUsageContextBadgeDisplayMode(mode)
  if (normalized === 'native_only') {
    return line.nativeContextBadge
  }
  if (normalized === 'both') {
    return line.requestBadge || line.nativeContextBadge
  }
  return line.requestBadge
}

export function resolveUsageNativeContextLabel(modelId: string | null | undefined): string {
  return buildUsageModelLinePresentation(modelId, 'model_only').nativeContextLabel
}
