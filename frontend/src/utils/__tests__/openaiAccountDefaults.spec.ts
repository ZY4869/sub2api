import { describe, expect, it } from 'vitest'
import {
  getOpenAIDefaultWhitelist,
  getDefaultOpenAIImageProtocolMode,
  isOpenAIImageCompatAllowedPlan,
  normalizeOpenAIPlanType,
  resolveOpenAIImageProtocolState,
  shouldAutoReplaceOpenAIWhitelist,
} from '../openaiAccountDefaults'

describe('openaiAccountDefaults', () => {
  it('returns the paid default whitelist for non-free plans', () => {
    expect(getOpenAIDefaultWhitelist()).toEqual(['gpt-5.2', 'gpt-5.4', 'gpt-5.4-mini', 'gpt-5.5'])
    expect(getOpenAIDefaultWhitelist('plus')).toEqual(['gpt-5.2', 'gpt-5.4', 'gpt-5.4-mini', 'gpt-5.5'])
    expect(getOpenAIDefaultWhitelist('team')).toEqual(['gpt-5.2', 'gpt-5.4', 'gpt-5.4-mini', 'gpt-5.5'])
  })

  it('returns the free default whitelist for free plans', () => {
    expect(getOpenAIDefaultWhitelist('free')).toEqual(['gpt-5.2', 'gpt-5.4', 'gpt-5.4-mini'])
  })

  it('adds spark to the default whitelist for pro plans', () => {
    expect(getOpenAIDefaultWhitelist('chatgpt-pro')).toEqual([
      'gpt-5.2',
      'gpt-5.4',
      'gpt-5.4-mini',
      'gpt-5.5',
      'gpt-5.3-codex-spark',
    ])
    expect(normalizeOpenAIPlanType('ChatGPT Team')).toBe('team')
  })

  it('only auto-replaces untouched default whitelist selections', () => {
    expect(shouldAutoReplaceOpenAIWhitelist([])).toBe(true)
    expect(shouldAutoReplaceOpenAIWhitelist(['gpt-5.4', 'gpt-5.2', 'gpt-5.4-mini'])).toBe(true)
    expect(
      shouldAutoReplaceOpenAIWhitelist(['gpt-5.4', 'gpt-5.2', 'gpt-5.4-mini', 'gpt-5.5']),
    ).toBe(true)
    expect(
      shouldAutoReplaceOpenAIWhitelist(['gpt-5.3-codex-spark', 'gpt-5.4-mini', 'gpt-5.4', 'gpt-5.2']),
    ).toBe(true)
    expect(
      shouldAutoReplaceOpenAIWhitelist(['gpt-5.3-codex-spark', 'gpt-5.4-mini', 'gpt-5.4', 'gpt-5.2', 'gpt-5.5']),
    ).toBe(true)
    expect(shouldAutoReplaceOpenAIWhitelist(['gpt-5.4', 'gpt-5.4-mini'])).toBe(false)
    expect(shouldAutoReplaceOpenAIWhitelist(['custom-model'])).toBe(false)
  })

  it('derives image protocol defaults from plan type', () => {
    expect(isOpenAIImageCompatAllowedPlan('free')).toBe(false)
    expect(isOpenAIImageCompatAllowedPlan('plus')).toBe(true)
    expect(isOpenAIImageCompatAllowedPlan('mystery-tier')).toBe(true)
    expect(getDefaultOpenAIImageProtocolMode('free')).toBe('native')
    expect(getDefaultOpenAIImageProtocolMode('team')).toBe('compat')
  })

  it('keeps free oauth accounts on native even if compat was stored', () => {
    expect(
      resolveOpenAIImageProtocolState({
        accountCategory: 'oauth-based',
        planType: 'free',
        storedMode: 'compat',
      }),
    ).toEqual({
      compatAllowed: false,
      mode: 'native',
    })
  })

  it('allows api key accounts to preserve explicit compat mode', () => {
    expect(
      resolveOpenAIImageProtocolState({
        accountCategory: 'apikey',
        storedMode: 'compat',
      }),
    ).toEqual({
      compatAllowed: true,
      mode: 'compat',
    })
  })
})
