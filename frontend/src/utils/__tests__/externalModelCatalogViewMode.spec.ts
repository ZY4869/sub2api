import { describe, expect, it } from 'vitest'
import { resolveEffectiveExternalModelCatalogViewMode } from '../externalModelCatalogViewMode'

describe('externalModelCatalogViewMode', () => {
  it('uses explicit group_first', () => {
    expect(resolveEffectiveExternalModelCatalogViewMode({
      external_model_catalog_view_mode: 'group_first',
      api_key_model_binding_mode: 'model_required',
    })).toBe('group_first')
  })

  it('uses explicit model_only', () => {
    expect(resolveEffectiveExternalModelCatalogViewMode({
      external_model_catalog_view_mode: 'model_only',
      api_key_model_binding_mode: 'group_allowed',
    })).toBe('model_only')
  })

  it('follows group_allowed key mode', () => {
    expect(resolveEffectiveExternalModelCatalogViewMode({
      external_model_catalog_view_mode: 'follow_key_binding',
      api_key_model_binding_mode: 'group_allowed',
    })).toBe('group_first')
  })

  it('follows model_required key mode', () => {
    expect(resolveEffectiveExternalModelCatalogViewMode({
      external_model_catalog_view_mode: 'follow_key_binding',
      api_key_model_binding_mode: 'model_required',
    })).toBe('model_only')
  })
})
