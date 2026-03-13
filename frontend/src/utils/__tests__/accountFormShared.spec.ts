import { describe, expect, it } from 'vitest'
import {
  buildMixedChannelWarningDetails,
  buildTempUnschedRules,
  createTempUnschedPresets,
  loadTempUnschedRulesFromCredentials,
  normalizePoolModeRetryCount,
  supportsMixedChannelCheck
} from '../accountFormShared'

describe('accountFormShared', () => {
  it('normalizes pool mode retry count into supported range', () => {
    expect(normalizePoolModeRetryCount(Number.NaN)).toBe(3)
    expect(normalizePoolModeRetryCount(-4)).toBe(0)
    expect(normalizePoolModeRetryCount(99)).toBe(10)
    expect(normalizePoolModeRetryCount(4.8)).toBe(4)
  })

  it('builds only valid temp-unsched rules', () => {
    expect(
      buildTempUnschedRules([
        {
          error_code: 429,
          keywords: 'rate limit, too many requests',
          duration_minutes: 15,
          description: 'valid'
        },
        {
          error_code: 99,
          keywords: 'skip',
          duration_minutes: 10,
          description: 'invalid'
        }
      ])
    ).toEqual([
      {
        error_code: 429,
        keywords: ['rate limit', 'too many requests'],
        duration_minutes: 15,
        description: 'valid'
      }
    ])
  })

  it('loads temp-unsched state from credentials payload', () => {
    expect(
      loadTempUnschedRulesFromCredentials({
        temp_unschedulable_enabled: true,
        temp_unschedulable_rules: [
          {
            error_code: 503,
            keywords: ['unavailable', 'maintenance'],
            duration_minutes: 30,
            description: 'server busy'
          }
        ]
      })
    ).toEqual({
      enabled: true,
      rules: [
        {
          error_code: 503,
          keywords: 'unavailable, maintenance',
          duration_minutes: 30,
          description: 'server busy'
        }
      ]
    })
  })

  it('creates temp-unsched presets from i18n labels', () => {
    const presets = createTempUnschedPresets((key) => key)
    expect(presets).toHaveLength(3)
    expect(presets[0]?.label).toBe('admin.accounts.tempUnschedulable.presets.overloadLabel')
  })

  it('builds mixed-channel warning details and platform support correctly', () => {
    expect(supportsMixedChannelCheck('anthropic')).toBe(true)
    expect(supportsMixedChannelCheck('openai')).toBe(false)
    expect(
      buildMixedChannelWarningDetails({
        has_risk: true,
        message: 'warn',
        details: {
          group_name: 'Group A',
          current_platform: 'anthropic',
          other_platform: 'gemini'
        }
      })
    ).toEqual({
      groupName: 'Group A',
      currentPlatform: 'anthropic',
      otherPlatform: 'gemini'
    })
  })
})
