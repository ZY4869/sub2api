import type {
  PublicModelCatalogDiscountPolicy,
  PublicModelCatalogDiscountWindow,
} from '@/api/meta'

const allDays = [0, 1, 2, 3, 4, 5, 6]

export function createDefaultDiscountPolicy(): PublicModelCatalogDiscountPolicy {
  return {
    enabled: false,
    reduction_percent: 20,
    timezone: 'Asia/Singapore',
    windows: [],
  }
}

export function ensureDiscountPolicy(
  policy?: PublicModelCatalogDiscountPolicy | null,
): PublicModelCatalogDiscountPolicy {
  const normalized = normalizeDiscountPolicy(policy)
  if (normalized.windows?.length) {
    return { ...normalized, enabled: true }
  }
  return {
    ...normalized,
    enabled: true,
    windows: [createDailyDiscountWindow()],
  }
}

export function normalizeDiscountPolicy(
  policy?: PublicModelCatalogDiscountPolicy | null,
): PublicModelCatalogDiscountPolicy {
  if (!policy) return createDefaultDiscountPolicy()
  return {
    enabled: Boolean(policy.enabled),
    reduction_percent: normalizeReduction(policy.reduction_percent),
    timezone: policy.timezone || 'Asia/Singapore',
    windows: (policy.windows || []).map(normalizeDiscountWindow),
  }
}

export function discountPolicyToPayload(
  policy?: PublicModelCatalogDiscountPolicy | null,
): PublicModelCatalogDiscountPolicy | null {
  if (!policy?.enabled) return null
  const normalized = ensureDiscountPolicy(policy)
  return {
    enabled: true,
    reduction_percent: normalizeReduction(normalized.reduction_percent),
    timezone: normalized.timezone || 'Asia/Singapore',
    windows: (normalized.windows || []).map(normalizeDiscountWindow),
  }
}

export function createDailyDiscountWindow(): PublicModelCatalogDiscountWindow {
  return {
    id: createDiscountWindowID(),
    type: 'daily',
    start_time: '00:00:00',
    end_time: '23:59:59',
    days: [...allDays],
  }
}

export function createOnceDiscountWindow(): PublicModelCatalogDiscountWindow {
  const start = new Date()
  start.setMinutes(0, 0, 0)
  const end = new Date(start.getTime() + 24 * 60 * 60 * 1000)
  return {
    id: createDiscountWindowID(),
    type: 'once',
    start_at: start.toISOString(),
    end_at: end.toISOString(),
  }
}

export function normalizeDiscountWindow(
  window: PublicModelCatalogDiscountWindow,
): PublicModelCatalogDiscountWindow {
  if (window.type === 'once') {
    return {
      id: window.id || createDiscountWindowID(),
      type: 'once',
      start_at: window.start_at || new Date().toISOString(),
      end_at: window.end_at || new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString(),
    }
  }
  return {
    id: window.id || createDiscountWindowID(),
    type: 'daily',
    start_time: normalizeClock(window.start_time, '00:00:00'),
    end_time: normalizeClock(window.end_time, '23:59:59'),
    days: normalizeDays(window.days),
  }
}

export function formatClockInput(value?: string): string {
  return normalizeClock(value, '00:00:00')
}

export function normalizeClock(value: string | undefined, fallback: string): string {
  const raw = String(value || '').trim()
  if (/^\d{2}:\d{2}:\d{2}$/.test(raw)) return raw
  if (/^\d{2}:\d{2}$/.test(raw)) return `${raw}:00`
  return fallback
}

function normalizeReduction(value?: number): number {
  const next = Number(value)
  if (!Number.isFinite(next) || next <= 0) return 20
  return Math.min(100, Number(next.toFixed(2)))
}

function normalizeDays(days?: number[]): number[] {
  const values = Array.isArray(days) && days.length ? days : allDays
  return Array.from(new Set(values.filter((day) => day >= 0 && day <= 6))).sort()
}

function createDiscountWindowID(): string {
  return `win_${Date.now().toString(36)}_${Math.random().toString(36).slice(2, 8)}`
}
