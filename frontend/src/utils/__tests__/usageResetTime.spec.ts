import { describe, expect, it, vi } from 'vitest'
import {
  formatLocalAbsoluteTime,
  formatLocalTimestamp,
  formatResetCountdown,
  parseEffectiveResetAt
} from '@/utils/usageResetTime'

describe('usageResetTime', () => {
  it('formats today reset time', () => {
    const now = new Date('2026-03-13T08:00:00')
    const target = new Date('2026-03-13T10:15:00')
    expect(formatLocalAbsoluteTime(target, now, { today: '今天', tomorrow: '明天' })).toBe('今天 10:15')
  })

  it('formats tomorrow reset time', () => {
    const now = new Date('2026-03-13T23:00:00')
    const target = new Date('2026-03-14T01:15:00')
    expect(formatLocalAbsoluteTime(target, now, { today: '今天', tomorrow: '明天' })).toBe('明天 01:15')
  })

  it('formats same-year cross-day reset time', () => {
    const now = new Date('2026-03-13T08:00:00')
    const target = new Date('2026-04-02T01:15:00')
    expect(formatLocalAbsoluteTime(target, now, { today: '今天', tomorrow: '明天' })).toBe('04-02 01:15')
  })

  it('formats cross-year reset time', () => {
    const now = new Date('2026-12-31T08:00:00')
    const target = new Date('2027-01-02T01:15:00')
    expect(formatLocalAbsoluteTime(target, now, { today: '今天', tomorrow: '明天' })).toBe('2027-01-02 01:15')
  })

  it('formats full local timestamp to seconds', () => {
    expect(formatLocalTimestamp(new Date('2026-03-13T08:09:10'))).toBe('2026-03-13 08:09:10')
  })

  it('formats elapsed reset countdown as now', () => {
    const now = new Date('2026-03-13T08:00:00')
    expect(formatResetCountdown(new Date('2026-03-13T07:59:59'), now, '现在')).toBe('现在')
  })

  it('parses reset time from remaining seconds', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-03-13T08:00:00'))

    const parsed = parseEffectiveResetAt(null, 3600)
    expect(parsed?.getHours()).toBe(9)
    expect(parsed?.getMinutes()).toBe(0)

    vi.useRealTimers()
  })
})
