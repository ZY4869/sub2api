import { describe, expect, it, vi } from 'vitest'

const translations: Record<string, string> = {
  'usage.protocolFamilies.openai': 'OpenAI',
  'usage.protocolFamilies.anthropic': 'Anthropic',
  'usage.protocolFamilies.gemini': 'Gemini',
  'usage.protocolFamilies.unknown': 'Unknown',
  'usage.protocolModes.native': 'Native',
  'usage.protocolModes.compatible': 'Compatible',
  'usage.protocolTransition': 'Inbound -> Upstream: {transition}'
}

vi.mock('@/i18n', () => ({
  i18n: {
    global: {
      te: (key: string) => key in translations,
      t: (key: string) => translations[key] ?? key
    }
  }
}))

import {
  formatUsageProtocolExportText,
  getProtocolBadgeMeta,
  resolveProtocolPairDisplay,
  resolveUsageProtocolDisplay
} from '../protocolDisplay'

describe('protocolDisplay', () => {
  it('resolves native OpenAI chat completions display', () => {
    const display = resolveUsageProtocolDisplay('/v1/chat/completions', '/v1/chat/completions')

    expect(display).not.toBeNull()
    expect(display?.badge.family).toBe('openai')
    expect(display?.badge.label).toBe('OpenAI')
    expect(display?.requestPath).toBe('/v1/chat/completions')
    expect(display?.mode).toBe('native')
    expect(display?.modeLabel).toBe('Native')
    expect(display?.tooltip).toBeUndefined()
  })

  it('resolves Anthropic native requests', () => {
    const display = resolveUsageProtocolDisplay('/v1/messages', '/v1/messages')

    expect(display?.badge.family).toBe('anthropic')
    expect(display?.badge.label).toBe('Anthropic')
    expect(display?.mode).toBe('native')
  })

  it('marks Gemini OpenAI compat paths as compatible', () => {
    const display = resolveUsageProtocolDisplay('/v1beta/openai/chat/completions', '/v1beta/models')

    expect(display?.badge.family).toBe('gemini')
    expect(display?.mode).toBe('compatible')
    expect(display?.modeLabel).toBe('Compatible')
  })

  it('builds transition tooltip when inbound and upstream protocols differ', () => {
    const display = resolveUsageProtocolDisplay('/v1/messages', '/v1/responses')

    expect(display?.mode).toBe('compatible')
    expect(display?.tooltip).toBe('Inbound -> Upstream: Anthropic /v1/messages -> OpenAI /v1/responses')
  })

  it('falls back to unknown family for unrecognized paths', () => {
    const badge = getProtocolBadgeMeta('/custom/endpoint')

    expect(badge.family).toBe('unknown')
    expect(badge.label).toBe('Unknown')
  })

  it('formats protocol pair labels and export text', () => {
    const pair = resolveProtocolPairDisplay('openai', 'anthropic')

    expect(pair.inboundBadge.family).toBe('openai')
    expect(pair.outboundBadge.family).toBe('anthropic')
    expect(pair.label).toBe('OpenAI -> Anthropic')
    expect(pair.title).toBe('openai -> anthropic')
    expect(pair.detailLabel).toBeUndefined()
    expect(formatUsageProtocolExportText('/v1/responses', '/v1/responses')).toBe('OpenAI /v1/responses Native')
  })

  it('keeps raw endpoint pair details when protocols are request format paths', () => {
    const pair = resolveProtocolPairDisplay('/v1/chat/completions', '/v1/responses')

    expect(pair.inboundBadge.family).toBe('openai')
    expect(pair.outboundBadge.family).toBe('openai')
    expect(pair.label).toBe('OpenAI -> OpenAI')
    expect(pair.detailLabel).toBe('/v1/chat/completions -> /v1/responses')
  })
})
