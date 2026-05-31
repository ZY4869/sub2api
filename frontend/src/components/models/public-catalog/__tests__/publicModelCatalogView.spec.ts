import { describe, expect, it } from 'vitest'
import { hasHealthMetrics } from '../publicModelCatalogView'

describe('publicModelCatalogView', () => {
  it('does not treat last_checked_at alone as SLA metrics', () => {
    expect(hasHealthMetrics({
      public_model_id: 'gpt-5.4',
      model: 'gpt-5.4',
      aliases: [],
      status: 'pending',
      health_source: 'none',
      status_reason: 'stale_history',
      last_checked_at: '2026-05-29T10:00:00Z',
      daily: [],
      trend: [],
    })).toBe(false)
  })

  it('detects explicit success and latency metrics', () => {
    expect(hasHealthMetrics({
      public_model_id: 'gpt-5.4',
      model: 'gpt-5.4',
      aliases: [],
      status: 'healthy',
      health_source: 'traffic',
      status_reason: 'traffic_recent',
      success_rate_7d: 1,
      daily: [],
      trend: [],
    })).toBe(true)
  })
})
