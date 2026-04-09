import { describe, expect, it } from 'vitest'
import { buildProviderDisplayName, formatProviderLabel, getProviderLabelCatalog } from '../providerLabels'

describe('providerLabels', () => {
  it('reads provider labels from the generated registry snapshot', () => {
    const catalog = getProviderLabelCatalog()

    expect(catalog.openai).toBe('OpenAI-GPT')
    expect(catalog.anthropic).toBe('Anthropic-Claude')
    expect(formatProviderLabel('grok')).toBe('xAI-Grok')
  })

  it('builds final display names with generated provider labels', () => {
    expect(buildProviderDisplayName({
      provider: 'gemini',
      displayName: '2.5 Flash',
      fallbackId: 'gemini-2.5-flash'
    })).toBe('Google-Gemini 2.5 Flash')
  })
})
