import { describe, expect, it } from 'vitest'
import {
  buildLobeIconSources,
  resolveLobeBadgeText,
  resolveModelIconSlugs,
  resolveProviderIconSlugs
} from '../lobeIconResolver'

describe('lobeIconResolver', () => {
  it('prefers company icons for provider slugs', () => {
    expect(resolveProviderIconSlugs('doubao')).toEqual(['bytedance'])
    expect(resolveProviderIconSlugs('gemini')).toEqual(['google'])
  })

  it('prefers model family first and then provider fallback', () => {
    expect(resolveModelIconSlugs({
      model: 'doubao-seed-1.6',
      provider: 'doubao'
    })).toEqual(['doubao', 'bytedance'])

    expect(resolveModelIconSlugs({
      model: 'claude-3-7-sonnet',
      provider: 'anthropic'
    })).toEqual(['claude', 'anthropic'])
  })

  it('builds color first then mono fallback sources', () => {
    expect(buildLobeIconSources(['qwen'])).toEqual([
      '/lobehub-icons-static-svg/icons/qwen-color.svg',
      '/lobehub-icons-static-svg/icons/qwen.svg'
    ])
  })

  it('returns a compact badge label when no icon matches', () => {
    expect(resolveLobeBadgeText('gpt-4.1')).toBe('GP')
    expect(resolveLobeBadgeText('', 'Claude')).toBe('CL')
  })
})
