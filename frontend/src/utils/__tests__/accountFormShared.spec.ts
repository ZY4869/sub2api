import { describe, expect, it } from 'vitest'
import {
  ACCOUNT_UPSTREAM_API_KEY_PLACEHOLDER,
  ACCOUNT_UPSTREAM_BASE_URL_PLACEHOLDER,
  buildMixedChannelWarningDetails,
  buildTempUnschedRules,
  createDefaultAccountCustomErrorCodesState,
  createDefaultAccountPoolModeState,
  createTempUnschedPresets,
  loadTempUnschedRulesFromCredentials,
  normalizeAccountConcurrency,
  normalizeAccountLoadFactor,
  normalizePoolModeRetryCount,
  resolveAccountUpstreamApiKeyHintKey,
  supportsMixedChannelCheck
} from '../accountFormShared'

describe('accountFormShared', () => {
  it('exposes stable upstream placeholders and mode-aware hint keys', () => {
    expect(ACCOUNT_UPSTREAM_BASE_URL_PLACEHOLDER).toBe('https://cloudcode-pa.googleapis.com')
    expect(ACCOUNT_UPSTREAM_API_KEY_PLACEHOLDER).toBe('sk-...')
    expect(resolveAccountUpstreamApiKeyHintKey('create')).toBe('admin.accounts.upstream.apiKeyHint')
    expect(resolveAccountUpstreamApiKeyHintKey('edit')).toBe('admin.accounts.leaveEmptyToKeep')
  })

  it('normalizes runtime settings fields', () => {
    expect(normalizeAccountConcurrency(undefined)).toBe(1)
    expect(normalizeAccountConcurrency(0)).toBe(1)
    expect(normalizeAccountConcurrency(-1)).toBe(1)
    expect(normalizeAccountConcurrency(8)).toBe(8)

    expect(normalizeAccountLoadFactor(undefined)).toBeNull()
    expect(normalizeAccountLoadFactor(0)).toBeNull()
    expect(normalizeAccountLoadFactor(-1)).toBeNull()
    expect(normalizeAccountLoadFactor(2)).toBe(2)
  })

  it('creates stable default states for API key advanced settings', () => {
    expect(createDefaultAccountPoolModeState(3)).toEqual({
      enabled: false,
      retryCount: 3
    })
    expect(createDefaultAccountCustomErrorCodesState()).toEqual({
      enabled: false,
      selectedCodes: [],
      input: null
    })
  })

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
