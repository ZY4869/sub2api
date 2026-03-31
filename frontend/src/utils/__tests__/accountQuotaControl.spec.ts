import { describe, expect, it } from 'vitest'
import type { Account } from '@/types'
import {
  DEFAULT_BASE_RPM,
  DEFAULT_CACHE_TTL_OVERRIDE_TARGET,
  DEFAULT_SESSION_IDLE_TIMEOUT_MINUTES,
  DEFAULT_WINDOW_COST_STICKY_RESERVE,
  buildAnthropicQuotaControlExtra,
  createDefaultAnthropicQuotaControlState,
  readAnthropicQuotaControlState
} from '../accountQuotaControl'

describe('accountQuotaControl', () => {
  it('reads anthropic quota control fields from an account payload', () => {
    const state = readAnthropicQuotaControlState({
      platform: 'anthropic',
      type: 'oauth',
      window_cost_limit: 18,
      window_cost_sticky_reserve: null,
      max_sessions: 4,
      session_idle_timeout_minutes: null,
      base_rpm: 22,
      rpm_strategy: 'sticky_exempt',
      rpm_sticky_buffer: 6,
      user_msg_queue_mode: 'serialize',
      enable_tls_fingerprint: true,
      tls_fingerprint_profile_id: -1,
      session_id_masking_enabled: true,
      cache_ttl_override_enabled: true,
      cache_ttl_override_target: '1h',
      custom_base_url_enabled: true,
      custom_base_url: 'https://relay.example.com'
    } as Account)

    expect(state).toMatchObject({
      windowCostEnabled: true,
      windowCostLimit: 18,
      windowCostStickyReserve: DEFAULT_WINDOW_COST_STICKY_RESERVE,
      sessionLimitEnabled: true,
      maxSessions: 4,
      sessionIdleTimeout: DEFAULT_SESSION_IDLE_TIMEOUT_MINUTES,
      rpmLimitEnabled: true,
      baseRpm: 22,
      rpmStrategy: 'sticky_exempt',
      rpmStickyBuffer: 6,
      userMsgQueueMode: 'serialize',
      tlsFingerprintEnabled: true,
      tlsFingerprintProfileId: -1,
      sessionIdMaskingEnabled: true,
      cacheTTLOverrideEnabled: true,
      cacheTTLOverrideTarget: '1h',
      customBaseUrlEnabled: true,
      customBaseUrl: 'https://relay.example.com'
    })
  })

  it('builds extra payload and removes deprecated fields when overrides are disabled', () => {
    const state = createDefaultAnthropicQuotaControlState()
    state.windowCostEnabled = true
    state.windowCostLimit = 12
    state.sessionLimitEnabled = true
    state.maxSessions = 3
    state.rpmLimitEnabled = true
    state.userMsgQueueMode = 'throttle'
    state.tlsFingerprintEnabled = true
    state.tlsFingerprintProfileId = -1
    state.sessionIdMaskingEnabled = true
    state.cacheTTLOverrideEnabled = true
    state.customBaseUrlEnabled = true
    state.customBaseUrl = 'https://relay.example.com'

    expect(
      buildAnthropicQuotaControlExtra(state, {
        keep_me: true,
        user_msg_queue_enabled: true,
        cache_ttl_override_target: 'legacy'
      })
    ).toEqual({
      keep_me: true,
      window_cost_limit: 12,
      window_cost_sticky_reserve: DEFAULT_WINDOW_COST_STICKY_RESERVE,
      max_sessions: 3,
      session_idle_timeout_minutes: DEFAULT_SESSION_IDLE_TIMEOUT_MINUTES,
      base_rpm: DEFAULT_BASE_RPM,
      rpm_strategy: 'tiered',
      user_msg_queue_mode: 'throttle',
      enable_tls_fingerprint: true,
      tls_fingerprint_profile_id: -1,
      session_id_masking_enabled: true,
      cache_ttl_override_enabled: true,
      cache_ttl_override_target: DEFAULT_CACHE_TTL_OVERRIDE_TARGET,
      custom_base_url_enabled: true,
      custom_base_url: 'https://relay.example.com'
    })
  })

  it('returns undefined when only legacy quota fields remain', () => {
    expect(
      buildAnthropicQuotaControlExtra(createDefaultAnthropicQuotaControlState(), {
        user_msg_queue_enabled: true
      })
    ).toBeUndefined()
  })
})
