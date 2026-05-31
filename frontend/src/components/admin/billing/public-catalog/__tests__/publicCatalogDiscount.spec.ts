import { describe, expect, it, vi } from 'vitest'
import {
  createDailyDiscountWindow,
  createOnceDiscountWindow,
  discountPolicyToPayload,
  normalizeDiscountPolicy,
} from '../publicCatalogDiscount'

describe('publicCatalogDiscount', () => {
  it('normalizes default policy values', () => {
    expect(normalizeDiscountPolicy(null)).toEqual({
      enabled: false,
      reduction_percent: 20,
      timezone: 'Asia/Singapore',
      windows: [],
    })

    expect(normalizeDiscountPolicy({
      enabled: true,
      reduction_percent: 250,
      windows: [{ type: 'daily', start_time: '08:30', end_time: '20:45', days: [6, 1, 1, 9] }],
    })).toMatchObject({
      enabled: true,
      reduction_percent: 100,
      timezone: 'Asia/Singapore',
      windows: [{
        type: 'daily',
        start_time: '08:30:00',
        end_time: '20:45:00',
        days: [1, 6],
      }],
    })
  })

  it('creates once and daily windows with stable payload shapes', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-06-01T10:20:30.000Z'))
    try {
      const once = createOnceDiscountWindow()
      expect(once).toMatchObject({
        type: 'once',
        start_at: '2026-06-01T10:00:00.000Z',
        end_at: '2026-06-02T10:00:00.000Z',
      })
      expect(once.id).toMatch(/^win_/)

      expect(createDailyDiscountWindow()).toMatchObject({
        type: 'daily',
        start_time: '00:00:00',
        end_time: '23:59:59',
        days: [0, 1, 2, 3, 4, 5, 6],
      })
    } finally {
      vi.useRealTimers()
    }
  })

  it('serializes enabled policies and drops disabled policies', () => {
    expect(discountPolicyToPayload({ enabled: false, reduction_percent: 20, windows: [] })).toBeNull()

    expect(discountPolicyToPayload({
      enabled: true,
      reduction_percent: 33.333,
      timezone: '',
      windows: [{ id: 'once-1', type: 'once', start_at: '2026-06-01T00:00:00Z', end_at: '2026-06-01T01:00:00Z' }],
    })).toEqual({
      enabled: true,
      reduction_percent: 33.33,
      timezone: 'Asia/Singapore',
      windows: [{ id: 'once-1', type: 'once', start_at: '2026-06-01T00:00:00Z', end_at: '2026-06-01T01:00:00Z' }],
    })
  })
})
