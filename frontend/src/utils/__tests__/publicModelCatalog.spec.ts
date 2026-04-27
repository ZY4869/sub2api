import { describe, expect, it } from 'vitest'

import { formatCatalogPrice } from '../publicModelCatalog'

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
})
