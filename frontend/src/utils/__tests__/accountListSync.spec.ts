import { describe, expect, it } from 'vitest'
import {
  accountMatchesFilters,
  buildDefaultTodayStats,
  mergeRuntimeFields,
  shouldReplaceAutoRefreshRow
} from '@/utils/accountListSync'
import type { Account } from '@/types'

const createAccount = (overrides: Partial<Account> = {}): Account => ({
  id: 1,
  name: 'Primary Account',
  platform: 'openai',
  type: 'oauth',
  proxy_id: null,
  concurrency: 1,
  priority: 0,
  status: 'active',
  error_message: null,
  last_used_at: null,
  expires_at: null,
  auto_pause_on_expired: false,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
  schedulable: true,
  rate_limited_at: null,
  rate_limit_reset_at: null,
  rate_limit_reason: null,
  overload_until: null,
  temp_unschedulable_until: null,
  temp_unschedulable_reason: null,
  session_window_start: null,
  session_window_end: null,
  session_window_status: null,
  ...overrides
})

describe('accountListSync', () => {
  it('builds default today stats payload', () => {
    expect(buildDefaultTodayStats()).toEqual({
      requests: 0,
      tokens: 0,
      cost: 0,
      standard_cost: 0,
      user_cost: 0
    })
  })

  it('matches group, rate-limited and temp-unschedulable filters', () => {
    const now = new Date('2026-03-14T10:00:00Z').getTime()

    expect(
      accountMatchesFilters(
        createAccount({ group_ids: [2] }),
        { group: '2' },
        now
      )
    ).toBe(true)

    expect(
      accountMatchesFilters(
        createAccount({ rate_limit_reset_at: '2026-03-14T10:05:00Z' }),
        { status: 'rate_limited' },
        now
      )
    ).toBe(true)

    expect(
      accountMatchesFilters(
        createAccount({ temp_unschedulable_until: '2026-03-14T10:05:00Z' }),
        { status: 'temp_unschedulable' },
        now
      )
    ).toBe(true)

    expect(
      accountMatchesFilters(
        createAccount({ group_ids: [1] }),
        { group: '2' },
        now
      )
    ).toBe(false)
  })

  it('matches limited view and limited reason filters', () => {
    const now = new Date('2026-03-14T10:00:00Z').getTime()
    const limitedAccount = createAccount({
      rate_limit_reset_at: '2026-03-14T10:05:00Z',
      rate_limit_reason: 'usage_7d'
    })

    expect(accountMatchesFilters(limitedAccount, { limited_view: 'limited_only' }, now)).toBe(true)
    expect(accountMatchesFilters(limitedAccount, { limited_view: 'normal_only' }, now)).toBe(false)
    expect(accountMatchesFilters(limitedAccount, { limited_reason: 'usage_7d' }, now)).toBe(true)
    expect(accountMatchesFilters(limitedAccount, { limited_reason: 'rate_429' }, now)).toBe(false)
  })

  it('matches runtime view filter for in-use accounts', () => {
    expect(
      accountMatchesFilters(
        createAccount({ current_concurrency: 1 }),
        { runtime_view: 'in_use_only' }
      )
    ).toBe(true)

    expect(
      accountMatchesFilters(
        createAccount({ active_sessions: 2 }),
        { runtime_view: 'in_use_only' }
      )
    ).toBe(true)

    expect(
      accountMatchesFilters(
        createAccount({ current_concurrency: 0, active_sessions: 0 }),
        { runtime_view: 'in_use_only' }
      )
    ).toBe(false)
  })

  it('preserves runtime fields when patch payload omits them', () => {
    const current = createAccount({
      current_concurrency: 3,
      current_window_cost: 12,
      active_sessions: 2
    })
    const updated = createAccount({
      updated_at: '2026-01-02T00:00:00Z',
      current_concurrency: undefined,
      current_window_cost: undefined,
      active_sessions: undefined
    })

    expect(mergeRuntimeFields(current, updated)).toEqual(
      expect.objectContaining({
        updated_at: '2026-01-02T00:00:00Z',
        current_concurrency: 3,
        current_window_cost: 12,
        active_sessions: 2
      })
    )
  })

  it('detects rows that should be replaced during incremental refresh', () => {
    const current = createAccount({
      extra: { codex_usage_updated_at: '2026-01-01T00:00:00Z' }
    })
    const next = createAccount({
      extra: { codex_usage_updated_at: '2026-01-01T00:01:00Z' }
    })

    expect(shouldReplaceAutoRefreshRow(current, next)).toBe(true)
    expect(shouldReplaceAutoRefreshRow(current, current)).toBe(false)
  })

  it('replaces rows when rate-limit reason changes', () => {
    const current = createAccount({
      rate_limit_reset_at: '2026-03-14T10:05:00Z',
      rate_limit_reason: 'rate_429'
    })
    const next = createAccount({
      rate_limit_reset_at: '2026-03-14T10:05:00Z',
      rate_limit_reason: 'usage_5h'
    })

    expect(shouldReplaceAutoRefreshRow(current, next)).toBe(true)
  })
})
