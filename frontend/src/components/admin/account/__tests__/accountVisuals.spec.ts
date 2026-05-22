import { afterAll, beforeAll, describe, expect, it, vi } from 'vitest'
import { resolveAccountRowVisualState } from '../accountVisuals'
import type { Account } from '@/types'

const FIXED_NOW = new Date('2026-05-16T12:00:00Z').getTime()

const baseAccount = (): Account => ({
  id: 1,
  name: 'Primary',
  platform: 'openai',
  type: 'apikey',
  concurrency: 2,
  proxy_id: null,
  priority: 0,
  status: 'active',
  error_message: null,
  last_used_at: null,
  expires_at: null,
  auto_pause_on_expired: false,
  created_at: '2026-05-16T00:00:00Z',
  updated_at: '2026-05-16T00:00:00Z',
  schedulable: true,
  rate_limited_at: null,
  rate_limit_reset_at: null,
  rate_limit_reason: null,
  overload_until: null,
  temp_unschedulable_until: null,
  temp_unschedulable_reason: null,
  extra: {},
})

describe('accountVisuals', () => {
  beforeAll(() => {
    vi.useFakeTimers()
    vi.setSystemTime(FIXED_NOW)
  })

  afterAll(() => {
    vi.useRealTimers()
  })

  it('returns available tone for normal accounts', () => {
    const visual = resolveAccountRowVisualState(baseAccount())
    expect(visual.tone).toBe('available')
  })

  it('maps usage_5h accounts into the 5h visual family based on recovery progress', () => {
    const visual = resolveAccountRowVisualState({
      ...baseAccount(),
      rate_limited_at: '2026-05-16T10:00:00Z',
      rate_limit_reset_at: '2026-05-16T14:00:00Z',
      rate_limit_reason: 'usage_5h',
    })

    expect(visual.tone).toBe('usage_5h_50_75')
  })

  it('maps usage_7d_all accounts into the 7d visual family', () => {
    const visual = resolveAccountRowVisualState({
      ...baseAccount(),
      rate_limited_at: '2026-05-09T12:00:00Z',
      rate_limit_reset_at: '2026-05-18T12:00:00Z',
      rate_limit_reason: 'usage_7d_all',
    })

    expect(visual.tone).toBe('usage_7d_75_100')
  })

  it('prioritizes overload over temp unsched and paused', () => {
    const visual = resolveAccountRowVisualState({
      ...baseAccount(),
      schedulable: false,
      overload_until: '2026-05-16T12:30:00Z',
      temp_unschedulable_until: '2026-05-16T13:00:00Z',
    })

    expect(visual.tone).toBe('usage_5h_0_25')
  })

  it('returns error tone for blacklisted and error accounts', () => {
    const blacklisted = resolveAccountRowVisualState({
      ...baseAccount(),
      lifecycle_state: 'blacklisted',
    })
    const errored = resolveAccountRowVisualState({
      ...baseAccount(),
      status: 'error',
    })

    expect(blacklisted.tone).toBe('error')
    expect(errored.tone).toBe('error')
  })

  it('returns paused tone for inactive or unschedulable accounts', () => {
    const inactive = resolveAccountRowVisualState({
      ...baseAccount(),
      status: 'inactive',
    })
    const paused = resolveAccountRowVisualState({
      ...baseAccount(),
      schedulable: false,
    })

    expect(inactive.tone).toBe('paused')
    expect(paused.tone).toBe('paused')
  })

  it('keeps airy row tones while disabling per-row background animation', () => {
    const available = resolveAccountRowVisualState(baseAccount())
    const rateLimited = resolveAccountRowVisualState({
      ...baseAccount(),
      rate_limited_at: '2026-05-16T10:00:00Z',
      rate_limit_reset_at: '2026-05-16T14:00:00Z',
      rate_limit_reason: 'usage_5h',
    })

    for (const visual of [available, rateLimited]) {
      const animation = String(visual.style.animation ?? '')
      const backgroundImage = String(visual.style.backgroundImage ?? '')

      expect(visual.className).toContain('transition-colors')
      expect(visual.className).not.toContain('transition-all')
      expect(animation).toBe('none')
      expect(backgroundImage).toBe('none')
    }
  })
})
