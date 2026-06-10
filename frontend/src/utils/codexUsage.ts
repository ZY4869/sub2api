import type { AccountUsageRowColor, CodexUsageSnapshot } from '@/types'

export interface ResolvedCodexUsageWindow {
  usedPercent: number | null
  resetAt: string | null
  windowMinutes: number | null
  label: string
}

type WindowKind = '5h' | '7d'
export type CodexUsageScope = 'normal' | 'spark'

type CodexWindowKeys = {
  usedPercent: string
  resetAfterSeconds: string
  resetAt: string
  windowMinutes: string
}

function asNumber(value: unknown): number | null {
  if (typeof value === 'number' && Number.isFinite(value)) return value
  if (typeof value === 'string' && value.trim() !== '') {
    const n = Number(value)
    if (Number.isFinite(n)) return n
  }
  return null
}

function asString(value: unknown): string | null {
  if (typeof value !== 'string') return null
  const trimmed = value.trim()
  return trimmed === '' ? null : trimmed
}

function asISOTime(value: unknown): string | null {
  const raw = asString(value)
  if (!raw) return null
  const date = new Date(raw)
  if (Number.isNaN(date.getTime())) return null
  return date.toISOString()
}

function formatWindowMinutesLabel(minutes: number): string {
  const normalized = Math.max(1, Math.round(minutes))
  if (normalized >= 24 * 60) {
    return `${Math.max(1, Math.round(normalized / (24 * 60)))}D`
  }
  if (normalized >= 60) {
    return `${Math.max(1, Math.round(normalized / 60))}H`
  }
  return `${normalized}M`
}

export function resolveCodexUsageWindowLabel(
  windowMinutes: number | null | undefined,
  fallbackWindow: WindowKind,
): string {
  if (typeof windowMinutes === 'number' && Number.isFinite(windowMinutes) && windowMinutes > 0) {
    return formatWindowMinutesLabel(windowMinutes)
  }
  return fallbackWindow === '5h' ? '5H' : '7D'
}

export function resolveUsageWindowColor(label: string): AccountUsageRowColor {
  const normalized = label.trim().toUpperCase()
  if (normalized === '5H') return 'indigo'
  if (normalized === '30D') return 'green'
  if (normalized === '7D') return 'orange'
  return 'emerald'
}

function resolveLegacy5h(snapshot: Record<string, unknown>): { used: number | null; resetAfterSeconds: number | null; windowMinutes: number | null } {
  const primaryWindow = asNumber(snapshot.codex_primary_window_minutes)
  const secondaryWindow = asNumber(snapshot.codex_secondary_window_minutes)
  const primaryUsed = asNumber(snapshot.codex_primary_used_percent)
  const secondaryUsed = asNumber(snapshot.codex_secondary_used_percent)
  const primaryReset = asNumber(snapshot.codex_primary_reset_after_seconds)
  const secondaryReset = asNumber(snapshot.codex_secondary_reset_after_seconds)

  if (primaryWindow != null && primaryWindow <= 360) {
    return { used: primaryUsed, resetAfterSeconds: primaryReset, windowMinutes: primaryWindow }
  }
  if (secondaryWindow != null && secondaryWindow <= 360) {
    return { used: secondaryUsed, resetAfterSeconds: secondaryReset, windowMinutes: secondaryWindow }
  }
  return { used: secondaryUsed, resetAfterSeconds: secondaryReset, windowMinutes: secondaryWindow }
}

function resolveLegacy7d(snapshot: Record<string, unknown>): { used: number | null; resetAfterSeconds: number | null; windowMinutes: number | null } {
  const primaryWindow = asNumber(snapshot.codex_primary_window_minutes)
  const secondaryWindow = asNumber(snapshot.codex_secondary_window_minutes)
  const primaryUsed = asNumber(snapshot.codex_primary_used_percent)
  const secondaryUsed = asNumber(snapshot.codex_secondary_used_percent)
  const primaryReset = asNumber(snapshot.codex_primary_reset_after_seconds)
  const secondaryReset = asNumber(snapshot.codex_secondary_reset_after_seconds)

  if (primaryWindow != null && primaryWindow >= 10000) {
    return { used: primaryUsed, resetAfterSeconds: primaryReset, windowMinutes: primaryWindow }
  }
  if (secondaryWindow != null && secondaryWindow >= 10000) {
    return { used: secondaryUsed, resetAfterSeconds: secondaryReset, windowMinutes: secondaryWindow }
  }
  return { used: primaryUsed, resetAfterSeconds: primaryReset, windowMinutes: primaryWindow }
}

