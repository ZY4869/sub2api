import { i18n } from '@/i18n'

export type ProtocolFamily = 'openai' | 'anthropic' | 'gemini' | 'unknown'
export type ProtocolMode = 'native' | 'compatible'

export interface ProtocolBadgeMeta {
  family: ProtocolFamily
  label: string
  iconName: 'sparkles' | 'chat' | 'cpu' | 'link'
  className: string
}

export interface UsageProtocolDisplay {
  badge: ProtocolBadgeMeta
  requestPath: string
  pathTitle: string
  mode: ProtocolMode
  modeLabel: string
  tooltip?: string
  inboundPath?: string
  upstreamPath?: string
}

export interface ProtocolPairDisplay {
  protocolIn: string
  protocolOut: string
  inboundBadge: ProtocolBadgeMeta
  outboundBadge: ProtocolBadgeMeta
  label: string
  title: string
  detailLabel?: string
}

const PROTOCOL_BADGE_META: Record<
  ProtocolFamily,
  Omit<ProtocolBadgeMeta, 'label'>
> = {
  openai: {
    family: 'openai',
    iconName: 'sparkles',
    className:
      'bg-emerald-100 text-emerald-700 ring-1 ring-inset ring-emerald-200 dark:bg-emerald-500/15 dark:text-emerald-300 dark:ring-emerald-500/25'
  },
  anthropic: {
    family: 'anthropic',
    iconName: 'chat',
    className:
      'bg-amber-100 text-amber-700 ring-1 ring-inset ring-amber-200 dark:bg-amber-500/15 dark:text-amber-300 dark:ring-amber-500/25'
  },
  gemini: {
    family: 'gemini',
    iconName: 'cpu',
    className:
      'bg-sky-100 text-sky-700 ring-1 ring-inset ring-sky-200 dark:bg-sky-500/15 dark:text-sky-300 dark:ring-sky-500/25'
  },
  unknown: {
    family: 'unknown',
    iconName: 'link',
    className:
      'bg-gray-100 text-gray-700 ring-1 ring-inset ring-gray-200 dark:bg-white/10 dark:text-gray-200 dark:ring-white/10'
  }
}

function translate(key: string, fallback: string): string {
  if (typeof i18n.global.te === 'function' && !i18n.global.te(key)) {
    return fallback
  }
  const message = i18n.global.t(key)
  return typeof message === 'string' && message !== key ? message : fallback
}

function isKnownCompatPath(path: string): boolean {
  return path.startsWith('/v1beta/openai')
}

function trimPath(path: string | null | undefined): string {
  return String(path || '').trim()
}

export function resolveProtocolFamilyFromValue(value: string | null | undefined): ProtocolFamily {
  const normalized = trimPath(value).toLowerCase()
  if (!normalized) {
    return 'unknown'
  }

  switch (normalized) {
    case 'openai':
    case 'anthropic':
    case 'gemini':
      return normalized
  }

  if (
    normalized.startsWith('/v1/chat/completions') ||
    normalized.startsWith('/v1/responses') ||
    normalized.startsWith('/v1/images/') ||
    normalized === '/v1/videos' ||
    normalized.startsWith('/v1/videos/')
  ) {
    return 'openai'
  }
  if (normalized.startsWith('/v1/messages')) {
    return 'anthropic'
  }
  if (
    normalized.startsWith('/v1beta/') ||
    normalized.startsWith('/upload/v1beta/') ||
    normalized.startsWith('/download/v1beta/') ||
    normalized.startsWith('/google/batch/archive/') ||
    normalized.startsWith('/v1/projects/')
  ) {
    return 'gemini'
  }

  return 'unknown'
}

export function getProtocolBadgeMeta(value: string | null | undefined): ProtocolBadgeMeta {
  const family = resolveProtocolFamilyFromValue(value)
  const badge = PROTOCOL_BADGE_META[family]
  return {
    ...badge,
    label: translate(`usage.protocolFamilies.${family}`, fallbackProtocolFamilyLabel(family))
  }
}

