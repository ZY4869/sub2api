import { describe, expect, it } from 'vitest'
import { resolveCodexUsageWindow, resolveCodexUsageWindowLabel } from '../codexUsage'

describe('codexUsage', () => {
  it('formats canonical codex window labels from minutes', () => {
    expect(resolveCodexUsageWindowLabel(300, '5h')).toBe('5H')
    expect(resolveCodexUsageWindowLabel(10080, '7d')).toBe('7D')
    expect(resolveCodexUsageWindowLabel(43200, '7d')).toBe('30D')
    expect(resolveCodexUsageWindowLabel(null, '5h')).toBe('5H')
    expect(resolveCodexUsageWindowLabel(undefined, '7d')).toBe('7D')
  })

  it('returns canonical window minutes and label with the usage snapshot', () => {
    const window = resolveCodexUsageWindow({
      codex_7d_used_percent: 42,
      codex_7d_reset_at: '2026-03-31T00:00:00Z',
      codex_7d_window_minutes: 43200,
    }, '7d', new Date('2026-03-01T00:00:00Z'))

    expect(window.usedPercent).toBe(42)
    expect(window.windowMinutes).toBe(43200)
    expect(window.label).toBe('30D')
  })

  it('uses legacy window minutes when canonical minutes are not available', () => {
    const window = resolveCodexUsageWindow({
      codex_primary_used_percent: 66,
      codex_primary_reset_after_seconds: 3600,
      codex_primary_window_minutes: 43200,
      codex_usage_updated_at: '2026-03-01T00:00:00Z',
    }, '7d', new Date('2026-03-01T00:10:00Z'))

    expect(window.usedPercent).toBe(66)
    expect(window.windowMinutes).toBe(43200)
    expect(window.label).toBe('30D')
    expect(window.resetAt).toBe('2026-03-01T01:00:00.000Z')
  })
})
