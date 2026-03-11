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
    expect(formatModelCatalogProvider('anthropic')).toBe('Anthropic')
    expect(formatModelCatalogProvider('openai')).toBe('OpenAI')
    expect(formatModelCatalogProvider('custom')).toBe('Custom')
    expect(formatModelCatalogPlatforms(['anthropic', 'gemini'])).toEqual(['Anthropic', 'Gemini'])
  })

  it('persists and restores price display mode', () => {
    expect(getModelCatalogPriceDisplayMode()).toBe('usd')

    setModelCatalogPriceDisplayMode('dual')

    expect(localStorage.getItem(MODEL_CATALOG_PRICE_DISPLAY_MODE_STORAGE_KEY)).toBe('dual')
    expect(getModelCatalogPriceDisplayMode()).toBe('dual')
  })
})
