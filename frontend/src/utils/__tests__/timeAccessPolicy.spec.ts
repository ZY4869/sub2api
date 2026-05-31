import { describe, expect, it } from 'vitest'
import {
  buildPresetTimeAccessPolicy,
  policyToPayload,
} from '../timeAccessPolicy'

describe('timeAccessPolicy', () => {
  it('builds fixed preset payloads for api key forms', () => {
    expect(policyToPayload(buildPresetTimeAccessPolicy('eight_hours'))).toEqual(expect.objectContaining({
      enabled: true,
      timezone: 'Asia/Singapore',
      daily_allowed_minutes: 480,
      weekly_windows: [
        { days: [0, 1, 2, 3, 4, 5, 6], start: '09:00', end: '17:00' },
      ],
    }))

    expect(policyToPayload(buildPresetTimeAccessPolicy('business_days_daytime'))).toEqual(expect.objectContaining({
      daily_allowed_minutes: 720,
      weekly_windows: [
        { days: [1, 2, 3, 4, 5], start: '08:00', end: '20:00' },
      ],
    }))
  })

  it('preserves multi-window days and cross-midnight ranges in payloads', () => {
    const payload = policyToPayload({
      enabled: true,
      timezone: 'Asia/Singapore',
      weekly_windows: [
        { days: [1, 3, 5], start: '08:00', end: '12:00' },
        { days: [6, 0], start: '22:00', end: '02:00' },
      ],
      daily_allowed_minutes: 480,
    })

    expect(payload?.weekly_windows).toEqual([
      { days: [1, 3, 5], start: '08:00', end: '12:00' },
      { days: [0, 6], start: '22:00', end: '02:00' },
    ])
  })

  it('omits disabled policies so update requests can clear them', () => {
    expect(policyToPayload({ enabled: false, timezone: 'Asia/Singapore' })).toBeUndefined()
  })
})
