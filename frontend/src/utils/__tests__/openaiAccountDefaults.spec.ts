import { describe, expect, it } from 'vitest'
import {
  getDefaultOpenAIImageProtocolMode,
  isOpenAIImageCompatAllowedPlan,
  normalizeOpenAIPlanType,
  resolveOpenAIImageProtocolState,
} from '../openaiAccountDefaults'

describe('openaiAccountDefaults', () => {
  it('normalizes OpenAI plan aliases for image protocol decisions', () => {
    expect(normalizeOpenAIPlanType('ChatGPT Team')).toBe('team')
    expect(normalizeOpenAIPlanType('chatgpt-pro')).toBe('pro')
    expect(normalizeOpenAIPlanType('')).toBe('')
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
