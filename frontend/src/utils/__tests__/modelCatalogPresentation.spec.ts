import { describe, expect, it } from 'vitest'
import {
  buildModelCatalogTierDescription,
  MODEL_CATALOG_DEFAULT_THRESHOLD,
  resolveModelCatalogDisplayName,
  resolveModelCatalogIcon
} from '@/utils/modelCatalogPresentation'

describe('modelCatalogPresentation utils', () => {
  it('returns display name fallback and icon urls', () => {
    expect(resolveModelCatalogDisplayName('gpt-4o-mini', 'GPT-4o-mini')).toBe('GPT-4o-mini')
    expect(resolveModelCatalogDisplayName('gpt-4o-mini')).toBe('gpt-4o-mini')
    expect(resolveModelCatalogIcon('claude')).toContain('claude')
    expect(resolveModelCatalogIcon('chatgpt')).toContain('chatgpt')
    expect(resolveModelCatalogIcon('gemini')).toContain('gemini')
  })

  it('builds tier description from the default threshold', () => {
    expect(buildModelCatalogTierDescription(MODEL_CATALOG_DEFAULT_THRESHOLD)).toEqual({
      low: '<= 200,000',
      high: '>= 200,001'
    })
  })
})
