import { describe, expect, it } from 'vitest'
import {
  getOpenAIDefaultWhitelist,
  normalizeOpenAIPlanType,
  shouldAutoReplaceOpenAIWhitelist,
} from '../openaiAccountDefaults'

describe('openaiAccountDefaults', () => {
  it('returns the base default whitelist for non-pro plans', () => {
    expect(getOpenAIDefaultWhitelist()).toEqual(['gpt-5.2', 'gpt-5.4', 'gpt-5.4-mini'])
    expect(getOpenAIDefaultWhitelist('plus')).toEqual(['gpt-5.2', 'gpt-5.4', 'gpt-5.4-mini'])
  })

  it('adds spark to the default whitelist for pro plans', () => {
    expect(getOpenAIDefaultWhitelist('chatgpt-pro')).toEqual([
      'gpt-5.2',
      'gpt-5.4',
      'gpt-5.4-mini',
      'gpt-5.3-codex-spark',
    ])
    expect(normalizeOpenAIPlanType('ChatGPT Team')).toBe('team')
  })

  it('only auto-replaces untouched default whitelist selections', () => {
    expect(shouldAutoReplaceOpenAIWhitelist([])).toBe(true)
    expect(shouldAutoReplaceOpenAIWhitelist(['gpt-5.4', 'gpt-5.2', 'gpt-5.4-mini'])).toBe(true)
    expect(
      shouldAutoReplaceOpenAIWhitelist(['gpt-5.3-codex-spark', 'gpt-5.4-mini', 'gpt-5.4', 'gpt-5.2']),
    ).toBe(true)
    expect(shouldAutoReplaceOpenAIWhitelist(['gpt-5.4', 'gpt-5.4-mini'])).toBe(false)
    expect(shouldAutoReplaceOpenAIWhitelist(['custom-model'])).toBe(false)
  })
})
