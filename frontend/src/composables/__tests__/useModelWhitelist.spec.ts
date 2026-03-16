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
  it('keeps Claude 4.6 models as independent anthropic selections', () => {
    const models = getModelsByPlatform('anthropic')

    expect(models).toEqual([
      'claude-opus-4.1',
      'claude-opus-4-6',
      'claude-sonnet-4.5',
      'claude-sonnet-4-6',
      'claude-haiku-4.5',
    ])
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

  it('use_key exposure stays curated per platform', () => {
    const openAIModels = getModelsByPlatform('openai', 'use_key')
    const geminiModels = getModelsByPlatform('gemini', 'use_key')

    expect(openAIModels).toContain('gpt-5-codex')
    expect(openAIModels).not.toContain('gpt-5.4')
    expect(geminiModels).toContain('gemini-2.0-flash')
    expect(geminiModels).toContain('gemini-2.5-flash')
    expect(geminiModels).not.toContain('gemini-3.1-flash-image')
  })

  it('test exposure includes runtime test models without leaking use_key-only curation', () => {
    const openAIModels = getModelsByPlatform('openai', 'test')

    expect(openAIModels).toContain('gpt-5.4')
    expect(openAIModels).not.toContain('gpt-5-codex')
  })

  it('keeps antigravity presets available without hard-filtering 4.6 ids', () => {
    const presets = getPresetMappingsByPlatform('antigravity')

    expect(presets.some((preset) => preset.to === 'claude-sonnet-4.5')).toBe(true)
    expect(presets.some((preset) => preset.to === 'claude-opus-4.1')).toBe(true)
    expect(presets.some((preset) => preset.from === 'gemini-2.5-flash-image' && preset.to === 'gemini-2.5-flash-image')).toBe(true)
    expect(presets.some((preset) => preset.from === 'gemini-3.1-flash-image' && preset.to === 'gemini-3.1-flash-image')).toBe(true)
    expect(presets.some((preset) => preset.from === 'gemini-3-pro-image' && preset.to === 'gemini-3.1-flash-image')).toBe(true)
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
