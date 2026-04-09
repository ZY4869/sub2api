import { beforeEach, describe, expect, it } from 'vitest'
import {
  formatModelCatalogPlatforms,
  formatModelCatalogProvider,
  getModelCatalogPriceDisplayMode,
  MODEL_CATALOG_PRICE_DISPLAY_MODE_STORAGE_KEY,
  setModelCatalogPriceDisplayMode
} from '../modelCatalogPresentation'

describe('modelCatalogPresentation', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('formats provider names with expected casing', () => {
    expect(formatModelCatalogProvider('anthropic')).toBe('Anthropic-Claude')
    expect(formatModelCatalogProvider('kiro')).toBe('Kiro')
    expect(formatModelCatalogProvider('openai')).toBe('OpenAI-GPT')
    expect(formatModelCatalogProvider('copilot')).toBe('GitHub-Copilot')
    expect(formatModelCatalogProvider('custom')).toBe('Custom')
    expect(formatModelCatalogPlatforms(['anthropic', 'kiro', 'copilot', 'gemini'])).toEqual([
      'Anthropic-Claude',
      'Kiro',
      'GitHub-Copilot',
      'Google-Gemini'
    ])
  })

  it('persists and restores price display mode', () => {
    expect(getModelCatalogPriceDisplayMode()).toBe('usd')

    setModelCatalogPriceDisplayMode('dual')

    expect(localStorage.getItem(MODEL_CATALOG_PRICE_DISPLAY_MODE_STORAGE_KEY)).toBe('dual')
    expect(getModelCatalogPriceDisplayMode()).toBe('dual')
  })
})
