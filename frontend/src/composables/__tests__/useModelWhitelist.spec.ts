import { describe, expect, it, vi } from 'vitest'

vi.mock('@/api/admin/accounts', () => ({
  getAntigravityDefaultModelMapping: vi.fn()
}))

import {
  buildModelMappingObject,
  getModelsByPlatform,
  getPresetMappingsByPlatform
} from '../useModelWhitelist'

describe('useModelWhitelist', () => {
  it('returns only current Claude aliases for anthropic', () => {
    const models = getModelsByPlatform('anthropic')

    expect(models).toEqual(['claude-opus-4.1', 'claude-sonnet-4.5', 'claude-haiku-4.5'])
    expect(models).not.toContain('claude-opus-4-6')
    expect(models).not.toContain('claude-sonnet-4-6')
  })

  it('openai models include GPT-5.4 official snapshot', () => {
    const models = getModelsByPlatform('openai')

    expect(models).toContain('gpt-5.4')
    expect(models).toContain('gpt-5.4-2026-03-05')
  })

  it('gemini models include prioritized native image models', () => {
    const models = getModelsByPlatform('gemini')

    expect(models).toContain('gemini-2.5-flash-image')
    expect(models).toContain('gemini-3.1-flash-image')
    expect(models.indexOf('gemini-3.1-flash-image')).toBeLessThan(models.indexOf('gemini-2.0-flash'))
    expect(models.indexOf('gemini-2.5-flash-image')).toBeLessThan(models.indexOf('gemini-2.5-flash'))
  })

  it('antigravity models include prioritized image compatibility entries', () => {
    const models = getModelsByPlatform('antigravity')

    expect(models).toContain('gemini-2.5-flash-image')
    expect(models).toContain('gemini-3.1-flash-image')
    expect(models).toContain('gemini-3-pro-image')
    expect(models.indexOf('gemini-3.1-flash-image')).toBeLessThan(models.indexOf('gemini-2.5-flash'))
    expect(models.indexOf('gemini-2.5-flash-image')).toBeLessThan(models.indexOf('gemini-2.5-flash-lite'))
  })

  it('drops legacy 4.6 entries from antigravity presets while keeping image passthroughs', () => {
    const presets = getPresetMappingsByPlatform('antigravity')

    expect(presets.some((preset) => preset.to === 'claude-sonnet-4.5')).toBe(true)
    expect(presets.some((preset) => preset.to === 'claude-opus-4.1')).toBe(true)
    expect(presets.some((preset) => preset.from === 'gemini-2.5-flash-image' && preset.to === 'gemini-2.5-flash-image')).toBe(true)
    expect(presets.some((preset) => preset.from === 'gemini-3.1-flash-image' && preset.to === 'gemini-3.1-flash-image')).toBe(true)
    expect(presets.some((preset) => preset.from === 'gemini-3-pro-image' && preset.to === 'gemini-3.1-flash-image')).toBe(true)
    expect(presets.some((preset) => preset.from.includes('4-6') || preset.to.includes('4-6'))).toBe(false)
  })

  it('ignores wildcard entries in whitelist mode', () => {
    const mapping = buildModelMappingObject('whitelist', ['claude-*', 'gemini-3.1-flash-image'], [])

    expect(mapping).toEqual({
      'gemini-3.1-flash-image': 'gemini-3.1-flash-image'
    })
  })

  it('keeps GPT-5.4 official snapshot as exact whitelist mapping', () => {
    const mapping = buildModelMappingObject('whitelist', ['gpt-5.4-2026-03-05'], [])

    expect(mapping).toEqual({
      'gpt-5.4-2026-03-05': 'gpt-5.4-2026-03-05'
    })
  })
})