function resolveFromSeconds(snapshot: Record<string, unknown>, resetAfterSeconds: number | null): string | null {
  if (resetAfterSeconds == null) return null

  const baseRaw = asString(snapshot.codex_usage_updated_at)
  const base = baseRaw ? new Date(baseRaw) : new Date()
  if (Number.isNaN(base.getTime())) {
    return null
  }

  const sec = Math.max(0, resetAfterSeconds)
  const resetAt = new Date(base.getTime() + sec * 1000)
  return resetAt.toISOString()
}

function applyExpiredRule(window: ResolvedCodexUsageWindow, now: Date): ResolvedCodexUsageWindow {
  if (window.usedPercent == null || !window.resetAt) return window
  const resetDate = new Date(window.resetAt)
  if (Number.isNaN(resetDate.getTime())) return window
  if (resetDate.getTime() <= now.getTime()) {
    return { ...window, usedPercent: 0, resetAt: resetDate.toISOString() }
  }
  return window
}

function resolveCanonicalKeys(window: WindowKind, scope: CodexUsageScope): CodexWindowKeys {
  if (scope === 'spark') {
    return window === '5h'
      ? {
          usedPercent: 'codex_spark_5h_used_percent',
          resetAfterSeconds: 'codex_spark_5h_reset_after_seconds',
          resetAt: 'codex_spark_5h_reset_at',
          windowMinutes: 'codex_spark_5h_window_minutes',
        }
      : {
          usedPercent: 'codex_spark_7d_used_percent',
          resetAfterSeconds: 'codex_spark_7d_reset_after_seconds',
          resetAt: 'codex_spark_7d_reset_at',
          windowMinutes: 'codex_spark_7d_window_minutes',
        }
  }

  return window === '5h'
    ? {
        usedPercent: 'codex_5h_used_percent',
        resetAfterSeconds: 'codex_5h_reset_after_seconds',
        resetAt: 'codex_5h_reset_at',
        windowMinutes: 'codex_5h_window_minutes',
      }
    : {
        usedPercent: 'codex_7d_used_percent',
        resetAfterSeconds: 'codex_7d_reset_after_seconds',
        resetAt: 'codex_7d_reset_at',
        windowMinutes: 'codex_7d_window_minutes',
      }
}

export function resolveCodexUsageWindow(
  snapshot: (CodexUsageSnapshot & Record<string, unknown>) | null | undefined,
  window: WindowKind,
  now: Date = new Date(),
  scope: CodexUsageScope = 'normal'
): ResolvedCodexUsageWindow {
  if (!snapshot) {
    return {
      usedPercent: null,
      resetAt: null,
      windowMinutes: null,
      label: resolveCodexUsageWindowLabel(null, window),
    }
  }

  const typedSnapshot = snapshot as Record<string, unknown>
  const canonicalKeys = resolveCanonicalKeys(window, scope)
  let usedPercent: number | null
  let resetAfterSeconds: number | null
  let resetAt: string | null
  let windowMinutes: number | null

  if (window === '5h') {
    usedPercent = asNumber(typedSnapshot[canonicalKeys.usedPercent])
    resetAfterSeconds = asNumber(typedSnapshot[canonicalKeys.resetAfterSeconds])
    resetAt = asISOTime(typedSnapshot[canonicalKeys.resetAt])
    windowMinutes = asNumber(typedSnapshot[canonicalKeys.windowMinutes])
    if (scope === 'normal' && (usedPercent == null || (resetAfterSeconds == null && !resetAt))) {
      const legacy = resolveLegacy5h(typedSnapshot)
      if (usedPercent == null) usedPercent = legacy.used
      if (resetAfterSeconds == null) resetAfterSeconds = legacy.resetAfterSeconds
      if (windowMinutes == null) windowMinutes = legacy.windowMinutes
    }
  } else {
    usedPercent = asNumber(typedSnapshot[canonicalKeys.usedPercent])
    resetAfterSeconds = asNumber(typedSnapshot[canonicalKeys.resetAfterSeconds])
    resetAt = asISOTime(typedSnapshot[canonicalKeys.resetAt])
    windowMinutes = asNumber(typedSnapshot[canonicalKeys.windowMinutes])
    if (scope === 'normal' && (usedPercent == null || (resetAfterSeconds == null && !resetAt))) {
      const legacy = resolveLegacy7d(typedSnapshot)
      if (usedPercent == null) usedPercent = legacy.used
      if (resetAfterSeconds == null) resetAfterSeconds = legacy.resetAfterSeconds
      if (windowMinutes == null) windowMinutes = legacy.windowMinutes
    }
  }

  if (!resetAt) {
    resetAt = resolveFromSeconds(typedSnapshot, resetAfterSeconds)
  }

  return applyExpiredRule({
    usedPercent,
    resetAt,
    windowMinutes,
    label: resolveCodexUsageWindowLabel(windowMinutes, window),
  }, now)
}
