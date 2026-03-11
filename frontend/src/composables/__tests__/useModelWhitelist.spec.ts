import { describe, expect, it } from 'vitest'
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

  it('drops legacy 4.6 entries from antigravity presets', () => {
    const presets = getPresetMappingsByPlatform('antigravity')

    expect(presets.some((preset) => preset.to === 'claude-sonnet-4.5')).toBe(true)
    expect(presets.some((preset) => preset.to === 'claude-opus-4.1')).toBe(true)
    expect(presets.some((preset) => preset.from.includes('4-6') || preset.to.includes('4-6'))).toBe(false)
  })

  it('ignores wildcard entries in whitelist mode', () => {
    const mapping = buildModelMappingObject('whitelist', ['claude-*', 'gemini-3.1-flash-image'], [])

    expect(mapping).toEqual({
      'gemini-3.1-flash-image': 'gemini-3.1-flash-image'
    })
  })
})