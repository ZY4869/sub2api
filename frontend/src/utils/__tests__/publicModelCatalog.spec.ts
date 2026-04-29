import { describe, expect, it } from 'vitest'

import {
  buildPublicModelCatalogDisplayItem,
  formatCatalogPrice,
  normalizePublicModelPriceDisplay,
  resolvePublicModelSubtitle,
  resolvePublicModelTitle,
} from '../publicModelCatalog'

const t = (key: string) => {
  const labels: Record<string, string> = {
    'ui.modelCatalog.units.perMillionTokens': '/ M Tokens',
    'ui.modelCatalog.units.perImage': '/ Image',
    'ui.modelCatalog.units.perRequest': '/ Request',
  }
  return labels[key] || key
}

describe('publicModelCatalog', () => {
  it('keeps CNY source prices unchanged even when an exchange rate is available', () => {
    expect(formatCatalogPrice(
      t,
      { id: 'input_price', value: 3e-7, unit: 'input_token' },
      'CNY',
      7.2,
    )).toBe('¥0.3 / M Tokens')
  })

  it('normalizes equivalent display names and hides duplicate subtitles', () => {
    const item = {
      model: 'gpt-5.4-mini',
      display_name: 'gpt_5_4_mini',
      currency: 'USD',
      price_display: { primary: [] },
      multiplier_summary: { enabled: false, kind: 'disabled' as const },
    }

    expect(resolvePublicModelTitle(item)).toBe('GPT 5.4 Mini')
    expect(resolvePublicModelSubtitle(item)).toBe('')
  })

  it('promotes cache pricing from secondary rows into the primary section', () => {
    const normalized = normalizePublicModelPriceDisplay({
      primary: [{ id: 'input_price', unit: 'input_token', value: 0.000001 }],
      secondary: [
        { id: 'cache_price', unit: 'input_token', value: 0.0000002 },
        { id: 'retrieval', unit: 'request', value: 1 },
      ],
    })

    expect(normalized.primary.map((entry) => entry.id)).toEqual(['input_price', 'cache_price'])
    expect(normalized.secondary?.map((entry) => entry.id)).toEqual(['retrieval'])
  })

  it('builds a searchable display item with normalized prices', () => {
    const displayItem = buildPublicModelCatalogDisplayItem({
      model: 'claude-sonnet-4.5',
      display_name: 'Claude Sonnet 4.5',
      currency: 'USD',
      source_ids: ['claude-source'],
      price_display: {
        primary: [{ id: 'output_price', unit: 'output_token', value: 0.000004 }],
        secondary: [{ id: 'batch_cache_price', unit: 'input_token', value: 0.000001 }],
      },
      multiplier_summary: { enabled: true, kind: 'uniform', value: 1, mode: 'shared' },
      status: 'ok' as const,
    })

    expect(displayItem.primaryPrices.map((entry) => entry.id)).toEqual([
      'output_price',
      'batch_cache_price',
    ])
    expect(displayItem.searchText).toContain('claude-source')
  })
})