function fallbackProtocolFamilyLabel(family: ProtocolFamily): string {
  switch (family) {
    case 'openai':
      return 'OpenAI'
    case 'anthropic':
      return 'Anthropic'
    case 'gemini':
      return 'Gemini'
    default:
      return 'Unknown'
  }
}

function formatProtocolTooltipSegment(path: string): string {
  const badge = getProtocolBadgeMeta(path)
  return `${badge.label} ${path}`
}

export function resolveUsageProtocolDisplay(
  inboundPath: string | null | undefined,
  upstreamPath: string | null | undefined
): UsageProtocolDisplay | null {
  const inbound = trimPath(inboundPath)
  const upstream = trimPath(upstreamPath)
  const primaryPath = inbound || upstream

  if (!primaryPath) {
    return null
  }

  const primaryBadge = getProtocolBadgeMeta(primaryPath)
  const primaryFamily = primaryBadge.family
  const upstreamFamily = getProtocolBadgeMeta(upstream || primaryPath).family
  const compatible = isKnownCompatPath(primaryPath) || (
    primaryFamily !== 'unknown' &&
    upstreamFamily !== 'unknown' &&
    primaryFamily !== upstreamFamily
  )

  const tooltip = inbound && upstream && inbound !== upstream
    ? translate('usage.protocolTransition', 'Inbound -> Upstream: {transition}')
        .replace('{transition}', `${formatProtocolTooltipSegment(inbound)} -> ${formatProtocolTooltipSegment(upstream)}`)
    : undefined

  return {
    badge: primaryBadge,
    requestPath: primaryPath,
    pathTitle: primaryPath,
    mode: compatible ? 'compatible' : 'native',
    modeLabel: translate(
      compatible ? 'usage.protocolModes.compatible' : 'usage.protocolModes.native',
      compatible ? 'Compatible' : 'Native'
    ),
    tooltip,
    inboundPath: inbound || undefined,
    upstreamPath: upstream || undefined
  }
}

export function resolveProtocolPairDisplay(
  protocolIn: string | null | undefined,
  protocolOut: string | null | undefined
): ProtocolPairDisplay {
  const inbound = trimPath(protocolIn)
  const outbound = trimPath(protocolOut)
  const inboundValue = inbound || 'unknown'
  const outboundValue = outbound || inboundValue
  const inboundBadge = getProtocolBadgeMeta(inboundValue)
  const outboundBadge = getProtocolBadgeMeta(outboundValue)

  return {
    protocolIn: inboundValue,
    protocolOut: outboundValue,
    inboundBadge,
    outboundBadge,
    label: `${inboundBadge.label} -> ${outboundBadge.label}`,
    title: `${inboundValue} -> ${outboundValue}`,
    detailLabel: shouldShowProtocolPairDetail(inboundValue, outboundValue)
      ? `${inboundValue} -> ${outboundValue}`
      : undefined
  }
}

function shouldShowProtocolPairDetail(inboundValue: string, outboundValue: string): boolean {
  return !isProtocolFamilyToken(inboundValue) || !isProtocolFamilyToken(outboundValue)
}

function isProtocolFamilyToken(value: string): boolean {
  switch (trimPath(value).toLowerCase()) {
    case 'openai':
    case 'anthropic':
    case 'gemini':
    case 'unknown':
      return true
    default:
      return false
  }
}

export function formatProtocolPairText(protocolIn: string | null | undefined, protocolOut: string | null | undefined): string {
  return resolveProtocolPairDisplay(protocolIn, protocolOut).label
}

export function formatUsageProtocolExportText(
  inboundPath: string | null | undefined,
  upstreamPath: string | null | undefined
): string {
  const display = resolveUsageProtocolDisplay(inboundPath, upstreamPath)
  if (!display) {
    return ''
  }

  return `${display.badge.label} ${display.requestPath} ${display.modeLabel}`
}
